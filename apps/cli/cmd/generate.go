package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/internal/ai/gemini"
	"github.com/shuhao/goviral/internal/ai/generator"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/pkg/models"
)

var (
	generatePlatform string
	generateAuto     bool
	generateCount    int
	generateMaxChars int
	generateImages   bool
	generateRepost   bool
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate viral content based on trending posts",
	RunE:  runGenerate,
}

func init() {
	generateCmd.Flags().StringVarP(&generatePlatform, "platform", "p", "x", "Target platform (x, linkedin)")
	generateCmd.Flags().BoolVar(&generateAuto, "auto", false, "Auto-pick top trending posts")
	generateCmd.Flags().IntVarP(&generateCount, "count", "c", 3, "Number of variations per trending post")
	generateCmd.Flags().IntVar(&generateMaxChars, "max-chars", 0, "Maximum characters per generated post (0 = no limit)")
	generateCmd.Flags().BoolVar(&generateImages, "images", false, "Auto-generate images via Gemini when suggested")
	generateCmd.Flags().BoolVar(&generateRepost, "repost", false, "Generate quote tweet commentary instead of full rewrites")
	rootCmd.AddCommand(generateCmd)
}

func runGenerate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	// Get persona
	personaModel, err := findPersona(database)
	if err != nil {
		return err
	}

	// Get trending posts
	trending, err := database.GetTrendingPosts(generatePlatform, 20)
	if err != nil {
		return fmt.Errorf("fetching trending posts: %w", err)
	}
	if len(trending) == 0 {
		return fmt.Errorf("no trending posts found; run 'goviral trending' first")
	}

	// Select posts
	var selected []models.TrendingPost
	if generateAuto {
		limit := 5
		if len(trending) < limit {
			limit = len(trending)
		}
		selected = trending[:limit]
	} else {
		selected, err = interactiveSelect(trending)
		if err != nil {
			return err
		}
	}

	if len(selected) == 0 {
		fmt.Println("No posts selected.")
		return nil
	}

	// Generate content
	claudeClient := claude.NewClient(cfg.Claude.APIKey, cfg.Claude.Model)
	gen := generator.NewGenerator(claudeClient)
	ctx := context.Background()

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))

	for _, tp := range selected {
		if generateRepost {
			fmt.Println(titleStyle.Render(fmt.Sprintf("\nGenerating %d repost commentaries for post by @%s...", generateCount, tp.AuthorUsername)))
		} else {
			fmt.Println(titleStyle.Render(fmt.Sprintf("\nGenerating %d variations for post by @%s...", generateCount, tp.AuthorUsername)))
		}

		maxChars := generateMaxChars
		if generateRepost && maxChars == 0 {
			maxChars = 200
		}

		req := models.GenerateRequest{
			TrendingPost:   tp,
			Persona:        *personaModel,
			TargetPlatform: generatePlatform,
			Niches:         cfg.Niches,
			Count:          generateCount,
			MaxChars:       maxChars,
			IsRepost:       generateRepost,
		}

		results, err := gen.Generate(ctx, req)
		if err != nil {
			fmt.Printf("  Error generating: %v\n", err)
			continue
		}

		for i, r := range results {
			displayGeneratedContent(i+1, &r)

			// Separate image flow
			var imagePath string
			var imagePrompt string

			if generateImages {
				// --images flag: skip decision, always generate image prompt
				imgPrompt, err := gen.GenerateImagePrompt(ctx, r.Content, generatePlatform)
				if err != nil {
					fmt.Printf("  Warning: image prompt generation failed: %v\n", err)
				} else {
					imagePrompt = imgPrompt
				}
			} else {
				// Ask Claude if an image would help
				decision, err := gen.DecideImage(ctx, r.Content, generatePlatform)
				if err != nil {
					// Silently skip image decision on error
				} else if decision.SuggestImage {
					displayImageSuggestion(decision.Reasoning)

					fmt.Print("Generate image? (y/N) ")
					reader := bufio.NewReader(os.Stdin)
					line, _ := reader.ReadString('\n')
					if strings.TrimSpace(strings.ToLower(line)) == "y" {
						imgPrompt, err := gen.GenerateImagePrompt(ctx, r.Content, generatePlatform)
						if err != nil {
							fmt.Printf("  Warning: image prompt generation failed: %v\n", err)
						} else {
							imagePrompt = imgPrompt
						}
					}
				}
			}

			// Generate actual image if we have a prompt
			if imagePrompt != "" {
				if cfg.Gemini.APIKey != "" {
					imgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
					fmt.Println(imgStyle.Render("Generating image via Gemini..."))

					geminiClient := gemini.NewClient(cfg.Gemini.APIKey, cfg.Gemini.Model)
					img, err := geminiClient.GenerateImage(ctx, imagePrompt)
					if err != nil {
						fmt.Printf("  Warning: image generation failed: %v\n", err)
					} else {
						name := fmt.Sprintf("gen_%d_%d_%d", tp.ID, i+1, time.Now().Unix())
						path, err := gemini.SaveImage(img, name)
						if err != nil {
							fmt.Printf("  Warning: failed to save image: %v\n", err)
						} else {
							imagePath = path
							successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
							fmt.Println(successStyle.Render(fmt.Sprintf("Image saved: %s", path)))
						}
					}
				} else {
					hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
					fmt.Println(hintStyle.Render("Add gemini.api_key to ~/.goviral/config.yaml to enable image generation"))
				}
			}

			// Store in database
			promptUsed := fmt.Sprintf("rewrite-%s", generatePlatform)
			if generateRepost {
				promptUsed = fmt.Sprintf("repost-%s", generatePlatform)
			}
			gc := &models.GeneratedContent{
				SourceTrendingID: tp.ID,
				TargetPlatform:   generatePlatform,
				OriginalContent:  tp.Content,
				GeneratedContent: r.Content,
				PersonaID:        personaModel.ID,
				PromptUsed:       promptUsed,
				Status:           "draft",
				ImagePrompt:      imagePrompt,
				ImagePath:        imagePath,
				IsRepost:         generateRepost,
			}
			if generateRepost {
				gc.QuoteTweetID = tp.PlatformPostID
			}
			if _, err := database.InsertGeneratedContent(gc); err != nil {
				fmt.Printf("  Warning: failed to save: %v\n", err)
			}
		}
	}

	return nil
}

func findPersona(database *db.DB) (*models.Persona, error) {
	for _, platform := range []string{"all", "x", "linkedin"} {
		p, err := database.GetPersona(platform)
		if err != nil {
			return nil, fmt.Errorf("fetching persona: %w", err)
		}
		if p != nil {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no persona profile found; run 'goviral profile build' first")
}

func interactiveSelect(trending []models.TrendingPost) ([]models.TrendingPost, error) {
	fmt.Println("\nAvailable trending posts:")
	for i, tp := range trending {
		preview := tp.Content
		if len(preview) > 80 {
			preview = preview[:77] + "..."
		}
		fmt.Printf("  [%d] @%s (%d likes) — %s\n", i+1, tp.AuthorUsername, tp.Likes, preview)
	}

	fmt.Print("\nSelect posts (comma-separated numbers, e.g., 1,3,5): ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)

	var selected []models.TrendingPost
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err == nil && idx >= 1 && idx <= len(trending) {
			selected = append(selected, trending[idx-1])
		}
	}

	return selected, nil
}

func displayImageSuggestion(reasoning string) {
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("14")).
		Padding(0, 1).
		Width(72)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))

	header := headerStyle.Render("Image Suggested")
	body := fmt.Sprintf("%s\n%s", header, reasoning)
	fmt.Println(cardStyle.Render(body))
}

func displayGeneratedContent(num int, r *models.GenerateResult) {
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
	mechanic := metaStyle.Render(fmt.Sprintf("Viral mechanic: %s", r.ViralMechanic))

	body := fmt.Sprintf("%s  %s\n%s\n\n%s", header, score, mechanic, r.Content)
	fmt.Println(cardStyle.Render(body))
	fmt.Println()
}
