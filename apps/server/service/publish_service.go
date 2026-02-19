package service

import (
	"context"
	"fmt"
	"log"
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
func (s *PublishService) Publish(ctx context.Context, contentID int64, numbered bool) ([]string, []string, error) {
	gc, err := s.db.GetGeneratedContentByID(contentID)
	if err != nil {
		return nil, nil, fmt.Errorf("getting content %d: %w", contentID, err)
	}
	if gc == nil {
		return nil, nil, fmt.Errorf("content %d not found", contentID)
	}

	switch gc.TargetPlatform {
	case "x":
		return s.PublishX(ctx, contentID, numbered)
	case "linkedin":
		return s.PublishLinkedIn(ctx, contentID)
	default:
		return nil, nil, fmt.Errorf("unsupported platform: %s", gc.TargetPlatform)
	}
}

// PublishX posts X content (threading + quote tweets).
func (s *PublishService) PublishX(ctx context.Context, contentID int64, numbered bool) ([]string, []string, error) {
	gc, err := s.db.GetGeneratedContentByID(contentID)
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

	// Quote tweet path
	if gc.IsRepost && gc.QuoteTweetID != "" {
		log.Printf("publishing quote tweet for content %d (quoting %s)", contentID, gc.QuoteTweetID)
		quotePoster := NewXQuotePoster(s.cfg.X)
		postID, err := quotePoster.PostQuoteTweet(ctx, gc.GeneratedContent, gc.QuoteTweetID)
		if err != nil {
			log.Printf("quote tweet publish failed for content %d: %v", contentID, err)
			return nil, nil, fmt.Errorf("posting quote tweet to X: %w", err)
		}
		if err := s.db.UpdateGeneratedContentPosted(contentID, postID); err != nil {
			return nil, nil, fmt.Errorf("updating content status: %w", err)
		}
		return []string{postID}, []string{gc.GeneratedContent}, nil
	}

	result := thread.Split(gc.GeneratedContent, numbered)
	parts := result.Parts

	poster := NewXPoster(s.cfg.X)
	postIDs, err := s.postThread(ctx, poster, parts)
	if err != nil {
		return nil, nil, fmt.Errorf("posting to X: %w", err)
	}

	if err := s.db.UpdateGeneratedContentPosted(contentID, strings.Join(postIDs, ",")); err != nil {
		return nil, nil, fmt.Errorf("updating content status: %w", err)
	}

	return postIDs, parts, nil
}

// PublishLinkedIn posts or reposts LinkedIn content via likit.
func (s *PublishService) PublishLinkedIn(ctx context.Context, contentID int64) ([]string, []string, error) {
	gc, err := s.db.GetGeneratedContentByID(contentID)
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

	// Repost path
	if gc.IsRepost && gc.QuoteTweetID != "" {
		log.Printf("publishing LinkedIn repost for content %d (quoting %s)", contentID, gc.QuoteTweetID)
		reposter := NewLinkedInReposter(s.cfg.LinkedIn)
		postID, err := reposter.Repost(ctx, gc.QuoteTweetID, gc.GeneratedContent)
		if err != nil {
			log.Printf("LinkedIn repost failed for content %d: %v", contentID, err)
			return nil, nil, fmt.Errorf("reposting to LinkedIn: %w", err)
		}
		if err := s.db.UpdateGeneratedContentPosted(contentID, postID); err != nil {
			return nil, nil, fmt.Errorf("updating content status: %w", err)
		}
		return []string{postID}, []string{gc.GeneratedContent}, nil
	}

	poster := NewLinkedInPoster(s.cfg.LinkedIn)
	postID, err := poster.CreatePost(ctx, gc.GeneratedContent)
	if err != nil {
		return nil, nil, fmt.Errorf("posting to LinkedIn: %w", err)
	}

	if err := s.db.UpdateGeneratedContentPosted(contentID, postID); err != nil {
		return nil, nil, fmt.Errorf("updating content status: %w", err)
	}

	return []string{postID}, []string{gc.GeneratedContent}, nil
}

// Schedule submits content to the platform's native scheduling for the given time.
// Returns the platform-specific schedule ID if available, otherwise an empty string for fallback scheduling.
func (s *PublishService) Schedule(ctx context.Context, contentID int64, scheduledAt time.Time) (string, error) {
	gc, err := s.db.GetGeneratedContentByID(contentID)
	if err != nil {
		return "", fmt.Errorf("getting content %d: %w", contentID, err)
	}
	if gc == nil {
		return "", fmt.Errorf("content %d not found", contentID)
	}

	switch gc.TargetPlatform {
	case "x":
		if gc.IsRepost && gc.QuoteTweetID != "" {
			quoteScheduler := NewXQuoteScheduler(s.cfg.X)
			return quoteScheduler.ScheduleQuoteTweet(ctx, gc.GeneratedContent, gc.QuoteTweetID, scheduledAt.Unix())
		}
		scheduler := NewXScheduler(s.cfg.X)
		return scheduler.ScheduleTweet(ctx, gc.GeneratedContent, scheduledAt.Unix())
	case "linkedin":
		if gc.IsRepost {
			return "", fmt.Errorf("native scheduling not supported for LinkedIn reposts: content %d will be executed via RunDue", contentID)
		}
		linkedInPoster := NewLinkedInPoster(s.cfg.LinkedIn)
		// Check if content has an image; if so, use CreateScheduledPostWithImage
		if gc.ImagePath != "" {
			imageData, err := readImageFile(gc.ImagePath)
			if err != nil {
				// Fall back to text-only scheduling if image read fails
				log.Printf("failed to read image from %s: %v, using text-only scheduling", gc.ImagePath, err)
				return linkedInPoster.CreateScheduledPost(ctx, gc.GeneratedContent, scheduledAt)
			}
			filename := filepath.Base(gc.ImagePath)
			return linkedInPoster.CreateScheduledPostWithImage(ctx, gc.GeneratedContent, imageData, filename, scheduledAt)
		}
		return linkedInPoster.CreateScheduledPost(ctx, gc.GeneratedContent, scheduledAt)
	default:
		return "", fmt.Errorf("scheduled posting not supported for platform: %s", gc.TargetPlatform)
	}
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
