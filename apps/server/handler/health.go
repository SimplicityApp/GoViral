package handler

import (
	"net/http"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/internal/config"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	cfg *config.Config
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(cfg *config.Config) *HealthHandler {
	return &HealthHandler{cfg: cfg}
}

// Get returns the health status and platform configuration state.
func (h *HealthHandler) Get(w http.ResponseWriter, r *http.Request) {
	platforms := make(map[string]string)

	if h.cfg.X.BearerToken != "" || h.cfg.X.AccessToken != "" {
		platforms["x"] = "configured"
	} else {
		platforms["x"] = "not_configured"
	}

	if h.cfg.LinkedIn.AccessToken != "" {
		platforms["linkedin"] = "configured"
	} else {
		platforms["linkedin"] = "not_configured"
	}

	middleware.WriteJSON(w, http.StatusOK, dto.HealthResponse{
		Status:    "ok",
		Platforms: platforms,
	})
}
