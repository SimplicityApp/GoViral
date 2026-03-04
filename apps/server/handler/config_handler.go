package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/internal/config"
)

// ConfigHandler handles requests for application config.
type ConfigHandler struct {
	cfg *config.Config
}

// NewConfigHandler creates a new ConfigHandler.
func NewConfigHandler(cfg *config.Config) *ConfigHandler {
	return &ConfigHandler{cfg: cfg}
}

// Get returns the current config with secrets masked.
func (h *ConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	resp := dto.ConfigResponse{
		Claude: dto.ConfigClaudeResponse{
			APIKey: maskSecret(h.cfg.Claude.APIKey),
			Model:  h.cfg.Claude.Model,
		},
		Gemini: dto.ConfigGeminiResponse{
			APIKey: maskSecret(h.cfg.Gemini.APIKey),
			Model:  h.cfg.Gemini.Model,
		},
		X: dto.ConfigXResponse{
			APIKey:        maskSecret(h.cfg.X.APIKey),
			APISecret:     maskSecret(h.cfg.X.APISecret),
			BearerToken:   maskSecret(h.cfg.X.BearerToken),
			ClientID:      maskSecret(h.cfg.X.ClientID),
			ClientSecret:  maskSecret(h.cfg.X.ClientSecret),
			Username:      h.cfg.X.Username,
			HasAuth:       h.cfg.X.AccessToken != "",
			HasTwikitAuth: twikitCookiesExist(),
			AuthToken:     maskedTwikitCookie("auth_token"),
			Ct0:           maskedTwikitCookie("ct0"),
		},
		LinkedIn: dto.ConfigLinkedInResponse{
			ClientID:        maskSecret(h.cfg.LinkedIn.ClientID),
			ClientSecret:    maskSecret(h.cfg.LinkedIn.ClientSecret),
			HasAuth:         h.cfg.LinkedIn.AccessToken != "",
			HasLinkitinAuth: linkitinCookiesExist(),
			LiAt:            maskedLinkitinCookie("li_at"),
			JSessionID:      maskedLinkitinCookie("JSESSIONID"),
		},
		GitHub: dto.ConfigGitHubResponse{
			PersonalAccessToken: maskSecret(h.cfg.GitHub.PersonalAccessToken),
			DefaultOwner:        h.cfg.GitHub.DefaultOwner,
			DefaultRepo:         h.cfg.GitHub.DefaultRepo,
		},
		YouTube: dto.ConfigYouTubeResponse{
			ClientID:     maskSecret(h.cfg.YouTube.ClientID),
			ClientSecret: maskSecret(h.cfg.YouTube.ClientSecret),
			ChannelID:    h.cfg.YouTube.ChannelID,
			HasAuth:      h.cfg.YouTube.AccessToken != "",
		},
		TikTok: dto.ConfigTikTokResponse{
			ClientKey:    maskSecret(h.cfg.TikTok.ClientKey),
			ClientSecret: maskSecret(h.cfg.TikTok.ClientSecret),
			Username:     h.cfg.TikTok.Username,
			HasAuth:      h.cfg.TikTok.AccessToken != "",
		},
		Niches:         h.cfg.Niches,
		LinkedInNiches: h.cfg.LinkedInNiches,
	}
	if resp.Niches == nil {
		resp.Niches = []string{}
	}
	if resp.LinkedInNiches == nil {
		resp.LinkedInNiches = []string{}
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

func twikitCookiesExist() bool {
	cookiePath := filepath.Join(config.DefaultConfigDir(), "twikit_cookies.json")
	_, err := os.Stat(cookiePath)
	return err == nil
}

func linkitinCookiesExist() bool {
	cookiePath := filepath.Join(config.DefaultConfigDir(), "linkitin_cookies.json")
	_, err := os.Stat(cookiePath)
	return err == nil
}

func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return s[:2] + "****"
	}
	return s[:4] + "****" + s[len(s)-2:]
}

func readCookieFile(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	result := make(map[string]string, len(raw))
	for k, v := range raw {
		var s string
		if json.Unmarshal(v, &s) == nil {
			result[k] = s
		}
	}
	return result
}

func maskedTwikitCookie(key string) string {
	cookies := readCookieFile(filepath.Join(config.DefaultConfigDir(), "twikit_cookies.json"))
	return maskSecret(cookies[key])
}

func maskedLinkitinCookie(key string) string {
	cookies := readCookieFile(filepath.Join(config.DefaultConfigDir(), "linkitin_cookies.json"))
	return maskSecret(cookies[key])
}
