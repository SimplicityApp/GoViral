package service

import (
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/platform/linkedin"
	"github.com/shuhao/goviral/internal/platform/x"
	"github.com/shuhao/goviral/pkg/models"
)

// NewXPoster creates an X platform poster from the config.
func NewXPoster(cfg config.XConfig) models.PlatformPoster {
	return x.NewFallbackClient(cfg)
}

// NewXScheduler creates an X platform scheduler from the config.
func NewXScheduler(cfg config.XConfig) models.PlatformScheduler {
	return x.NewFallbackClient(cfg)
}

// NewXQuotePoster creates an X platform quote poster from the config.
func NewXQuotePoster(cfg config.XConfig) models.QuotePoster {
	return x.NewFallbackClient(cfg)
}

// NewXQuoteScheduler creates an X platform quote tweet scheduler from the config.
func NewXQuoteScheduler(cfg config.XConfig) models.QuoteScheduler {
	return x.NewFallbackClient(cfg)
}

// NewLinkedInPoster creates a LinkedIn platform poster from the config.
func NewLinkedInPoster(cfg config.LinkedInConfig) models.LinkedInPoster {
	return linkedin.NewFallbackClient(cfg, nil)
}

// NewLinkedInReposter creates a LinkedIn reposter from the config.
func NewLinkedInReposter(cfg config.LinkedInConfig) models.LinkedInReposter {
	return linkedin.NewFallbackClient(cfg, nil)
}
