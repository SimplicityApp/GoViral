package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
)

type ConfigWriteHandler struct {
	cfg *config.Config
	db  *db.DB
}

func NewConfigWriteHandler(cfg *config.Config, database *db.DB) *ConfigWriteHandler {
	return &ConfigWriteHandler{cfg: cfg, db: database}
}

func (h *ConfigWriteHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req dto.UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid request body", reqID)
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	uc, err := h.db.GetUserConfig(userID)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to load user config", reqID)
		return
	}

	// Apply per-user field updates
	if req.Claude != nil {
		if req.Claude.APIKey != nil {
			uc.ClaudeAPIKey = *req.Claude.APIKey
		}
		if req.Claude.Model != nil {
			uc.ClaudeModel = *req.Claude.Model
		}
	}

	if req.Gemini != nil {
		if req.Gemini.APIKey != nil {
			uc.GeminiAPIKey = *req.Gemini.APIKey
		}
		if req.Gemini.Model != nil {
			uc.GeminiModel = *req.Gemini.Model
		}
	}

	if req.X != nil {
		if req.X.Username != nil {
			uc.XUsername = *req.X.Username
		}
	}

	if req.LinkedIn != nil {
		if req.LinkedIn.PersonURN != nil {
			uc.LinkedInPersonURN = *req.LinkedIn.PersonURN
		}
	}

	if req.YouTube != nil {
		if req.YouTube.ChannelID != nil {
			uc.YouTubeChannelID = *req.YouTube.ChannelID
		}
	}

	if req.TikTok != nil {
		if req.TikTok.Username != nil {
			uc.TikTokUsername = *req.TikTok.Username
		}
	}

	if req.Niches != nil {
		uc.Niches = *req.Niches
	}

	if req.LinkedInNiches != nil {
		uc.LinkedInNiches = *req.LinkedInNiches
	}

	if req.SelfDescription != nil {
		uc.SelfDescription = *req.SelfDescription
	}

	// Save to DB (NOT to config.yaml)
	if err := h.db.SaveUserConfig(userID, uc); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to save config", reqID)
		return
	}

	// Return updated config
	ch := NewConfigHandler(h.cfg, h.db)
	ch.Get(w, r)
}
