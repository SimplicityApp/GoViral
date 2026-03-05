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

// FetchPostsHandler handles requests to fetch posts from platforms.
type FetchPostsHandler struct {
	db    *db.DB
	cfg   *config.Config
	store *service.OperationStore
}

// NewFetchPostsHandler creates a new FetchPostsHandler.
func NewFetchPostsHandler(database *db.DB, cfg *config.Config, store *service.OperationStore) *FetchPostsHandler {
	return &FetchPostsHandler{db: database, cfg: cfg, store: store}
}

// Post triggers fetching posts from the specified platform(s).
func (h *FetchPostsHandler) Post(w http.ResponseWriter, r *http.Request) {
	var req dto.FetchPostsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = dto.FetchPostsRequest{} // default: all platforms
	}

	userID := middleware.UserIDFromContext(r.Context())
	progress := make(chan dto.ProgressEvent, 10)

	if WantsSSE(r) {
		go h.doFetch(r.Context(), userID, req.Platform, progress)
		StreamProgress(w, r, progress)
		return
	}

	// Background mode: return 202 with operation ID
	opID := h.store.Create()
	go func() {
		h.doFetch(context.Background(), userID, req.Platform, progress)
		// Drain and use the last event
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

func (h *FetchPostsHandler) doFetch(ctx context.Context, userID string, platform string, progress chan<- dto.ProgressEvent) {
	defer close(progress)

	platforms := []string{"x", "linkedin"}
	if platform != "" {
		platforms = []string{platform}
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
			if xCfg.Username == "" && uc.TwikitCookiesJSON == "" {
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

		progress <- dto.ProgressEvent{
			Type:       "progress",
			Message:    fmt.Sprintf("Fetching posts from %s...", p),
			Percentage: 0,
		}

		var posts []models.Post
		var err error

		switch p {
		case "x":
			client := x.NewFallbackClientWithCookiePath(uc.MergedXConfig(*h.cfg), twikitCookiePath)
			posts, err = client.FetchMyPosts(ctx, 100)
		case "linkedin":
			client := linkedin.NewFallbackClientWithConfigDir(uc.MergedLinkedInConfig(*h.cfg), nil, linkitinConfigDir)
			posts, err = client.FetchMyPosts(ctx, 100)
		default:
			progress <- dto.ProgressEvent{
				Type:    "error",
				Message: fmt.Sprintf("unknown platform: %s", p),
			}
			continue
		}

		if err != nil {
			slog.Error("fetching posts", "platform", p, "error", err)
			progress <- dto.ProgressEvent{
				Type:    "error",
				Message: fmt.Sprintf("failed to fetch from %s: %v", p, err),
			}
			errCount++
			continue
		}

		for _, post := range posts {
			post := post
			if err := h.db.UpsertPost(userID, &post); err != nil {
				slog.Error("saving post", "platform", p, "error", err)
			}
		}

		progress <- dto.ProgressEvent{
			Type:       "progress",
			Message:    fmt.Sprintf("Fetched %d posts from %s", len(posts), p),
			Percentage: 100,
		}
	}

	if errCount < len(platforms) {
		progress <- dto.ProgressEvent{
			Type:       "complete",
			Message:    "Finished fetching posts",
			Percentage: 100,
		}
	}
}
