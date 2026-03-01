package youtube

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/pkg/models"
)

// Compile-time interface compliance checks.
var _ models.YouTubePoster = (*FallbackClient)(nil)

// youtubePoster is an internal interface for testability.
type youtubePoster interface {
	UploadVideo(ctx context.Context, videoPath string, title string, description string, tags []string) (string, error)
	UploadVideoWithThumbnail(ctx context.Context, videoPath string, thumbnailPath string, title string, description string, tags []string) (string, error)
}

// FallbackClient wraps the official YouTube API client with a Python bridge fallback.
type FallbackClient struct {
	primary         youtubePoster
	fallback        youtubePoster // may be nil if python is unavailable
	primaryDisabled bool
}

// NewFallbackClient creates a FallbackClient with the official YouTube API as primary
// and a Python google-api-python-client bridge as fallback.
func NewFallbackClient(cfg config.YouTubeConfig) *FallbackClient {
	primary := NewClient(cfg)

	fc := &FallbackClient{
		primary:         primary,
		primaryDisabled: cfg.AccessToken == "",
	}

	if fc.primaryDisabled {
		slog.Info("youtube primary API disabled (no access token)")
	}

	bridge, err := NewBridgeClient()
	if err != nil {
		slog.Warn("youtube python bridge unavailable, official API only", "error", err)
	} else {
		fc.fallback = bridge
	}

	return fc
}

// UploadVideo uploads a video, falling back to Python bridge on failure.
func (fc *FallbackClient) UploadVideo(ctx context.Context, videoPath string, title string, description string, tags []string) (string, error) {
	if !fc.primaryDisabled {
		id, err := fc.primary.UploadVideo(ctx, videoPath, title, description, tags)
		if err == nil {
			return id, nil
		}
		fc.checkDisablePrimary(err)

		if fc.fallback == nil {
			return "", fmt.Errorf("YouTube API failed: %w (python bridge unavailable)", err)
		}

		slog.Info("youtube primary API failed, trying python bridge", "error", err)
		id, fallbackErr := fc.fallback.UploadVideo(ctx, videoPath, title, description, tags)
		if fallbackErr != nil {
			return "", fmt.Errorf("YouTube API failed: %w; python bridge also failed: %w", err, fallbackErr)
		}
		return id, nil
	}

	if fc.fallback == nil {
		return "", fmt.Errorf("YouTube API disabled (python bridge unavailable)")
	}
	slog.Info("youtube using python bridge for UploadVideo")
	return fc.fallback.UploadVideo(ctx, videoPath, title, description, tags)
}

// UploadVideoWithThumbnail uploads a video with thumbnail, falling back to Python bridge.
func (fc *FallbackClient) UploadVideoWithThumbnail(ctx context.Context, videoPath string, thumbnailPath string, title string, description string, tags []string) (string, error) {
	if !fc.primaryDisabled {
		id, err := fc.primary.UploadVideoWithThumbnail(ctx, videoPath, thumbnailPath, title, description, tags)
		if err == nil {
			return id, nil
		}
		fc.checkDisablePrimary(err)

		if fc.fallback == nil {
			return "", fmt.Errorf("YouTube API failed: %w (python bridge unavailable)", err)
		}

		slog.Info("youtube primary API failed, trying python bridge", "error", err)
		id, fallbackErr := fc.fallback.UploadVideoWithThumbnail(ctx, videoPath, thumbnailPath, title, description, tags)
		if fallbackErr != nil {
			return "", fmt.Errorf("YouTube API failed: %w; python bridge also failed: %w", err, fallbackErr)
		}
		return id, nil
	}

	if fc.fallback == nil {
		return "", fmt.Errorf("YouTube API disabled (python bridge unavailable)")
	}
	slog.Info("youtube using python bridge for UploadVideoWithThumbnail")
	return fc.fallback.UploadVideoWithThumbnail(ctx, videoPath, thumbnailPath, title, description, tags)
}

// checkDisablePrimary disables the primary client on auth errors.
func (fc *FallbackClient) checkDisablePrimary(err error) {
	msg := err.Error()
	if strings.Contains(msg, "status 401") ||
		strings.Contains(msg, "status 403") ||
		strings.Contains(msg, "no YouTube access token") ||
		strings.Contains(msg, "access_token") {
		fc.primaryDisabled = true
	}
}
