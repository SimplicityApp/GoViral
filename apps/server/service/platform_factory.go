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
var newXPoster = func(cfg config.XConfig) models.PlatformPoster {
	return x.NewFallbackClient(cfg)
}

var newXScheduler = func(cfg config.XConfig) models.PlatformScheduler {
	return x.NewFallbackClient(cfg)
}

var newXQuotePoster = func(cfg config.XConfig) models.QuotePoster {
	return x.NewFallbackClient(cfg)
}

var newXQuoteScheduler = func(cfg config.XConfig) models.QuoteScheduler {
	return x.NewFallbackClient(cfg)
}

var newLinkedInPoster = func(cfg config.LinkedInConfig) models.LinkedInPoster {
	return linkedin.NewFallbackClient(cfg, nil)
}

var newLinkedInReposter = func(cfg config.LinkedInConfig) models.LinkedInReposter {
	return linkedin.NewFallbackClient(cfg, nil)
}

var newLinkedInCommenter = func(cfg config.LinkedInConfig) models.LinkedInCommenter {
	return linkedin.NewFallbackClient(cfg, nil)
}

var newYouTubePoster = func(cfg config.YouTubeConfig) models.YouTubePoster {
	return youtube.NewFallbackClient(cfg)
}

var newTikTokPoster = func(cfg config.TikTokConfig) models.TikTokPoster {
	return tiktok.NewFallbackClient(cfg)
}
