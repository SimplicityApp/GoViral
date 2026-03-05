package service

import (
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/platform/linkedin"
	"github.com/shuhao/goviral/internal/platform/tiktok"
	"github.com/shuhao/goviral/internal/platform/x"
	"github.com/shuhao/goviral/internal/platform/youtube"
	"github.com/shuhao/goviral/pkg/models"
)

// Package-level factory variables. Tests can swap these to inject mocks.
// Each factory accepts a cookie path (twikit) or config dir (linkitin) for per-user isolation.
// Pass empty string to use the default global path.
var newXPoster = func(cfg config.XConfig, cookiePath string) models.PlatformPoster {
	return x.NewFallbackClientWithCookiePath(cfg, cookiePath)
}

var newXScheduler = func(cfg config.XConfig, cookiePath string) models.PlatformScheduler {
	return x.NewFallbackClientWithCookiePath(cfg, cookiePath)
}

var newXQuotePoster = func(cfg config.XConfig, cookiePath string) models.QuotePoster {
	return x.NewFallbackClientWithCookiePath(cfg, cookiePath)
}

var newXQuoteScheduler = func(cfg config.XConfig, cookiePath string) models.QuoteScheduler {
	return x.NewFallbackClientWithCookiePath(cfg, cookiePath)
}

var newLinkedInPoster = func(cfg config.LinkedInConfig, configDir string) models.LinkedInPoster {
	return linkedin.NewFallbackClientWithConfigDir(cfg, nil, configDir)
}

var newLinkedInReposter = func(cfg config.LinkedInConfig, configDir string) models.LinkedInReposter {
	return linkedin.NewFallbackClientWithConfigDir(cfg, nil, configDir)
}

var newLinkedInCommenter = func(cfg config.LinkedInConfig, configDir string) models.LinkedInCommenter {
	return linkedin.NewFallbackClientWithConfigDir(cfg, nil, configDir)
}

var newYouTubePoster = func(cfg config.YouTubeConfig) models.YouTubePoster {
	return youtube.NewFallbackClient(cfg)
}

var newTikTokPoster = func(cfg config.TikTokConfig) models.TikTokPoster {
	return tiktok.NewFallbackClient(cfg)
}
