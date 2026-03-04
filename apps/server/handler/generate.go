package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/apps/server/service"
	"github.com/shuhao/goviral/pkg/models"
)

// GenerateHandler handles content generation requests.
type GenerateHandler struct {
	svc   *service.GenerateService
	store *service.OperationStore
}

// NewGenerateHandler creates a new GenerateHandler.
func NewGenerateHandler(svc *service.GenerateService, store *service.OperationStore) *GenerateHandler {
	return &GenerateHandler{svc: svc, store: store}
}

// Post triggers content generation from trending posts.
func (h *GenerateHandler) Post(w http.ResponseWriter, r *http.Request) {
	var req dto.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid request body", reqID)
		return
	}

	if len(req.TrendingPostIDs) == 0 {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "trending_post_ids is required", reqID)
		return
	}

	userID := middleware.UserIDFromContext(r.Context())

	if WantsSSE(r) {
		svcProgress := make(chan dto.ProgressEvent, 10)
		clientProgress := make(chan dto.ProgressEvent, 10)

		go func() {
			var result []models.GeneratedContent
			var genErr error

			done := make(chan struct{})
			go func() {
				defer close(done)
				result, genErr = h.svc.Generate(r.Context(), userID, req, svcProgress)
			}()

			for evt := range svcProgress {
				if evt.Type == "complete" {
					<-done
					if result != nil {
						responses := make([]dto.GeneratedContentResponse, len(result))
						for i, gc := range result {
							responses[i] = contentToResponse(gc)
						}
						evt.Data = responses
					}
				}
				clientProgress <- evt
			}

			<-done
			if genErr != nil {
				clientProgress <- dto.ProgressEvent{
					Type:    "error",
					Message: genErr.Error(),
				}
			}
			close(clientProgress)
		}()

		StreamProgress(w, r, clientProgress)
		return
	}

	// Background mode
	opID := h.store.Create()
	go func() {
		progress := make(chan dto.ProgressEvent, 10)
		go func() {
			result, err := h.svc.Generate(context.Background(), userID, req, progress)
			if err != nil {
				h.store.Fail(opID, err.Error())
				return
			}
			responses := make([]dto.GeneratedContentResponse, len(result))
			for i, gc := range result {
				responses[i] = contentToResponse(gc)
			}
			h.store.Complete(opID, responses)
		}()
		// Drain the progress channel
		for range progress {
		}
	}()

	middleware.WriteJSON(w, http.StatusAccepted, dto.OperationResponse{
		ID:     opID,
		Status: "running",
	})
}
