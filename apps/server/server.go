package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
)

// Server holds the HTTP server and its dependencies.
type Server struct {
	Cfg        *config.Config
	DB         *db.DB
	Router     chi.Router
	httpServer *http.Server
}

// NewServer creates a new Server with the given config and database.
func NewServer(cfg *config.Config, database *db.DB) *Server {
	r := chi.NewRouter()
	return &Server{
		Cfg:    cfg,
		DB:     database,
		Router: r,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
			Handler: r,
		},
	}
}

// Start begins listening and serving HTTP requests.
func (s *Server) Start() error {
	slog.Info("server starting", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("starting server: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("server shutting down")
	return s.httpServer.Shutdown(ctx)
}
