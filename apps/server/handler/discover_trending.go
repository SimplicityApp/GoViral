package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/apps/server/service"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/internal/platform/linkedin"
	"github.com/shuhao/goviral/internal/platform/x"
	"github.com/shuhao/goviral/pkg/models"
)

// DiscoverTrendingHandler handles requests to discover trending posts.
type DiscoverTrendingHandler struct {
	db    *db.DB
	cfg   *config.Config
	store *service.OperationStore
}

// NewDiscoverTrendingHandler creates a new DiscoverTrendingHandler.
func NewDiscoverTrendingHandler(database *db.DB, cfg *config.Config, store *service.OperationStore) *DiscoverTrendingHandler {
	return &DiscoverTrendingHandler{db: database, cfg: cfg, store: store}
}

// Post triggers trending post discovery.
func (h *DiscoverTrendingHandler) Post(w http.ResponseWriter, r *http.Request) {
	var req dto.DiscoverTrendingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = dto.DiscoverTrendingRequest{}
	}
	if req.Period == "" {
		req.Period = "week"
	}
	if req.MinLikes <= 0 {
		req.MinLikes = 10
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}

	progress := make(chan dto.ProgressEvent, 10)

	if WantsSSE(r) {
		go h.doDiscover(r.Context(), req, progress)
		StreamProgress(w, r, progress)
		return
	}

	opID := h.store.Create()
	go func() {
		h.doDiscover(context.Background(), req, progress)
		var lastErr string
		for evt := range progress {
			if evt.Type == "error" {
				lastErr = evt.Message
			}
		}
		if lastErr != "" {
			h.store.Fail(opID, lastErr)
		} else {
			h.store.Complete(opID, nil)
		}
	}()

	middleware.WriteJSON(w, http.StatusAccepted, dto.OperationResponse{
		ID:     opID,
		Status: "running",
	})
}

func (h *DiscoverTrendingHandler) doDiscover(ctx context.Context, req dto.DiscoverTrendingRequest, progress chan<- dto.ProgressEvent) {
	defer close(progress)

	platforms := []string{"x", "linkedin"}
	if req.Platform != "" {
		platforms = []string{req.Platform}
	}

	for _, p := range platforms {
		var niches []string
		switch p {
		case "x":
			niches = h.cfg.Niches
		case "linkedin":
			niches = h.cfg.LinkedInNiches
			if len(niches) == 0 {
				niches = []string{"AI", "Programming", "Technology"}
			}
		}
		if len(niches) == 0 {
			progress <- dto.ProgressEvent{
				Type:    "error",
				Message: fmt.Sprintf("no niches configured for %s", p),
			}
			return
		}

		progress <- dto.ProgressEvent{
			Type:       "progress",
			Message:    fmt.Sprintf("Discovering trending posts on %s...", p),
			Percentage: 0,
		}

		var posts []models.TrendingPost
		var err error

		switch p {
		case "x":
			client := x.NewFallbackClient(h.cfg.X)
			posts, err = client.FetchTrendingPosts(ctx, niches, req.Period, req.MinLikes, req.Limit)
		case "linkedin":
			client := linkedin.NewFallbackClient(h.cfg.LinkedIn, nil)
			posts, err = client.FetchTrendingPosts(ctx, niches, req.Period, req.MinLikes, req.Limit)
		}

		if err != nil {
			slog.Error("discovering trending", "platform", p, "error", err)
			progress <- dto.ProgressEvent{
				Type:    "error",
				Message: fmt.Sprintf("failed to discover trending on %s: %v", p, err),
			}
			return
		}

		for _, tp := range posts {
			tp := tp
			if err := h.db.UpsertTrendingPost(&tp); err != nil {
				slog.Error("saving trending post", "platform", p, "error", err)
			}
		}

		progress <- dto.ProgressEvent{
			Type:       "progress",
			Message:    fmt.Sprintf("Found %d trending posts on %s", len(posts), p),
			Percentage: 100,
		}
	}

	progress <- dto.ProgressEvent{
		Type:       "complete",
		Message:    "Finished discovering trending posts",
		Percentage: 100,
	}
}
