package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/shuhao/goviral/internal/auth"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/internal/platform/x"
	"github.com/shuhao/goviral/internal/thread"
	"github.com/shuhao/goviral/pkg/models"
)

var (
	postID           int64
	postNumbered     bool
	postAt           string
	postScheduled    bool
	postRunScheduled bool
	postDryRun       bool
	postVideo        string
	postThumbnail    string
)

var postCmd = &cobra.Command{
	Use:   "post",
	Short: "Post or schedule generated content to X, LinkedIn, YouTube, or TikTok",
	Long: `Post generated content to social media platforms.
Supports X threads, LinkedIn posts, YouTube Shorts, and TikTok videos.

Examples:
  goviral post                          # Interactive selection
  goviral post --id 5                   # Post specific content
  goviral post --id 5 --dry-run         # Preview thread splitting
  goviral post --id 5 --at "2025-03-01 09:00"  # Schedule for later
  goviral post --id 5 --video /path/to/video.mp4  # Post with video
  goviral post --scheduled              # List pending scheduled posts
  goviral post --run-scheduled          # Post due scheduled posts (for cron)`,
	RunE: runPost,
}

func init() {
	postCmd.Flags().Int64Var(&postID, "id", 0, "Post specific generated content by ID")
	postCmd.Flags().BoolVar(&postNumbered, "numbered", true, "Add thread numbering (1/N)")
	postCmd.Flags().StringVar(&postAt, "at", "", "Schedule for later (e.g., \"2025-03-01 09:00\")")
	postCmd.Flags().BoolVar(&postScheduled, "scheduled", false, "List pending scheduled posts")
	postCmd.Flags().BoolVar(&postRunScheduled, "run-scheduled", false, "Post any due scheduled posts")
	postCmd.Flags().BoolVar(&postDryRun, "dry-run", false, "Preview thread splitting without posting")
	postCmd.Flags().StringVar(&postVideo, "video", "", "Path to video file (for YouTube/TikTok)")
	postCmd.Flags().StringVar(&postThumbnail, "thumbnail", "", "Path to thumbnail image (for YouTube)")
	rootCmd.AddCommand(postCmd)
}

func runPost(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	if postScheduled {
		return listScheduledPosts(database)
	}

	if postRunScheduled {
		return runScheduledPosts(cfg, database)
	}

	// Get content to post
	var gc *models.GeneratedContent
	if postID > 0 {
		gc, err = database.GetGeneratedContentByID("", postID)
		if err != nil {
			return fmt.Errorf("fetching content: %w", err)
		}
		if gc == nil {
			return fmt.Errorf("no generated content found with ID %d", postID)
		}
	} else {
		gc, err = interactiveContentSelect(database)
		if err != nil {
			return err
		}
		if gc == nil {
			fmt.Println("No content selected.")
			return nil
		}
	}

	if gc.Status == "posted" {
		return fmt.Errorf("content ID %d has already been posted (tweet IDs: %s)", gc.ID, gc.PlatformPostIDs)
	}

	isQuoteTweet := gc.IsRepost && gc.QuoteTweetID != ""

	if isQuoteTweet {
		// Quote tweets are a single tweet, no threading
		displayThreadPreview([]string{gc.GeneratedContent})
		repostStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
		fmt.Println(repostStyle.Render(fmt.Sprintf("Quoting: https://x.com/i/status/%s", gc.QuoteTweetID)))
	} else {
		// Split into thread parts
		result := thread.Split(gc.GeneratedContent, postNumbered)
		displayThreadPreview(result.Parts)
	}

	// Handle scheduling
	if postAt != "" {
		return schedulePost(database, gc.ID, postAt)
	}

	// Show image info if present
	if gc.ImagePath != "" {
		imgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
		fmt.Println(imgStyle.Render(fmt.Sprintf("Image: %s", gc.ImagePath)))
	}

	// Handle dry run
	if postDryRun {
		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
		fmt.Println(infoStyle.Render("(dry run — nothing was posted)"))
		return nil
	}

	// Validate and refresh access token if needed
	if err := ensureFreshToken(cfg); err != nil {
		return err
	}

	// Confirm
	fmt.Print("Post this content? (y/N) ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(line)) != "y" {
		fmt.Println("Cancelled.")
		return nil
	}

	// Post
	client := x.NewFallbackClient(cfg.X)

	if isQuoteTweet {
		tweetID, err := postQuoteTweet(context.Background(), client, gc.GeneratedContent, gc.QuoteTweetID)
		if err != nil {
			if strings.Contains(err.Error(), "401") {
				return fmt.Errorf("authentication failed; try re-running 'goviral auth x': %w", err)
			}
			return fmt.Errorf("posting quote tweet: %w", err)
		}

		if err := database.UpdateGeneratedContentPosted("", gc.ID, tweetID); err != nil {
			return fmt.Errorf("updating database: %w", err)
		}

		successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
		fmt.Println(successStyle.Render("\nQuote tweet posted successfully!"))
		fmt.Printf("View: https://x.com/i/status/%s\n", tweetID)
	} else {
		result := thread.Split(gc.GeneratedContent, postNumbered)

		// Upload media if image is attached
		var mediaIDs []string
		if gc.ImagePath != "" {
			imageData, err := os.ReadFile(gc.ImagePath)
			if err != nil {
				fmt.Printf("  Warning: failed to read image %s: %v\n", gc.ImagePath, err)
			} else {
				imgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
				fmt.Println(imgStyle.Render("Uploading image..."))

				mediaID, err := client.UploadMedia(context.Background(), imageData, "image/png")
				if err != nil {
					fmt.Printf("  Warning: media upload failed: %v\n", err)
				} else {
					mediaIDs = append(mediaIDs, mediaID)
					fmt.Println(imgStyle.Render(fmt.Sprintf("Media uploaded: %s", mediaID)))
				}
			}
		}

		tweetIDs, err := postThread(context.Background(), client, result.Parts, mediaIDs)
		if err != nil {
			// Partial failure: save what was posted
			if len(tweetIDs) > 0 {
				partialIDs := strings.Join(tweetIDs, ",")
				_ = database.UpdateGeneratedContentPosted("", gc.ID, partialIDs)
				warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
				fmt.Println(warnStyle.Render(fmt.Sprintf("Partial thread posted (%d/%d tweets). IDs: %s", len(tweetIDs), len(result.Parts), partialIDs)))
			}
			if strings.Contains(err.Error(), "401") {
				return fmt.Errorf("authentication failed; try re-running 'goviral auth x': %w", err)
			}
			return fmt.Errorf("posting thread: %w", err)
		}

		// Update DB
		allIDs := strings.Join(tweetIDs, ",")
		if err := database.UpdateGeneratedContentPosted("", gc.ID, allIDs); err != nil {
			return fmt.Errorf("updating database: %w", err)
		}

		// Display success
		successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
		fmt.Println(successStyle.Render(fmt.Sprintf("\nPosted successfully! %d tweet(s)", len(tweetIDs))))
		fmt.Printf("View: https://x.com/i/status/%s\n", tweetIDs[0])
	}

	return nil
}

func interactiveContentSelect(database *db.DB) (*models.GeneratedContent, error) {
	// Fetch draft and approved content
	drafts, err := database.GetGeneratedContent("", "draft", "", 0)
	if err != nil {
		return nil, fmt.Errorf("fetching drafts: %w", err)
	}
	approved, err := database.GetGeneratedContent("", "approved", "", 0)
	if err != nil {
		return nil, fmt.Errorf("fetching approved content: %w", err)
	}

	all := append(approved, drafts...)
	if len(all) == 0 {
		return nil, fmt.Errorf("no draft or approved content found; run 'goviral generate' first")
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	fmt.Println(headerStyle.Render("\nAvailable content:"))
	repostStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	for i, gc := range all {
		preview := gc.GeneratedContent
		if len(preview) > 80 {
			preview = preview[:77] + "..."
		}
		status := statusStyle.Render(fmt.Sprintf("[%s]", gc.Status))
		repostTag := ""
		if gc.IsRepost {
			repostTag = repostStyle.Render(" [REPOST]")
		}
		fmt.Printf("  [%d] %s%s ID:%d — %s\n", i+1, status, repostTag, gc.ID, preview)
	}

	fmt.Print("\nSelect content (number): ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)

	var idx int
	if _, err := fmt.Sscanf(line, "%d", &idx); err != nil || idx < 1 || idx > len(all) {
		return nil, nil
	}

	return &all[idx-1], nil
}

func displayThreadPreview(parts []string) {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("14")).
		Padding(0, 1).
		Width(72)
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	if len(parts) == 1 {
		fmt.Println(headerStyle.Render("\nTweet preview:"))
	} else {
		fmt.Println(headerStyle.Render(fmt.Sprintf("\nThread preview (%d tweets):", len(parts))))
	}

	for i, part := range parts {
		label := ""
		if len(parts) > 1 {
			label = fmt.Sprintf("Tweet %d/%d", i+1, len(parts))
		}
		charCount := countStyle.Render(fmt.Sprintf("%d/280 chars", len(part)))
		header := ""
		if label != "" {
			header = fmt.Sprintf("%s  %s\n", label, charCount)
		} else {
			header = fmt.Sprintf("%s\n", charCount)
		}
		fmt.Println(cardStyle.Render(header + part))
	}
	fmt.Println()
}

func postQuoteTweet(ctx context.Context, client *x.FallbackClient, text string, quoteTweetID string) (string, error) {
	return client.PostQuoteTweet(ctx, text, quoteTweetID)
}

func postThread(ctx context.Context, client models.PlatformPoster, parts []string, mediaIDs []string) ([]string, error) {
	var tweetIDs []string

	for i, part := range parts {
		var tweetID string
		var err error

		if i == 0 && len(mediaIDs) > 0 {
			// First tweet with media — use MediaPoster interface via type assertion
			if mp, ok := client.(models.MediaPoster); ok {
				tweetID, err = mp.PostTweetWithMedia(ctx, part, mediaIDs)
			} else {
				tweetID, err = client.PostTweet(ctx, part)
			}
		} else if i == 0 {
			tweetID, err = client.PostTweet(ctx, part)
		} else {
			tweetID, err = client.PostReply(ctx, part, tweetIDs[i-1])
		}

		if err != nil {
			return tweetIDs, fmt.Errorf("posting tweet %d/%d: %w", i+1, len(parts), err)
		}

		tweetIDs = append(tweetIDs, tweetID)

		// Delay between thread tweets to avoid rate limits
		if i < len(parts)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	return tweetIDs, nil
}

func schedulePost(database *db.DB, contentID int64, atStr string) error {
	scheduledAt, err := time.ParseInLocation("2006-01-02 15:04", atStr, time.Local)
	if err != nil {
		return fmt.Errorf("parsing schedule time %q (expected format: YYYY-MM-DD HH:MM): %w", atStr, err)
	}

	if scheduledAt.Before(time.Now()) {
		return fmt.Errorf("scheduled time %s is in the past", atStr)
	}

	id, err := database.InsertScheduledPost("", contentID, scheduledAt)
	if err != nil {
		return fmt.Errorf("scheduling post: %w", err)
	}

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render(fmt.Sprintf("Scheduled! Post ID %d will be posted at %s", id, scheduledAt.Format("2006-01-02 15:04"))))
	fmt.Println("Run scheduled posts with: goviral post --run-scheduled")
	fmt.Println("Or set up cron: */5 * * * * goviral post --run-scheduled")

	return nil
}

func listScheduledPosts(database *db.DB) error {
	posts, err := database.GetScheduledPosts("", "pending", 20)
	if err != nil {
		return fmt.Errorf("fetching scheduled posts: %w", err)
	}

	if len(posts) == 0 {
		fmt.Println("No pending scheduled posts.")
		return nil
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	fmt.Println(headerStyle.Render("\nPending scheduled posts:"))

	for _, sp := range posts {
		fmt.Printf("  ID:%d  Content:%d  Scheduled: %s\n",
			sp.ID, sp.GeneratedContentID, sp.ScheduledAt.Format("2006-01-02 15:04"))
	}

	return nil
}

func runScheduledPosts(cfg *config.Config, database *db.DB) error {
	pending, err := database.GetPendingScheduledPosts()
	if err != nil {
		return fmt.Errorf("fetching pending posts: %w", err)
	}

	if len(pending) == 0 {
		return nil // Silent for cron usage
	}

	if err := ensureFreshToken(cfg); err != nil {
		return err
	}

	client := x.NewFallbackClient(cfg.X)
	ctx := context.Background()
	posted := 0

	for _, sp := range pending {
		gc, err := database.GetGeneratedContentByID("", sp.GeneratedContentID)
		if err != nil {
			_ = database.UpdateScheduledPostStatus(sp.ID, "failed", err.Error())
			continue
		}
		if gc == nil {
			_ = database.UpdateScheduledPostStatus(sp.ID, "failed", "content not found")
			continue
		}
		if gc.Status == "posted" {
			_ = database.UpdateScheduledPostStatus(sp.ID, "failed", "content already posted")
			continue
		}

		if gc.IsRepost && gc.QuoteTweetID != "" {
			tweetID, err := postQuoteTweet(ctx, client, gc.GeneratedContent, gc.QuoteTweetID)
			if err != nil {
				_ = database.UpdateScheduledPostStatus(sp.ID, "failed", err.Error())
				fmt.Printf("Failed to post scheduled ID %d: %s\n", sp.ID, err.Error())
				continue
			}
			_ = database.UpdateGeneratedContentPosted("", gc.ID, tweetID)
			_ = database.UpdateScheduledPostStatus(sp.ID, "posted", "")
			posted++
			fmt.Printf("Posted scheduled ID %d (quote tweet) → https://x.com/i/status/%s\n", sp.ID, tweetID)
		} else {
			result := thread.Split(gc.GeneratedContent, postNumbered)
			// For scheduled posts, upload media if image is attached
			var scheduledMediaIDs []string
			if gc.ImagePath != "" {
				imageData, readErr := os.ReadFile(gc.ImagePath)
				if readErr == nil {
					mediaID, uploadErr := client.UploadMedia(ctx, imageData, "image/png")
					if uploadErr == nil {
						scheduledMediaIDs = append(scheduledMediaIDs, mediaID)
					}
				}
			}
			tweetIDs, err := postThread(ctx, client, result.Parts, scheduledMediaIDs)
			if err != nil {
				errMsg := err.Error()
				if len(tweetIDs) > 0 {
					partialIDs := strings.Join(tweetIDs, ",")
					_ = database.UpdateGeneratedContentPosted("", gc.ID, partialIDs)
					errMsg = fmt.Sprintf("partial post (%d/%d): %s", len(tweetIDs), len(result.Parts), errMsg)
				}
				_ = database.UpdateScheduledPostStatus(sp.ID, "failed", errMsg)
				fmt.Printf("Failed to post scheduled ID %d: %s\n", sp.ID, errMsg)
				continue
			}

			allIDs := strings.Join(tweetIDs, ",")
			_ = database.UpdateGeneratedContentPosted("", gc.ID, allIDs)
			_ = database.UpdateScheduledPostStatus(sp.ID, "posted", "")
			posted++
			fmt.Printf("Posted scheduled ID %d → https://x.com/i/status/%s\n", sp.ID, tweetIDs[0])
		}
	}

	if posted > 0 {
		fmt.Printf("Posted %d scheduled item(s).\n", posted)
	}

	return nil
}

// ensureFreshToken checks if the X access token is present and not expired.
// If expired and a refresh token is available, it refreshes automatically.
func ensureFreshToken(cfg *config.Config) error {
	if cfg.X.AccessToken == "" {
		return fmt.Errorf("no X access token configured; run 'goviral auth x' first")
	}

	// Check if token has expired
	if cfg.X.TokenExpiry != "" {
		expiry, err := time.Parse(time.RFC3339, cfg.X.TokenExpiry)
		if err == nil && time.Now().After(expiry) {
			fmt.Println("Access token expired, refreshing...")
			if err := auth.RefreshXToken(cfg); err != nil {
				return fmt.Errorf("token expired and refresh failed: %w\nRun 'goviral auth x' to re-authenticate", err)
			}
			fmt.Println("Token refreshed successfully.")
		}
	}

	return nil
}
