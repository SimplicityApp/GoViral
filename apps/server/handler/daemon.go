package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/daemon"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/internal/telegram"
	"github.com/shuhao/goviral/pkg/models"
)

// DaemonHandler handles daemon management endpoints.
type DaemonHandler struct {
	daemon *daemon.Daemon
	db     *db.DB
	cfg    *config.Config
}

// NewDaemonHandler creates a new DaemonHandler.
func NewDaemonHandler(d *daemon.Daemon, database *db.DB, cfg *config.Config) *DaemonHandler {
	return &DaemonHandler{daemon: d, db: database, cfg: cfg}
}

// GetStatus returns the daemon status.
func (h *DaemonHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.daemon.GetStatus()

	resp := dto.DaemonStatusResponse{
		Running:   status.Running,
		Platforms: make(map[string]dto.PlatformDaemonInfo),
	}

	for p, ps := range status.Platforms {
		info := dto.PlatformDaemonInfo{
			Schedule:    ps.Schedule,
			LastBatchID: ps.LastBatchID,
		}
		if ps.NextRun != nil {
			s := ps.NextRun.Format(time.RFC3339)
			info.NextRun = &s
		}
		if ps.LastRun != nil {
			s := ps.LastRun.Format(time.RFC3339)
			info.LastRun = &s
		}
		resp.Platforms[p] = info
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// ListBatches returns daemon batches.
func (h *DaemonHandler) ListBatches(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform")
	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	batches, err := h.db.GetDaemonBatches(platform, status, limit)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to list batches", reqID)
		return
	}

	resp := make([]dto.DaemonBatchResponse, 0, len(batches))
	for _, b := range batches {
		resp = append(resp, batchToResponse(&b))
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

// GetBatch returns a single batch with its content.
func (h *DaemonHandler) GetBatch(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid batch ID", reqID)
		return
	}

	batch, err := h.db.GetDaemonBatch(id)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to get batch", reqID)
		return
	}
	if batch == nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusNotFound, dto.ErrCodeNotFound, "batch not found", reqID)
		return
	}

	resp := batchToResponse(batch)

	// Include content details
	contents, err := h.db.GetGeneratedContentByBatchID(id)
	if err == nil {
		for _, gc := range contents {
			resp.Contents = append(resp.Contents, dto.GeneratedContentResponse{
				ID:               gc.ID,
				SourceTrendingID: gc.SourceTrendingID,
				TargetPlatform:   gc.TargetPlatform,
				OriginalContent:  gc.OriginalContent,
				GeneratedContent: gc.GeneratedContent,
				PersonaID:        gc.PersonaID,
				CreatedAt:        gc.CreatedAt,
				Status:           gc.Status,
				PlatformPostIDs:  gc.PlatformPostIDs,
				PostedAt:         gc.PostedAt,
				ImagePrompt:      gc.ImagePrompt,
				IsRepost:         gc.IsRepost,
				QuoteTweetID:     gc.QuoteTweetID,
			})
		}
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// BatchAction handles approve/reject/edit from the web UI.
func (h *DaemonHandler) BatchAction(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid batch ID", reqID)
		return
	}

	var req dto.BatchActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid request body", reqID)
		return
	}

	if err := h.daemon.HandleBatchAction(r.Context(), id, req.Action, req.ContentIDs, req.Edits, req.ScheduleAt); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, err.Error(), reqID)
		return
	}

	// Return updated batch
	batch, _ := h.db.GetDaemonBatch(id)
	if batch != nil {
		middleware.WriteJSON(w, http.StatusOK, batchToResponse(batch))
	} else {
		middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// RunNow triggers an immediate pipeline run.
func (h *DaemonHandler) RunNow(w http.ResponseWriter, r *http.Request) {
	var req dto.DaemonRunNowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Platform = "x"
	}
	if req.Platform == "" {
		req.Platform = "x"
	}

	if err := h.daemon.RunNow(r.Context(), req.Platform); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, err.Error(), reqID)
		return
	}

	middleware.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "started", "platform": req.Platform})
}

// GetConfig returns the daemon configuration.
func (h *DaemonHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	schedules := h.cfg.Daemon.Schedules
	if schedules == nil {
		schedules = make(map[string]string)
	}

	resp := dto.DaemonConfigResponse{
		Daemon: dto.DaemonSettingsResponse{
			Enabled:       h.cfg.Daemon.Enabled,
			Schedules:     schedules,
			MaxPerBatch:   h.cfg.Daemon.MaxPerBatch,
			AutoSkipAfter: h.cfg.Daemon.AutoSkipAfter,
			TrendingLimit: h.cfg.Daemon.TrendingLimit,
			MinLikes:      h.cfg.Daemon.MinLikes,
			Period:        h.cfg.Daemon.Period,
		},
		Telegram: dto.TelegramSettingsResponse{
			BotToken:   maskSecret(h.cfg.Telegram.BotToken),
			ChatID:     h.cfg.Telegram.ChatID,
			WebhookURL: h.cfg.Telegram.WebhookURL,
			Connected:  h.cfg.Telegram.BotToken != "" && h.cfg.Telegram.ChatID != 0,
		},
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// UpdateConfig updates daemon configuration.
func (h *DaemonHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req dto.DaemonConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid request body", reqID)
		return
	}

	if req.Enabled != nil {
		h.cfg.Daemon.Enabled = *req.Enabled
	}
	if len(req.Schedules) > 0 {
		if h.cfg.Daemon.Schedules == nil {
			h.cfg.Daemon.Schedules = make(map[string]string)
		}
		for k, v := range req.Schedules {
			h.cfg.Daemon.Schedules[k] = v
		}
	}
	if req.MaxPerBatch != nil {
		h.cfg.Daemon.MaxPerBatch = *req.MaxPerBatch
	}
	if req.AutoSkipAfter != nil {
		h.cfg.Daemon.AutoSkipAfter = *req.AutoSkipAfter
	}
	if req.TrendingLimit != nil {
		h.cfg.Daemon.TrendingLimit = *req.TrendingLimit
	}
	if req.MinLikes != nil {
		h.cfg.Daemon.MinLikes = *req.MinLikes
	}
	if req.Period != nil {
		h.cfg.Daemon.Period = *req.Period
	}
	if req.BotToken != nil {
		h.cfg.Telegram.BotToken = *req.BotToken
	}
	if req.ChatID != nil {
		h.cfg.Telegram.ChatID = *req.ChatID
	}
	if req.WebhookURL != nil {
		h.cfg.Telegram.WebhookURL = *req.WebhookURL
	}

	if err := config.Save(h.cfg, config.DefaultConfigPath()); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to save config", reqID)
		return
	}

	h.GetConfig(w, r)
}

// StartDaemon starts the daemon.
func (h *DaemonHandler) StartDaemon(w http.ResponseWriter, r *http.Request) {
	if err := h.daemon.Start(r.Context()); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, err.Error(), reqID)
		return
	}
	h.GetStatus(w, r)
}

// StopDaemon stops the daemon.
func (h *DaemonHandler) StopDaemon(w http.ResponseWriter, r *http.Request) {
	h.daemon.Stop()
	h.GetStatus(w, r)
}

// TelegramWebhook handles incoming Telegram webhook updates.
func (h *DaemonHandler) TelegramWebhook(w http.ResponseWriter, r *http.Request) {
	// Validate secret
	secret := chi.URLParam(r, "secret")
	expected := webhookSecret(h.cfg.Telegram.BotToken)
	if secret != expected {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	update, err := telegram.ParseWebhookUpdate(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	h.daemon.HandleWebhookUpdate(update)
	w.WriteHeader(http.StatusOK)
}

// WebhookSecret returns the webhook secret path segment.
func WebhookSecret(botToken string) string {
	return webhookSecret(botToken)
}

func webhookSecret(botToken string) string {
	h := sha256.Sum256([]byte(botToken))
	return hex.EncodeToString(h[:16])
}

func batchToResponse(b *models.DaemonBatch) dto.DaemonBatchResponse {
	resp := dto.DaemonBatchResponse{
		ID:                b.ID,
		Platform:          b.Platform,
		Status:            b.Status,
		TelegramMessageID: b.TelegramMessageID,
		ApprovalSource:    b.ApprovalSource,
		ReplyText:         b.ReplyText,
		ErrorMessage:      b.ErrorMessage,
		CreatedAt:         b.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         b.UpdatedAt.Format(time.RFC3339),
	}

	// Parse JSON arrays
	var contentIDs []int64
	json.Unmarshal([]byte(b.ContentIDs), &contentIDs)
	resp.ContentIDs = contentIDs
	if resp.ContentIDs == nil {
		resp.ContentIDs = []int64{}
	}

	var trendingIDs []int64
	json.Unmarshal([]byte(b.TrendingIDs), &trendingIDs)
	resp.TrendingIDs = trendingIDs
	if resp.TrendingIDs == nil {
		resp.TrendingIDs = []int64{}
	}

	if b.NotifiedAt != nil {
		s := b.NotifiedAt.Format(time.RFC3339)
		resp.NotifiedAt = &s
	}
	if b.ResolvedAt != nil {
		s := b.ResolvedAt.Format(time.RFC3339)
		resp.ResolvedAt = &s
	}

	return resp
}
