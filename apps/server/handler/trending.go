package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/apps/server/service"
	"github.com/shuhao/goviral/pkg/models"
)

// TrendingHandler handles requests for trending posts.
type TrendingHandler struct {
	svc *service.TrendingService
}

// NewTrendingHandler creates a new TrendingHandler.
func NewTrendingHandler(svc *service.TrendingService) *TrendingHandler {
	return &TrendingHandler{svc: svc}
}

// List returns trending posts filtered by query parameters.
func (h *TrendingHandler) List(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	posts, err := h.svc.List(platform, limit)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to list trending posts", reqID)
		return
	}

	resp := make([]dto.TrendingPostResponse, len(posts))
	for i, p := range posts {
		resp[i] = trendingToResponse(p)
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// GetByID returns a single trending post.
func (h *TrendingHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid trending post ID", reqID)
		return
	}

	post, err := h.svc.GetByID(id)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to get trending post", reqID)
		return
	}
	if post == nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusNotFound, dto.ErrCodeNotFound, "trending post not found", reqID)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, trendingToResponse(*post))
}

func trendingToResponse(p models.TrendingPost) dto.TrendingPostResponse {
	tags := p.NicheTags
	if tags == nil {
		tags = []string{}
	}
	return dto.TrendingPostResponse{
		ID:             p.ID,
		Platform:       p.Platform,
		PlatformPostID: p.PlatformPostID,
		AuthorUsername: p.AuthorUsername,
		AuthorName:     p.AuthorName,
		Content:        p.Content,
		Likes:          p.Likes,
		Reposts:        p.Reposts,
		Comments:       p.Comments,
		Impressions:    p.Impressions,
		NicheTags:      tags,
		PostedAt:       p.PostedAt,
		FetchedAt:      p.FetchedAt,
	}
}
