package handler

import (
	"net/http"
	"strconv"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/apps/server/service"
)

// PostsHandler handles requests for user posts.
type PostsHandler struct {
	svc *service.PostsService
}

// NewPostsHandler creates a new PostsHandler.
func NewPostsHandler(svc *service.PostsService) *PostsHandler {
	return &PostsHandler{svc: svc}
}

// List returns posts filtered by query parameters.
func (h *PostsHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	platform := r.URL.Query().Get("platform")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	posts, err := h.svc.List(userID, platform, limit)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to list posts", reqID)
		return
	}

	resp := make([]dto.PostResponse, len(posts))
	for i, p := range posts {
		resp[i] = dto.PostResponse{
			ID:             p.ID,
			Platform:       p.Platform,
			PlatformPostID: p.PlatformPostID,
			Content:        p.Content,
			Likes:          p.Likes,
			Reposts:        p.Reposts,
			Comments:       p.Comments,
			Impressions:    p.Impressions,
			PostedAt:       p.PostedAt,
			FetchedAt:      p.FetchedAt,
		}
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}
