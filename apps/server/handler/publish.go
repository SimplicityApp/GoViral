package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/apps/server/service"
)

// PublishHandler handles content publishing requests.
type PublishHandler struct {
	svc *service.PublishService
}

// NewPublishHandler creates a new PublishHandler.
func NewPublishHandler(svc *service.PublishService) *PublishHandler {
	return &PublishHandler{svc: svc}
}

// Post publishes content to the target platform.
func (h *PublishHandler) Post(w http.ResponseWriter, r *http.Request) {
	var req dto.PublishRequest
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

	postIDs, threadParts, err := h.svc.Publish(r.Context(), req.ContentID, req.Numbered)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodePlatformError, err.Error(), reqID)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, dto.PublishResponse{
		PostIDs:     postIDs,
		ThreadParts: threadParts,
	})
}

// PostX handles X-specific publish requests.
func (h *PublishHandler) PostX(w http.ResponseWriter, r *http.Request) {
	var req dto.PublishRequest
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

	postIDs, threadParts, err := h.svc.PublishX(r.Context(), req.ContentID, req.Numbered)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodePlatformError, err.Error(), reqID)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, dto.PublishResponse{
		PostIDs:     postIDs,
		ThreadParts: threadParts,
	})
}

// PostYouTube handles YouTube-specific publish requests.
func (h *PublishHandler) PostYouTube(w http.ResponseWriter, r *http.Request) {
	var req dto.PublishRequest
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

	postIDs, threadParts, err := h.svc.PublishYouTube(r.Context(), req.ContentID)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodePlatformError, err.Error(), reqID)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, dto.PublishResponse{
		PostIDs:     postIDs,
		ThreadParts: threadParts,
	})
}

// PostTikTok handles TikTok-specific publish requests.
func (h *PublishHandler) PostTikTok(w http.ResponseWriter, r *http.Request) {
	var req dto.PublishRequest
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

	postIDs, threadParts, err := h.svc.PublishTikTok(r.Context(), req.ContentID)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodePlatformError, err.Error(), reqID)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, dto.PublishResponse{
		PostIDs:     postIDs,
		ThreadParts: threadParts,
	})
}

// PostLinkedIn handles LinkedIn-specific publish requests.
func (h *PublishHandler) PostLinkedIn(w http.ResponseWriter, r *http.Request) {
	var req dto.PublishRequest
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

	postIDs, threadParts, err := h.svc.PublishLinkedIn(r.Context(), req.ContentID)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodePlatformError, err.Error(), reqID)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, dto.PublishResponse{
		PostIDs:     postIDs,
		ThreadParts: threadParts,
	})
}
