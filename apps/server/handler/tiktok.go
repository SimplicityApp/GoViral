package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/apps/server/service"
)

// TikTokHandler handles TikTok video upload requests.
type TikTokHandler struct {
	publishSvc *service.PublishService
}

// NewTikTokHandler creates a new TikTokHandler.
func NewTikTokHandler(publishSvc *service.PublishService) *TikTokHandler {
	return &TikTokHandler{publishSvc: publishSvc}
}

// Upload handles POST /api/v1/tiktok/upload — publish video to TikTok.
func (h *TikTokHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	var req dto.VideoUploadRequest
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

	postIDs, _, err := h.publishSvc.PublishTikTok(r.Context(), userID, req.ContentID)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodePlatformError, err.Error(), reqID)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, dto.VideoUploadResponse{
		VideoID:  postIDs[0],
		Platform: "tiktok",
	})
}
