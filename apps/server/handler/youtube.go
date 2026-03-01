package handler

import (
	"encoding/json"
	"net/http"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/apps/server/service"
)

// YouTubeHandler handles YouTube video upload requests.
type YouTubeHandler struct {
	publishSvc *service.PublishService
}

// NewYouTubeHandler creates a new YouTubeHandler.
func NewYouTubeHandler(publishSvc *service.PublishService) *YouTubeHandler {
	return &YouTubeHandler{publishSvc: publishSvc}
}

// Upload handles POST /api/v1/youtube/upload — publish video to YouTube Shorts.
func (h *YouTubeHandler) Upload(w http.ResponseWriter, r *http.Request) {
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

	postIDs, _, err := h.publishSvc.PublishYouTube(r.Context(), req.ContentID)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodePlatformError, err.Error(), reqID)
		return
	}

	url := ""
	if len(postIDs) > 0 {
		url = "https://youtube.com/shorts/" + postIDs[0]
	}

	middleware.WriteJSON(w, http.StatusOK, dto.VideoUploadResponse{
		VideoID:  postIDs[0],
		Platform: "youtube",
		URL:      url,
	})
}
