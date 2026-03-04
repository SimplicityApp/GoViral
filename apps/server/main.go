package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/handler"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/apps/server/service"
	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/internal/ai/generator"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/daemon"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/internal/platform/linkedin"
	"github.com/shuhao/goviral/internal/platform/x"
	"github.com/shuhao/goviral/pkg/models"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		slog.Error("loading config", "error", err)
		os.Exit(1)
	}

	database, err := db.New(cfg.DBPath)
	if err != nil {
		slog.Error("opening database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	srv := NewServer(cfg, database)

	// Create daemon with adapter functions
	generateSvc := service.NewGenerateService(database, cfg)
	publishSvc := service.NewPublishService(database, cfg)

	d := daemon.New(cfg, database,
		makeDaemonGenerateFn(generateSvc),
		makeDaemonPublishFn(publishSvc),
		makeDaemonDiscoverFn(database, cfg),
		makeDaemonClassifyFn(database, cfg),
		makeDaemonCommentGenerateFn(generateSvc),
		makeDaemonCompeteFn(cfg),
		makeDaemonScheduleFn(publishSvc),
		makeDaemonActionSelectFn(database, cfg),
	)

	setupRoutes(srv, d)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start daemon if enabled
	if cfg.Daemon.Enabled && cfg.Telegram.BotToken != "" {
		if err := d.Start(ctx); err != nil {
			slog.Error("starting daemon", "error", err)
		}
	}

	// Background goroutine: execute pending scheduled posts as they become due
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pending, err := database.GetPendingScheduledPosts()
				if err != nil || len(pending) == 0 {
					continue
				}
				for _, sp := range pending {
					postIDs, _, err := publishSvc.Publish(ctx, sp.UserID, sp.GeneratedContentID, false)
					if err != nil {
						slog.Error("running due scheduled post", "schedule_id", sp.ID, "error", err)
						database.UpdateScheduledPostStatus(sp.ID, "failed", err.Error())
					} else {
						slog.Info("ran due scheduled post", "schedule_id", sp.ID, "post_ids", postIDs)
						database.UpdateScheduledPostStatus(sp.ID, "posted", "")
					}
				}
			}
		}
	}()

	go func() {
		if err := srv.Start(); err != nil {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received")

	d.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}

// setupRoutes configures all middleware and API route groups.
func setupRoutes(s *Server, d *daemon.Daemon) {
	s.Router.Use(
		middleware.Recovery,
		middleware.Logging,
		middleware.CORS(s.Cfg.Server.AllowedOrigins),
	)

	// Services
	postsSvc := service.NewPostsService(s.DB)
	trendingSvc := service.NewTrendingService(s.DB)
	personaSvc := service.NewPersonaService(s.DB)
	scheduleSvc := service.NewScheduleService(s.DB)
	publishSvc := service.NewPublishService(s.DB, s.Cfg)
	generateSvc := service.NewGenerateService(s.DB, s.Cfg)
	opStore := service.NewOperationStore(30 * time.Minute)
	repoSvc := service.NewRepoService(s.DB, s.Cfg)
	repoH := handler.NewRepoHandler(repoSvc, opStore)

	// Read handlers
	healthH := handler.NewHealthHandler(s.Cfg)
	postsH := handler.NewPostsHandler(postsSvc)
	trendingH := handler.NewTrendingHandler(trendingSvc)
	personaH := handler.NewPersonaHandler(personaSvc)
	historyH := handler.NewHistoryHandler(s.DB)
	scheduleH := handler.NewScheduleHandler(scheduleSvc, s.DB)
	configH := handler.NewConfigHandler(s.Cfg)

	// Write handlers
	publishH := handler.NewPublishHandler(publishSvc)
	scheduleWriteH := handler.NewScheduleWriteHandler(s.DB, publishSvc)
	historyWriteH := handler.NewHistoryWriteHandler(s.DB)
	configWriteH := handler.NewConfigWriteHandler(s.Cfg)

	// Long-running operation handlers
	operationsH := handler.NewOperationsHandler(opStore)
	fetchPostsH := handler.NewFetchPostsHandler(s.DB, s.Cfg, opStore)
	discoverH := handler.NewDiscoverTrendingHandler(s.DB, s.Cfg, opStore)
	generateH := handler.NewGenerateHandler(generateSvc, opStore)
	buildPersonaH := handler.NewBuildPersonaHandler(s.DB, s.Cfg, opStore)

	// Video upload handlers
	youtubeH := handler.NewYouTubeHandler(publishSvc)
	tiktokH := handler.NewTikTokHandler(publishSvc)

	// Comment handler
	commentH := handler.NewCommentHandler(publishSvc, generateSvc, s.DB)

	// Cookie management handlers
	xCookiesH := handler.NewXCookiesHandler(s.Cfg)
	linkedinCookiesH := handler.NewLinkedInCookiesHandler(s.Cfg)

	// Extension download handler
	extensionH := handler.NewExtensionHandler(extensionFS())

	// Auth handler
	authH := handler.NewAuthHandler(s.Cfg)

	// Daemon handler
	daemonH := handler.NewDaemonHandler(d, s.DB, s.Cfg)

	// Telegram webhook (unauthenticated, validated by secret)
	if s.Cfg.Telegram.BotToken != "" {
		secret := handler.WebhookSecret(s.Cfg.Telegram.BotToken)
		s.Router.Post("/api/v1/telegram/webhook/"+secret, daemonH.TelegramWebhook)
	}

	// Public (unauthenticated) endpoints
	s.Router.Get("/api/v1/extension/download", extensionH.Download)

	s.Router.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.UserID(s.DB.GetOrCreateUser))

		// Read-only endpoints
		r.Get("/health", healthH.Get)
		r.Get("/posts", postsH.List)

		r.Get("/trending", trendingH.List)
		r.Get("/trending/{id}", trendingH.GetByID)

		r.Get("/persona", personaH.Get)

		r.Get("/history", historyH.List)
		r.Get("/history/{id}", historyH.GetByID)

		r.Get("/schedule", scheduleH.List)

		r.Get("/config", configH.Get)

		r.Get("/operations/{id}", operationsH.Get)

		// Write endpoints
		r.Post("/publish", publishH.Post)
		r.Post("/x/publish", publishH.PostX)
		r.Post("/linkedin/publish", publishH.PostLinkedIn)
		r.Post("/youtube/publish", publishH.PostYouTube)
		r.Post("/tiktok/publish", publishH.PostTikTok)

		r.Post("/schedule", scheduleWriteH.Create)
		r.Patch("/schedule/{id}/ack", scheduleWriteH.Acknowledge)
		r.Delete("/schedule/{id}", scheduleWriteH.Delete)
		r.Post("/schedule/run", scheduleWriteH.RunDue)

		r.Patch("/history/{id}", historyWriteH.UpdateStatus)
		r.Delete("/history/{id}", historyWriteH.Delete)

		r.Patch("/config", configWriteH.Update)

		// Ingest endpoints (Chrome extension)
		ingestPostsH := handler.NewIngestPostsHandler(s.DB)
		ingestTrendingH := handler.NewIngestTrendingHandler(s.DB)
		r.Post("/posts/ingest", ingestPostsH.Post)
		r.Post("/trending/ingest", ingestTrendingH.Post)

		// Long-running operations (SSE or 202)
		r.Post("/posts/fetch", fetchPostsH.Post)
		r.Post("/trending/discover", discoverH.Post)
		r.Post("/generate", generateH.Post)
		r.Post("/persona/build", buildPersonaH.Post)

		// LinkedIn comments
		r.Post("/linkedin/comment/generate", commentH.GenerateComment)
		r.Post("/linkedin/comment", commentH.PostComment)

		// X comments
		r.Post("/x/comment/generate", commentH.GenerateComment)
		r.Post("/x/comment", commentH.PostComment)

		// YouTube video upload
		r.Post("/youtube/upload", youtubeH.Upload)

		// TikTok video upload
		r.Post("/tiktok/upload", tiktokH.Upload)

		// X cookie management
		r.Post("/x/extract-cookies", xCookiesH.ExtractCookies)
		r.Post("/x/login-cookies", xCookiesH.LoginCookies)
		r.Get("/x/cookies/status", xCookiesH.Status)

		// LinkedIn cookie management
		r.Post("/linkedin/extract-cookies", linkedinCookiesH.ExtractCookies)
		r.Post("/linkedin/login-cookies", linkedinCookiesH.LoginCookies)
		r.Get("/linkedin/cookies/status", linkedinCookiesH.Status)

		// OAuth flow
		r.Post("/auth/{platform}/start", authH.Start)
		r.Get("/auth/{platform}/status", authH.Status)

		// Daemon endpoints
		r.Get("/daemon/status", daemonH.GetStatus)
		r.Get("/daemon/batches", daemonH.ListBatches)
		r.Get("/daemon/batches/{id}", daemonH.GetBatch)
		r.Post("/daemon/batches/{id}/action", daemonH.BatchAction)
		r.Post("/daemon/run", daemonH.RunNow)
		r.Post("/daemon/digest", daemonH.RunDigestNow)
		r.Get("/daemon/config", daemonH.GetConfig)
		r.Patch("/daemon/config", daemonH.UpdateConfig)
		r.Post("/daemon/start", daemonH.StartDaemon)
		r.Post("/daemon/stop", daemonH.StopDaemon)

		// Repo-to-post endpoints
		r.Get("/repos", repoH.ListRepos)
		r.Post("/repos", repoH.AddRepo)
		r.Get("/repos/available", repoH.ListAvailableRepos)
		r.Delete("/repos/{id}", repoH.DeleteRepo)
		r.Patch("/repos/{id}/settings", repoH.UpdateSettings)
		r.Get("/repos/{id}/commits", repoH.ListCommits)
		r.Post("/repos/{id}/fetch", repoH.FetchCommits)
		r.Post("/repos/generate", repoH.GeneratePosts)
		r.Get("/repos/code-image-options", repoH.ListCodeImageOptions)
		r.Get("/repos/code-image-previews", repoH.ListCodeImagePreviews)
		r.Post("/repos/code-image", repoH.RenderCodeImage)
		r.Get("/repos/commits/{commitId}/image", repoH.GetCodeImage)
		r.Get("/content/{contentId}/code-image", repoH.GetContentCodeImage)
		r.Post("/content/{contentId}/re-render-code-image", repoH.ReRenderContentCodeImage)
		r.Get("/content/{id}/image", historyH.GetContentImage)
	})

	// Static file serving (SPA fallback)
	s.Router.Handle("/*", staticHandler())
}

// --- Daemon adapter functions ---

func makeDaemonGenerateFn(svc *service.GenerateService) daemon.GenerateFunc {
	return func(ctx context.Context, platform string, trendingIDs []int64, count int, isRepost bool) ([]int64, error) {
		progress := make(chan dto.ProgressEvent, 10)
		go func() {
			for range progress {
			}
		}()

		contents, err := svc.Generate(ctx, "", dto.GenerateRequest{
			TrendingPostIDs: trendingIDs,
			TargetPlatform:  platform,
			Count:           count,
			IsRepost:        isRepost,
		}, progress)
		if err != nil {
			return nil, err
		}

		var ids []int64
		for _, c := range contents {
			ids = append(ids, c.ID)
		}
		return ids, nil
	}
}

func makeDaemonPublishFn(svc *service.PublishService) daemon.PublishFunc {
	return func(ctx context.Context, contentID int64) ([]string, error) {
		postIDs, _, err := svc.Publish(ctx, "", contentID, false)
		return postIDs, err
	}
}

func makeDaemonDiscoverFn(database *db.DB, cfg *config.Config) daemon.DiscoverFunc {
	return func(ctx context.Context, platform string, period string, minLikes, limit int) ([]int64, error) {
		var niches []string
		switch platform {
		case "x":
			niches = cfg.Niches
		case "linkedin":
			niches = cfg.LinkedInNiches
			if len(niches) == 0 {
				niches = []string{"AI", "Programming", "Technology"}
			}
		}
		if len(niches) == 0 {
			return nil, fmt.Errorf("no niches configured for %s", platform)
		}

		var posts []models.TrendingPost
		var err error
		switch platform {
		case "x":
			client := x.NewFallbackClient(cfg.X)
			posts, err = client.FetchTrendingPosts(ctx, niches, period, minLikes, limit)
		case "linkedin":
			client := linkedin.NewFallbackClient(cfg.LinkedIn, nil)
			posts, err = client.FetchTrendingPosts(ctx, niches, period, minLikes, limit)
		default:
			return nil, fmt.Errorf("unsupported platform: %s", platform)
		}
		if err != nil {
			return nil, fmt.Errorf("discovering trending on %s: %w", platform, err)
		}

		var ids []int64
		for _, tp := range posts {
			tp := tp
			if err := database.UpsertTrendingPost(&tp); err != nil {
				slog.Error("saving trending post", "platform", platform, "error", err)
				continue
			}
			ids = append(ids, tp.ID)
		}
		return ids, nil
	}
}

func makeDaemonCommentGenerateFn(svc *service.GenerateService) daemon.CommentGenerateFunc {
	return func(ctx context.Context, platform string, trendingIDs []int64, count int) ([]int64, error) {
		var allIDs []int64
		for _, tpID := range trendingIDs {
			contents, err := svc.GenerateComment(ctx, "", tpID, platform, count)
			if err != nil {
				slog.Error("generating comment for trending post", "trending_id", tpID, "error", err)
				continue
			}
			for _, c := range contents {
				allIDs = append(allIDs, c.ID)
			}
		}
		return allIDs, nil
	}
}

func makeDaemonCompeteFn(cfg *config.Config) daemon.CompeteFunc {
	if cfg.Claude.APIKey == "" {
		return nil
	}
	return func(ctx context.Context, entries []models.CompeteEntry, maxWinners int, platform string) ([]models.CompeteResult, error) {
		claudeClient := claude.NewClient(cfg.Claude.APIKey, cfg.Claude.Model)
		gen := generator.NewGenerator(claudeClient)
		return gen.CompeteContent(ctx, entries, maxWinners, platform)
	}
}

func makeDaemonScheduleFn(publishSvc *service.PublishService) daemon.ScheduleFunc {
	return func(ctx context.Context, contentID int64, scheduledAt time.Time) (string, error) {
		return publishSvc.Schedule(ctx, "", contentID, scheduledAt)
	}
}

func makeDaemonActionSelectFn(database *db.DB, cfg *config.Config) daemon.ActionSelectFunc {
	if cfg.Claude.APIKey == "" {
		return nil
	}
	return func(ctx context.Context, posts []models.TrendingPost, platform string) ([]models.ActionSelectResult, error) {
		claudeClient := claude.NewClient(cfg.Claude.APIKey, cfg.Claude.Model)
		gen := generator.NewGenerator(claudeClient)
		return gen.SelectActions(ctx, posts, platform)
	}
}

func makeDaemonClassifyFn(database *db.DB, cfg *config.Config) daemon.ClassifyFunc {
	if cfg.Claude.APIKey == "" {
		return nil
	}
	return func(ctx context.Context, trendingIDs []int64) (rewriteIDs, repostIDs []int64, err error) {
		// Fetch trending posts from DB
		var posts []models.TrendingPost
		for _, id := range trendingIDs {
			tp, err := database.GetTrendingPostByID(id)
			if err != nil {
				slog.Error("fetching trending post for classification", "id", id, "error", err)
				continue
			}
			if tp != nil {
				posts = append(posts, *tp)
			}
		}
		if len(posts) == 0 {
			return trendingIDs, nil, nil // default all to rewrite
		}

		claudeClient := claude.NewClient(cfg.Claude.APIKey, cfg.Claude.Model)
		gen := generator.NewGenerator(claudeClient)

		results, err := gen.ClassifyPosts(ctx, posts)
		if err != nil {
			return trendingIDs, nil, nil // on error, default all to rewrite
		}

		// Map results back to IDs
		for i, r := range results {
			if i >= len(posts) {
				break
			}
			id := posts[i].ID
			if r.Decision == "repost" {
				repostIDs = append(repostIDs, id)
			} else {
				rewriteIDs = append(rewriteIDs, id)
			}
			slog.Info("classified trending post", "id", id, "decision", r.Decision, "confidence", r.Confidence, "reasoning", r.Reasoning)
		}

		return rewriteIDs, repostIDs, nil
	}
}
