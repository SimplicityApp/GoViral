package auth

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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
	xAuthURL  = "https://x.com/i/oauth2/authorize"
	xTokenURL = "https://api.x.com/2/oauth2/token"
	xScopes   = "tweet.read tweet.write users.read offline.access"
)

// XAuth runs the X OAuth 2.0 authorization code flow with PKCE.
func XAuth(cfg *config.Config, port int) error {
	reader := bufio.NewReader(os.Stdin)

	if cfg.X.ClientID != "" {
		fmt.Printf("Using X Client ID from config: %s\n", maskValue(cfg.X.ClientID))
	} else {
		fmt.Print("X Client ID: ")
		cfg.X.ClientID = readLine(reader)
	}

	if cfg.X.ClientSecret != "" {
		fmt.Printf("Using X Client Secret from config: %s\n", maskValue(cfg.X.ClientSecret))
	} else {
		fmt.Print("X Client Secret: ")
		cfg.X.ClientSecret = readLine(reader)
	}

	if cfg.X.ClientID == "" {
		return fmt.Errorf("X client_id is required")
	}

	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return fmt.Errorf("generating PKCE code verifier: %w", err)
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	state, err := generateRandomString(32)
	if err != nil {
		return fmt.Errorf("generating state parameter: %w", err)
	}

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

	authURL := fmt.Sprintf("%s?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&code_challenge=%s&code_challenge_method=S256&state=%s",
		xAuthURL,
		url.QueryEscape(cfg.X.ClientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(xScopes),
		url.QueryEscape(codeChallenge),
		url.QueryEscape(state),
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
	tokenResp, err := exchangeXCode(code, cfg.X.ClientID, cfg.X.ClientSecret, codeVerifier, redirectURI)
	if err != nil {
		return fmt.Errorf("exchanging authorization code: %w", err)
	}

	cfg.X.AccessToken = tokenResp.AccessToken
	cfg.X.RefreshToken = tokenResp.RefreshToken
	if tokenResp.ExpiresIn > 0 {
		cfg.X.TokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Format(time.RFC3339)
	}

	if err := config.Save(cfg, config.DefaultConfigPath()); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	return nil
}

// xTokenResponse holds the fields returned by the X token endpoint.
type xTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func exchangeXCode(code, clientID, clientSecret, codeVerifier, redirectURI string) (*xTokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequest("POST", xTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("posting token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp xTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("no access_token in response: %s", string(body))
	}

	return &tokenResp, nil
}

// RefreshXToken uses a refresh token to obtain a new access token.
// It updates the config in place and saves it to disk.
func RefreshXToken(cfg *config.Config) error {
	if cfg.X.RefreshToken == "" {
		return fmt.Errorf("no refresh token available; run 'goviral auth x' to re-authenticate")
	}
	if cfg.X.ClientID == "" {
		return fmt.Errorf("no client_id configured; run 'goviral auth x' to re-authenticate")
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {cfg.X.RefreshToken},
	}

	req, err := http.NewRequest("POST", xTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("creating refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(cfg.X.ClientID, cfg.X.ClientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("posting refresh request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp xTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("parsing refresh response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return fmt.Errorf("no access_token in refresh response")
	}

	cfg.X.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		cfg.X.RefreshToken = tokenResp.RefreshToken
	}
	if tokenResp.ExpiresIn > 0 {
		cfg.X.TokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Format(time.RFC3339)
	}

	if err := config.Save(cfg, config.DefaultConfigPath()); err != nil {
		return fmt.Errorf("saving refreshed config: %w", err)
	}

	return nil
}

// generateCodeVerifier creates a cryptographically random PKCE code verifier.
func generateCodeVerifier() (string, error) {
	return generateRandomString(64)
}

// generateCodeChallenge derives a S256 code challenge from a code verifier.
func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// generateRandomString creates a cryptographically random base64url string.
func generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b)[:length], nil
}
