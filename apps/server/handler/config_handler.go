package handler

import (
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
			APIKey:       maskSecret(h.cfg.X.APIKey),
			APISecret:    maskSecret(h.cfg.X.APISecret),
			BearerToken:  maskSecret(h.cfg.X.BearerToken),
			ClientID:     maskSecret(h.cfg.X.ClientID),
			ClientSecret: maskSecret(h.cfg.X.ClientSecret),
			Username:     h.cfg.X.Username,
			HasAuth:      h.cfg.X.AccessToken != "",
		},
		LinkedIn: dto.ConfigLinkedInResponse{
			ClientID:     maskSecret(h.cfg.LinkedIn.ClientID),
			ClientSecret: maskSecret(h.cfg.LinkedIn.ClientSecret),
			HasAuth:      h.cfg.LinkedIn.AccessToken != "",
			HasLikitAuth: likitCookiesExist(),
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

func likitCookiesExist() bool {
	cookiePath := filepath.Join(config.DefaultConfigDir(), "likit_cookies.json")
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
