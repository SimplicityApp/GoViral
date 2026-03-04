package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/internal/ai/generator"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/internal/platform/linkedin"
	"github.com/shuhao/goviral/pkg/models"
)

var (
	commentPostURN string
	commentText    string
	commentID      int64
	commentDryRun  bool
	commentCount   int
)

var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Generate and post AI comments on LinkedIn posts",
	Long: `Generate AI-powered comments on LinkedIn posts using your persona profile.

Examples:
  goviral comment --post-urn "urn:li:activity:1234"              # AI flow: generate and pick
  goviral comment --post-urn "urn:li:activity:1234" --count 5   # Generate 5 variations
  goviral comment --post-urn "urn:li:activity:1234" --dry-run   # Preview without posting
  goviral comment --text "Great post!" --post-urn "urn:li:activity:1234"  # Post directly
  goviral comment --id 42                                         # Post by stored content ID`,
	RunE: runComment,
}

func init() {
	commentCmd.Flags().StringVar(&commentPostURN, "post-urn", "", "LinkedIn post URN to comment on")
	commentCmd.Flags().StringVar(&commentText, "text", "", "Comment text (skip AI generation)")
	commentCmd.Flags().Int64Var(&commentID, "id", 0, "Use existing generated comment content ID")
	commentCmd.Flags().BoolVar(&commentDryRun, "dry-run", false, "Preview without posting")
	commentCmd.Flags().IntVar(&commentCount, "count", 3, "Number of comment variations to generate")
	rootCmd.AddCommand(commentCmd)
}

func runComment(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	// Flow 1: post by stored content ID
	if commentID > 0 {
		return runCommentByID(cfg, database, commentID)
	}

	// Flow 2: post directly with --text and --post-urn
	if commentText != "" {
		if commentPostURN == "" {
			return fmt.Errorf("--post-urn is required when using --text")
		}
		return runCommentDirect(cfg, commentPostURN, commentText)
	}

	// Flow 3: AI generation flow
	if commentPostURN == "" {
		return fmt.Errorf("--post-urn is required; specify the LinkedIn post URN to comment on")
	}
	return runCommentAI(cfg, database, commentPostURN)
}

// runCommentByID fetches a stored comment by ID, previews it, and posts it.
func runCommentByID(cfg *config.Config, database *db.DB, id int64) error {
	gc, err := database.GetGeneratedContentByID("", id)
	if err != nil {
		return fmt.Errorf("fetching content: %w", err)
	}
	if gc == nil {
		return fmt.Errorf("no generated content found with ID %d", id)
	}
	if !gc.IsComment {
		return fmt.Errorf("content ID %d is not a comment (IsComment=false)", id)
	}
	if gc.Status == "posted" {
		return fmt.Errorf("content ID %d has already been posted (URN: %s)", gc.ID, gc.PlatformPostIDs)
	}

	postURN := gc.QuoteTweetID // URN stored in QuoteTweetID for comments

	displayCommentPreview(gc.GeneratedContent, postURN)

	if commentDryRun {
		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
		fmt.Println(infoStyle.Render("(dry run — nothing was posted)"))
		return nil
	}

	if !confirmAction("Post this comment?") {
		fmt.Println("Cancelled.")
		return nil
	}

	client := linkedin.NewFallbackClient(cfg.LinkedIn, nil)
	commentURN, err := client.CreateComment(context.Background(), postURN, "", gc.GeneratedContent)
	if err != nil {
		return fmt.Errorf("posting comment: %w", err)
	}

	if err := database.UpdateGeneratedContentPosted("", gc.ID, commentURN); err != nil {
		return fmt.Errorf("updating database: %w", err)
	}

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render("\nComment posted successfully!"))
	if commentURN != "" {
		fmt.Printf("Comment URN: %s\n", commentURN)
	}

	return nil
}

// runCommentDirect posts a comment with the provided text directly, skipping AI generation.
func runCommentDirect(cfg *config.Config, postURN, text string) error {
	displayCommentPreview(text, postURN)

	if commentDryRun {
		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
		fmt.Println(infoStyle.Render("(dry run — nothing was posted)"))
		return nil
	}

	if !confirmAction("Post this comment?") {
		fmt.Println("Cancelled.")
		return nil
	}

	client := linkedin.NewFallbackClient(cfg.LinkedIn, nil)
	commentURN, err := client.CreateComment(context.Background(), postURN, "", text)
	if err != nil {
		return fmt.Errorf("posting comment: %w", err)
	}

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render("\nComment posted successfully!"))
	if commentURN != "" {
		fmt.Printf("Comment URN: %s\n", commentURN)
	}

	return nil
}

// runCommentAI generates comment variations with AI, lets the user pick one, and posts it.
func runCommentAI(cfg *config.Config, database *db.DB, postURN string) error {
	// Look up persona
	persona, err := database.GetPersona("", "linkedin")
	if err != nil {
		return fmt.Errorf("fetching persona: %w", err)
	}
	if persona == nil {
		persona, err = database.GetPersona("", "all")
		if err != nil {
			return fmt.Errorf("fetching persona: %w", err)
		}
	}
	if persona == nil {
		return fmt.Errorf("no persona profile found; run 'goviral profile build' first")
	}

	// Resolve trending post by platform_post_id or synthesize from URN
	trendingPost := resolveTrendingPost(database, postURN)

	// Generate comment variations
	claudeClient := claude.NewClient(cfg.Claude.APIKey, cfg.Claude.Model)
	gen := generator.NewGenerator(claudeClient)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	fmt.Println(titleStyle.Render(fmt.Sprintf("\nGenerating %d comment variations...", commentCount)))

	results, err := gen.GenerateComment(context.Background(), models.GenerateCommentRequest{
		TrendingPost:   trendingPost,
		Persona:        *persona,
		TargetPlatform: "linkedin",
		Count:          commentCount,
	})
	if err != nil {
		return fmt.Errorf("generating comments: %w", err)
	}

	if len(results) == 0 {
		return fmt.Errorf("no comment variations were generated")
	}

	// Display variations
	fmt.Println()
	for i, r := range results {
		displayCommentVariation(i+1, &r)
	}

	// Let user pick one
	chosen, err := pickCommentVariation(results)
	if err != nil {
		return err
	}
	if chosen == nil {
		fmt.Println("No variation selected.")
		return nil
	}

	displayCommentPreview(chosen.Content, postURN)

	// Save to DB
	gc := &models.GeneratedContent{
		SourceTrendingID: trendingPost.ID,
		TargetPlatform:   "linkedin",
		OriginalContent:  trendingPost.Content,
		GeneratedContent: chosen.Content,
		PersonaID:        persona.ID,
		PromptUsed:       "comment-linkedin",
		Status:           "draft",
		IsComment:        true,
		QuoteTweetID:     postURN, // reuse field to store the parent post URN
	}
	contentID, err := database.InsertGeneratedContent("", gc)
	if err != nil {
		warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
		fmt.Println(warnStyle.Render(fmt.Sprintf("Warning: failed to save to database: %v", err)))
	} else {
		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		fmt.Println(infoStyle.Render(fmt.Sprintf("Saved as content ID %d", contentID)))
	}

	if commentDryRun {
		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
		fmt.Println(infoStyle.Render("(dry run — nothing was posted)"))
		return nil
	}

	if !confirmAction("Post this comment?") {
		fmt.Println("Cancelled.")
		return nil
	}

	client := linkedin.NewFallbackClient(cfg.LinkedIn, nil)
	commentURN, err := client.CreateComment(context.Background(), postURN, "", chosen.Content)
	if err != nil {
		return fmt.Errorf("posting comment: %w", err)
	}

	if contentID > 0 {
		if err := database.UpdateGeneratedContentPosted("", contentID, commentURN); err != nil {
			warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
			fmt.Println(warnStyle.Render(fmt.Sprintf("Warning: failed to update database: %v", err)))
		}
	}

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render("\nComment posted successfully!"))
	if commentURN != "" {
		fmt.Printf("Comment URN: %s\n", commentURN)
	}

	return nil
}

// resolveTrendingPost finds a trending post by its platform_post_id (URN) in the DB.
// If not found, it returns a minimal TrendingPost constructed from the URN so generation still works.
func resolveTrendingPost(database *db.DB, postURN string) models.TrendingPost {
	posts, err := database.GetTrendingPosts("linkedin", 0)
	if err == nil {
		for _, p := range posts {
			if p.PlatformPostID == postURN {
				return p
			}
		}
	}

	// Not in DB — synthesize a minimal TrendingPost so the generator has something to work with.
	return models.TrendingPost{
		Platform:       "linkedin",
		PlatformPostID: postURN,
		Content:        fmt.Sprintf("[Post URN: %s]", postURN),
	}
}

// displayCommentPreview renders a single comment preview card.
func displayCommentPreview(text, postURN string) {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("14")).
		Padding(0, 1).
		Width(72)
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	fmt.Println(headerStyle.Render("\nComment preview:"))
	meta := metaStyle.Render(fmt.Sprintf("Post: %s\n", postURN))
	fmt.Println(cardStyle.Render(meta + text))
	fmt.Println()
}

// displayCommentVariation renders a single AI-generated comment variation.
func displayCommentVariation(num int, r *models.GenerateResult) {
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("10")).
		Padding(0, 1).
		Width(72)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	scoreStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))

	header := headerStyle.Render(fmt.Sprintf("Variation #%d", num))
	score := scoreStyle.Render(fmt.Sprintf("Confidence: %d/10", r.ConfidenceScore))
	mechanic := metaStyle.Render(fmt.Sprintf("Tactic: %s", r.ViralMechanic))

	body := fmt.Sprintf("%s  %s\n%s\n\n%s", header, score, mechanic, r.Content)
	fmt.Println(cardStyle.Render(body))
	fmt.Println()
}

// pickCommentVariation prompts the user to select one of the generated comment variations.
func pickCommentVariation(results []models.GenerateResult) (*models.GenerateResult, error) {
	fmt.Print("Select a variation (number): ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)

	var idx int
	if _, err := fmt.Sscanf(line, "%d", &idx); err != nil || idx < 1 || idx > len(results) {
		return nil, nil
	}

	chosen := results[idx-1]
	return &chosen, nil
}

// confirmAction prompts the user for a yes/no confirmation and returns true for "y".
func confirmAction(prompt string) bool {
	fmt.Printf("%s (y/N) ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(strings.ToLower(line)) == "y"
}
