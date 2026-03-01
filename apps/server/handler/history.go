package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/pkg/models"
)

// HistoryHandler handles requests for generated content history.
type HistoryHandler struct {
	db *db.DB
}

// NewHistoryHandler creates a new HistoryHandler.
func NewHistoryHandler(database *db.DB) *HistoryHandler {
	return &HistoryHandler{db: database}
}

// List returns generated content with optional status filter.
func (h *HistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	platform := r.URL.Query().Get("platform")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	contents, err := h.db.GetGeneratedContent(status, platform, limit)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to list history", reqID)
		return
	}

	resp := make([]dto.GeneratedContentResponse, len(contents))
	for i, gc := range contents {
		resp[i] = contentToResponse(gc)
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// GetByID returns a single generated content record.
func (h *HistoryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid content ID", reqID)
		return
	}

	gc, err := h.db.GetGeneratedContentByID(id)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to get content", reqID)
		return
	}
	if gc == nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusNotFound, dto.ErrCodeNotFound, "content not found", reqID)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, contentToResponse(*gc))
}

// GetContentImage serves the AI-generated image for a generated content item.
func (h *HistoryHandler) GetContentImage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid content ID", reqID)
		return
	}

	gc, err := h.db.GetGeneratedContentByID(id)
	if err != nil || gc == nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusNotFound, dto.ErrCodeNotFound, "content not found", reqID)
		return
	}

	if gc.ImagePath == "" {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusNotFound, dto.ErrCodeNotFound, "no image available", reqID)
		return
	}

	http.ServeFile(w, r, gc.ImagePath)
}

func contentToResponse(gc models.GeneratedContent) dto.GeneratedContentResponse {
	return dto.GeneratedContentResponse{
		ID:               gc.ID,
		SourceTrendingID: gc.SourceTrendingID,
		TargetPlatform:   gc.TargetPlatform,
		OriginalContent:  gc.OriginalContent,
		GeneratedContent: gc.GeneratedContent,
		PersonaID:        gc.PersonaID,
		PromptUsed:       gc.PromptUsed,
		CreatedAt:        gc.CreatedAt,
		Status:           gc.Status,
		PlatformPostIDs:  gc.PlatformPostIDs,
		PostedAt:         gc.PostedAt,
		ImagePrompt:      gc.ImagePrompt,
		ImagePath:        gc.ImagePath,
		IsRepost:         gc.IsRepost,
		QuoteTweetID:     gc.QuoteTweetID,
		IsComment:        gc.IsComment,
		SourceType:           gc.SourceType,
		SourceCommitID:       gc.SourceCommitID,
		CodeImagePath:        gc.CodeImagePath,
		CodeImageDescription: gc.CodeImageDescription,
		VideoPath:            gc.VideoPath,
		ThumbnailPath:        gc.ThumbnailPath,
		VideoDuration:        gc.VideoDuration,
		VideoTitle:           gc.VideoTitle,
	}
}
