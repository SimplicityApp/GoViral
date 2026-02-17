package x

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/pkg/models"
)

// Compile-time interface compliance checks.
var _ models.PlatformClient = (*FallbackClient)(nil)
var _ models.PlatformPoster = (*FallbackClient)(nil)
var _ models.MediaPoster = (*FallbackClient)(nil)

// fetcher is an internal interface for testability, matching the PlatformClient methods.
type fetcher interface {
	FetchMyPosts(ctx context.Context, limit int) ([]models.Post, error)
	FetchTrendingPosts(ctx context.Context, niches []string, period string, minLikes int, limit int) ([]models.TrendingPost, error)
}

// poster is an internal interface for testability, matching the PlatformPoster and MediaPoster methods.
type poster interface {
	PostTweet(ctx context.Context, text string) (string, error)
	PostReply(ctx context.Context, text string, inReplyToID string) (string, error)
	UploadMedia(ctx context.Context, imageData []byte, mimeType string) (string, error)
	PostTweetWithMedia(ctx context.Context, text string, mediaIDs []string) (string, error)
	PostReplyWithMedia(ctx context.Context, text string, inReplyToID string, mediaIDs []string) (string, error)
}

// FallbackClient wraps a primary X API client with an optional twikit fallback.
// If the primary client fails, it falls back to twikit (cookie-based auth).
// Once the primary fails with an account-level error (e.g. 402 credits depleted),
// subsequent calls skip the primary and go directly to twikit.
type FallbackClient struct {
	primary         fetcher
	primaryPoster   poster
	twikit          fetcher // may be nil if python is unavailable
	twikitPoster    poster  // may be nil if python is unavailable
	primaryDisabled bool    // set to true after an account-level primary failure
}

// NewFallbackClient creates a FallbackClient with the official API as primary
// and twikit as fallback. If twikit setup fails (e.g. no Python),
// the client operates with primary only and logs a warning.
func NewFallbackClient(cfg config.XConfig) *FallbackClient {
	primary := NewClient(cfg)

	fc := &FallbackClient{
		primary:       primary,
		primaryPoster: primary,
	}

	tc, err := NewTwikitClient(cfg.Username)
	if err != nil {
		log.Printf("twikit fallback unavailable: %v (primary API only)", err)
	} else {
		fc.twikit = tc
		fc.twikitPoster = tc
	}

	return fc
}

// FetchMyPosts tries the primary API first. On failure, falls back to twikit if available.
func (fc *FallbackClient) FetchMyPosts(ctx context.Context, limit int) ([]models.Post, error) {
	if !fc.primaryDisabled {
		posts, primaryErr := fc.primary.FetchMyPosts(ctx, limit)
		if primaryErr == nil {
			return posts, nil
		}
		fc.checkDisablePrimary(primaryErr)

		if fc.twikit == nil {
			return nil, fmt.Errorf("primary API failed: %w (twikit fallback unavailable)", primaryErr)
		}

		log.Printf("primary X API failed (%v), trying twikit fallback...", primaryErr)
		posts, twikitErr := fc.twikit.FetchMyPosts(ctx, limit)
		if twikitErr != nil {
			return nil, fmt.Errorf("primary API failed: %w; twikit fallback also failed: %w", primaryErr, twikitErr)
		}
		return posts, nil
	}

	// Primary already known to be down — go straight to twikit.
	if fc.twikit == nil {
		return nil, fmt.Errorf("primary API disabled (twikit fallback unavailable)")
	}
	return fc.twikit.FetchMyPosts(ctx, limit)
}

// FetchTrendingPosts tries the primary API first. On failure, falls back to twikit if available.
func (fc *FallbackClient) FetchTrendingPosts(ctx context.Context, niches []string, period string, minLikes int, limit int) ([]models.TrendingPost, error) {
	if !fc.primaryDisabled {
		posts, primaryErr := fc.primary.FetchTrendingPosts(ctx, niches, period, minLikes, limit)
		if primaryErr == nil {
			return posts, nil
		}
		fc.checkDisablePrimary(primaryErr)

		if fc.twikit == nil {
			return nil, fmt.Errorf("primary API failed: %w (twikit fallback unavailable)", primaryErr)
		}

		log.Printf("primary X API failed (%v), using twikit fallback", primaryErr)
		posts, twikitErr := fc.twikit.FetchTrendingPosts(ctx, niches, period, minLikes, limit)
		if twikitErr != nil {
			return nil, fmt.Errorf("primary API failed: %w; twikit fallback also failed: %w", primaryErr, twikitErr)
		}
		return posts, nil
	}

	// Primary already known to be down — go straight to twikit.
	if fc.twikit == nil {
		return nil, fmt.Errorf("primary API disabled (twikit fallback unavailable)")
	}
	return fc.twikit.FetchTrendingPosts(ctx, niches, period, minLikes, limit)
}

// PostTweet posts a tweet, falling back to twikit on account-level errors.
func (fc *FallbackClient) PostTweet(ctx context.Context, text string) (string, error) {
	if !fc.primaryDisabled {
		id, primaryErr := fc.primaryPoster.PostTweet(ctx, text)
		if primaryErr == nil {
			return id, nil
		}
		fc.checkDisablePrimary(primaryErr)

		if fc.twikitPoster == nil {
			return "", fmt.Errorf("primary API failed: %w (twikit fallback unavailable)", primaryErr)
		}

		log.Printf("primary X API failed (%v), trying twikit fallback...", primaryErr)
		id, twikitErr := fc.twikitPoster.PostTweet(ctx, text)
		if twikitErr != nil {
			return "", fmt.Errorf("primary API failed: %w; twikit fallback also failed: %w", primaryErr, twikitErr)
		}
		return id, nil
	}

	if fc.twikitPoster == nil {
		return "", fmt.Errorf("primary API disabled (twikit fallback unavailable)")
	}
	return fc.twikitPoster.PostTweet(ctx, text)
}

// PostReply posts a reply, falling back to twikit on account-level errors.
func (fc *FallbackClient) PostReply(ctx context.Context, text string, inReplyToID string) (string, error) {
	if !fc.primaryDisabled {
		id, primaryErr := fc.primaryPoster.PostReply(ctx, text, inReplyToID)
		if primaryErr == nil {
			return id, nil
		}
		fc.checkDisablePrimary(primaryErr)

		if fc.twikitPoster == nil {
			return "", fmt.Errorf("primary API failed: %w (twikit fallback unavailable)", primaryErr)
		}

		log.Printf("primary X API failed (%v), trying twikit fallback...", primaryErr)
		id, twikitErr := fc.twikitPoster.PostReply(ctx, text, inReplyToID)
		if twikitErr != nil {
			return "", fmt.Errorf("primary API failed: %w; twikit fallback also failed: %w", primaryErr, twikitErr)
		}
		return id, nil
	}

	if fc.twikitPoster == nil {
		return "", fmt.Errorf("primary API disabled (twikit fallback unavailable)")
	}
	return fc.twikitPoster.PostReply(ctx, text, inReplyToID)
}

// UploadMedia uploads media, falling back to twikit on account-level errors.
func (fc *FallbackClient) UploadMedia(ctx context.Context, imageData []byte, mimeType string) (string, error) {
	if !fc.primaryDisabled {
		id, primaryErr := fc.primaryPoster.UploadMedia(ctx, imageData, mimeType)
		if primaryErr == nil {
			return id, nil
		}
		fc.checkDisablePrimary(primaryErr)

		if fc.twikitPoster == nil {
			return "", fmt.Errorf("primary API failed: %w (twikit fallback unavailable)", primaryErr)
		}

		log.Printf("primary X API failed (%v), trying twikit fallback...", primaryErr)
		id, twikitErr := fc.twikitPoster.UploadMedia(ctx, imageData, mimeType)
		if twikitErr != nil {
			return "", fmt.Errorf("primary API failed: %w; twikit fallback also failed: %w", primaryErr, twikitErr)
		}
		return id, nil
	}

	if fc.twikitPoster == nil {
		return "", fmt.Errorf("primary API disabled (twikit fallback unavailable)")
	}
	return fc.twikitPoster.UploadMedia(ctx, imageData, mimeType)
}

// PostTweetWithMedia posts a tweet with media, falling back to twikit on account-level errors.
func (fc *FallbackClient) PostTweetWithMedia(ctx context.Context, text string, mediaIDs []string) (string, error) {
	if !fc.primaryDisabled {
		id, primaryErr := fc.primaryPoster.PostTweetWithMedia(ctx, text, mediaIDs)
		if primaryErr == nil {
			return id, nil
		}
		fc.checkDisablePrimary(primaryErr)

		if fc.twikitPoster == nil {
			return "", fmt.Errorf("primary API failed: %w (twikit fallback unavailable)", primaryErr)
		}

		log.Printf("primary X API failed (%v), trying twikit fallback...", primaryErr)
		id, twikitErr := fc.twikitPoster.PostTweetWithMedia(ctx, text, mediaIDs)
		if twikitErr != nil {
			return "", fmt.Errorf("primary API failed: %w; twikit fallback also failed: %w", primaryErr, twikitErr)
		}
		return id, nil
	}

	if fc.twikitPoster == nil {
		return "", fmt.Errorf("primary API disabled (twikit fallback unavailable)")
	}
	return fc.twikitPoster.PostTweetWithMedia(ctx, text, mediaIDs)
}

// PostReplyWithMedia posts a reply with media, falling back to twikit on account-level errors.
func (fc *FallbackClient) PostReplyWithMedia(ctx context.Context, text string, inReplyToID string, mediaIDs []string) (string, error) {
	if !fc.primaryDisabled {
		id, primaryErr := fc.primaryPoster.PostReplyWithMedia(ctx, text, inReplyToID, mediaIDs)
		if primaryErr == nil {
			return id, nil
		}
		fc.checkDisablePrimary(primaryErr)

		if fc.twikitPoster == nil {
			return "", fmt.Errorf("primary API failed: %w (twikit fallback unavailable)", primaryErr)
		}

		log.Printf("primary X API failed (%v), trying twikit fallback...", primaryErr)
		id, twikitErr := fc.twikitPoster.PostReplyWithMedia(ctx, text, inReplyToID, mediaIDs)
		if twikitErr != nil {
			return "", fmt.Errorf("primary API failed: %w; twikit fallback also failed: %w", primaryErr, twikitErr)
		}
		return id, nil
	}

	if fc.twikitPoster == nil {
		return "", fmt.Errorf("primary API disabled (twikit fallback unavailable)")
	}
	return fc.twikitPoster.PostReplyWithMedia(ctx, text, inReplyToID, mediaIDs)
}

// checkDisablePrimary disables the primary client for subsequent calls if the
// error indicates an account-level issue (credits depleted, auth failure, etc.).
func (fc *FallbackClient) checkDisablePrimary(err error) {
	msg := err.Error()
	if strings.Contains(msg, "status 402") ||
		strings.Contains(msg, "CreditsDepleted") ||
		strings.Contains(msg, "status 401") ||
		strings.Contains(msg, "status 403") {
		fc.primaryDisabled = true
	}
}
