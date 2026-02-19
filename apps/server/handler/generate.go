package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/apps/server/service"
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

	if WantsSSE(r) {
		svcProgress := make(chan dto.ProgressEvent, 10)
		clientProgress := make(chan dto.ProgressEvent, 10)
		go func() {
			result, err := h.svc.Generate(r.Context(), req, svcProgress)
			if err != nil {
				_ = err
			}
			// Forward all service events to the client, injecting result data into the complete event
			for evt := range svcProgress {
				if evt.Type == "complete" && result != nil {
					responses := make([]dto.GeneratedContentResponse, len(result))
					for i, gc := range result {
						responses[i] = contentToResponse(gc)
					}
					evt.Data = responses
				}
				clientProgress <- evt
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
			result, err := h.svc.Generate(context.Background(), req, progress)
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
