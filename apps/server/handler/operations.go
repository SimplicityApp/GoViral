package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/apps/server/service"
)

// OperationsHandler handles operation polling requests.
type OperationsHandler struct {
	store *service.OperationStore
}

// NewOperationsHandler creates a new OperationsHandler.
func NewOperationsHandler(store *service.OperationStore) *OperationsHandler {
	return &OperationsHandler{store: store}
}

// Get returns the status of a long-running operation.
func (h *OperationsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	op := h.store.Get(id)
	if op == nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusNotFound, dto.ErrCodeNotFound, "operation not found", reqID)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, op)
}
