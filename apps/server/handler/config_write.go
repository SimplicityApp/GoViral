package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/internal/config"
)

// ConfigWriteHandler handles config update requests.
type ConfigWriteHandler struct {
	cfg *config.Config
}

// NewConfigWriteHandler creates a new ConfigWriteHandler.
func NewConfigWriteHandler(cfg *config.Config) *ConfigWriteHandler {
	return &ConfigWriteHandler{cfg: cfg}
}

// Update applies partial config updates.
func (h *ConfigWriteHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req dto.UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid request body", reqID)
		return
	}

	if req.Claude != nil {
		if req.Claude.APIKey != nil {
			h.cfg.Claude.APIKey = *req.Claude.APIKey
		}
		if req.Claude.Model != nil {
			h.cfg.Claude.Model = *req.Claude.Model
		}
	}

	if req.Gemini != nil {
		if req.Gemini.APIKey != nil {
			h.cfg.Gemini.APIKey = *req.Gemini.APIKey
		}
		if req.Gemini.Model != nil {
			h.cfg.Gemini.Model = *req.Gemini.Model
		}
	}

	if req.X != nil {
		if req.X.APIKey != nil {
			h.cfg.X.APIKey = *req.X.APIKey
		}
		if req.X.APISecret != nil {
			h.cfg.X.APISecret = *req.X.APISecret
		}
		if req.X.BearerToken != nil {
			h.cfg.X.BearerToken = *req.X.BearerToken
		}
		if req.X.ClientID != nil {
			h.cfg.X.ClientID = *req.X.ClientID
		}
		if req.X.ClientSecret != nil {
			h.cfg.X.ClientSecret = *req.X.ClientSecret
		}
		if req.X.Username != nil {
			h.cfg.X.Username = *req.X.Username
		}
	}

	if req.LinkedIn != nil {
		if req.LinkedIn.ClientID != nil {
			h.cfg.LinkedIn.ClientID = *req.LinkedIn.ClientID
		}
		if req.LinkedIn.ClientSecret != nil {
			h.cfg.LinkedIn.ClientSecret = *req.LinkedIn.ClientSecret
		}
		if req.LinkedIn.PersonURN != nil {
			h.cfg.LinkedIn.PersonURN = *req.LinkedIn.PersonURN
		}
	}

	if req.GitHub != nil {
		if req.GitHub.PersonalAccessToken != nil {
			h.cfg.GitHub.PersonalAccessToken = *req.GitHub.PersonalAccessToken
		}
		if req.GitHub.DefaultOwner != nil {
			h.cfg.GitHub.DefaultOwner = *req.GitHub.DefaultOwner
		}
		if req.GitHub.DefaultRepo != nil {
			h.cfg.GitHub.DefaultRepo = *req.GitHub.DefaultRepo
		}
	}

	if req.YouTube != nil {
		if req.YouTube.ClientID != nil {
			h.cfg.YouTube.ClientID = *req.YouTube.ClientID
		}
		if req.YouTube.ClientSecret != nil {
			h.cfg.YouTube.ClientSecret = *req.YouTube.ClientSecret
		}
		if req.YouTube.ChannelID != nil {
			h.cfg.YouTube.ChannelID = *req.YouTube.ChannelID
		}
	}

	if req.TikTok != nil {
		if req.TikTok.ClientKey != nil {
			h.cfg.TikTok.ClientKey = *req.TikTok.ClientKey
		}
		if req.TikTok.ClientSecret != nil {
			h.cfg.TikTok.ClientSecret = *req.TikTok.ClientSecret
		}
		if req.TikTok.Username != nil {
			h.cfg.TikTok.Username = *req.TikTok.Username
		}
	}

	if req.Niches != nil {
		h.cfg.Niches = *req.Niches
	}

	if req.LinkedInNiches != nil {
		h.cfg.LinkedInNiches = *req.LinkedInNiches
	}

	// Persist to disk
	if err := config.Save(h.cfg, config.DefaultConfigPath()); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to save config", reqID)
		return
	}

	// Return the updated config (with secrets masked)
	ch := NewConfigHandler(h.cfg)
	ch.Get(w, r)
}
