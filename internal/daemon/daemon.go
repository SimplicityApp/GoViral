package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/internal/telegram"
	"github.com/shuhao/goviral/pkg/models"
)

// GenerateFunc generates content for the given trending post IDs and returns content IDs.
// The isRepost parameter indicates whether to generate repost commentary or full rewrites.
type GenerateFunc func(ctx context.Context, platform string, trendingIDs []int64, count int, isRepost bool) ([]int64, error)

// PublishFunc publishes a content item and returns the post IDs.
type PublishFunc func(ctx context.Context, contentID int64) ([]string, error)

// DiscoverFunc fetches trending posts for a platform and returns their IDs.
type DiscoverFunc func(ctx context.Context, platform string, period string, minLikes, limit int) ([]int64, error)

// ClassifyFunc classifies trending posts as rewrite or repost, returning the split IDs.
type ClassifyFunc func(ctx context.Context, trendingIDs []int64) (rewriteIDs, repostIDs []int64, err error)

// Status reports daemon running state per platform.
type Status struct {
	Running   bool
	Platforms map[string]PlatformStatus
}

// PlatformStatus tracks per-platform daemon state.
type PlatformStatus struct {
	Schedule    string
	NextRun     *time.Time
	LastRun     *time.Time
	LastBatchID *int64
}

// Daemon is the autopilot daemon that runs the trending→generate→notify→publish pipeline.
type Daemon struct {
	cfg          *config.Config
	db           *db.DB
	tg           *telegram.Client
	intentParser *IntentParser
	scheduler    *CronScheduler

	generateFn GenerateFunc
	publishFn  PublishFunc
	discoverFn DiscoverFunc
	classifyFn ClassifyFunc

	mu        sync.RWMutex
	running   bool
	cancel    context.CancelFunc
	lastRun   map[string]*time.Time
	lastBatch map[string]*int64
}

// New creates a new Daemon instance. classifyFn is optional — if nil, all posts default to rewrite.
func New(cfg *config.Config, database *db.DB, generateFn GenerateFunc, publishFn PublishFunc, discoverFn DiscoverFunc, classifyFn ClassifyFunc) *Daemon {
	var tg *telegram.Client
	if cfg.Telegram.BotToken != "" {
		tg = telegram.NewClient(cfg.Telegram.BotToken)
	}

	var intentParser *IntentParser
	if cfg.Claude.APIKey != "" {
		claudeClient := claude.NewClient(cfg.Claude.APIKey, cfg.Claude.Model)
		intentParser = NewIntentParser(claudeClient)
	}

	return &Daemon{
		cfg:          cfg,
		db:           database,
		tg:           tg,
		intentParser: intentParser,
		scheduler:    NewScheduler(),
		generateFn:   generateFn,
		publishFn:    publishFn,
		discoverFn:   discoverFn,
		classifyFn:   classifyFn,
		lastRun:      make(map[string]*time.Time),
		lastBatch:    make(map[string]*int64),
	}
}

// Start launches the daemon scheduler and Telegram receiver.
func (d *Daemon) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("daemon already running")
	}

	ctx, d.cancel = context.WithCancel(ctx)
	d.running = true
	d.mu.Unlock()

	// Register cron jobs per platform
	for platform, expr := range d.cfg.Daemon.Schedules {
		p := platform
		if err := d.scheduler.Add(p, expr, func() {
			d.runPipeline(ctx, p)
		}); err != nil {
			slog.Error("adding cron job", "platform", p, "error", err)
		}
	}

	d.scheduler.Start(ctx)

	// Start Telegram receiver
	if d.tg != nil && d.cfg.Telegram.ChatID != 0 {
		if d.cfg.Telegram.WebhookURL != "" {
			if err := d.tg.SetWebhook(ctx, d.cfg.Telegram.WebhookURL); err != nil {
				slog.Error("setting telegram webhook", "error", err)
			} else {
				slog.Info("telegram webhook registered", "url", d.cfg.Telegram.WebhookURL)
			}
		} else {
			// Clear any previously-registered webhook so getUpdates long-polling works.
			if err := d.tg.DeleteWebhook(ctx); err != nil {
				slog.Warn("clearing telegram webhook before polling (may be harmless)", "error", err)
			}
			go d.startTelegramPoller(ctx)
		}
	}

	// Start auto-skip goroutine
	go d.autoSkipLoop(ctx)

	slog.Info("daemon started", "platforms", len(d.cfg.Daemon.Schedules))
	return nil
}

// Stop gracefully shuts down the daemon.
func (d *Daemon) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return
	}

	d.scheduler.Stop()
	if d.cancel != nil {
		d.cancel()
	}

	// Deregister webhook if applicable
	if d.tg != nil && d.cfg.Telegram.WebhookURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		d.tg.DeleteWebhook(ctx)
	}

	d.running = false
	slog.Info("daemon stopped")
}

// GetStatus returns the current daemon status.
func (d *Daemon) GetStatus() Status {
	d.mu.RLock()
	defer d.mu.RUnlock()

	platforms := make(map[string]PlatformStatus)
	for p, expr := range d.cfg.Daemon.Schedules {
		ps := PlatformStatus{Schedule: expr}
		next := d.scheduler.NextRun(p)
		ps.NextRun = next
		ps.LastRun = d.lastRun[p]
		ps.LastBatchID = d.lastBatch[p]
		platforms[p] = ps
	}

	return Status{
		Running:   d.running,
		Platforms: platforms,
	}
}

// IsRunning returns whether the daemon is running.
func (d *Daemon) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// RunNow triggers an immediate pipeline run for the given platform.
func (d *Daemon) RunNow(ctx context.Context, platform string) error {
	if !d.IsRunning() {
		return fmt.Errorf("daemon is not running")
	}
	go d.runPipeline(ctx, platform)
	return nil
}

// ApproveBatch approves a batch from the web UI.
func (d *Daemon) ApproveBatch(ctx context.Context, batchID int64) error {
	batch, err := d.db.GetDaemonBatch(batchID)
	if err != nil {
		return fmt.Errorf("getting batch: %w", err)
	}
	if batch == nil {
		return fmt.Errorf("batch %d not found", batchID)
	}

	return d.executeBatchAction(ctx, batch, &models.DaemonIntent{
		Action:  "approve",
		Message: "approved from web UI",
	}, "web")
}

// RejectBatch rejects a batch from the web UI.
func (d *Daemon) RejectBatch(ctx context.Context, batchID int64) error {
	batch, err := d.db.GetDaemonBatch(batchID)
	if err != nil {
		return fmt.Errorf("getting batch: %w", err)
	}
	if batch == nil {
		return fmt.Errorf("batch %d not found", batchID)
	}

	now := time.Now()
	return d.db.UpdateDaemonBatchStatus(batchID, models.BatchStatusRejected, map[string]interface{}{
		"approval_source": "web",
		"resolved_at":     &now,
	})
}

// HandleWebhookUpdate processes a Telegram webhook update.
func (d *Daemon) HandleWebhookUpdate(update *telegram.Update) {
	d.handleUpdate(update)
}

// HandleBatchAction processes a batch action request from the web UI.
func (d *Daemon) HandleBatchAction(ctx context.Context, batchID int64, action string, contentIDs []int64, edits map[int64]string, scheduleAt string) error {
	batch, err := d.db.GetDaemonBatch(batchID)
	if err != nil {
		return fmt.Errorf("getting batch: %w", err)
	}
	if batch == nil {
		return fmt.Errorf("batch %d not found", batchID)
	}

	intent := &models.DaemonIntent{
		Action:     action,
		ContentIDs: contentIDs,
		Edits:      make(map[int64]string),
		Message:    fmt.Sprintf("%s from web UI", action),
	}

	for k, v := range edits {
		intent.Edits[k] = v
	}

	if scheduleAt != "" {
		t, err := time.Parse(time.RFC3339, scheduleAt)
		if err != nil {
			return fmt.Errorf("parsing schedule_at: %w", err)
		}
		intent.ScheduleAt = &t
	}

	return d.executeBatchAction(ctx, batch, intent, "web")
}

// --- Internal methods ---

func (d *Daemon) runPipeline(ctx context.Context, platform string) {
	slog.Info("daemon pipeline starting", "platform", platform)

	now := time.Now()
	d.mu.Lock()
	d.lastRun[platform] = &now
	d.mu.Unlock()

	// 1. Discover trending posts
	trendingIDs, err := d.discoverFn(ctx, platform, d.cfg.Daemon.Period, d.cfg.Daemon.MinLikes, d.cfg.Daemon.TrendingLimit)
	if err != nil {
		slog.Error("daemon discover failed", "platform", platform, "error", err)
		return
	}
	if len(trendingIDs) == 0 {
		slog.Info("daemon: no trending posts found", "platform", platform)
		return
	}

	// Take up to TrendingLimit
	limit := d.cfg.Daemon.TrendingLimit
	if limit > 0 && len(trendingIDs) > limit {
		trendingIDs = trendingIDs[:limit]
	}

	// 2. Classify posts as rewrite vs repost
	var rewriteIDs, repostIDs []int64
	if d.classifyFn != nil {
		rewriteIDs, repostIDs, err = d.classifyFn(ctx, trendingIDs)
		if err != nil {
			slog.Error("daemon classify failed, defaulting all to rewrite", "platform", platform, "error", err)
			rewriteIDs = trendingIDs
			repostIDs = nil
		}
		slog.Info("daemon classify complete", "platform", platform, "rewrites", len(rewriteIDs), "reposts", len(repostIDs))
	} else {
		// No classifier — all posts default to rewrite
		rewriteIDs = trendingIDs
	}

	// 3. Generate rewrites + reposts
	var contentIDs []int64
	if len(rewriteIDs) > 0 {
		ids, err := d.generateFn(ctx, platform, rewriteIDs, d.cfg.Daemon.MaxPerBatch, false)
		if err != nil {
			slog.Error("daemon generate rewrites failed", "platform", platform, "error", err)
		} else {
			contentIDs = append(contentIDs, ids...)
		}
	}
	if len(repostIDs) > 0 {
		ids, err := d.generateFn(ctx, platform, repostIDs, d.cfg.Daemon.MaxPerBatch, true)
		if err != nil {
			slog.Error("daemon generate reposts failed", "platform", platform, "error", err)
		} else {
			contentIDs = append(contentIDs, ids...)
		}
	}
	if len(contentIDs) == 0 {
		slog.Info("daemon: no content generated", "platform", platform)
		return
	}

	// 3. Create batch record
	trendingJSON, _ := json.Marshal(trendingIDs)
	contentJSON, _ := json.Marshal(contentIDs)

	batch := &models.DaemonBatch{
		Platform:    platform,
		Status:      models.BatchStatusPending,
		ContentIDs:  string(contentJSON),
		TrendingIDs: string(trendingJSON),
	}

	batchID, err := d.db.InsertDaemonBatch(batch)
	if err != nil {
		slog.Error("daemon insert batch failed", "error", err)
		return
	}
	batch.ID = batchID

	// Link content to batch
	for _, cid := range contentIDs {
		if err := d.db.SetGeneratedContentBatchID(cid, batchID); err != nil {
			slog.Error("linking content to batch", "content_id", cid, "error", err)
		}
	}

	d.mu.Lock()
	d.lastBatch[platform] = &batchID
	d.mu.Unlock()

	// 4. Send Telegram notification
	if d.tg != nil && d.cfg.Telegram.ChatID != 0 {
		contents, err := d.db.GetGeneratedContentByBatchID(batchID)
		if err != nil {
			slog.Error("getting batch contents for notification", "error", err)
			return
		}

		msg := telegram.FormatBatchNotification(batch, contents)
		msgID, err := d.tg.SendMessageWithMarkdown(ctx, d.cfg.Telegram.ChatID, msg)
		if err != nil {
			slog.Error("sending telegram notification", "error", err)
			return
		}

		d.db.UpdateDaemonBatchStatus(batchID, models.BatchStatusNotified, map[string]interface{}{
			"telegram_message_id": msgID,
			"notified_at":        &now,
		})
	}

	slog.Info("daemon pipeline completed", "platform", platform, "batch_id", batchID, "content_count", len(contentIDs))
}

func (d *Daemon) startTelegramPoller(ctx context.Context) {
	slog.Info("starting telegram long-poll receiver")
	var offset int64

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		updates, err := d.tg.GetUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("polling telegram updates", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for i := range updates {
			offset = updates[i].UpdateID + 1
			d.handleUpdate(&updates[i])
		}
	}
}

func (d *Daemon) handleUpdate(update *telegram.Update) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	slog.Info("telegram update received", "chat_id", msg.Chat.ID, "text_preview", truncate(msg.Text, 60))

	if msg.Chat.ID != d.cfg.Telegram.ChatID {
		slog.Warn("telegram message from unexpected chat, ignoring", "got", msg.Chat.ID, "want", d.cfg.Telegram.ChatID)
		return
	}

	// Must be a reply to one of our messages; otherwise treat as standalone command
	if msg.ReplyToMessage == nil {
		slog.Info("telegram standalone command received", "text", truncate(msg.Text, 60))
		go d.handleStandaloneCommand(context.Background(), msg)
		return
	}

	replyToMsgID := msg.ReplyToMessage.MessageID

	batch, err := d.db.GetDaemonBatchByTelegramMsgID(replyToMsgID)
	if err != nil {
		slog.Error("finding batch for telegram reply", "msg_id", replyToMsgID, "error", err)
		return
	}
	if batch == nil {
		return
	}

	// Update status to awaiting_reply
	d.db.UpdateDaemonBatchStatus(batch.ID, models.BatchStatusAwaitingReply, map[string]interface{}{
		"reply_text": msg.Text,
	})

	// Get batch contents
	contents, err := d.db.GetGeneratedContentByBatchID(batch.ID)
	if err != nil {
		slog.Error("getting batch contents for intent parsing", "error", err)
		return
	}

	// Parse intent
	if d.intentParser == nil {
		slog.Error("intent parser not configured (missing Claude API key)")
		return
	}

	ctx := context.Background()
	intent, err := d.intentParser.Parse(ctx, batch, contents, msg.Text)
	if err != nil {
		slog.Error("parsing intent", "batch_id", batch.ID, "error", err)
		d.sendTelegramReply(ctx, fmt.Sprintf("Failed to understand your reply: %v", err))
		return
	}

	// Store parsed intent
	intentJSON, _ := json.Marshal(intent)
	d.db.UpdateDaemonBatchStatus(batch.ID, batch.Status, map[string]interface{}{
		"parsed_intent": string(intentJSON),
	})

	// Execute the action
	if err := d.executeBatchAction(ctx, batch, intent, "telegram"); err != nil {
		slog.Error("executing batch action", "batch_id", batch.ID, "error", err)
		d.sendTelegramReply(ctx, fmt.Sprintf("Failed to execute action: %v", err))
	}
}

func (d *Daemon) handleStandaloneCommand(ctx context.Context, msg *telegram.Message) {
	if d.intentParser == nil {
		d.sendTelegramReply(ctx, "Intent parser not configured (missing Claude API key)")
		return
	}

	// Find the most recent active batch across all platforms
	batch, err := d.db.GetLatestActiveDaemonBatch()
	if err != nil {
		slog.Error("finding latest active batch for standalone command", "error", err)
		d.sendTelegramReply(ctx, fmt.Sprintf("Failed to find active batch: %v", err))
		return
	}
	if batch == nil {
		d.sendTelegramReply(ctx, "No active batch found. Run the pipeline first.")
		return
	}

	contents, err := d.db.GetGeneratedContentByBatchID(batch.ID)
	if err != nil {
		slog.Error("getting batch contents for standalone command", "error", err)
		return
	}

	// Reuse existing intent parser (regex fast-path + Claude AI fallback)
	intent, err := d.intentParser.Parse(ctx, batch, contents, msg.Text)
	if err != nil {
		slog.Error("parsing standalone command intent", "batch_id", batch.ID, "error", err)
		d.sendTelegramReply(ctx, fmt.Sprintf("Could not understand command: %v", err))
		return
	}

	// Store parsed intent on the batch
	intentJSON, _ := json.Marshal(intent)
	d.db.UpdateDaemonBatchStatus(batch.ID, batch.Status, map[string]interface{}{
		"parsed_intent": string(intentJSON),
		"reply_text":    msg.Text,
	})

	// Reuse existing batch action executor
	if err := d.executeBatchAction(ctx, batch, intent, "telegram"); err != nil {
		slog.Error("executing standalone command", "batch_id", batch.ID, "error", err)
		d.sendTelegramReply(ctx, fmt.Sprintf("Failed to execute: %v", err))
	}
}

func (d *Daemon) executeBatchAction(ctx context.Context, batch *models.DaemonBatch, intent *models.DaemonIntent, source string) error {
	now := time.Now()

	switch intent.Action {
	case "approve":
		d.db.UpdateDaemonBatchStatus(batch.ID, models.BatchStatusApproved, map[string]interface{}{
			"approval_source": source,
			"resolved_at":     &now,
		})

		// Get content IDs to post
		var contentIDs []int64
		if len(intent.ContentIDs) > 0 {
			contentIDs = intent.ContentIDs
		} else {
			if err := json.Unmarshal([]byte(batch.ContentIDs), &contentIDs); err != nil {
				return fmt.Errorf("parsing content IDs: %w", err)
			}
		}

		// Post each content item
		var allPostIDs []string
		for _, cid := range contentIDs {
			postIDs, err := d.publishFn(ctx, cid)
			if err != nil {
				slog.Error("publishing content", "content_id", cid, "error", err)
				d.db.UpdateDaemonBatchStatus(batch.ID, models.BatchStatusFailed, map[string]interface{}{
					"error_message": err.Error(),
				})
				return err
			}
			allPostIDs = append(allPostIDs, postIDs...)
		}

		d.db.UpdateDaemonBatchStatus(batch.ID, models.BatchStatusPosted, nil)

		// Notify via Telegram
		if d.tg != nil && d.cfg.Telegram.ChatID != 0 {
			msg := telegram.FormatPostResult(batch, allPostIDs)
			d.sendTelegramReply(ctx, msg)
		}

	case "reject":
		d.db.UpdateDaemonBatchStatus(batch.ID, models.BatchStatusRejected, map[string]interface{}{
			"approval_source": source,
			"resolved_at":     &now,
		})
		if d.tg != nil && d.cfg.Telegram.ChatID != 0 {
			d.sendTelegramReply(ctx, telegram.FormatBatchApproved(batch, "rejected"))
		}

	case "edit":
		// Apply edits to content
		for contentID, newText := range intent.Edits {
			if err := d.db.UpdateGeneratedContentText(contentID, newText); err != nil {
				slog.Error("applying edit", "content_id", contentID, "error", err)
			}
		}

		// Re-notify with updated content
		contents, _ := d.db.GetGeneratedContentByBatchID(batch.ID)
		d.db.UpdateDaemonBatchStatus(batch.ID, models.BatchStatusNotified, nil)

		if d.tg != nil && d.cfg.Telegram.ChatID != 0 {
			msg := telegram.FormatBatchNotification(batch, contents)
			msgID, err := d.tg.SendMessageWithMarkdown(ctx, d.cfg.Telegram.ChatID, msg)
			if err == nil {
				d.db.UpdateDaemonBatchStatus(batch.ID, models.BatchStatusNotified, map[string]interface{}{
					"telegram_message_id": msgID,
				})
			}
		}

	case "schedule":
		if intent.ScheduleAt == nil {
			return fmt.Errorf("schedule action requires schedule_at")
		}

		d.db.UpdateDaemonBatchStatus(batch.ID, models.BatchStatusScheduled, map[string]interface{}{
			"approval_source": source,
			"resolved_at":     &now,
		})

		// Schedule each content item
		var contentIDs []int64
		if len(intent.ContentIDs) > 0 {
			contentIDs = intent.ContentIDs
		} else {
			if err := json.Unmarshal([]byte(batch.ContentIDs), &contentIDs); err != nil {
				return fmt.Errorf("parsing content IDs: %w", err)
			}
		}

		for _, cid := range contentIDs {
			if _, err := d.db.InsertScheduledPost(cid, *intent.ScheduleAt); err != nil {
				slog.Error("scheduling content", "content_id", cid, "error", err)
			}
		}

		if d.tg != nil && d.cfg.Telegram.ChatID != 0 {
			d.sendTelegramReply(ctx, fmt.Sprintf("Batch #%d scheduled for %s", batch.ID, intent.ScheduleAt.Format(time.RFC822)))
		}

	default:
		return fmt.Errorf("unknown action: %s", intent.Action)
	}

	return nil
}

func (d *Daemon) autoSkipLoop(ctx context.Context) {
	dur, err := time.ParseDuration(d.cfg.Daemon.AutoSkipAfter)
	if err != nil {
		dur = 2 * time.Hour
	}

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			batches, err := d.db.GetPendingDaemonBatches(dur)
			if err != nil {
				slog.Error("checking stale batches", "error", err)
				continue
			}
			now := time.Now()
			for _, b := range batches {
				slog.Info("auto-skipping stale batch", "batch_id", b.ID)
				d.db.UpdateDaemonBatchStatus(b.ID, models.BatchStatusArchived, map[string]interface{}{
					"resolved_at": &now,
				})
			}
		}
	}
}

func (d *Daemon) sendTelegramReply(ctx context.Context, text string) {
	if d.tg == nil || d.cfg.Telegram.ChatID == 0 {
		return
	}
	_, err := d.tg.SendMessageWithMarkdown(ctx, d.cfg.Telegram.ChatID, text)
	if err != nil {
		slog.Error("sending telegram reply", "error", err)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
