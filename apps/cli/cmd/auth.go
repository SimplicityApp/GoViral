package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/shuhao/goviral/internal/auth"
	"github.com/shuhao/goviral/internal/config"
)

var authPort int

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with social media platforms",
	Long:  "Run the OAuth flow to obtain access tokens for X or LinkedIn.",
	RunE:  runAuthPrompt,
}

var authLinkedinCmd = &cobra.Command{
	Use:   "linkedin",
	Short: "Authenticate with LinkedIn via OAuth 2.0",
	RunE:  runAuthLinkedin,
}

var authXCmd = &cobra.Command{
	Use:   "x",
	Short: "Authenticate with X via OAuth 2.0 with PKCE",
	RunE:  runAuthX,
}

func init() {
	authCmd.PersistentFlags().IntVar(&authPort, "port", 8080, "Local port for OAuth callback server")
	authCmd.AddCommand(authLinkedinCmd)
	authCmd.AddCommand(authXCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthPrompt(cmd *cobra.Command, args []string) error {
	var platform string

	err := huh.NewSelect[string]().
		Title("Which platform do you want to authenticate with?").
		Options(
			huh.NewOption("LinkedIn", "linkedin"),
			huh.NewOption("X", "x"),
		).
		Value(&platform).
		Run()
	if err != nil {
		return fmt.Errorf("selecting platform: %w", err)
	}

	switch platform {
	case "linkedin":
		return runAuthLinkedin(cmd, args)
	case "x":
		return runAuthX(cmd, args)
	default:
		return fmt.Errorf("unknown platform %q", platform)
	}
}

func runAuthLinkedin(cmd *cobra.Command, args []string) error {
	cfg, err := loadOrCreateConfig()
	if err != nil {
		return err
	}

	if err := auth.LinkedInAuth(cfg, authPort); err != nil {
		return fmt.Errorf("LinkedIn authentication: %w", err)
	}

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render("\nLinkedIn authentication successful!"))
	fmt.Println("Access token saved to config.")
	return nil
}

func runAuthX(cmd *cobra.Command, args []string) error {
	cfg, err := loadOrCreateConfig()
	if err != nil {
		return err
	}

	if err := auth.XAuth(cfg, authPort); err != nil {
		return fmt.Errorf("X authentication: %w", err)
	}

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render("\nX authentication successful!"))
	fmt.Println("Access token saved to config.")
	return nil
}

// loadOrCreateConfig loads config from ~/.goviral/config.yaml, then fills in
// any empty credential fields from ./config.yaml if it exists in the current directory.
func loadOrCreateConfig() (*config.Config, error) {
	cfg, err := config.Load("")
	if err != nil {
		// No global config — try local only.
		cfg, err = config.Load("config.yaml")
		if err != nil {
			if err := config.EnsureConfigDir(); err != nil {
				return nil, fmt.Errorf("creating config directory: %w", err)
			}
			return &config.Config{
				Claude: config.ClaudeConfig{Model: "claude-sonnet-4-20250514"},
				DBPath: config.DefaultDBPath(),
			}, nil
		}
		return cfg, nil
	}

	// Global config loaded — merge any missing fields from local config.yaml.
	local, err := config.Load("config.yaml")
	if err != nil {
		return cfg, nil
	}
	mergeConfig(cfg, local)
	return cfg, nil
}

// mergeConfig fills empty fields in dst with non-empty values from src.
func mergeConfig(dst, src *config.Config) {
	if dst.X.ClientID == "" {
		dst.X.ClientID = src.X.ClientID
	}
	if dst.X.ClientSecret == "" {
		dst.X.ClientSecret = src.X.ClientSecret
	}
	if dst.X.BearerToken == "" {
		dst.X.BearerToken = src.X.BearerToken
	}
	if dst.X.AccessToken == "" {
		dst.X.AccessToken = src.X.AccessToken
	}
	if dst.X.Username == "" {
		dst.X.Username = src.X.Username
	}
	if dst.LinkedIn.ClientID == "" {
		dst.LinkedIn.ClientID = src.LinkedIn.ClientID
	}
	if dst.LinkedIn.ClientSecret == "" {
		dst.LinkedIn.ClientSecret = src.LinkedIn.ClientSecret
	}
	if dst.LinkedIn.AccessToken == "" {
		dst.LinkedIn.AccessToken = src.LinkedIn.AccessToken
	}
	if dst.Claude.APIKey == "" {
		dst.Claude.APIKey = src.Claude.APIKey
	}
}
