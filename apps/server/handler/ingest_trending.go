package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/pkg/models"
)

// IngestTrendingHandler handles POST /api/v1/trending/ingest
type IngestTrendingHandler struct {
	db *db.DB
}

func NewIngestTrendingHandler(database *db.DB) *IngestTrendingHandler {
	return &IngestTrendingHandler{db: database}
}

func (h *IngestTrendingHandler) Post(w http.ResponseWriter, r *http.Request) {
	var req dto.IngestTrendingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid request body", "")
		return
	}

	if req.Platform == "" {
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "platform is required", "")
		return
	}
	if len(req.Posts) == 0 {
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "at least one post is required", "")
		return
	}

	for _, p := range req.Posts {
		if p.PlatformPostID == "" {
			middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "each post must have a platform_post_id", "")
			return
		}

		var postedAt time.Time
		if p.PostedAt != "" {
			t, err := time.Parse(time.RFC3339, p.PostedAt)
			if err == nil {
				postedAt = t
			}
		}

		nicheTags := p.NicheTags
		if nicheTags == nil {
			nicheTags = []string{}
		}

		post := models.TrendingPost{
			Platform:       req.Platform,
			PlatformPostID: p.PlatformPostID,
			AuthorUsername: p.AuthorUsername,
			AuthorName:     p.AuthorName,
			Content:        p.Content,
			Likes:          p.Likes,
			Reposts:        p.Reposts,
			Comments:       p.Comments,
			Impressions:    p.Impressions,
			NicheTags:      nicheTags,
			PostedAt:       postedAt,
		}
		if err := h.db.UpsertTrendingPost(&post); err != nil {
			middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to upsert trending post", "")
			return
		}
	}

	middleware.WriteJSON(w, http.StatusOK, dto.IngestResponse{Count: len(req.Posts)})
}
