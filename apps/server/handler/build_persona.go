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
	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/internal/ai/persona"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/internal/ratelimit"
	"github.com/shuhao/goviral/pkg/models"
)

// BuildPersonaHandler handles requests to build persona profiles.
type BuildPersonaHandler struct {
	db    *db.DB
	cfg   *config.Config
	store *service.OperationStore
}

// NewBuildPersonaHandler creates a new BuildPersonaHandler.
func NewBuildPersonaHandler(database *db.DB, cfg *config.Config, store *service.OperationStore) *BuildPersonaHandler {
	return &BuildPersonaHandler{db: database, cfg: cfg, store: store}
}

// Post triggers persona building for the specified platform.
func (h *BuildPersonaHandler) Post(w http.ResponseWriter, r *http.Request) {
	var req dto.BuildPersonaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid request body", reqID)
		return
	}

	if req.Platform == "" {
		req.Platform = "x"
	}

	userID := middleware.UserIDFromContext(r.Context())

	if WantsSSE(r) {
		progress := make(chan dto.ProgressEvent, 10)
		go h.doBuild(r.Context(), userID, req.Platform, progress)
		StreamProgress(w, r, progress)
		return
	}

	opID := h.store.Create()
	go func() {
		progress := make(chan dto.ProgressEvent, 10)
		go h.doBuild(context.Background(), userID, req.Platform, progress)
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

func (h *BuildPersonaHandler) doBuild(ctx context.Context, userID string, platform string, progress chan<- dto.ProgressEvent) {
	defer close(progress)

	uc, _ := h.db.GetUserConfig(userID)
	claudeCfg := uc.ResolvedClaudeConfig(*h.cfg)
	if claudeCfg.APIKey == "" {
		progress <- dto.ProgressEvent{Type: "error", Message: "Claude API key not configured"}
		return
	}
	if !uc.UsingOwnClaudeKey() {
		if err := ratelimit.CheckAIRateLimit(h.db, userID, "claude", h.cfg.Claude.DailyLimit); err != nil {
			progress <- dto.ProgressEvent{Type: "error", Message: err.Error()}
			return
		}
	}

	progress <- dto.ProgressEvent{
		Type:       "progress",
		Message:    fmt.Sprintf("Loading %s posts...", platform),
		Percentage: 10,
	}

	var posts []models.Post
	var err error
	if platform != "" {
		posts, err = h.db.GetPostsByPlatform(userID, platform)
	} else {
		posts, err = h.db.GetAllPosts(userID)
	}
	if err != nil {
		progress <- dto.ProgressEvent{Type: "error", Message: fmt.Sprintf("loading posts: %v", err)}
		return
	}
	if len(posts) == 0 {
		progress <- dto.ProgressEvent{Type: "error", Message: "no posts found; fetch posts first"}
		return
	}

	progress <- dto.ProgressEvent{
		Type:       "progress",
		Message:    fmt.Sprintf("Analyzing %d posts with Claude...", len(posts)),
		Percentage: 30,
	}

	client := claude.NewClient(claudeCfg.APIKey, claudeCfg.Model)
	analyzer := persona.NewAnalyzer(client)

	profile, err := analyzer.BuildProfile(ctx, posts, platform)
	if err != nil {
		slog.Error("building persona", "error", err)
		progress <- dto.ProgressEvent{Type: "error", Message: fmt.Sprintf("building persona: %v", err)}
		return
	}

	progress <- dto.ProgressEvent{
		Type:       "progress",
		Message:    "Saving persona profile...",
		Percentage: 90,
	}

	p := &models.Persona{
		Platform: platform,
		Profile:  *profile,
	}
	if err := h.db.UpsertPersona(userID, p); err != nil {
		progress <- dto.ProgressEvent{Type: "error", Message: fmt.Sprintf("saving persona: %v", err)}
		return
	}

	if !uc.UsingOwnClaudeKey() {
		ratelimit.RecordAIUsage(h.db, userID, "claude")
	}

	progress <- dto.ProgressEvent{
		Type:       "complete",
		Message:    fmt.Sprintf("Persona built for %s", platform),
		Percentage: 100,
	}
}
