package service

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/internal/thread"
	"github.com/shuhao/goviral/pkg/models"
)

// PublishService handles publishing content to platforms.
type PublishService struct {
	db  *db.DB
	cfg *config.Config
}

// NewPublishService creates a new PublishService.
func NewPublishService(database *db.DB, cfg *config.Config) *PublishService {
	return &PublishService{db: database, cfg: cfg}
}

// Publish posts content to the target platform and returns the post IDs and thread parts.
// It dispatches to PublishX or PublishLinkedIn based on the content's target platform.
func (s *PublishService) Publish(ctx context.Context, userID string, contentID int64, numbered bool) ([]string, []string, error) {
	gc, err := s.db.GetGeneratedContentByID(userID, contentID)
	if err != nil {
		return nil, nil, fmt.Errorf("getting content %d: %w", contentID, err)
	}
	if gc == nil {
		return nil, nil, fmt.Errorf("content %d not found", contentID)
	}

	// Route comments to platform-specific handler
	if gc.IsComment {
		commentID, err := s.Comment(ctx, userID, contentID)
		if err != nil {
			return nil, nil, err
		}
		return []string{commentID}, []string{gc.GeneratedContent}, nil
	}

	switch gc.TargetPlatform {
	case "x":
		return s.PublishX(ctx, userID, contentID, numbered)
	case "linkedin":
		return s.PublishLinkedIn(ctx, userID, contentID)
	case "youtube":
		return s.PublishYouTube(ctx, userID, contentID)
	case "tiktok":
		return s.PublishTikTok(ctx, userID, contentID)
	default:
		return nil, nil, fmt.Errorf("unsupported platform: %s", gc.TargetPlatform)
	}
}

// PublishX posts X content (threading + quote tweets).
func (s *PublishService) PublishX(ctx context.Context, userID string, contentID int64, numbered bool) ([]string, []string, error) {
	gc, err := s.db.GetGeneratedContentByID(userID, contentID)
	if err != nil {
		return nil, nil, fmt.Errorf("getting content %d: %w", contentID, err)
	}
	if gc == nil {
		return nil, nil, fmt.Errorf("content %d not found", contentID)
	}
	if gc.Status == "posted" {
		return nil, nil, fmt.Errorf("content %d already posted", contentID)
	}
	if gc.TargetPlatform != "x" {
		return nil, nil, fmt.Errorf("content %d targets %q, not x", contentID, gc.TargetPlatform)
	}

	uc, _ := s.db.GetUserConfig(userID)
	xCfg := uc.MergedXConfig(*s.cfg)

	// Comment path
	if gc.IsComment {
		commentID, err := s.CommentX(ctx, userID, contentID)
		if err != nil {
			return nil, nil, err
		}
		return []string{commentID}, []string{gc.GeneratedContent}, nil
	}

	// Quote tweet path
	if gc.IsRepost && gc.QuoteTweetID != "" {
		log.Printf("publishing quote tweet for content %d (quoting %s)", contentID, gc.QuoteTweetID)
		quotePoster := newXQuotePoster(xCfg)
		postID, err := quotePoster.PostQuoteTweet(ctx, gc.GeneratedContent, gc.QuoteTweetID)
		if err != nil {
			log.Printf("quote tweet publish failed for content %d: %v", contentID, err)
			return nil, nil, fmt.Errorf("posting quote tweet to X: %w", err)
		}
		if err := s.db.UpdateGeneratedContentPosted(userID, contentID, postID); err != nil {
			return nil, nil, fmt.Errorf("updating content status: %w", err)
		}
		return []string{postID}, []string{gc.GeneratedContent}, nil
	}

	result := thread.Split(gc.GeneratedContent, numbered)
	parts := result.Parts

	poster := newXPoster(xCfg)

	// If the content has a code image, attempt to post the first tweet with the image
	// attached. Subsequent thread parts are posted as plain text.
	if gc.SourceType == "commit" && gc.CodeImagePath != "" {
		if mediaPoster, ok := poster.(models.MediaPoster); ok {
			imageData, err := readImageFile(gc.CodeImagePath)
			if err != nil {
				log.Printf("failed to read code image %s for content %d: %v; posting without image", gc.CodeImagePath, contentID, err)
			} else {
				mediaID, err := mediaPoster.UploadMedia(ctx, imageData, "image/png")
				if err != nil {
					log.Printf("failed to upload code image for content %d: %v; posting without image", contentID, err)
				} else {
					// Post the first part with the media attached, then thread the rest normally
					firstID, err := mediaPoster.PostTweetWithMedia(ctx, parts[0], []string{mediaID})
					if err != nil {
						log.Printf("failed to post tweet with media for content %d: %v; falling back to plain post", contentID, err)
					} else {
						var postIDs []string
						postIDs = append(postIDs, firstID)
						if len(parts) > 1 {
							remainingIDs, err := s.postThreadFrom(ctx, poster, parts[1:], firstID)
							if err != nil {
								return nil, nil, fmt.Errorf("posting thread remainder to X: %w", err)
							}
							postIDs = append(postIDs, remainingIDs...)
						}
						if err := s.db.UpdateGeneratedContentPosted(userID, contentID, strings.Join(postIDs, ",")); err != nil {
							return nil, nil, fmt.Errorf("updating content status: %w", err)
						}
						return postIDs, parts, nil
					}
				}
			}
		}
	}

	postIDs, err := s.postThread(ctx, poster, parts)
	if err != nil {
		return nil, nil, fmt.Errorf("posting to X: %w", err)
	}

	if err := s.db.UpdateGeneratedContentPosted(userID, contentID, strings.Join(postIDs, ",")); err != nil {
		return nil, nil, fmt.Errorf("updating content status: %w", err)
	}

	return postIDs, parts, nil
}

// PublishLinkedIn posts or reposts LinkedIn content via linkitin.
func (s *PublishService) PublishLinkedIn(ctx context.Context, userID string, contentID int64) ([]string, []string, error) {
	gc, err := s.db.GetGeneratedContentByID(userID, contentID)
	if err != nil {
		return nil, nil, fmt.Errorf("getting content %d: %w", contentID, err)
	}
	if gc == nil {
		return nil, nil, fmt.Errorf("content %d not found", contentID)
	}
	if gc.Status == "posted" {
		return nil, nil, fmt.Errorf("content %d already posted", contentID)
	}
	if gc.TargetPlatform != "linkedin" {
		return nil, nil, fmt.Errorf("content %d targets %q, not linkedin", contentID, gc.TargetPlatform)
	}

	uc, _ := s.db.GetUserConfig(userID)
	linkedInCfg := uc.MergedLinkedInConfig(*s.cfg)

	// Comment path
	if gc.IsComment {
		commentID, err := s.CommentLinkedIn(ctx, userID, contentID)
		if err != nil {
			return nil, nil, err
		}
		return []string{commentID}, []string{gc.GeneratedContent}, nil
	}

	// Repost path
	if gc.IsRepost && gc.QuoteTweetID != "" {
		log.Printf("publishing LinkedIn repost for content %d (quoting %s)", contentID, gc.QuoteTweetID)
		reposter := newLinkedInReposter(linkedInCfg)
		postID, err := reposter.Repost(ctx, gc.QuoteTweetID, gc.GeneratedContent)
		if err != nil {
			log.Printf("LinkedIn repost failed for content %d: %v", contentID, err)
			return nil, nil, fmt.Errorf("reposting to LinkedIn: %w", err)
		}
		if err := s.db.UpdateGeneratedContentPosted(userID, contentID, postID); err != nil {
			return nil, nil, fmt.Errorf("updating content status: %w", err)
		}
		return []string{postID}, []string{gc.GeneratedContent}, nil
	}

	poster := newLinkedInPoster(linkedInCfg)

	// If the content has a code image, post with it attached.
	if gc.SourceType == "commit" && gc.CodeImagePath != "" {
		imageData, err := readImageFile(gc.CodeImagePath)
		if err != nil {
			log.Printf("failed to read code image %s for content %d: %v; posting without image", gc.CodeImagePath, contentID, err)
		} else {
			filename := filepath.Base(gc.CodeImagePath)
			postID, err := poster.CreatePostWithImage(ctx, gc.GeneratedContent, imageData, filename)
			if err != nil {
				log.Printf("failed to post LinkedIn content with image for content %d: %v; falling back to plain post", contentID, err)
			} else {
				if err := s.db.UpdateGeneratedContentPosted(userID, contentID, postID); err != nil {
					return nil, nil, fmt.Errorf("updating content status: %w", err)
				}
				return []string{postID}, []string{gc.GeneratedContent}, nil
			}
		}
	}

	postID, err := poster.CreatePost(ctx, gc.GeneratedContent)
	if err != nil {
		return nil, nil, fmt.Errorf("posting to LinkedIn: %w", err)
	}

	if err := s.db.UpdateGeneratedContentPosted(userID, contentID, postID); err != nil {
		return nil, nil, fmt.Errorf("updating content status: %w", err)
	}

	return []string{postID}, []string{gc.GeneratedContent}, nil
}

// CommentLinkedIn posts a comment on a LinkedIn post via linkitin.
func (s *PublishService) CommentLinkedIn(ctx context.Context, userID string, contentID int64) (string, error) {
	gc, err := s.db.GetGeneratedContentByID(userID, contentID)
	if err != nil {
		return "", fmt.Errorf("getting content %d: %w", contentID, err)
	}
	if gc == nil {
		return "", fmt.Errorf("content %d not found", contentID)
	}
	if !gc.IsComment {
		return "", fmt.Errorf("content %d is not a comment", contentID)
	}
	if gc.QuoteTweetID == "" {
		return "", fmt.Errorf("content %d has no parent post URN", contentID)
	}
	if gc.Status == "posted" {
		return "", fmt.Errorf("content %d already posted", contentID)
	}

	// Reject known-bad URNs early. LinkedIn returns 400/422 for these.
	// urn:li:dom:* are linkitin-internal DOM-parsed IDs (any N, not just :0)
	// — not real LinkedIn URNs. urn:li:content:* is our synthetic content-hash
	// ID assigned to DOM-parsed trending posts; also not a real LinkedIn URN.
	// Sponsored content URNs also can't be commented on.
	if strings.Contains(gc.QuoteTweetID, "urn:li:dom:") ||
		strings.Contains(gc.QuoteTweetID, "urn:li:content:") ||
		strings.Contains(gc.QuoteTweetID, "sponsoredContent") ||
		strings.HasSuffix(gc.QuoteTweetID, ":0") {
		return "", fmt.Errorf("content %d has an unpostable LinkedIn URN %q (no real LinkedIn URN — post was scraped without a resolvable URN); re-fetch trending posts to get a commentable target", contentID, gc.QuoteTweetID)
	}

	// Look up the thread_urn from the source trending post (needed for ugcPost comments).
	var threadURN string
	if gc.SourceTrendingID != 0 {
		if tp, tpErr := s.db.GetTrendingPostByID("",gc.SourceTrendingID); tpErr == nil && tp != nil {
			threadURN = tp.ThreadURN
		}
	}

	uc, _ := s.db.GetUserConfig(userID)
	commenter := newLinkedInCommenter(uc.MergedLinkedInConfig(*s.cfg))
	commentURN, err := commenter.CreateComment(ctx, gc.QuoteTweetID, threadURN, gc.GeneratedContent)
	if err != nil {
		slog.Error("linkedin comment failed", "content_id", contentID, "post_urn", gc.QuoteTweetID, "error", err)
		return "", fmt.Errorf("commenting on LinkedIn post: %w", err)
	}

	if err := s.db.UpdateGeneratedContentPosted(userID, contentID, commentURN); err != nil {
		return "", fmt.Errorf("updating content status: %w", err)
	}

	return commentURN, nil
}

// CommentX posts a reply to an X (Twitter) tweet via twikit.
func (s *PublishService) CommentX(ctx context.Context, userID string, contentID int64) (string, error) {
	gc, err := s.db.GetGeneratedContentByID(userID, contentID)
	if err != nil {
		return "", fmt.Errorf("getting content %d: %w", contentID, err)
	}
	if gc == nil {
		return "", fmt.Errorf("content %d not found", contentID)
	}
	if !gc.IsComment {
		return "", fmt.Errorf("content %d is not a comment", contentID)
	}
	if gc.QuoteTweetID == "" {
		return "", fmt.Errorf("content %d has no parent tweet ID", contentID)
	}
	if gc.Status == "posted" {
		return "", fmt.Errorf("content %d already posted", contentID)
	}

	uc, _ := s.db.GetUserConfig(userID)
	poster := newXPoster(uc.MergedXConfig(*s.cfg))
	replyID, err := poster.PostReply(ctx, gc.GeneratedContent, gc.QuoteTweetID)
	if err != nil {
		slog.Error("x comment (reply) failed", "content_id", contentID, "tweet_id", gc.QuoteTweetID, "error", err)
		return "", fmt.Errorf("replying to X tweet: %w", err)
	}

	if err := s.db.UpdateGeneratedContentPosted(userID, contentID, replyID); err != nil {
		return "", fmt.Errorf("updating content status: %w", err)
	}

	return replyID, nil
}

// Comment dispatches to the platform-specific comment handler based on content's target platform.
func (s *PublishService) Comment(ctx context.Context, userID string, contentID int64) (string, error) {
	gc, err := s.db.GetGeneratedContentByID(userID, contentID)
	if err != nil {
		return "", fmt.Errorf("getting content %d: %w", contentID, err)
	}
	if gc == nil {
		return "", fmt.Errorf("content %d not found", contentID)
	}

	switch gc.TargetPlatform {
	case "x":
		return s.CommentX(ctx, userID, contentID)
	case "linkedin":
		return s.CommentLinkedIn(ctx, userID, contentID)
	default:
		return "", fmt.Errorf("unsupported platform for comment: %s", gc.TargetPlatform)
	}
}

// Schedule submits content to the platform's native scheduling for the given time.
// Returns the platform-specific schedule ID if available, otherwise an empty string for fallback scheduling.
func (s *PublishService) Schedule(ctx context.Context, userID string, contentID int64, scheduledAt time.Time) (string, error) {
	gc, err := s.db.GetGeneratedContentByID(userID, contentID)
	if err != nil {
		return "", fmt.Errorf("getting content %d: %w", contentID, err)
	}
	if gc == nil {
		return "", fmt.Errorf("content %d not found", contentID)
	}

	uc, _ := s.db.GetUserConfig(userID)

	switch gc.TargetPlatform {
	case "x":
		if gc.IsRepost && gc.QuoteTweetID != "" {
			quoteScheduler := newXQuoteScheduler(uc.MergedXConfig(*s.cfg))
			return quoteScheduler.ScheduleQuoteTweet(ctx, gc.GeneratedContent, gc.QuoteTweetID, scheduledAt.Unix())
		}
		scheduler := newXScheduler(uc.MergedXConfig(*s.cfg))
		return scheduler.ScheduleTweet(ctx, gc.GeneratedContent, scheduledAt.Unix())
	case "linkedin":
		if gc.IsRepost {
			return "", fmt.Errorf("native scheduling not supported for LinkedIn reposts: content %d will be executed via RunDue", contentID)
		}
		linkedInPoster := newLinkedInPoster(uc.MergedLinkedInConfig(*s.cfg))
		// Commit posts store their image in CodeImagePath; regular AI-image posts use ImagePath.
		imagePath := gc.ImagePath
		if gc.SourceType == "commit" && gc.CodeImagePath != "" {
			imagePath = gc.CodeImagePath
		}
		if imagePath != "" {
			imageData, err := readImageFile(imagePath)
			if err != nil {
				// Fall back to text-only scheduling if image read fails
				log.Printf("failed to read image from %s: %v, using text-only scheduling", imagePath, err)
				return linkedInPoster.CreateScheduledPost(ctx, gc.GeneratedContent, scheduledAt)
			}
			filename := filepath.Base(imagePath)
			return linkedInPoster.CreateScheduledPostWithImage(ctx, gc.GeneratedContent, imageData, filename, scheduledAt)
		}
		return linkedInPoster.CreateScheduledPost(ctx, gc.GeneratedContent, scheduledAt)
	default:
		return "", fmt.Errorf("scheduled posting not supported for platform: %s", gc.TargetPlatform)
	}
}

// PublishYouTube uploads video content to YouTube Shorts.
func (s *PublishService) PublishYouTube(ctx context.Context, userID string, contentID int64) ([]string, []string, error) {
	gc, err := s.db.GetGeneratedContentByID(userID, contentID)
	if err != nil {
		return nil, nil, fmt.Errorf("getting content %d: %w", contentID, err)
	}
	if gc == nil {
		return nil, nil, fmt.Errorf("content %d not found", contentID)
	}
	if gc.Status == "posted" {
		return nil, nil, fmt.Errorf("content %d already posted", contentID)
	}
	if gc.TargetPlatform != "youtube" {
		return nil, nil, fmt.Errorf("content %d targets %q, not youtube", contentID, gc.TargetPlatform)
	}
	if gc.VideoPath == "" {
		return nil, nil, fmt.Errorf("content %d has no video path", contentID)
	}

	// Validate video file exists
	if _, err := os.Stat(gc.VideoPath); err != nil {
		return nil, nil, fmt.Errorf("video file not found at %s: %w", gc.VideoPath, err)
	}

	uc, _ := s.db.GetUserConfig(userID)
	poster := newYouTubePoster(uc.MergedYouTubeConfig(*s.cfg))

	title := gc.VideoTitle
	if title == "" {
		// Use first line of content as title
		title = gc.GeneratedContent
		if len(title) > 100 {
			title = title[:97] + "..."
		}
	}

	var videoID string
	if gc.ThumbnailPath != "" {
		videoID, err = poster.UploadVideoWithThumbnail(ctx, gc.VideoPath, gc.ThumbnailPath, title, gc.GeneratedContent, nil)
	} else {
		videoID, err = poster.UploadVideo(ctx, gc.VideoPath, title, gc.GeneratedContent, nil)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("uploading to YouTube: %w", err)
	}

	if err := s.db.UpdateGeneratedContentPosted(userID, contentID, videoID); err != nil {
		return nil, nil, fmt.Errorf("updating content status: %w", err)
	}

	return []string{videoID}, []string{gc.GeneratedContent}, nil
}

// PublishTikTok uploads video content to TikTok.
func (s *PublishService) PublishTikTok(ctx context.Context, userID string, contentID int64) ([]string, []string, error) {
	gc, err := s.db.GetGeneratedContentByID(userID, contentID)
	if err != nil {
		return nil, nil, fmt.Errorf("getting content %d: %w", contentID, err)
	}
	if gc == nil {
		return nil, nil, fmt.Errorf("content %d not found", contentID)
	}
	if gc.Status == "posted" {
		return nil, nil, fmt.Errorf("content %d already posted", contentID)
	}
	if gc.TargetPlatform != "tiktok" {
		return nil, nil, fmt.Errorf("content %d targets %q, not tiktok", contentID, gc.TargetPlatform)
	}
	if gc.VideoPath == "" {
		return nil, nil, fmt.Errorf("content %d has no video path", contentID)
	}

	if _, err := os.Stat(gc.VideoPath); err != nil {
		return nil, nil, fmt.Errorf("video file not found at %s: %w", gc.VideoPath, err)
	}

	uc, _ := s.db.GetUserConfig(userID)
	poster := newTikTokPoster(uc.MergedTikTokConfig(*s.cfg))

	videoID, err := poster.UploadVideo(ctx, gc.VideoPath, gc.GeneratedContent, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("uploading to TikTok: %w", err)
	}

	if err := s.db.UpdateGeneratedContentPosted(userID, contentID, videoID); err != nil {
		return nil, nil, fmt.Errorf("updating content status: %w", err)
	}

	return []string{videoID}, []string{gc.GeneratedContent}, nil
}

func (s *PublishService) postThread(ctx context.Context, poster models.PlatformPoster, parts []string) ([]string, error) {
	var postIDs []string
	var lastID string

	for i, part := range parts {
		var id string
		var err error

		if i == 0 {
			id, err = poster.PostTweet(ctx, part)
		} else {
			// Small delay between thread parts to avoid X rate limiting / anti-spam.
			select {
			case <-ctx.Done():
				return postIDs, ctx.Err()
			case <-time.After(2 * time.Second):
			}
			id, err = poster.PostReply(ctx, part, lastID)
		}
		if err != nil {
			return postIDs, fmt.Errorf("posting part %d: %w", i+1, err)
		}

		postIDs = append(postIDs, id)
		lastID = id
	}

	return postIDs, nil
}

// postThreadFrom posts a sequence of thread parts starting as replies to replyToID.
// It is used when the first tweet of a thread has already been posted (e.g. with media).
func (s *PublishService) postThreadFrom(ctx context.Context, poster models.PlatformPoster, parts []string, replyToID string) ([]string, error) {
	var postIDs []string
	lastID := replyToID

	for i, part := range parts {
		select {
		case <-ctx.Done():
			return postIDs, ctx.Err()
		case <-time.After(2 * time.Second):
		}
		id, err := poster.PostReply(ctx, part, lastID)
		if err != nil {
			return postIDs, fmt.Errorf("posting thread part %d: %w", i+2, err)
		}
		postIDs = append(postIDs, id)
		lastID = id
	}

	return postIDs, nil
}

// readImageFile reads an image file from disk and returns its contents as bytes.
func readImageFile(imagePath string) ([]byte, error) {
	if imagePath == "" {
		return nil, fmt.Errorf("image path is empty")
	}
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("reading image file %s: %w", imagePath, err)
	}
	return imageData, nil
}
