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
	userID := middleware.UserIDFromContext(r.Context())
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
		go h.doDiscover(r.Context(), userID, req, progress)
		StreamProgress(w, r, progress)
		return
	}

	opID := h.store.Create()
	go func() {
		h.doDiscover(context.Background(), userID, req, progress)
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

func (h *DiscoverTrendingHandler) doDiscover(ctx context.Context, userID string, req dto.DiscoverTrendingRequest, progress chan<- dto.ProgressEvent) {
	defer close(progress)

	platforms := []string{"x", "linkedin"}
	if req.Platform != "" {
		platforms = []string{req.Platform}
	}

	uc, _ := h.db.GetUserConfig(userID)

	// Write per-user cookies to temp files for process isolation.
	twikitCookiePath, twikitCleanup, _ := writeTwikitCookieTempFile(uc.TwikitCookiesJSON)
	defer twikitCleanup()
	linkitinConfigDir, linkitinCleanup, _ := writeLinkitinCookieTempDir(uc.LinkitinCookiesJSON)
	defer linkitinCleanup()

	var errCount int
	for _, p := range platforms {
		// Pre-flight credential check: fail fast if no credentials are configured.
		switch p {
		case "x":
			xCfg := uc.MergedXConfig(*h.cfg)
			if xCfg.BearerToken == "" && xCfg.Username == "" && uc.TwikitCookiesJSON == "" {
				progress <- dto.ProgressEvent{
					Type:    "error",
					Message: "X is not connected — install the GoViral browser extension to sync cookies, or paste your X cookies manually in Settings",
				}
				errCount++
				continue
			}
		case "linkedin":
			liCfg := uc.MergedLinkedInConfig(*h.cfg)
			if liCfg.AccessToken == "" && liCfg.PersonURN == "" && uc.LinkitinCookiesJSON == "" {
				progress <- dto.ProgressEvent{
					Type:    "error",
					Message: "LinkedIn is not connected — install the GoViral browser extension to sync cookies, or connect via OAuth in Settings",
				}
				errCount++
				continue
			}
		}

		var niches []string
		switch p {
		case "x":
			niches = uc.MergedNiches(*h.cfg)
		case "linkedin":
			niches = uc.MergedLinkedInNiches(*h.cfg)
			if len(niches) == 0 {
				niches = []string{"AI", "Programming", "Technology"}
			}
		}
		if len(niches) == 0 {
			progress <- dto.ProgressEvent{
				Type:    "error",
				Message: fmt.Sprintf("no niches configured for %s", p),
			}
			errCount++
			continue
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
			client := x.NewFallbackClientWithCookiePath(uc.MergedXConfig(*h.cfg), twikitCookiePath)
			posts, err = client.FetchTrendingPosts(ctx, niches, req.Period, req.MinLikes, req.Limit)
		case "linkedin":
			client := linkedin.NewFallbackClientWithConfigDir(uc.MergedLinkedInConfig(*h.cfg), nil, linkitinConfigDir)
			posts, err = client.FetchTrendingPosts(ctx, niches, req.Period, req.MinLikes, req.Limit)
		}

		if err != nil {
			slog.Error("discovering trending", "platform", p, "error", err)
			progress <- dto.ProgressEvent{
				Type:    "error",
				Message: fmt.Sprintf("failed to discover trending on %s: %v", p, err),
			}
			errCount++
			continue
		}

		for _, tp := range posts {
			tp := tp
			if err := h.db.UpsertTrendingPost(userID, &tp); err != nil {
				slog.Error("saving trending post", "platform", p, "error", err)
			}
		}

		progress <- dto.ProgressEvent{
			Type:       "progress",
			Message:    fmt.Sprintf("Found %d trending posts on %s", len(posts), p),
			Percentage: 100,
		}
	}

	if errCount < len(platforms) {
		progress <- dto.ProgressEvent{
			Type:       "complete",
			Message:    "Finished discovering trending posts",
			Percentage: 100,
		}
	}
}
