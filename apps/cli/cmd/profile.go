package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/internal/ai/persona"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/pkg/models"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage your persona profile",
}

var profileBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build your persona profile from fetched posts",
	RunE:  runProfileBuild,
}

var profileShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display your current persona profile",
	RunE:  runProfileShow,
}

var profileRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh your persona profile with latest posts",
	RunE:  runProfileBuild, // Same logic as build
}

func init() {
	profileCmd.AddCommand(profileBuildCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileRefreshCmd)
	rootCmd.AddCommand(profileCmd)
}

func runProfileBuild(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	posts, err := database.GetAllPosts()
	if err != nil {
		return fmt.Errorf("fetching posts from database: %w", err)
	}

	if len(posts) == 0 {
		return fmt.Errorf("no posts found; run 'goviral fetch' first")
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	fmt.Println(titleStyle.Render(fmt.Sprintf("Analyzing %d posts to build persona profile...", len(posts))))

	claudeClient := claude.NewClient(cfg.Claude.APIKey, cfg.Claude.Model)
	analyzer := persona.NewAnalyzer(claudeClient)

	ctx := context.Background()

	// Determine platform — use "all" if we have posts from multiple platforms
	platform := determinePlatform(posts)

	profile, err := analyzer.BuildProfile(ctx, posts, platform)
	if err != nil {
		return fmt.Errorf("building persona profile: %w", err)
	}

	p := &models.Persona{
		Platform: platform,
		Profile:  *profile,
	}
	if err := database.UpsertPersona(p); err != nil {
		return fmt.Errorf("saving persona: %w", err)
	}

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render("Persona profile built and saved!"))
	fmt.Println()
	displayProfile(profile)

	return nil
}

func runProfileShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	// Try "all" first, then individual platforms
	for _, platform := range []string{"all", "x", "linkedin"} {
		p, err := database.GetPersona(platform)
		if err != nil {
			return fmt.Errorf("fetching persona: %w", err)
		}
		if p != nil {
			titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
			fmt.Println(titleStyle.Render(fmt.Sprintf("Persona Profile (%s)", p.Platform)))
			fmt.Printf("Last updated: %s\n\n", p.UpdatedAt.Format("2006-01-02 15:04:05"))
			displayProfile(&p.Profile)
			return nil
		}
	}

	return fmt.Errorf("no persona profile found; run 'goviral profile build' first")
}

func displayProfile(p *models.PersonaProfile) {
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))

	fields := []struct {
		label string
		value string
	}{
		{"Writing Tone", p.WritingTone},
		{"Typical Length", p.TypicalLength},
		{"Vocabulary Level", p.VocabularyLevel},
		{"Engagement Patterns", p.EngagementPatterns},
		{"Structural Patterns", strings.Join(p.StructuralPatterns, ", ")},
		{"Emoji Usage", p.EmojiUsage},
		{"Hashtag Usage", p.HashtagUsage},
		{"Call to Action Style", p.CallToActionStyle},
	}

	for _, f := range fields {
		fmt.Printf("%s %s\n", labelStyle.Render(f.label+":"), valueStyle.Render(f.value))
	}

	if len(p.CommonThemes) > 0 {
		themesJSON, _ := json.Marshal(p.CommonThemes)
		fmt.Printf("%s %s\n", labelStyle.Render("Common Themes:"), valueStyle.Render(string(themesJSON)))
	}

	if len(p.UniqueQuirks) > 0 {
		quirksJSON, _ := json.Marshal(p.UniqueQuirks)
		fmt.Printf("%s %s\n", labelStyle.Render("Unique Quirks:"), valueStyle.Render(string(quirksJSON)))
	}

	fmt.Printf("\n%s\n%s\n", labelStyle.Render("Voice Summary:"), valueStyle.Render(p.VoiceSummary))
}

func determinePlatform(posts []models.Post) string {
	platforms := make(map[string]bool)
	for _, p := range posts {
		platforms[p.Platform] = true
	}
	if len(platforms) > 1 {
		return "all"
	}
	for p := range platforms {
		return p
	}
	return "all"
}
