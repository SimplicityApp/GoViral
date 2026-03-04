package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
)

// Setup configures the chi router with middleware and route groups.
// It accepts the router, config, and database directly to avoid
// importing the main server package.
func Setup(r chi.Router, cfg *config.Config, database *db.DB) {
	// Global middleware
	r.Use(middleware.Recovery)
	r.Use(middleware.Logging)
	r.Use(middleware.CORS(cfg.Server.AllowedOrigins))

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Handlers will be registered here by subsequent tasks.
	})
}
