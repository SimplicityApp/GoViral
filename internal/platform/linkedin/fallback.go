package linkedin

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

// linkedinFetcher is an internal interface for testability.
type linkedinFetcher interface {
	FetchMyPosts(ctx context.Context, limit int) ([]models.Post, error)
	FetchTrendingPosts(ctx context.Context, niches []string, period string, minLikes int, limit int) ([]models.TrendingPost, error)
}

// FallbackClient wraps the official LinkedIn API client with a likit fallback.
// If the official API fails, it falls back to likit (cookie-based Voyager API).
type FallbackClient struct {
	primary         linkedinFetcher
	likit           linkedinFetcher // may be nil if python is unavailable
	primaryDisabled bool
}

// NewFallbackClient creates a FallbackClient with the official API as primary
// and likit as fallback. If likit setup fails (e.g. no Python),
// the client operates with primary only and logs a warning.
func NewFallbackClient(cfg config.LinkedInConfig, influencerURNs []string) *FallbackClient {
	primary := NewClient(cfg, influencerURNs)

	fc := &FallbackClient{
		primary: primary,
	}

	lc, err := NewLikitClient()
	if err != nil {
		log.Printf("likit fallback unavailable: %v (official LinkedIn API only)", err)
	} else {
		fc.likit = lc
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
