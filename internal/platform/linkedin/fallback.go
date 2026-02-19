package linkedin

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/pkg/models"
)

// Compile-time interface compliance checks.
var _ models.PlatformClient = (*FallbackClient)(nil)
var _ models.LinkedInPoster = (*FallbackClient)(nil)
var _ models.LinkedInReposter = (*FallbackClient)(nil)

// linkedinFetcher is an internal interface for testability.
type linkedinFetcher interface {
	FetchMyPosts(ctx context.Context, limit int) ([]models.Post, error)
	FetchTrendingPosts(ctx context.Context, niches []string, period string, minLikes int, limit int) ([]models.TrendingPost, error)
}

// linkedinPoster is an internal interface for testability.
type linkedinPoster interface {
	CreatePost(ctx context.Context, text string) (string, error)
	UploadImage(ctx context.Context, imageData []byte, filename string) (string, error)
	CreatePostWithImage(ctx context.Context, text string, imageData []byte, filename string) (string, error)
	Repost(ctx context.Context, postURN string, text string) (string, error)
	CreateScheduledPost(ctx context.Context, text string, scheduledAt time.Time) (string, error)
	CreateScheduledPostWithImage(ctx context.Context, text string, imageData []byte, filename string, scheduledAt time.Time) (string, error)
}

// FallbackClient wraps the official LinkedIn API client with a likit fallback.
// If the official API fails, it falls back to likit (cookie-based Voyager API).
type FallbackClient struct {
	primary         linkedinFetcher
	likit           linkedinFetcher // may be nil if python is unavailable
	likitPoster     linkedinPoster  // may be nil if python is unavailable
	primaryDisabled bool
}

// NewFallbackClient creates a FallbackClient with the official API as primary
// and likit as fallback. If likit setup fails (e.g. no Python),
// the client operates with primary only and logs a warning.
func NewFallbackClient(cfg config.LinkedInConfig, influencerURNs []string) *FallbackClient {
	primary := NewClient(cfg, influencerURNs)

	fc := &FallbackClient{
		primary:         primary,
		primaryDisabled: cfg.AccessToken == "" || cfg.PersonURN == "",
	}

	lc, err := NewLikitClient()
	if err != nil {
		log.Printf("likit fallback unavailable: %v (official LinkedIn API only)", err)
	} else {
		fc.likit = lc
		fc.likitPoster = lc
	}

	return fc
}

// FetchMyPosts tries the official API first. On failure, falls back to likit if available.
func (fc *FallbackClient) FetchMyPosts(ctx context.Context, limit int) ([]models.Post, error) {
	if !fc.primaryDisabled {
		posts, primaryErr := fc.primary.FetchMyPosts(ctx, limit)
		if primaryErr == nil {
			return posts, nil
		}
		fc.checkDisablePrimary(primaryErr)

		if fc.likit == nil {
			return nil, fmt.Errorf("official LinkedIn API failed: %w (likit fallback unavailable)", primaryErr)
		}

		log.Printf("official LinkedIn API failed (%v), trying likit fallback...", primaryErr)
		posts, likitErr := fc.likit.FetchMyPosts(ctx, limit)
		if likitErr != nil {
			return nil, fmt.Errorf("official API failed: %w; likit fallback also failed: %w", primaryErr, likitErr)
		}
		return posts, nil
	}

	// Primary already known to be down - go straight to likit.
	if fc.likit == nil {
		return nil, fmt.Errorf("official LinkedIn API disabled (likit fallback unavailable)")
	}
	return fc.likit.FetchMyPosts(ctx, limit)
}

// FetchTrendingPosts tries the official API first. On failure, falls back to likit if available.
func (fc *FallbackClient) FetchTrendingPosts(ctx context.Context, niches []string, period string, minLikes int, limit int) ([]models.TrendingPost, error) {
	if !fc.primaryDisabled {
		posts, primaryErr := fc.primary.FetchTrendingPosts(ctx, niches, period, minLikes, limit)
		if primaryErr == nil {
			return posts, nil
		}
		fc.checkDisablePrimary(primaryErr)

		if fc.likit == nil {
			return nil, fmt.Errorf("official LinkedIn API failed: %w (likit fallback unavailable)", primaryErr)
		}

		log.Printf("official LinkedIn API failed (%v), trying likit fallback...", primaryErr)
		posts, likitErr := fc.likit.FetchTrendingPosts(ctx, niches, period, minLikes, limit)
		if likitErr != nil {
			return nil, fmt.Errorf("official API failed: %w; likit fallback also failed: %w", primaryErr, likitErr)
		}
		return posts, nil
	}

	if fc.likit == nil {
		return nil, fmt.Errorf("official LinkedIn API disabled (likit fallback unavailable)")
	}
	return fc.likit.FetchTrendingPosts(ctx, niches, period, minLikes, limit)
}

// CreatePost creates a LinkedIn post via likit (official API has no posting support).
func (fc *FallbackClient) CreatePost(ctx context.Context, text string) (string, error) {
	if fc.likitPoster == nil {
		return "", fmt.Errorf("LinkedIn posting requires likit (cookie-based auth); likit is unavailable")
	}
	return fc.likitPoster.CreatePost(ctx, text)
}

// UploadImage uploads an image to LinkedIn via likit (official API has no posting support).
func (fc *FallbackClient) UploadImage(ctx context.Context, imageData []byte, filename string) (string, error) {
	if fc.likitPoster == nil {
		return "", fmt.Errorf("LinkedIn image upload requires likit (cookie-based auth); likit is unavailable")
	}
	return fc.likitPoster.UploadImage(ctx, imageData, filename)
}

// CreatePostWithImage creates a LinkedIn post with an image via likit (official API has no posting support).
func (fc *FallbackClient) CreatePostWithImage(ctx context.Context, text string, imageData []byte, filename string) (string, error) {
	if fc.likitPoster == nil {
		return "", fmt.Errorf("LinkedIn posting requires likit (cookie-based auth); likit is unavailable")
	}
	return fc.likitPoster.CreatePostWithImage(ctx, text, imageData, filename)
}

// Repost reshares an existing LinkedIn post via likit (official API has no repost support).
func (fc *FallbackClient) Repost(ctx context.Context, postURN string, text string) (string, error) {
	if fc.likitPoster == nil {
		return "", fmt.Errorf("LinkedIn repost requires likit (cookie-based auth); likit is unavailable")
	}
	return fc.likitPoster.Repost(ctx, postURN, text)
}

// CreateScheduledPost schedules a LinkedIn post via likit (official API has no scheduling support).
func (fc *FallbackClient) CreateScheduledPost(ctx context.Context, text string, scheduledAt time.Time) (string, error) {
	if fc.likitPoster == nil {
		return "", fmt.Errorf("LinkedIn scheduling requires likit (cookie-based auth); likit is unavailable")
	}
	return fc.likitPoster.CreateScheduledPost(ctx, text, scheduledAt)
}

// CreateScheduledPostWithImage schedules a LinkedIn post with an image via likit (official API has no scheduling support).
func (fc *FallbackClient) CreateScheduledPostWithImage(ctx context.Context, text string, imageData []byte, filename string, scheduledAt time.Time) (string, error) {
	if fc.likitPoster == nil {
		return "", fmt.Errorf("LinkedIn scheduling requires likit (cookie-based auth); likit is unavailable")
	}
	return fc.likitPoster.CreateScheduledPostWithImage(ctx, text, imageData, filename, scheduledAt)
}

// checkDisablePrimary disables the primary client for subsequent calls if the
// error indicates an account-level issue (auth failure, no access token, etc.).
func (fc *FallbackClient) checkDisablePrimary(err error) {
	msg := err.Error()
	if strings.Contains(msg, "status 401") ||
		strings.Contains(msg, "status 403") ||
		strings.Contains(msg, "no access token") ||
		strings.Contains(msg, "access_token") {
		fc.primaryDisabled = true
	}
}
