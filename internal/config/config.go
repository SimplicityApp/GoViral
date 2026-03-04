package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	X        XConfig        `yaml:"x"`
	LinkedIn LinkedInConfig `yaml:"linkedin"`
	YouTube  YouTubeConfig  `yaml:"youtube"`
	TikTok   TikTokConfig   `yaml:"tiktok"`
	Claude   ClaudeConfig   `yaml:"claude"`
	Gemini   GeminiConfig   `yaml:"gemini"`
	GitHub   GitHubConfig   `yaml:"github"`
	Server   ServerConfig   `yaml:"server"`
	Daemon   DaemonConfig   `yaml:"daemon"`
	Telegram TelegramConfig `yaml:"telegram"`
	Niches         []string `yaml:"niches"`
	LinkedInNiches []string `yaml:"linkedin_niches"`
	DBPath         string   `yaml:"db_path"`
}

// YouTubeConfig contains YouTube Data API v3 credentials.
type YouTubeConfig struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	AccessToken  string `yaml:"access_token"`
	RefreshToken string `yaml:"refresh_token"`
	TokenExpiry  string `yaml:"token_expiry,omitempty"`
	ChannelID    string `yaml:"channel_id,omitempty"`
}

// TikTokConfig contains TikTok Content Posting API credentials.
type TikTokConfig struct {
	ClientKey    string `yaml:"client_key"`
	ClientSecret string `yaml:"client_secret"`
	AccessToken  string `yaml:"access_token"`
	RefreshToken string `yaml:"refresh_token"`
	TokenExpiry  string `yaml:"token_expiry,omitempty"`
	Username     string `yaml:"username"`
}

// GitHubConfig contains GitHub API settings.
type GitHubConfig struct {
	PersonalAccessToken string `yaml:"personal_access_token"`
	DefaultOwner        string `yaml:"default_owner"`
	DefaultRepo         string `yaml:"default_repo"`
}

// DaemonConfig contains autopilot daemon settings.
type DaemonConfig struct {
	Enabled            bool              `yaml:"enabled"`
	Schedules          map[string]string `yaml:"schedules"`             // platform → cron expr
	MaxPerBatch        int               `yaml:"max_per_batch"`
	AutoSkipAfter      string            `yaml:"auto_skip_after"`       // duration string e.g. "2h"
	TrendingLimit      int               `yaml:"trending_limit"`
	MinLikes           int               `yaml:"min_likes"`
	Period             string            `yaml:"period"`
	DedupActionedPosts bool              `yaml:"dedup_actioned_posts"`  // skip posts the user already acted on
	DedupLookbackHours int              `yaml:"dedup_lookback_hours"`  // how far back to look for actioned posts
	CommentsEnabled    bool             `yaml:"comments_enabled"`      // enable comment generation in daemon
	CommentsPerBatch   int              `yaml:"comments_per_batch"`    // number of comments per batch (default 3)
	DigestMode         bool             `yaml:"digest_mode"`           // true = accumulate-then-digest; false = immediate
	DigestSchedule     string           `yaml:"digest_schedule"`       // cron expr for nightly digest, e.g. "0 21 * * *"
	DigestMaxPosts     int              `yaml:"digest_max_posts"`      // max winners per digest (default 5)
	AutoPublish        bool             `yaml:"auto_publish"`          // auto-publish best content at digest time
	AutoPublishMaxPosts int             `yaml:"auto_publish_max_posts"` // safety cap per digest (default 1)
}

// TelegramConfig contains Telegram Bot API settings.
type TelegramConfig struct {
	BotToken   string `yaml:"bot_token"`
	ChatID     int64  `yaml:"chat_id"`
	WebhookURL string `yaml:"webhook_url"`
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	APIKey         string   `yaml:"api_key"`
	Port           int      `yaml:"port"`
	AllowedOrigins []string `yaml:"allowed_origins"`
}

// GeminiConfig contains Google Gemini API settings.
type GeminiConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

// XConfig contains X/Twitter API credentials.
type XConfig struct {
	APIKey            string `yaml:"api_key"`
	APISecret         string `yaml:"api_secret"`
	BearerToken       string `yaml:"bearer_token"`
	AccessToken       string `yaml:"access_token"`
	AccessTokenSecret string `yaml:"access_token_secret"`
	RefreshToken      string `yaml:"refresh_token,omitempty"`
	TokenExpiry       string `yaml:"token_expiry,omitempty"` // RFC3339 timestamp
	ClientID          string `yaml:"client_id"`
	ClientSecret      string `yaml:"client_secret"`
	Username          string `yaml:"username"`
}

// LinkedInConfig contains LinkedIn API credentials.
type LinkedInConfig struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	AccessToken  string `yaml:"access_token"`
	PersonURN    string `yaml:"person_urn,omitempty"` // LinkedIn member URN (e.g. urn:li:person:12345678)
}

// ClaudeConfig contains Anthropic Claude API settings.
type ClaudeConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

// DefaultConfigDir returns the default config directory path.
func DefaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".goviral"
	}
	return filepath.Join(home, ".goviral")
}

// DefaultConfigPath returns the default config file path.
func DefaultConfigPath() string {
	return filepath.Join(DefaultConfigDir(), "config.yaml")
}

// DefaultDBPath returns the default database path.
func DefaultDBPath() string {
	return filepath.Join(DefaultConfigDir(), "goviral.db")
}

// Load reads the config from the given path.
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if cfg.DBPath == "" {
		cfg.DBPath = DefaultDBPath()
	}

	if cfg.Claude.Model == "" {
		cfg.Claude.Model = "claude-haiku-4-5-20251001"
	}

	if cfg.Gemini.Model == "" {
		cfg.Gemini.Model = "gemini-2.0-flash-exp"
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}

	if cfg.Daemon.MaxPerBatch == 0 {
		cfg.Daemon.MaxPerBatch = 3
	}
	if cfg.Daemon.AutoSkipAfter == "" {
		cfg.Daemon.AutoSkipAfter = "2h"
	}
	if cfg.Daemon.TrendingLimit == 0 {
		cfg.Daemon.TrendingLimit = 10
	}
	if cfg.Daemon.MinLikes == 0 {
		cfg.Daemon.MinLikes = 10
	}
	if cfg.Daemon.Period == "" {
		cfg.Daemon.Period = "week"
	}
	// DedupActionedPosts defaults to true via zero-value awareness:
	// we use a separate "was it explicitly set?" check — but since YAML
	// unmarshals missing bool as false, we always default it on.
	// A user must explicitly set dedup_actioned_posts: false to disable.
	if !cfg.Daemon.DedupActionedPosts && cfg.Daemon.DedupLookbackHours == 0 {
		cfg.Daemon.DedupActionedPosts = true
	}
	if cfg.Daemon.DedupLookbackHours == 0 {
		cfg.Daemon.DedupLookbackHours = 24
	}

	if cfg.Daemon.CommentsPerBatch == 0 {
		cfg.Daemon.CommentsPerBatch = 3
	}

	if cfg.Daemon.AutoPublishMaxPosts <= 0 {
		cfg.Daemon.AutoPublishMaxPosts = 1
	}

	if cfg.Daemon.DigestSchedule == "" {
		cfg.Daemon.DigestSchedule = "0 21 * * *" // 9 PM daily
	}
	if cfg.Daemon.DigestMaxPosts <= 0 {
		cfg.Daemon.DigestMaxPosts = 5
	}

	if len(cfg.Niches) == 0 {
		cfg.Niches = []string{"AI", "Programming", "Technology"}
	}

	if len(cfg.LinkedInNiches) == 0 {
		cfg.LinkedInNiches = []string{"AI", "Programming", "Technology"}
	}

	return &cfg, nil
}

// Save writes the config to the given path.
func Save(cfg *Config, path string) error {
	if path == "" {
		path = DefaultConfigPath()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// EnsureConfigDir creates the config directory if it doesn't exist.
func EnsureConfigDir() error {
	dir := DefaultConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	return nil
}
