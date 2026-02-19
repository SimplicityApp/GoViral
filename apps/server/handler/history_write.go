package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/internal/db"
)

// HistoryWriteHandler handles write operations for generated content.
type HistoryWriteHandler struct {
	db *db.DB
}

// NewHistoryWriteHandler creates a new HistoryWriteHandler.
func NewHistoryWriteHandler(database *db.DB) *HistoryWriteHandler {
	return &HistoryWriteHandler{db: database}
}

// UpdateStatus updates the status of a generated content record.
func (h *HistoryWriteHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid content ID", reqID)
		return
	}

	var req dto.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid request body", reqID)
		return
	}

	// At least one field must be provided
	if req.Status == "" && req.GeneratedContent == nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "must provide status or generated_content", reqID)
		return
	}

	// Validate status if provided
	if req.Status != "" {
		validStatuses := map[string]bool{"draft": true, "approved": true, "posted": true}
		if !validStatuses[req.Status] {
			reqID := middleware.RequestIDFromContext(r.Context())
			middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "status must be draft, approved, or posted", reqID)
			return
		}
	}

	// Verify content exists
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

	// Update content text if provided
	if req.GeneratedContent != nil {
		if err := h.db.UpdateGeneratedContentText(id, *req.GeneratedContent); err != nil {
			reqID := middleware.RequestIDFromContext(r.Context())
			middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to update content", reqID)
			return
		}
		gc.GeneratedContent = *req.GeneratedContent
	}

	// Update status if provided
	if req.Status != "" {
		if err := h.db.UpdateGeneratedContentStatus(id, req.Status); err != nil {
			reqID := middleware.RequestIDFromContext(r.Context())
			middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to update status", reqID)
			return
		}
		gc.Status = req.Status
	}

	middleware.WriteJSON(w, http.StatusOK, contentToResponse(*gc))
}

// Delete removes a generated content record.
func (h *HistoryWriteHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	if err := h.db.DeleteGeneratedContent(id); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to delete content", reqID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
