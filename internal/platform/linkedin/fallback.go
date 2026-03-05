package linkedin

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/pkg/models"
)

// Compile-time interface compliance checks.
var _ models.PlatformClient = (*FallbackClient)(nil)
var _ models.LinkedInPoster = (*FallbackClient)(nil)
var _ models.LinkedInReposter = (*FallbackClient)(nil)
var _ models.LinkedInCommenter = (*FallbackClient)(nil)

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
	CreateComment(ctx context.Context, postURN string, threadURN string, text string) (string, error)
	CreateScheduledPost(ctx context.Context, text string, scheduledAt time.Time) (string, error)
	CreateScheduledPostWithImage(ctx context.Context, text string, imageData []byte, filename string, scheduledAt time.Time) (string, error)
}

// FallbackClient wraps the official LinkedIn API client with a linkitin fallback.
// If the official API fails, it falls back to linkitin (cookie-based Voyager API).
type FallbackClient struct {
	primary         linkedinFetcher
	linkitin        linkedinFetcher // may be nil if python is unavailable
	linkitinPoster  linkedinPoster  // may be nil if python is unavailable
	primaryDisabled bool
}

// NewFallbackClient creates a FallbackClient with the official API as primary
// and linkitin as fallback. If linkitin setup fails (e.g. no Python),
// the client operates with primary only and logs a warning.
func NewFallbackClient(cfg config.LinkedInConfig, influencerURNs []string) *FallbackClient {
	return NewFallbackClientWithConfigDir(cfg, influencerURNs, "")
}

// NewFallbackClientWithConfigDir creates a FallbackClient with a custom linkitin config directory.
// If configDir is empty, the default global path is used.
func NewFallbackClientWithConfigDir(cfg config.LinkedInConfig, influencerURNs []string, configDir string) *FallbackClient {
	primary := NewClient(cfg, influencerURNs)

	fc := &FallbackClient{
		primary:         primary,
		primaryDisabled: cfg.AccessToken == "" || cfg.PersonURN == "",
	}

	if fc.primaryDisabled {
		slog.Info("linkedin primary API disabled (no access token or person URN)")
	}

	var lc *LinkitinClient
	var err error
	if configDir != "" {
		lc, err = NewLinkitinClientWithConfigDir(configDir)
	} else {
		lc, err = NewLinkitinClient()
	}
	if err != nil {
		slog.Warn("linkitin fallback unavailable, official LinkedIn API only", "error", err)
	} else {
		fc.linkitin = lc
		fc.linkitinPoster = lc
	}

	return fc
}

// FetchMyPosts tries the official API first. On failure, falls back to linkitin if available.
func (fc *FallbackClient) FetchMyPosts(ctx context.Context, limit int) ([]models.Post, error) {
	if !fc.primaryDisabled {
		posts, primaryErr := fc.primary.FetchMyPosts(ctx, limit)
		if primaryErr == nil {
			return posts, nil
		}
		fc.checkDisablePrimary(primaryErr)

		if fc.linkitin == nil {
			return nil, fmt.Errorf("official LinkedIn API failed: %w (linkitin fallback unavailable)", primaryErr)
		}

		slog.Info("linkedin primary API failed, trying linkitin fallback", "error", primaryErr)
		posts, linkitinErr := fc.linkitin.FetchMyPosts(ctx, limit)
		if linkitinErr != nil {
			return nil, fmt.Errorf("official API failed: %w; linkitin fallback also failed: %w", primaryErr, linkitinErr)
		}
		return posts, nil
	}

	// Primary already known to be down - go straight to linkitin.
	if fc.linkitin == nil {
		return nil, fmt.Errorf("official LinkedIn API disabled (linkitin fallback unavailable)")
	}
	slog.Info("linkedin using linkitin fallback for FetchMyPosts")
	return fc.linkitin.FetchMyPosts(ctx, limit)
}

// FetchTrendingPosts tries the official API first. On failure, falls back to linkitin if available.
func (fc *FallbackClient) FetchTrendingPosts(ctx context.Context, niches []string, period string, minLikes int, limit int) ([]models.TrendingPost, error) {
	if !fc.primaryDisabled {
		posts, primaryErr := fc.primary.FetchTrendingPosts(ctx, niches, period, minLikes, limit)
		if primaryErr == nil {
			return posts, nil
		}
		fc.checkDisablePrimary(primaryErr)

		if fc.linkitin == nil {
			return nil, fmt.Errorf("official LinkedIn API failed: %w (linkitin fallback unavailable)", primaryErr)
		}

		slog.Info("linkedin primary API failed, trying linkitin fallback", "error", primaryErr)
		posts, linkitinErr := fc.linkitin.FetchTrendingPosts(ctx, niches, period, minLikes, limit)
		if linkitinErr != nil {
			return nil, fmt.Errorf("official API failed: %w; linkitin fallback also failed: %w", primaryErr, linkitinErr)
		}
		return posts, nil
	}

	if fc.linkitin == nil {
		return nil, fmt.Errorf("official LinkedIn API disabled (linkitin fallback unavailable)")
	}
	slog.Info("linkedin using linkitin fallback for FetchTrendingPosts", "niches", niches)
	return fc.linkitin.FetchTrendingPosts(ctx, niches, period, minLikes, limit)
}

// CreatePost creates a LinkedIn post via linkitin (official API has no posting support).
func (fc *FallbackClient) CreatePost(ctx context.Context, text string) (string, error) {
	if fc.linkitinPoster == nil {
		return "", fmt.Errorf("LinkedIn posting requires linkitin (cookie-based auth); linkitin is unavailable")
	}
	return fc.linkitinPoster.CreatePost(ctx, text)
}

// UploadImage uploads an image to LinkedIn via linkitin (official API has no posting support).
func (fc *FallbackClient) UploadImage(ctx context.Context, imageData []byte, filename string) (string, error) {
	if fc.linkitinPoster == nil {
		return "", fmt.Errorf("LinkedIn image upload requires linkitin (cookie-based auth); linkitin is unavailable")
	}
	return fc.linkitinPoster.UploadImage(ctx, imageData, filename)
}

// CreatePostWithImage creates a LinkedIn post with an image via linkitin (official API has no posting support).
func (fc *FallbackClient) CreatePostWithImage(ctx context.Context, text string, imageData []byte, filename string) (string, error) {
	if fc.linkitinPoster == nil {
		return "", fmt.Errorf("LinkedIn posting requires linkitin (cookie-based auth); linkitin is unavailable")
	}
	return fc.linkitinPoster.CreatePostWithImage(ctx, text, imageData, filename)
}

// Repost reshares an existing LinkedIn post via linkitin (official API has no repost support).
func (fc *FallbackClient) Repost(ctx context.Context, postURN string, text string) (string, error) {
	if fc.linkitinPoster == nil {
		return "", fmt.Errorf("LinkedIn repost requires linkitin (cookie-based auth); linkitin is unavailable")
	}
	return fc.linkitinPoster.Repost(ctx, postURN, text)
}

// CreateComment posts a comment on a LinkedIn post via linkitin (official API has no comment support).
// threadURN is the optional urn:li:ugcPost:N for ugcPost threads; pass "" to let linkitin derive it.
func (fc *FallbackClient) CreateComment(ctx context.Context, postURN string, threadURN string, text string) (string, error) {
	if fc.linkitinPoster == nil {
		slog.Error("linkedin comment skipped: linkitin unavailable (run 'goviral linkitin-login')")
		return "", fmt.Errorf("LinkedIn commenting requires linkitin (cookie-based auth); linkitin is unavailable")
	}
	slog.Info("linkedin using linkitin for comment", "post_urn", postURN)
	return fc.linkitinPoster.CreateComment(ctx, postURN, threadURN, text)
}

// CreateScheduledPost schedules a LinkedIn post via linkitin (official API has no scheduling support).
func (fc *FallbackClient) CreateScheduledPost(ctx context.Context, text string, scheduledAt time.Time) (string, error) {
	if fc.linkitinPoster == nil {
		return "", fmt.Errorf("LinkedIn scheduling requires linkitin (cookie-based auth); linkitin is unavailable")
	}
	return fc.linkitinPoster.CreateScheduledPost(ctx, text, scheduledAt)
}

// CreateScheduledPostWithImage schedules a LinkedIn post with an image via linkitin (official API has no scheduling support).
func (fc *FallbackClient) CreateScheduledPostWithImage(ctx context.Context, text string, imageData []byte, filename string, scheduledAt time.Time) (string, error) {
	if fc.linkitinPoster == nil {
		return "", fmt.Errorf("LinkedIn scheduling requires linkitin (cookie-based auth); linkitin is unavailable")
	}
	return fc.linkitinPoster.CreateScheduledPostWithImage(ctx, text, imageData, filename, scheduledAt)
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
