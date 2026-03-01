package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/shuhao/goviral/internal/config"
)

var tiktokLoginCookies bool

var tiktokLoginCmd = &cobra.Command{
	Use:   "tiktok-login",
	Short: "Authenticate with TikTok via OAuth 2.0 or cookies",
	Long: `Authenticate with TikTok to enable video uploads.

Methods:
  goviral tiktok-login            # OAuth 2.0 flow (opens browser)
  goviral tiktok-login --cookies  # Extract cookies from Chrome

OAuth 2.0 prerequisites:
  - Set tiktok.client_key and tiktok.client_secret in config
  - Register http://localhost:8990/callback as a redirect URI in TikTok Developer Portal

Cookie method:
  - Log into TikTok in Chrome first
  - Cookies are extracted and saved to ~/.goviral/tiktok_cookies.json`,
	RunE: runTikTokLogin,
}

func init() {
	tiktokLoginCmd.Flags().BoolVar(&tiktokLoginCookies, "cookies", false, "Extract TikTok cookies from Chrome instead of OAuth")
	rootCmd.AddCommand(tiktokLoginCmd)
}

func runTikTokLogin(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if tiktokLoginCookies {
		return runTikTokCookieLogin()
	}

	return runTikTokOAuthLogin(cfg)
}

func runTikTokOAuthLogin(cfg *config.Config) error {
	if cfg.TikTok.ClientKey == "" || cfg.TikTok.ClientSecret == "" {
		return fmt.Errorf("tiktok.client_key and tiktok.client_secret must be set in config")
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))

	fmt.Println(headerStyle.Render("TikTok OAuth 2.0 Login"))
	fmt.Println()

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

	server := &http.Server{Addr: ":8990", Handler: mux}
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server error: %w", err)
		}
	}()
	defer server.Shutdown(context.Background()) //nolint:errcheck

	scopes := "user.info.basic,video.upload,video.publish"
	authURL := fmt.Sprintf(
		"https://www.tiktok.com/v2/auth/authorize/?client_key=%s&redirect_uri=http://localhost:8990/callback&response_type=code&scope=%s",
		cfg.TikTok.ClientKey, scopes,
	)

	fmt.Println(infoStyle.Render("Opening browser for authentication..."))
	fmt.Printf("If the browser doesn't open, visit:\n%s\n\n", authURL)
	openBrowser(authURL)

	fmt.Println("Waiting for authorization...")

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("timeout waiting for authorization (5 minutes)")
	}

	fmt.Println(infoStyle.Render("Authorization code received, exchanging for tokens..."))

	tokenResp, err := exchangeTikTokCode(cfg.TikTok.ClientKey, cfg.TikTok.ClientSecret, code)
	if err != nil {
		return fmt.Errorf("exchanging authorization code: %w", err)
	}

	cfg.TikTok.AccessToken = tokenResp.AccessToken
	cfg.TikTok.RefreshToken = tokenResp.RefreshToken
	if tokenResp.ExpiresIn > 0 {
		expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		cfg.TikTok.TokenExpiry = expiry.Format(time.RFC3339)
	}

	if err := config.Save(cfg, config.DefaultConfigPath()); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println(successStyle.Render("\nTikTok OAuth authentication successful!"))
	fmt.Println("You can now upload videos with: goviral post --video <path>")

	return nil
}

func runTikTokCookieLogin() error {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))

	fmt.Println(headerStyle.Render("TikTok Cookie Login"))
	fmt.Println()
	fmt.Println(infoStyle.Render("Make sure you are logged into TikTok in Chrome."))
	fmt.Println("This will extract session cookies for the tiktok-uploader fallback client.")
	fmt.Println()

	// The tiktok-uploader library handles cookie extraction via Playwright.
	// We just need to ensure the cookie file exists.
	fmt.Println(infoStyle.Render("To use cookie-based auth, manually export your TikTok cookies"))
	fmt.Println("to ~/.goviral/tiktok_cookies.json in Netscape cookie format.")
	fmt.Println()
	fmt.Println("You can use a browser extension like 'Get cookies.txt' to export them.")
	fmt.Println()

	cookiePath := filepath.Join(config.DefaultConfigDir(), "tiktok_cookies.json")
	fmt.Printf("Cookie file location: %s\n", cookiePath)
	fmt.Println()

	fmt.Println(successStyle.Render("Once cookies are saved, you can upload videos with: goviral post --video <path>"))

	return nil
}

type tiktokTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	OpenID       string `json:"open_id"`
}

func exchangeTikTokCode(clientKey, clientSecret, code string) (*tiktokTokenResponse, error) {
	resp, err := http.PostForm("https://open.tiktokapis.com/v2/oauth/token/", map[string][]string{
		"client_key":    {clientKey},
		"client_secret": {clientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {"http://localhost:8990/callback"},
	})
	if err != nil {
		return nil, fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	var wrapper struct {
		Data tiktokTokenResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	if wrapper.Data.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	return &wrapper.Data, nil
}
