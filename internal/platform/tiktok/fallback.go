package tiktok

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
var _ models.TikTokPoster = (*FallbackClient)(nil)

// tiktokPoster is an internal interface for testability.
type tiktokPoster interface {
	UploadVideo(ctx context.Context, videoPath string, description string, tags []string) (string, error)
	ScheduleVideo(ctx context.Context, videoPath string, description string, tags []string, scheduledAt time.Time) (string, error)
}

// FallbackClient wraps the official TikTok API client with a tiktok-uploader fallback.
type FallbackClient struct {
	primary         tiktokPoster
	fallback        tiktokPoster // may be nil if python is unavailable
	primaryDisabled bool
}

// NewFallbackClient creates a FallbackClient with the official TikTok API as primary
// and tiktok-uploader (Playwright-based) as fallback.
func NewFallbackClient(cfg config.TikTokConfig) *FallbackClient {
	primary := NewClient(cfg)

	fc := &FallbackClient{
		primary:         primary,
		primaryDisabled: cfg.AccessToken == "",
	}

	if fc.primaryDisabled {
		slog.Info("tiktok primary API disabled (no access token)")
	}

	uploader, err := NewUploaderClient()
	if err != nil {
		slog.Warn("tiktok-uploader fallback unavailable, official API only", "error", err)
	} else {
		fc.fallback = uploader
	}

	return fc
}

// UploadVideo uploads a video, falling back to tiktok-uploader on failure.
func (fc *FallbackClient) UploadVideo(ctx context.Context, videoPath string, description string, tags []string) (string, error) {
	if !fc.primaryDisabled {
		id, err := fc.primary.UploadVideo(ctx, videoPath, description, tags)
		if err == nil {
			return id, nil
		}
		fc.checkDisablePrimary(err)

		if fc.fallback == nil {
			return "", fmt.Errorf("TikTok API failed: %w (tiktok-uploader fallback unavailable)", err)
		}

		slog.Info("tiktok primary API failed, trying tiktok-uploader fallback", "error", err)
		id, fallbackErr := fc.fallback.UploadVideo(ctx, videoPath, description, tags)
		if fallbackErr != nil {
			return "", fmt.Errorf("TikTok API failed: %w; tiktok-uploader also failed: %w", err, fallbackErr)
		}
		return id, nil
	}

	if fc.fallback == nil {
		return "", fmt.Errorf("TikTok API disabled (tiktok-uploader fallback unavailable)")
	}
	slog.Info("tiktok using tiktok-uploader fallback for UploadVideo")
	return fc.fallback.UploadVideo(ctx, videoPath, description, tags)
}

// ScheduleVideo schedules a video, falling back to tiktok-uploader on failure.
func (fc *FallbackClient) ScheduleVideo(ctx context.Context, videoPath string, description string, tags []string, scheduledAt time.Time) (string, error) {
	if !fc.primaryDisabled {
		id, err := fc.primary.ScheduleVideo(ctx, videoPath, description, tags, scheduledAt)
		if err == nil {
			return id, nil
		}
		fc.checkDisablePrimary(err)

		if fc.fallback == nil {
			return "", fmt.Errorf("TikTok API failed: %w (tiktok-uploader fallback unavailable)", err)
		}

		slog.Info("tiktok primary API failed, trying tiktok-uploader fallback", "error", err)
		id, fallbackErr := fc.fallback.ScheduleVideo(ctx, videoPath, description, tags, scheduledAt)
		if fallbackErr != nil {
			return "", fmt.Errorf("TikTok API failed: %w; tiktok-uploader also failed: %w", err, fallbackErr)
		}
		return id, nil
	}

	if fc.fallback == nil {
		return "", fmt.Errorf("TikTok API disabled (tiktok-uploader fallback unavailable)")
	}
	slog.Info("tiktok using tiktok-uploader fallback for ScheduleVideo")
	return fc.fallback.ScheduleVideo(ctx, videoPath, description, tags, scheduledAt)
}

// checkDisablePrimary disables the primary client on auth errors.
func (fc *FallbackClient) checkDisablePrimary(err error) {
	msg := err.Error()
	if strings.Contains(msg, "status 401") ||
		strings.Contains(msg, "status 403") ||
		strings.Contains(msg, "no TikTok access token") ||
		strings.Contains(msg, "access_token") {
		fc.primaryDisabled = true
	}
}
