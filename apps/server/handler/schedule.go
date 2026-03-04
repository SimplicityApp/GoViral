package handler

import (
	"net/http"
	"strconv"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/apps/server/service"
	"github.com/shuhao/goviral/internal/db"
)

// ScheduleHandler handles requests for scheduled posts.
type ScheduleHandler struct {
	svc *service.ScheduleService
	db  *db.DB
}

// NewScheduleHandler creates a new ScheduleHandler.
func NewScheduleHandler(svc *service.ScheduleService, database *db.DB) *ScheduleHandler {
	return &ScheduleHandler{svc: svc, db: database}
}

// List returns scheduled posts with optional status filter.
func (h *ScheduleHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	posts, err := h.svc.List(userID, status, limit)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to list scheduled posts", reqID)
		return
	}

	resp := make([]dto.ScheduledPostResponse, len(posts))
	for i, sp := range posts {
		resp[i] = dto.ScheduledPostResponse{
			ID:                 sp.ID,
			GeneratedContentID: sp.GeneratedContentID,
			ScheduledAt:        sp.ScheduledAt,
			Status:             sp.Status,
			ErrorMessage:       sp.ErrorMessage,
			CreatedAt:          sp.CreatedAt,
		}
		// Look up content preview
		gc, err := h.db.GetGeneratedContentByID(userID, sp.GeneratedContentID)
		if err == nil && gc != nil {
			preview := gc.GeneratedContent
			if len(preview) > 120 {
				preview = preview[:120] + "..."
			}
			resp[i].ContentPreview = preview
			resp[i].TargetPlatform = gc.TargetPlatform
		}
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}
