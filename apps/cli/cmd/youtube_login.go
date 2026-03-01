package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/shuhao/goviral/internal/config"
)

var youtubeLoginCmd = &cobra.Command{
	Use:   "youtube-login",
	Short: "Authenticate with YouTube via OAuth 2.0",
	Long: `Authenticate with YouTube to enable video uploads to YouTube Shorts.

This command starts an OAuth 2.0 flow:
1. Opens your browser to Google's consent screen
2. After you approve, captures the authorization code
3. Exchanges it for access and refresh tokens
4. Saves tokens to ~/.goviral/config.yaml

Prerequisites:
  - Set youtube.client_id and youtube.client_secret in config
  - Add http://localhost:8989/callback as an authorized redirect URI in Google Cloud Console`,
	RunE: runYouTubeLogin,
}

func init() {
	rootCmd.AddCommand(youtubeLoginCmd)
}

func runYouTubeLogin(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.YouTube.ClientID == "" || cfg.YouTube.ClientSecret == "" {
		return fmt.Errorf("youtube.client_id and youtube.client_secret must be set in config")
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))

	fmt.Println(headerStyle.Render("YouTube OAuth 2.0 Login"))
	fmt.Println()

	// Start local server to receive callback.
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no authorization code received")
			fmt.Fprintf(w, "<html><body><h1>Error</h1><p>No authorization code received.</p></body></html>")
			return
		}
		codeCh <- code
		fmt.Fprintf(w, "<html><body><h1>Success!</h1><p>You can close this window and return to the terminal.</p></body></html>")
	})

	server := &http.Server{Addr: ":8989", Handler: mux}
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server error: %w", err)
		}
	}()
	defer server.Shutdown(context.Background()) //nolint:errcheck

	// Build OAuth URL.
	scopes := "https://www.googleapis.com/auth/youtube.upload https://www.googleapis.com/auth/youtube"
	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=http://localhost:8989/callback&response_type=code&scope=%s&access_type=offline&prompt=consent",
		cfg.YouTube.ClientID, scopes,
	)

	fmt.Println(infoStyle.Render("Opening browser for authentication..."))
	fmt.Printf("If the browser doesn't open, visit:\n%s\n\n", authURL)
	openBrowser(authURL)

	fmt.Println("Waiting for authorization...")

	// Wait for callback.
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("timeout waiting for authorization (5 minutes)")
	}

	fmt.Println(infoStyle.Render("Authorization code received, exchanging for tokens..."))

	// Exchange code for tokens.
	tokenResp, err := exchangeYouTubeCode(cfg.YouTube.ClientID, cfg.YouTube.ClientSecret, code)
	if err != nil {
		return fmt.Errorf("exchanging authorization code: %w", err)
	}

	// Save tokens to config.
	cfg.YouTube.AccessToken = tokenResp.AccessToken
	cfg.YouTube.RefreshToken = tokenResp.RefreshToken
	if tokenResp.ExpiresIn > 0 {
		expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		cfg.YouTube.TokenExpiry = expiry.Format(time.RFC3339)
	}

	if err := config.Save(cfg, config.DefaultConfigPath()); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Also save to youtube_token.json for any Python bridge consumers.
	tokenFile := filepath.Join(config.DefaultConfigDir(), "youtube_token.json")
	tokenData := map[string]string{
		"access_token":  tokenResp.AccessToken,
		"refresh_token": tokenResp.RefreshToken,
		"client_id":     cfg.YouTube.ClientID,
		"client_secret": cfg.YouTube.ClientSecret,
	}
	tokenJSON, _ := json.MarshalIndent(tokenData, "", "  ")
	os.WriteFile(tokenFile, tokenJSON, 0600) //nolint:errcheck

	fmt.Println(successStyle.Render("\nYouTube authentication successful!"))
	fmt.Println("You can now upload videos with: goviral post --video <path>")

	return nil
}

type youtubeTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func exchangeYouTubeCode(clientID, clientSecret, code string) (*youtubeTokenResponse, error) {
	resp, err := http.PostForm("https://oauth2.googleapis.com/token", map[string][]string{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {"http://localhost:8989/callback"},
	})
	if err != nil {
		return nil, fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp youtubeTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	return &tokenResp, nil
}

// openBrowser opens the given URL in the default system browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		cmd = exec.Command("open", url)
	}
	cmd.Start() //nolint:errcheck
}
