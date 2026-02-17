package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/shuhao/goviral/internal/config"
)

const (
	linkedinAuthURL  = "https://www.linkedin.com/oauth/v2/authorization"
	linkedinTokenURL = "https://www.linkedin.com/oauth/v2/accessToken"
	linkedinScopes   = "openid profile w_member_social"
)

// LinkedInAuth runs the LinkedIn OAuth 2.0 authorization code flow.
func LinkedInAuth(cfg *config.Config, port int) error {
	reader := bufio.NewReader(os.Stdin)

	if cfg.LinkedIn.ClientID != "" {
		fmt.Printf("Using LinkedIn Client ID from config: %s\n", maskValue(cfg.LinkedIn.ClientID))
	} else {
		fmt.Print("LinkedIn Client ID: ")
		cfg.LinkedIn.ClientID = readLine(reader)
	}

	if cfg.LinkedIn.ClientSecret != "" {
		fmt.Printf("Using LinkedIn Client Secret from config: %s\n", maskValue(cfg.LinkedIn.ClientSecret))
	} else {
		fmt.Print("LinkedIn Client Secret: ")
		cfg.LinkedIn.ClientSecret = readLine(reader)
	}

	if cfg.LinkedIn.ClientID == "" || cfg.LinkedIn.ClientSecret == "" {
		return fmt.Errorf("LinkedIn client_id and client_secret are required")
	}

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

	authURL := fmt.Sprintf("%s?response_type=code&client_id=%s&redirect_uri=%s&scope=%s",
		linkedinAuthURL,
		url.QueryEscape(cfg.LinkedIn.ClientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(linkedinScopes),
	)

	fmt.Println("\nOpen this URL in your browser to authorize:")
	fmt.Println(authURL)
	fmt.Println()

	if err := OpenBrowser(authURL); err != nil {
		fmt.Println("Could not open browser automatically. Please open the URL above manually.")
	}

	fmt.Println("Waiting for authorization callback...")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	code, err := StartCallbackServer(ctx, port)
	if err != nil {
		return fmt.Errorf("receiving authorization code: %w", err)
	}

	// Exchange code for access token.
	token, err := exchangeLinkedInCode(code, cfg.LinkedIn.ClientID, cfg.LinkedIn.ClientSecret, redirectURI)
	if err != nil {
		return fmt.Errorf("exchanging authorization code: %w", err)
	}

	cfg.LinkedIn.AccessToken = token

	if err := config.Save(cfg, config.DefaultConfigPath()); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	return nil
}

func exchangeLinkedInCode(code, clientID, clientSecret, redirectURI string) (string, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}

	resp, err := http.Post(linkedinTokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("posting token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parsing token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access_token in response: %s", string(body))
	}

	return tokenResp.AccessToken, nil
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// maskValue shows the first 4 and last 2 characters, masking the rest.
func maskValue(s string) string {
	if len(s) <= 8 {
		return s[:2] + "****"
	}
	return s[:4] + "****" + s[len(s)-2:]
}
