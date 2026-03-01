package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/apps/server/service"
	dbpkg "github.com/shuhao/goviral/internal/db"
)

// CommentHandler handles LinkedIn comment generation and posting.
type CommentHandler struct {
	publishSvc  *service.PublishService
	generateSvc *service.GenerateService
	db          *dbpkg.DB
}

// NewCommentHandler creates a new CommentHandler.
func NewCommentHandler(publishSvc *service.PublishService, generateSvc *service.GenerateService, database *dbpkg.DB) *CommentHandler {
	return &CommentHandler{publishSvc: publishSvc, generateSvc: generateSvc, db: database}
}

// GenerateComment generates AI comment variations for a trending post.
func (h *CommentHandler) GenerateComment(w http.ResponseWriter, r *http.Request) {
	var req dto.GenerateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid request body", reqID)
		return
	}

	if req.TrendingPostID == 0 {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "trending_post_id is required", reqID)
		return
	}

	contents, err := h.generateSvc.GenerateComment(r.Context(), req.TrendingPostID, req.Platform, req.Count)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, err.Error(), reqID)
		return
	}

	responses := make([]dto.GeneratedContentResponse, len(contents))
	for i, gc := range contents {
		responses[i] = contentToResponse(gc)
	}

	middleware.WriteJSON(w, http.StatusOK, responses)
}

// PostComment publishes a generated comment to the appropriate platform.
func (h *CommentHandler) PostComment(w http.ResponseWriter, r *http.Request) {
	var req dto.PostCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid request body", reqID)
		return
	}

	if req.ContentID == 0 {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "content_id is required", reqID)
		return
	}

	commentURN, err := h.publishSvc.Comment(r.Context(), req.ContentID)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		slog.Error("comment publish failed",
			"request_id", reqID,
			"content_id", req.ContentID,
			"error", err,
		)
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodePlatformError, err.Error(), reqID)
		return
	}

	gc, _ := h.db.GetGeneratedContentByID(req.ContentID)
	var content dto.GeneratedContentResponse
	if gc != nil {
		content = contentToResponse(*gc)
	}

	middleware.WriteJSON(w, http.StatusOK, dto.CommentResponse{
		CommentURN: commentURN,
		Content:    content,
	})
}
