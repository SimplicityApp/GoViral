package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const githubTokenURL = "https://github.com/login/oauth/access_token"

// ExchangeGitHubCode exchanges an authorization code for a GitHub access token.
func ExchangeGitHubCode(code, clientID, clientSecret, redirectURI string) (string, error) {
	data := url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
	}

	req, err := http.NewRequest("POST", githubTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
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
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parsing token response: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("GitHub OAuth error: %s — %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access_token in response: %s", string(body))
	}

	return tokenResp.AccessToken, nil
}
