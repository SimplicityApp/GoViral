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
	Claude   ClaudeConfig   `yaml:"claude"`
	Gemini   GeminiConfig   `yaml:"gemini"`
	Niches   []string       `yaml:"niches"`
	DBPath   string         `yaml:"db_path"`
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
		cfg.Claude.Model = "claude-sonnet-4-20250514"
	}

	if cfg.Gemini.Model == "" {
		cfg.Gemini.Model = "gemini-2.0-flash-exp"
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
