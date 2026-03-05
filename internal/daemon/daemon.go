package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/internal/ai/generator"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/internal/platform/linkedin"
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

// CommentGenerateFunc generates comments for trending post IDs and returns content IDs.
type CommentGenerateFunc func(ctx context.Context, platform string, trendingIDs []int64, count int) ([]int64, error)

// CompeteFunc ranks generated content entries and returns the results.
type CompeteFunc func(ctx context.Context, entries []models.CompeteEntry, maxWinners int, platform string) ([]models.CompeteResult, error)

// ActionSelectFunc selects the optimal action (post, repost, comment) for each trending post.
type ActionSelectFunc func(ctx context.Context, posts []models.TrendingPost, platform string) ([]models.ActionSelectResult, error)

// ScheduleFunc attempts native platform scheduling and returns the platform schedule ID.
type ScheduleFunc func(ctx context.Context, contentID int64, scheduledAt time.Time) (string, error)

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
	NextDigest  *time.Time
	Paused      bool
	PausedAt    *time.Time
	PauseReason string
}

// Daemon is the autopilot daemon that runs the trending→generate→notify→publish pipeline.
type Daemon struct {
	cfg          *config.Config
	db           *db.DB
	tg           *telegram.Client
	intentParser *IntentParser
	generator    *generator.Generator // for single-draft rewrites
	scheduler    *CronScheduler

	generateFn        GenerateFunc
	publishFn         PublishFunc
	discoverFn        DiscoverFunc
	classifyFn        ClassifyFunc
	commentGenerateFn CommentGenerateFunc
	competeFn         CompeteFunc
	scheduleFn        ScheduleFunc
	actionSelectFn    ActionSelectFunc

	mu              sync.RWMutex
	running         bool
	cancel          context.CancelFunc
	lastRun         map[string]*time.Time
	lastBatch       map[string]*int64
	linkedinPaused  bool
	linkedinPausedAt time.Time
}

// New creates a new Daemon instance. classifyFn, commentGenerateFn, competeFn, scheduleFn, and actionSelectFn are optional —
// if nil, all posts default to rewrite, comment generation is skipped, digest mode won't rank,
// scheduled posts are stored as pending without attempting native platform scheduling,
// and auto-publish falls back to the standard digest flow.
func New(cfg *config.Config, database *db.DB, generateFn GenerateFunc, publishFn PublishFunc, discoverFn DiscoverFunc, classifyFn ClassifyFunc, commentGenerateFn CommentGenerateFunc, competeFn CompeteFunc, scheduleFn ScheduleFunc, actionSelectFn ActionSelectFunc) *Daemon {
	var tg *telegram.Client
	if cfg.Telegram.BotToken != "" {
		tg = telegram.NewClient(cfg.Telegram.BotToken)
	}

	var intentParser *IntentParser
	var gen *generator.Generator
	if cfg.Claude.APIKey != "" {
		claudeClient := claude.NewClient(cfg.Claude.APIKey, cfg.Claude.Model)
		intentParser = NewIntentParser(claudeClient)
		gen = generator.NewGenerator(claudeClient)
	}

	return &Daemon{
		cfg:          cfg,
		db:           database,
		tg:           tg,
		intentParser: intentParser,
		generator:    gen,
		scheduler:    NewScheduler(),
		generateFn:        generateFn,
		publishFn:         publishFn,
		discoverFn:        discoverFn,
		classifyFn:        classifyFn,
		commentGenerateFn: commentGenerateFn,
		competeFn:         competeFn,
		scheduleFn:        scheduleFn,
		actionSelectFn:    actionSelectFn,
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

	// Register digest cron jobs (one per platform) when digest mode is enabled
	if d.cfg.Daemon.DigestMode && d.cfg.Daemon.DigestSchedule != "" {
		for platform := range d.cfg.Daemon.Schedules {
			p := platform
			entryName := p + "_digest"
			if err := d.scheduler.Add(entryName, d.cfg.Daemon.DigestSchedule, func() {
				d.runDigest(ctx, p)
			}); err != nil {
				slog.Error("adding digest cron job", "platform", p, "error", err)
			}
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
		if d.cfg.Daemon.DigestMode {
			ps.NextDigest = d.scheduler.NextRun(p + "_digest")
		}
		if p == "linkedin" && d.linkedinPaused {
			ps.Paused = true
			ps.PausedAt = &d.linkedinPausedAt
			ps.PauseReason = "LinkedIn cookies expired — re-sync cookies via the browser extension or update them in Settings"
		}
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

// pauseLinkedIn sets the LinkedIn pause flag and sends a one-time Telegram notification.
// Subsequent calls are no-ops (idempotent).
func (d *Daemon) pauseLinkedIn(ctx context.Context) {
	d.mu.Lock()
	if d.linkedinPaused {
		d.mu.Unlock()
		return
	}
	d.linkedinPaused = true
	d.linkedinPausedAt = time.Now()
	d.mu.Unlock()

	slog.Warn("linkedin paused: cookies expired")
	if d.tg != nil && d.cfg.Telegram.ChatID != 0 {
		msg := "⚠️ *LinkedIn paused* — cookies expired\\.\n\nRe\\-sync cookies via the browser extension or update them in Settings\\. The daemon will auto\\-resume\\."
		d.sendTelegramReply(ctx, msg)
	}
}

// isLinkedInPaused checks whether LinkedIn operations are paused. If the cookie
// file has been updated since the pause was set, it auto-resumes and returns false.
func (d *Daemon) isLinkedInPaused() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.linkedinPaused {
		return false
	}

	// Check if cookie file has been refreshed since pause
	info, err := os.Stat(linkedin.CookieFilePath())
	if err == nil && info.ModTime().After(d.linkedinPausedAt) {
		d.linkedinPaused = false
		slog.Info("linkedin auto-resumed: cookie file updated")
		return false
	}

	return true
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

// RunDigestNow triggers an immediate digest run for the given platform.
func (d *Daemon) RunDigestNow(ctx context.Context, platform string) error {
	if !d.IsRunning() {
		return fmt.Errorf("daemon is not running")
	}
	go d.runDigest(ctx, platform)
	return nil
}

// --- Internal methods ---

func (d *Daemon) runDigest(ctx context.Context, platform string) {
	if platform == "linkedin" && d.isLinkedInPaused() {
		slog.Info("daemon digest skipped: linkedin paused (cookies expired)")
		return
	}

	slog.Info("daemon digest starting", "platform", platform)

	// 1. Get unbatched trending IDs (accumulated pool from last 24h)
	lookback := 24 * time.Hour
	trendingIDs, err := d.db.GetUnbatchedTrendingIDs("", platform, lookback)
	if err != nil {
		slog.Error("daemon digest: getting unbatched trending IDs", "platform", platform, "error", err)
		return
	}
	if len(trendingIDs) == 0 {
		slog.Info("daemon digest: no unbatched trending posts found", "platform", platform)
		return
	}
	slog.Info("daemon digest: found unbatched posts", "platform", platform, "count", len(trendingIDs))

	// Branch: auto-publish mode
	if d.cfg.Daemon.AutoPublish && d.actionSelectFn != nil {
		d.runAutoPublishDigest(ctx, platform, trendingIDs)
		return
	}

	// 2. Classify: rewrite vs repost
	var rewriteIDs, repostIDs []int64
	if d.classifyFn != nil {
		rewriteIDs, repostIDs, err = d.classifyFn(ctx, trendingIDs)
		if err != nil {
			slog.Error("daemon digest classify failed, defaulting all to rewrite", "platform", platform, "error", err)
			rewriteIDs = trendingIDs
			repostIDs = nil
		}
		slog.Info("daemon digest classify complete", "platform", platform, "rewrites", len(rewriteIDs), "reposts", len(repostIDs))
	} else {
		rewriteIDs = trendingIDs
	}

	// 3. Generate content for all of them
	var contentIDs []int64
	if len(rewriteIDs) > 0 {
		ids, err := d.generateFn(ctx, platform, rewriteIDs, d.cfg.Daemon.MaxPerBatch, false)
		if err != nil {
			slog.Error("daemon digest generate rewrites failed", "platform", platform, "error", err)
		} else {
			contentIDs = append(contentIDs, ids...)
		}
	}
	if len(repostIDs) > 0 {
		ids, err := d.generateFn(ctx, platform, repostIDs, d.cfg.Daemon.MaxPerBatch, true)
		if err != nil {
			slog.Error("daemon digest generate reposts failed", "platform", platform, "error", err)
		} else {
			contentIDs = append(contentIDs, ids...)
		}
	}
	if len(contentIDs) == 0 {
		slog.Info("daemon digest: no content generated", "platform", platform)
		return
	}

	totalCandidates := len(contentIDs)
	maxWinners := totalCandidates / 2
	if maxWinners < 1 {
		maxWinners = 1
	}
	// DigestMaxPosts still acts as an upper cap if configured
	if d.cfg.Daemon.DigestMaxPosts > 0 && d.cfg.Daemon.DigestMaxPosts < maxWinners {
		maxWinners = d.cfg.Daemon.DigestMaxPosts
	}

	// 4. Compete: rank all generated content (if competeFn is available)
	var rankings []models.CompeteResult
	var winnerContentIDs []int64

	if d.competeFn != nil && len(contentIDs) > maxWinners {
		// Build CompeteEntry list
		var entries []models.CompeteEntry
		for _, cid := range contentIDs {
			content, err := d.db.GetGeneratedContentByID("", cid)
			if err != nil || content == nil {
				slog.Warn("digest: skipping content for competition", "content_id", cid, "error", err)
				continue
			}
			var tp models.TrendingPost
			if content.SourceTrendingID != 0 {
				post, err := d.db.GetTrendingPostByID("",content.SourceTrendingID)
				if err == nil && post != nil {
					tp = *post
				}
			}
			entries = append(entries, models.CompeteEntry{
				TrendingPost:     tp,
				GeneratedContent: *content,
			})
		}

		rankings, err = d.competeFn(ctx, entries, maxWinners, platform)
		if err != nil {
			slog.Error("daemon digest competition failed, using all content", "platform", platform, "error", err)
			winnerContentIDs = contentIDs
		} else {
			// Split winners vs losers
			winnerSet := make(map[int64]bool, len(rankings))
			for _, r := range rankings {
				winnerSet[r.ContentID] = true
				winnerContentIDs = append(winnerContentIDs, r.ContentID)
			}

			// Archive losers
			for _, cid := range contentIDs {
				if !winnerSet[cid] {
					if err := d.db.UpdateGeneratedContentStatus("", cid, "archived"); err != nil {
						slog.Error("digest: archiving loser content", "content_id", cid, "error", err)
					}
				}
			}
			slog.Info("daemon digest competition complete", "platform", platform, "winners", len(winnerContentIDs), "archived", totalCandidates-len(winnerContentIDs))
		}
	} else {
		// Not enough content to compete, or no competeFn — use all
		winnerContentIDs = contentIDs
	}

	// 5. Create batch record with winners only
	trendingJSON, _ := json.Marshal(trendingIDs)
	winnerJSON, _ := json.Marshal(winnerContentIDs)

	batch := &models.DaemonBatch{
		Platform:    platform,
		Status:      models.BatchStatusPending,
		ContentIDs:  string(winnerJSON),
		TrendingIDs: string(trendingJSON),
		BatchType:   "digest",
	}

	batchID, err := d.db.InsertDaemonBatch(batch)
	if err != nil {
		slog.Error("daemon digest insert batch failed", "error", err)
		return
	}
	batch.ID = batchID

	// Link winner content to batch
	for _, cid := range winnerContentIDs {
		if err := d.db.SetGeneratedContentBatchID(cid, batchID); err != nil {
			slog.Error("digest: linking content to batch", "content_id", cid, "error", err)
		}
	}

	d.mu.Lock()
	d.lastBatch[platform] = &batchID
	d.mu.Unlock()

	// 6. Send single Telegram digest notification
	if d.tg != nil && d.cfg.Telegram.ChatID != 0 {
		contents, err := d.db.GetGeneratedContentByBatchID(batchID)
		if err != nil {
			slog.Error("getting digest batch contents for notification", "error", err)
			return
		}

		trendingPosts := d.fetchTrendingPostsForContents(contents)
		now := time.Now()
		msg := telegram.FormatDigestNotification(batch, contents, trendingPosts, rankings, totalCandidates)
		msgID, err := d.tg.SendLongMessageWithMarkdown(ctx, d.cfg.Telegram.ChatID, msg)
		if err != nil {
			slog.Error("sending digest telegram notification", "error", err)
			return
		}

		d.db.UpdateDaemonBatchStatus(batchID, models.BatchStatusNotified, map[string]interface{}{
			"telegram_message_id": msgID,
			"notified_at":        &now,
		})
	}

	slog.Info("daemon digest completed", "platform", platform, "batch_id", batchID, "winners", len(winnerContentIDs), "total_candidates", totalCandidates)

	// 7. Comment generation (if enabled, same as current pipeline step 5)
	if d.cfg.Daemon.CommentsEnabled && d.commentGenerateFn != nil && platform == "linkedin" {
		commentsPerBatch := d.cfg.Daemon.CommentsPerBatch
		if commentsPerBatch <= 0 {
			commentsPerBatch = 3
		}

		commentTrendingIDs := trendingIDs
		if len(commentTrendingIDs) > commentsPerBatch {
			commentTrendingIDs = commentTrendingIDs[:commentsPerBatch]
		}

		commentContentIDs, err := d.commentGenerateFn(ctx, platform, commentTrendingIDs, 1)
		if err != nil {
			slog.Error("daemon digest generate comments failed", "platform", platform, "error", err)
		} else if len(commentContentIDs) > 0 {
			commentTrendingJSON, _ := json.Marshal(commentTrendingIDs)
			commentContentJSON, _ := json.Marshal(commentContentIDs)

			commentBatch := &models.DaemonBatch{
				Platform:    platform,
				Status:      models.BatchStatusPending,
				ContentIDs:  string(commentContentJSON),
				TrendingIDs: string(commentTrendingJSON),
				BatchType:   "comment",
			}

			commentBatchID, err := d.db.InsertDaemonBatch(commentBatch)
			if err != nil {
				slog.Error("daemon digest insert comment batch failed", "error", err)
			} else {
				for _, cid := range commentContentIDs {
					if err := d.db.SetGeneratedContentBatchID(cid, commentBatchID); err != nil {
						slog.Error("digest: linking comment content to batch", "content_id", cid, "error", err)
					}
				}
				slog.Info("daemon digest comment batch created", "platform", platform, "batch_id", commentBatchID, "count", len(commentContentIDs))
			}
		}
	}
}

// runAutoPublishDigest implements the auto-publish flow: action selection → generate → compete → publish → report.
// On action selection failure, falls back to the standard digest flow.
func (d *Daemon) runAutoPublishDigest(ctx context.Context, platform string, trendingIDs []int64) {
	slog.Info("daemon auto-publish digest starting", "platform", platform, "trending_count", len(trendingIDs))

	// 1. Fetch trending posts from DB
	var posts []models.TrendingPost
	idToPost := make(map[int64]models.TrendingPost)
	for _, id := range trendingIDs {
		tp, err := d.db.GetTrendingPostByID("",id)
		if err != nil || tp == nil {
			slog.Warn("auto-publish: skipping trending post", "id", id, "error", err)
			continue
		}
		posts = append(posts, *tp)
		idToPost[id] = *tp
	}
	if len(posts) == 0 {
		slog.Info("auto-publish: no valid trending posts found", "platform", platform)
		return
	}

	// 2. AI action selection
	actionResults, err := d.actionSelectFn(ctx, posts, platform)
	if err != nil {
		slog.Error("auto-publish action selection failed, falling back to standard digest", "platform", platform, "error", err)
		d.runStandardDigest(ctx, platform, trendingIDs)
		return
	}
	slog.Info("auto-publish action selection complete", "platform", platform, "count", len(actionResults))

	// 3. Split into postIDs, repostIDs, commentIDs based on action decisions
	var postIDs, repostIDs, commentIDs []int64
	for i, r := range actionResults {
		if i >= len(posts) {
			break
		}
		id := posts[i].ID
		switch r.Action {
		case "repost":
			repostIDs = append(repostIDs, id)
		case "comment":
			commentIDs = append(commentIDs, id)
		default: // "post" or unknown
			postIDs = append(postIDs, id)
		}
		slog.Info("action selected", "id", id, "action", r.Action, "confidence", r.Confidence, "reasoning", r.Reasoning)
	}

	// 4. Generate content for each group
	var contentIDs []int64
	if len(postIDs) > 0 {
		ids, err := d.generateFn(ctx, platform, postIDs, 1, false)
		if err != nil {
			slog.Error("auto-publish generate posts failed", "platform", platform, "error", err)
		} else {
			contentIDs = append(contentIDs, ids...)
		}
	}
	if len(repostIDs) > 0 {
		ids, err := d.generateFn(ctx, platform, repostIDs, 1, true)
		if err != nil {
			slog.Error("auto-publish generate reposts failed", "platform", platform, "error", err)
		} else {
			contentIDs = append(contentIDs, ids...)
		}
	}
	if len(commentIDs) > 0 && d.commentGenerateFn != nil {
		ids, err := d.commentGenerateFn(ctx, platform, commentIDs, 1)
		if err != nil {
			slog.Error("auto-publish generate comments failed", "platform", platform, "error", err)
		} else {
			contentIDs = append(contentIDs, ids...)
		}
	}

	if len(contentIDs) == 0 {
		slog.Info("auto-publish: no content generated", "platform", platform)
		return
	}

	totalCandidates := len(contentIDs)
	maxWinners := d.cfg.Daemon.AutoPublishMaxPosts

	// 5. Compete to select top winners
	var rankings []models.CompeteResult
	var winnerContentIDs []int64

	if d.competeFn != nil && len(contentIDs) > maxWinners {
		var entries []models.CompeteEntry
		for _, cid := range contentIDs {
			content, err := d.db.GetGeneratedContentByID("", cid)
			if err != nil || content == nil {
				continue
			}
			var tp models.TrendingPost
			if content.SourceTrendingID != 0 {
				if p, ok := idToPost[content.SourceTrendingID]; ok {
					tp = p
				} else if post, err := d.db.GetTrendingPostByID("",content.SourceTrendingID); err == nil && post != nil {
					tp = *post
				}
			}
			entries = append(entries, models.CompeteEntry{
				TrendingPost:     tp,
				GeneratedContent: *content,
			})
		}

		rankings, err = d.competeFn(ctx, entries, maxWinners, platform)
		if err != nil {
			slog.Error("auto-publish competition failed, using first content", "platform", platform, "error", err)
			winnerContentIDs = contentIDs
			if len(winnerContentIDs) > maxWinners {
				winnerContentIDs = winnerContentIDs[:maxWinners]
			}
		} else {
			winnerSet := make(map[int64]bool, len(rankings))
			for _, r := range rankings {
				winnerSet[r.ContentID] = true
				winnerContentIDs = append(winnerContentIDs, r.ContentID)
			}
			// Archive non-winners
			for _, cid := range contentIDs {
				if !winnerSet[cid] {
					d.db.UpdateGeneratedContentStatus("", cid, "archived")
				}
			}
		}
	} else {
		winnerContentIDs = contentIDs
		if len(winnerContentIDs) > maxWinners {
			winnerContentIDs = winnerContentIDs[:maxWinners]
			for _, cid := range contentIDs[maxWinners:] {
				d.db.UpdateGeneratedContentStatus("", cid, "archived")
			}
		}
	}

	// 6. Create batch record
	trendingJSON, _ := json.Marshal(trendingIDs)
	winnerJSON, _ := json.Marshal(winnerContentIDs)

	batch := &models.DaemonBatch{
		Platform:       platform,
		Status:         models.BatchStatusApproved,
		ContentIDs:     string(winnerJSON),
		TrendingIDs:    string(trendingJSON),
		BatchType:      "auto_publish",
		ApprovalSource: "auto",
	}

	batchID, err := d.db.InsertDaemonBatch(batch)
	if err != nil {
		slog.Error("auto-publish insert batch failed", "error", err)
		return
	}
	batch.ID = batchID

	for _, cid := range winnerContentIDs {
		d.db.SetGeneratedContentBatchID(cid, batchID)
	}

	d.mu.Lock()
	d.lastBatch[platform] = &batchID
	d.mu.Unlock()

	// 7. Publish each winner
	var publishResults []models.AutoPublishResult
	allSuccess := true
	for _, cid := range winnerContentIDs {
		// Determine action type from content
		content, _ := d.db.GetGeneratedContentByID("", cid)
		action := "post"
		if content != nil {
			if content.IsComment {
				action = "comment"
			} else if content.IsRepost {
				action = "repost"
			}
		}

		postIDs, err := d.publishFn(ctx, cid)
		if err != nil {
			slog.Error("auto-publish: publishing failed", "content_id", cid, "error", err)
			if platform == "linkedin" && linkedin.IsLinkitinAuthError(err) {
				d.pauseLinkedIn(ctx)
				allSuccess = false
				break
			}
			allSuccess = false
			continue
		}
		publishResults = append(publishResults, models.AutoPublishResult{
			ContentID: cid,
			PostIDs:   postIDs,
			Action:    action,
		})
		slog.Info("auto-publish: published", "content_id", cid, "action", action, "post_ids", postIDs)
	}

	// 8. Update batch status
	now := time.Now()
	if allSuccess && len(publishResults) > 0 {
		d.db.UpdateDaemonBatchStatus(batchID, models.BatchStatusPosted, map[string]interface{}{
			"resolved_at": &now,
		})
	} else if len(publishResults) > 0 {
		d.db.UpdateDaemonBatchStatus(batchID, models.BatchStatusPosted, map[string]interface{}{
			"resolved_at":  &now,
			"error_message": "some items failed to publish",
		})
	} else {
		d.db.UpdateDaemonBatchStatus(batchID, models.BatchStatusFailed, map[string]interface{}{
			"error_message": "all items failed to publish",
		})
	}

	// 9. Send Telegram report
	if d.tg != nil && d.cfg.Telegram.ChatID != 0 && len(publishResults) > 0 {
		contents, err := d.db.GetGeneratedContentByBatchID(batchID)
		if err != nil {
			slog.Error("auto-publish: getting batch contents for report", "error", err)
		} else {
			trendingPosts := d.fetchTrendingPostsForContents(contents)
			msg := telegram.FormatAutoPublishReport(batch, publishResults, contents, trendingPosts, rankings, totalCandidates)
			_, err := d.tg.SendLongMessageWithMarkdown(ctx, d.cfg.Telegram.ChatID, msg)
			if err != nil {
				slog.Error("auto-publish: sending telegram report", "error", err)
			}
		}
	}

	slog.Info("daemon auto-publish digest completed", "platform", platform, "batch_id", batchID,
		"published", len(publishResults), "total_candidates", totalCandidates)
}

// runStandardDigest runs the standard digest flow (classify → generate → compete → notify → await approval).
// This is extracted to allow fallback from auto-publish when action selection fails.
func (d *Daemon) runStandardDigest(ctx context.Context, platform string, trendingIDs []int64) {
	slog.Info("daemon standard digest starting (fallback)", "platform", platform)

	// Classify: rewrite vs repost
	var rewriteIDs, repostIDs []int64
	var err error
	if d.classifyFn != nil {
		rewriteIDs, repostIDs, err = d.classifyFn(ctx, trendingIDs)
		if err != nil {
			slog.Error("daemon digest classify failed, defaulting all to rewrite", "platform", platform, "error", err)
			rewriteIDs = trendingIDs
			repostIDs = nil
		}
	} else {
		rewriteIDs = trendingIDs
	}

	// Generate content
	var contentIDs []int64
	if len(rewriteIDs) > 0 {
		ids, err := d.generateFn(ctx, platform, rewriteIDs, d.cfg.Daemon.MaxPerBatch, false)
		if err != nil {
			slog.Error("daemon digest generate rewrites failed", "platform", platform, "error", err)
		} else {
			contentIDs = append(contentIDs, ids...)
		}
	}
	if len(repostIDs) > 0 {
		ids, err := d.generateFn(ctx, platform, repostIDs, d.cfg.Daemon.MaxPerBatch, true)
		if err != nil {
			slog.Error("daemon digest generate reposts failed", "platform", platform, "error", err)
		} else {
			contentIDs = append(contentIDs, ids...)
		}
	}
	if len(contentIDs) == 0 {
		slog.Info("daemon digest: no content generated", "platform", platform)
		return
	}

	totalCandidates := len(contentIDs)
	maxWinners := totalCandidates / 2
	if maxWinners < 1 {
		maxWinners = 1
	}
	if d.cfg.Daemon.DigestMaxPosts > 0 && d.cfg.Daemon.DigestMaxPosts < maxWinners {
		maxWinners = d.cfg.Daemon.DigestMaxPosts
	}

	// Compete
	var rankings []models.CompeteResult
	var winnerContentIDs []int64

	if d.competeFn != nil && len(contentIDs) > maxWinners {
		var entries []models.CompeteEntry
		for _, cid := range contentIDs {
			content, err := d.db.GetGeneratedContentByID("", cid)
			if err != nil || content == nil {
				continue
			}
			var tp models.TrendingPost
			if content.SourceTrendingID != 0 {
				post, err := d.db.GetTrendingPostByID("",content.SourceTrendingID)
				if err == nil && post != nil {
					tp = *post
				}
			}
			entries = append(entries, models.CompeteEntry{
				TrendingPost:     tp,
				GeneratedContent: *content,
			})
		}

		rankings, err = d.competeFn(ctx, entries, maxWinners, platform)
		if err != nil {
			slog.Error("daemon digest competition failed, using all content", "platform", platform, "error", err)
			winnerContentIDs = contentIDs
		} else {
			winnerSet := make(map[int64]bool, len(rankings))
			for _, r := range rankings {
				winnerSet[r.ContentID] = true
				winnerContentIDs = append(winnerContentIDs, r.ContentID)
			}
			for _, cid := range contentIDs {
				if !winnerSet[cid] {
					d.db.UpdateGeneratedContentStatus("", cid, "archived")
				}
			}
		}
	} else {
		winnerContentIDs = contentIDs
	}

	// Create batch and notify
	trendingJSON, _ := json.Marshal(trendingIDs)
	winnerJSON, _ := json.Marshal(winnerContentIDs)

	batch := &models.DaemonBatch{
		Platform:    platform,
		Status:      models.BatchStatusPending,
		ContentIDs:  string(winnerJSON),
		TrendingIDs: string(trendingJSON),
		BatchType:   "digest",
	}

	batchID, err := d.db.InsertDaemonBatch(batch)
	if err != nil {
		slog.Error("daemon digest insert batch failed", "error", err)
		return
	}
	batch.ID = batchID

	for _, cid := range winnerContentIDs {
		d.db.SetGeneratedContentBatchID(cid, batchID)
	}

	d.mu.Lock()
	d.lastBatch[platform] = &batchID
	d.mu.Unlock()

	if d.tg != nil && d.cfg.Telegram.ChatID != 0 {
		contents, err := d.db.GetGeneratedContentByBatchID(batchID)
		if err != nil {
			slog.Error("getting digest batch contents for notification", "error", err)
			return
		}
		trendingPosts := d.fetchTrendingPostsForContents(contents)
		now := time.Now()
		msg := telegram.FormatDigestNotification(batch, contents, trendingPosts, rankings, totalCandidates)
		msgID, err := d.tg.SendLongMessageWithMarkdown(ctx, d.cfg.Telegram.ChatID, msg)
		if err != nil {
			slog.Error("sending digest telegram notification", "error", err)
			return
		}
		d.db.UpdateDaemonBatchStatus(batchID, models.BatchStatusNotified, map[string]interface{}{
			"telegram_message_id": msgID,
			"notified_at":        &now,
		})
	}

	slog.Info("daemon standard digest completed", "platform", platform, "batch_id", batchID, "winners", len(winnerContentIDs))
}

func (d *Daemon) runPipeline(ctx context.Context, platform string) {
	if platform == "linkedin" && d.isLinkedInPaused() {
		slog.Info("daemon pipeline skipped: linkedin paused (cookies expired)")
		return
	}

	slog.Info("daemon pipeline starting", "platform", platform)

	now := time.Now()
	d.mu.Lock()
	d.lastRun[platform] = &now
	d.mu.Unlock()

	// 1. Discover trending posts (over-fetch 3x to compensate for dedup filtering)
	fetchLimit := d.cfg.Daemon.TrendingLimit
	if d.cfg.Daemon.DedupActionedPosts {
		fetchLimit *= 3
	}
	trendingIDs, err := d.discoverFn(ctx, platform, d.cfg.Daemon.Period, d.cfg.Daemon.MinLikes, fetchLimit)
	if err != nil {
		slog.Error("daemon discover failed", "platform", platform, "error", err)
		if platform == "linkedin" && linkedin.IsLinkitinAuthError(err) {
			d.pauseLinkedIn(ctx)
			return
		}
		if d.tg != nil && d.cfg.Telegram.ChatID != 0 {
			d.sendTelegramReply(ctx, fmt.Sprintf("⚠️ *%s* trending fetch failed: %s", platform, err.Error()))
		}
		return
	}
	if len(trendingIDs) == 0 {
		slog.Info("daemon: no trending posts found", "platform", platform)
		return
	}

	// 1b. Filter out posts the user already acted on
	if d.cfg.Daemon.DedupActionedPosts {
		lookback := time.Duration(d.cfg.Daemon.DedupLookbackHours) * time.Hour
		actioned, err := d.db.GetActionedTrendingIDs("", platform, lookback)
		if err != nil {
			slog.Error("daemon dedup lookup failed, proceeding unfiltered", "platform", platform, "error", err)
		} else if len(actioned) > 0 {
			candidates := len(trendingIDs)
			filtered := trendingIDs[:0]
			for _, id := range trendingIDs {
				if !actioned[id] {
					filtered = append(filtered, id)
				}
			}
			trendingIDs = filtered
			slog.Info("daemon dedup", "platform", platform, "candidates", candidates, "excluded", candidates-len(trendingIDs), "remaining", len(trendingIDs))
		}
	}

	if len(trendingIDs) == 0 {
		slog.Info("daemon: all trending posts already actioned", "platform", platform)
		return
	}

	// In digest mode, stop after discovery — defer generation to the digest cron.
	if d.cfg.Daemon.DigestMode {
		slog.Info("daemon digest mode: trending posts saved, deferring generation", "platform", platform, "count", len(trendingIDs))
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

		trendingPosts := d.fetchTrendingPostsForContents(contents)
		msg := telegram.FormatBatchNotification(batch, contents, trendingPosts)
		msgID, err := d.tg.SendLongMessageWithMarkdown(ctx, d.cfg.Telegram.ChatID, msg)
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

	// 5. Comment batch generation (if enabled)
	if d.cfg.Daemon.CommentsEnabled && d.commentGenerateFn != nil && platform == "linkedin" {
		commentsPerBatch := d.cfg.Daemon.CommentsPerBatch
		if commentsPerBatch <= 0 {
			commentsPerBatch = 3
		}

		// Pick trending posts for commenting (use the same discovered set)
		commentTrendingIDs := trendingIDs
		if len(commentTrendingIDs) > commentsPerBatch {
			commentTrendingIDs = commentTrendingIDs[:commentsPerBatch]
		}

		commentContentIDs, err := d.commentGenerateFn(ctx, platform, commentTrendingIDs, 1)
		if err != nil {
			slog.Error("daemon generate comments failed", "platform", platform, "error", err)
		} else if len(commentContentIDs) > 0 {
			// Create comment batch
			commentTrendingJSON, _ := json.Marshal(commentTrendingIDs)
			commentContentJSON, _ := json.Marshal(commentContentIDs)

			commentBatch := &models.DaemonBatch{
				Platform:    platform,
				Status:      models.BatchStatusPending,
				ContentIDs:  string(commentContentJSON),
				TrendingIDs: string(commentTrendingJSON),
				BatchType:   "comment",
			}

			commentBatchID, err := d.db.InsertDaemonBatch(commentBatch)
			if err != nil {
				slog.Error("daemon insert comment batch failed", "error", err)
			} else {
				commentBatch.ID = commentBatchID

				for _, cid := range commentContentIDs {
					if err := d.db.SetGeneratedContentBatchID(cid, commentBatchID); err != nil {
						slog.Error("linking comment content to batch", "content_id", cid, "error", err)
					}
				}

				// Send Telegram notification for comment batch
				if d.tg != nil && d.cfg.Telegram.ChatID != 0 {
					commentContents, err := d.db.GetGeneratedContentByBatchID(commentBatchID)
					if err != nil {
						slog.Error("getting comment batch contents for notification", "error", err)
					} else {
						trendingPosts := d.fetchTrendingPostsForContents(commentContents)
						msg := telegram.FormatCommentBatchNotification(commentBatch, commentContents, trendingPosts)
						msgID, err := d.tg.SendLongMessageWithMarkdown(ctx, d.cfg.Telegram.ChatID, msg)
						if err != nil {
							slog.Error("sending comment batch telegram notification", "error", err)
						} else {
							d.db.UpdateDaemonBatchStatus(commentBatchID, models.BatchStatusNotified, map[string]interface{}{
								"telegram_message_id": msgID,
								"notified_at":         &now,
							})
						}
					}
				}

				slog.Info("daemon comment batch created", "platform", platform, "batch_id", commentBatchID, "count", len(commentContentIDs))
			}
		}
	}
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
			if isTransientNetErr(err) {
				slog.Warn("polling telegram updates (transient, retrying)", "error", err)
			} else {
				slog.Error("polling telegram updates", "error", err)
			}
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
	contents = orderContentsByBatch(batch, contents)

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

	// Check if the message references a specific batch (e.g. "approve batch 12")
	var batch *models.DaemonBatch
	var err error

	if batchID := extractBatchID(msg.Text); batchID != nil {
		batch, err = d.db.GetDaemonBatch(*batchID)
		if err != nil {
			slog.Error("getting referenced batch", "batch_id", *batchID, "error", err)
			d.sendTelegramReply(ctx, fmt.Sprintf("Failed to find batch %d: %v", *batchID, err))
			return
		}
		if batch == nil {
			d.sendTelegramReply(ctx, fmt.Sprintf("Batch %d not found.", *batchID))
			return
		}
		slog.Info("standalone command targeting specific batch", "batch_id", batch.ID, "status", batch.Status)
	} else {
		// No batch reference — fall back to the most recent active batch
		batch, err = d.db.GetLatestActiveDaemonBatch()
		if err != nil {
			slog.Error("finding latest active batch for standalone command", "error", err)
			d.sendTelegramReply(ctx, fmt.Sprintf("Failed to find active batch: %v", err))
			return
		}
		if batch == nil {
			d.sendTelegramReply(ctx, "No active batch found. Run the pipeline first.")
			return
		}
		slog.Info("standalone command matched latest batch", "batch_id", batch.ID, "status", batch.Status)
	}

	contents, err := d.db.GetGeneratedContentByBatchID(batch.ID)
	if err != nil {
		slog.Error("getting batch contents for standalone command", "error", err)
		d.sendTelegramReply(ctx, fmt.Sprintf("Failed to load batch contents: %v", err))
		return
	}
	contents = orderContentsByBatch(batch, contents)

	// Reuse existing intent parser (regex fast-path + Claude AI fallback)
	intent, err := d.intentParser.Parse(ctx, batch, contents, msg.Text)
	if err != nil {
		slog.Error("parsing standalone command intent", "batch_id", batch.ID, "error", err)
		d.sendTelegramReply(ctx, fmt.Sprintf("Could not understand command: %v", err))
		return
	}

	slog.Info("standalone command intent parsed", "batch_id", batch.ID, "action", intent.Action)

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

		// Validate X post length before publishing
		if batch.Platform == "x" {
			for _, cid := range contentIDs {
				c, err := d.db.GetGeneratedContentByID("", cid)
				if err != nil {
					return fmt.Errorf("fetching content %d for validation: %w", cid, err)
				}
				if c != nil && len([]rune(c.GeneratedContent)) > 280 {
					errMsg := fmt.Sprintf("Draft %d exceeds 280 characters (%d chars). Edit it shorter before approving.", cid, len([]rune(c.GeneratedContent)))
					d.db.UpdateDaemonBatchStatus(batch.ID, models.BatchStatusNotified, nil)
					d.sendTelegramReply(ctx, errMsg)
					return fmt.Errorf("content %d exceeds X 280-char limit", cid)
				}
			}
		}

		// Post each content item
		var allPostIDs []string
		for _, cid := range contentIDs {
			postIDs, err := d.publishFn(ctx, cid)
			if err != nil {
				slog.Error("publishing content", "content_id", cid, "error", err)
				if batch.Platform == "linkedin" && linkedin.IsLinkitinAuthError(err) {
					d.db.UpdateDaemonBatchStatus(batch.ID, models.BatchStatusFailed, map[string]interface{}{
						"error_message": err.Error(),
						"auth_expired":  true,
					})
					d.pauseLinkedIn(ctx)
					return err
				}
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
			if err := d.db.UpdateGeneratedContentText("", contentID, newText); err != nil {
				slog.Error("applying edit", "content_id", contentID, "error", err)
			}
		}

		// Re-notify with updated content
		contents, _ := d.db.GetGeneratedContentByBatchID(batch.ID)
		d.db.UpdateDaemonBatchStatus(batch.ID, models.BatchStatusNotified, nil)

		if d.tg != nil && d.cfg.Telegram.ChatID != 0 {
			trendingPosts := d.fetchTrendingPostsForContents(contents)
			msg := telegram.FormatBatchNotification(batch, contents, trendingPosts)
			msgID, err := d.tg.SendLongMessageWithMarkdown(ctx, d.cfg.Telegram.ChatID, msg)
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
			id, err := d.db.InsertScheduledPost("", cid, *intent.ScheduleAt)
			if err != nil {
				slog.Error("scheduling content", "content_id", cid, "error", err)
				continue
			}
			method := "pending"
			if d.scheduleFn != nil {
				schedID, err := d.scheduleFn(ctx, cid, *intent.ScheduleAt)
				if err == nil {
					d.db.UpdateScheduledPostStatus(id, "scheduled", "")
					method = "native"
					_ = schedID
				} else {
					slog.Warn("native scheduling failed, using pending fallback", "content_id", cid, "platform", batch.Platform, "error", err)
				}
			}
			slog.Info("content scheduled", "content_id", cid, "platform", batch.Platform, "method", method, "at", intent.ScheduleAt)
		}

		if d.tg != nil && d.cfg.Telegram.ChatID != 0 {
			d.sendTelegramReply(ctx, fmt.Sprintf("Batch #%d scheduled for %s on %s", batch.ID, intent.ScheduleAt.Format(time.RFC822), strings.ToUpper(batch.Platform)))
		}

	case "rewrite":
		if len(intent.ContentIDs) == 0 {
			return fmt.Errorf("rewrite requires a draft number")
		}
		if d.generator == nil {
			return fmt.Errorf("rewrite requires Claude API key to be configured")
		}

		content, err := d.db.GetGeneratedContentByID("", intent.ContentIDs[0])
		if err != nil {
			return fmt.Errorf("fetching content for rewrite: %w", err)
		}
		if content == nil {
			return fmt.Errorf("content not found")
		}

		// Fetch source trending post
		var tp models.TrendingPost
		if content.SourceTrendingID != 0 {
			post, err := d.db.GetTrendingPostByID("",content.SourceTrendingID)
			if err != nil {
				return fmt.Errorf("fetching trending post for rewrite: %w", err)
			}
			if post != nil {
				tp = *post
			}
		}

		// Fetch persona for the platform
		var persona models.Persona
		p, err := d.db.GetPersona("", batch.Platform)
		if err != nil {
			slog.Warn("fetching persona for rewrite, using empty", "error", err)
		}
		if p != nil {
			persona = *p
		}

		// Determine effective isRepost: use toggle if provided, otherwise preserve existing
		isRepost := content.IsRepost
		if intent.IsRepost != nil {
			isRepost = *intent.IsRepost
			if isRepost != content.IsRepost {
				if err := d.db.UpdateGeneratedContentIsRepost("", intent.ContentIDs[0], isRepost); err != nil {
					slog.Error("updating is_repost flag", "content_id", intent.ContentIDs[0], "error", err)
				}
				// Set or clear quote_tweet_id when toggling repost mode
				if isRepost {
					if err := d.db.UpdateGeneratedContentQuoteTweetID("", intent.ContentIDs[0], tp.PlatformPostID); err != nil {
						slog.Error("setting quote_tweet_id", "content_id", intent.ContentIDs[0], "error", err)
					}
				} else {
					if err := d.db.UpdateGeneratedContentQuoteTweetID("", intent.ContentIDs[0], ""); err != nil {
						slog.Error("clearing quote_tweet_id", "content_id", intent.ContentIDs[0], "error", err)
					}
				}
			}
		}

		// Determine platform-specific settings
		niches := d.cfg.Niches
		if batch.Platform == "linkedin" && len(d.cfg.LinkedInNiches) > 0 {
			niches = d.cfg.LinkedInNiches
		}
		maxChars := 0
		if batch.Platform == "x" {
			maxChars = 280
		}

		results, err := d.generator.Generate(ctx, models.GenerateRequest{
			TrendingPost:   tp,
			Persona:        persona,
			TargetPlatform: batch.Platform,
			Niches:         niches,
			Count:          1,
			MaxChars:       maxChars,
			IsRepost:       isRepost,
			StyleDirection: intent.Message,
		})
		if err != nil {
			return fmt.Errorf("rewriting draft: %w", err)
		}
		if len(results) == 0 {
			return fmt.Errorf("rewrite produced no results")
		}

		// Update draft text in DB
		if err := d.db.UpdateGeneratedContentText("", intent.ContentIDs[0], results[0].Content); err != nil {
			return fmt.Errorf("saving rewritten content: %w", err)
		}

		// Re-notify with updated content (same pattern as edit case)
		contents, _ := d.db.GetGeneratedContentByBatchID(batch.ID)
		d.db.UpdateDaemonBatchStatus(batch.ID, models.BatchStatusNotified, nil)

		if d.tg != nil && d.cfg.Telegram.ChatID != 0 {
			trendingPosts := d.fetchTrendingPostsForContents(contents)
			msg := telegram.FormatBatchNotification(batch, contents, trendingPosts)
			msgID, err := d.tg.SendLongMessageWithMarkdown(ctx, d.cfg.Telegram.ChatID, msg)
			if err == nil {
				d.db.UpdateDaemonBatchStatus(batch.ID, models.BatchStatusNotified, map[string]interface{}{
					"telegram_message_id": msgID,
				})
			}
		}

	case "read":
		if len(intent.ContentIDs) == 0 {
			return fmt.Errorf("read requires a draft number")
		}
		content, err := d.db.GetGeneratedContentByID("", intent.ContentIDs[0])
		if err != nil {
			return fmt.Errorf("fetching content for read: %w", err)
		}
		if content == nil {
			return fmt.Errorf("content not found")
		}

		// Resolve the 1-indexed draft display number
		contents, err := d.db.GetGeneratedContentByBatchID(batch.ID)
		if err != nil {
			return fmt.Errorf("fetching batch contents for draft number: %w", err)
		}
		draftNum := 1
		for i, c := range contents {
			if c.ID == intent.ContentIDs[0] {
				draftNum = i + 1
				break
			}
		}

		// Fetch source trending post (may be nil)
		var tp *models.TrendingPost
		if content.SourceTrendingID != 0 {
			tp, err = d.db.GetTrendingPostByID("",content.SourceTrendingID)
			if err != nil {
				slog.Warn("fetching trending post for read", "error", err)
			}
		}

		d.sendTelegramReply(ctx, telegram.FormatDraftDetail(draftNum, tp, content))

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
	_, err := d.tg.SendLongMessageWithMarkdown(ctx, d.cfg.Telegram.ChatID, text)
	if err != nil {
		slog.Error("sending telegram reply", "error", err)
	}
}

// fetchTrendingPostsForContents collects unique SourceTrendingID values from contents
// and fetches the corresponding TrendingPost records from the database.
func (d *Daemon) fetchTrendingPostsForContents(contents []models.GeneratedContent) map[int64]models.TrendingPost {
	result := make(map[int64]models.TrendingPost)
	for _, c := range contents {
		if c.SourceTrendingID == 0 {
			continue
		}
		if _, ok := result[c.SourceTrendingID]; ok {
			continue
		}
		tp, err := d.db.GetTrendingPostByID("",c.SourceTrendingID)
		if err != nil {
			slog.Warn("fetching trending post for notification", "trending_id", c.SourceTrendingID, "error", err)
			continue
		}
		if tp != nil {
			result[c.SourceTrendingID] = *tp
		}
	}
	return result
}

// orderContentsByBatch reorders contents to match the display order stored
// in the batch's content_ids JSON array. Falls back to the original order
// if parsing fails or IDs don't match.
func orderContentsByBatch(batch *models.DaemonBatch, contents []models.GeneratedContent) []models.GeneratedContent {
	var orderedIDs []int64
	if err := json.Unmarshal([]byte(batch.ContentIDs), &orderedIDs); err != nil || len(orderedIDs) == 0 {
		return contents
	}

	byID := make(map[int64]models.GeneratedContent, len(contents))
	for _, c := range contents {
		byID[c.ID] = c
	}

	ordered := make([]models.GeneratedContent, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		if c, ok := byID[id]; ok {
			ordered = append(ordered, c)
		}
	}

	if len(ordered) != len(contents) {
		return contents // fallback if mismatch
	}
	return ordered
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// isTransientNetErr returns true for network errors that are expected to
// self-resolve (connection resets, timeouts, etc.).
func isTransientNetErr(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	msg := err.Error()
	return strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "network is unreachable")
}
