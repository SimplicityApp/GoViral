package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/apps/server/service"
	"github.com/shuhao/goviral/internal/db"
)

// ScheduleWriteHandler handles write operations for scheduled posts.
type ScheduleWriteHandler struct {
	db         *db.DB
	publishSvc *service.PublishService
}

// NewScheduleWriteHandler creates a new ScheduleWriteHandler.
func NewScheduleWriteHandler(database *db.DB, publishSvc *service.PublishService) *ScheduleWriteHandler {
	return &ScheduleWriteHandler{db: database, publishSvc: publishSvc}
}

// Create schedules a post for future publishing.
func (h *ScheduleWriteHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	var req dto.ScheduleRequest
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

	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "scheduled_at must be RFC3339 format", reqID)
		return
	}

	// Verify the content exists
	gc, err := h.db.GetGeneratedContentByID(userID, req.ContentID)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to verify content", reqID)
		return
	}
	if gc == nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusNotFound, dto.ErrCodeNotFound, "content not found", reqID)
		return
	}

	// Try native scheduling via platform; fall back to pending for RunDue if it fails
	scheduledPostID, schedErr := h.publishSvc.Schedule(r.Context(), userID, req.ContentID, scheduledAt)

	id, err := h.db.InsertScheduledPost(userID, req.ContentID, scheduledAt)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to save schedule record", reqID)
		return
	}

	status := "pending"
	errorMsg := ""
	if schedErr == nil {
		status = "scheduled"
	} else {
		// Log the error but still store as pending for manual execution
		errorMsg = schedErr.Error()
		log.Printf("native scheduling failed for content %d: %v", req.ContentID, schedErr)
	}
	h.db.UpdateScheduledPostStatus(id, status, errorMsg)
	if schedErr != nil {
		// Don't fail the request - we have a fallback (pending scheduling)
		// but notify the client of the issue
		log.Printf("falling back to pending execution for content %d", req.ContentID)
	}

	middleware.WriteJSON(w, http.StatusCreated, dto.ScheduledPostResponse{
		ID:                 id,
		GeneratedContentID: req.ContentID,
		ScheduledAt:        scheduledAt,
		Status:             status,
		CreatedAt:          time.Now(),
		PlatformScheduleID: scheduledPostID,
	})
}

// Acknowledge manually marks a scheduled post as posted.
func (h *ScheduleWriteHandler) Acknowledge(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid schedule ID", reqID)
		return
	}

	if err := h.db.UpdateScheduledPostStatus(id, "posted", ""); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to update status", reqID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete cancels a scheduled post.
func (h *ScheduleWriteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid schedule ID", reqID)
		return
	}

	if err := h.db.DeleteScheduledPost(userID, id); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusNotFound, dto.ErrCodeNotFound, "scheduled post not found", reqID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RunDue executes all due scheduled posts.
func (h *ScheduleWriteHandler) RunDue(w http.ResponseWriter, r *http.Request) {
	pending, err := h.db.GetPendingScheduledPosts()
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to get pending posts", reqID)
		return
	}

	var results []map[string]interface{}
	for _, sp := range pending {
		postIDs, _, pubErr := h.publishSvc.Publish(r.Context(), sp.UserID, sp.GeneratedContentID, false)
		if pubErr != nil {
			h.db.UpdateScheduledPostStatus(sp.ID, "failed", pubErr.Error())
			results = append(results, map[string]interface{}{
				"schedule_id": sp.ID,
				"status":      "failed",
				"error":       pubErr.Error(),
			})
			continue
		}

		h.db.UpdateScheduledPostStatus(sp.ID, "posted", "")
		results = append(results, map[string]interface{}{
			"schedule_id": sp.ID,
			"status":      "posted",
			"post_ids":    postIDs,
		})
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"executed": len(results),
		"results":  results,
	})
}
