package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
)

// ConfigHandler handles requests for application config.
type ConfigHandler struct {
	cfg *config.Config
	db  *db.DB
}

// NewConfigHandler creates a new ConfigHandler.
func NewConfigHandler(cfg *config.Config, database *db.DB) *ConfigHandler {
	return &ConfigHandler{cfg: cfg, db: database}
}

// Get returns the current config with secrets masked, merged with per-user overrides.
func (h *ConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	uc, _ := h.db.GetUserConfig(userID)
	if uc == nil {
		uc = &config.UserConfig{}
	}

	claudeCfg := uc.ResolvedClaudeConfig(*h.cfg)
	claudeUsed, _ := h.db.GetAIUsage(userID, "claude")

	geminiCfg := uc.ResolvedGeminiConfig(*h.cfg)
	geminiUsed, _ := h.db.GetAIUsage(userID, "gemini")

	xUsername := h.cfg.X.Username
	if uc.XUsername != "" {
		xUsername = uc.XUsername
	}

	channelID := h.cfg.YouTube.ChannelID
	if uc.YouTubeChannelID != "" {
		channelID = uc.YouTubeChannelID
	}

	tiktokUsername := h.cfg.TikTok.Username
	if uc.TikTokUsername != "" {
		tiktokUsername = uc.TikTokUsername
	}

	hasTwikitAuth := uc.TwikitCookiesJSON != "" || twikitCookiesExist()
	hasLinkitinAuth := uc.LinkitinCookiesJSON != "" || linkitinCookiesExist()

	var authToken, ct0 string
	if uc.TwikitCookiesJSON != "" {
		authToken = maskedCookieFromJSON(uc.TwikitCookiesJSON, "auth_token")
		ct0 = maskedCookieFromJSON(uc.TwikitCookiesJSON, "ct0")
	} else {
		authToken = maskedTwikitCookie("auth_token")
		ct0 = maskedTwikitCookie("ct0")
	}

	var liAt, jsessionID string
	if uc.LinkitinCookiesJSON != "" {
		liAt = maskedCookieFromJSON(uc.LinkitinCookiesJSON, "li_at")
		jsessionID = maskedCookieFromJSON(uc.LinkitinCookiesJSON, "JSESSIONID")
	} else {
		liAt = maskedLinkitinCookie("li_at")
		jsessionID = maskedLinkitinCookie("JSESSIONID")
	}

	niches := uc.MergedNiches(*h.cfg)
	linkedInNiches := uc.MergedLinkedInNiches(*h.cfg)
	if niches == nil {
		niches = []string{}
	}
	if linkedInNiches == nil {
		linkedInNiches = []string{}
	}

	resp := dto.ConfigResponse{
		Claude: dto.ConfigClaudeResponse{
			HasGlobalKey: h.cfg.Claude.APIKey != "",
			UserAPIKey:   maskSecret(uc.ClaudeAPIKey),
			Model:        claudeCfg.Model,
			DailyLimit:   h.cfg.Claude.DailyLimit,
			DailyUsed:    claudeUsed,
		},
		Gemini: dto.ConfigGeminiResponse{
			HasGlobalKey: h.cfg.Gemini.APIKey != "",
			UserAPIKey:   maskSecret(uc.GeminiAPIKey),
			Model:        geminiCfg.Model,
			DailyLimit:   h.cfg.Gemini.DailyLimit,
			DailyUsed:    geminiUsed,
		},
		X: dto.ConfigXResponse{
			HasAPIKey:       h.cfg.X.APIKey != "",
			HasAPISecret:    h.cfg.X.APISecret != "",
			HasBearerToken:  h.cfg.X.BearerToken != "",
			HasClientID:     h.cfg.X.ClientID != "",
			HasClientSecret: h.cfg.X.ClientSecret != "",
			Username:        xUsername,
			HasAuth:         uc.XAccessToken != "",
			HasTwikitAuth:   hasTwikitAuth,
			AuthToken:       authToken,
			Ct0:             ct0,
		},
		LinkedIn: dto.ConfigLinkedInResponse{
			HasClientID:     h.cfg.LinkedIn.ClientID != "",
			HasClientSecret: h.cfg.LinkedIn.ClientSecret != "",
			HasAuth:         uc.LinkedInAccessToken != "",
			HasLinkitinAuth: hasLinkitinAuth,
			LiAt:            liAt,
			JSessionID:      jsessionID,
		},
		GitHub: dto.ConfigGitHubResponse{
			HasPAT:       h.cfg.GitHub.PersonalAccessToken != "",
			HasOAuth:     h.cfg.GitHub.ClientID != "",
			HasAuth:      uc.GitHubAccessToken != "" || h.cfg.GitHub.PersonalAccessToken != "",
			DefaultOwner: h.cfg.GitHub.DefaultOwner,
			DefaultRepo:  h.cfg.GitHub.DefaultRepo,
		},
		YouTube: dto.ConfigYouTubeResponse{
			HasClientID: h.cfg.YouTube.ClientID != "",
			HasAuth:     uc.YouTubeAccessToken != "",
			ChannelID:   channelID,
		},
		TikTok: dto.ConfigTikTokResponse{
			HasClientKey: h.cfg.TikTok.ClientKey != "",
			HasAuth:      uc.TikTokAccessToken != "",
			Username:     tiktokUsername,
		},
		Niches:         niches,
		LinkedInNiches: linkedInNiches,
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

// maskedCookieFromJSON parses a JSON cookie string and returns the masked value for key.
func maskedCookieFromJSON(cookiesJSON, key string) string {
	if cookiesJSON == "" {
		return ""
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(cookiesJSON), &raw); err != nil {
		return ""
	}
	v, ok := raw[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return ""
	}
	return maskSecret(s)
}

func maskedTwikitCookie(key string) string {
	cookies := readCookieFile(filepath.Join(config.DefaultConfigDir(), "twikit_cookies.json"))
	return maskSecret(cookies[key])
}

func maskedLinkitinCookie(key string) string {
	cookies := readCookieFile(filepath.Join(config.DefaultConfigDir(), "linkitin_cookies.json"))
	return maskSecret(cookies[key])
}
