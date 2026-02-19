package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"sort"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/internal/platform/linkedin"
	xplatform "github.com/shuhao/goviral/internal/platform/x"
	"github.com/shuhao/goviral/pkg/models"
)

var (
	trendingPlatform string
	trendingPeriod   string
	trendingMinLikes int
	trendingLimit    int
	trendingPerNiche int
)

var trendingCmd = &cobra.Command{
	Use:   "trending",
	Short: "Discover trending posts in your niches",
	RunE:  runTrending,
}

func init() {
	trendingCmd.Flags().StringVarP(&trendingPlatform, "platform", "p", "all", "Platform (x, linkedin, all)")
	trendingCmd.Flags().StringVar(&trendingPeriod, "period", "day", "Time period (day, week, month)")
	trendingCmd.Flags().IntVar(&trendingMinLikes, "min-likes", 100, "Minimum likes threshold")
	trendingCmd.Flags().IntVarP(&trendingLimit, "limit", "l", 0, "Max total results (0 = no limit)")
	trendingCmd.Flags().IntVar(&trendingPerNiche, "per-niche", 5, "Minimum posts per niche (0 = all niches in one query)")
	rootCmd.AddCommand(trendingCmd)
}

func runTrending(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	ctx := context.Background()
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	var allTrending []models.TrendingPost

	if trendingPlatform == "x" || trendingPlatform == "all" {
		fmt.Println(titleStyle.Render("Searching X for trending posts..."))
		client := xplatform.NewFallbackClient(cfg.X)
		posts, err := fetchTrending(ctx, client, cfg.Niches, trendingPeriod, trendingMinLikes, trendingLimit, trendingPerNiche)
		if err != nil {
			fmt.Printf("  Warning: X trending search failed: %v\n", err)
		} else {
			allTrending = append(allTrending, posts...)
		}
	}

	if trendingPlatform == "linkedin" || trendingPlatform == "all" {
		fmt.Println(titleStyle.Render("Searching LinkedIn for trending posts..."))
		client := linkedin.NewFallbackClient(cfg.LinkedIn, nil)
		posts, err := fetchTrending(ctx, client, cfg.Niches, trendingPeriod, trendingMinLikes, trendingLimit, trendingPerNiche)
		if err != nil {
			fmt.Printf("  Warning: LinkedIn trending search failed: %v\n", err)
		} else {
			allTrending = append(allTrending, posts...)
		}
	}

	// Store and display
	for i := range allTrending {
		if err := database.UpsertTrendingPost(&allTrending[i]); err != nil {
			return fmt.Errorf("storing trending post: %w", err)
		}
	}

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render(fmt.Sprintf("\nFound %d trending posts", len(allTrending))))
	fmt.Println()

	for i, tp := range allTrending {
		displayTrendingPost(i+1, &tp)
	}

	return nil
}

// fetchTrending fetches trending posts. When perNiche > 0, it searches each
// niche separately to guarantee at least perNiche results per niche, then
// sorts all results by engagement. Otherwise it searches all niches at once.
func fetchTrending(ctx context.Context, client models.PlatformClient, niches []string, period string, minLikes int, limit int, perNiche int) ([]models.TrendingPost, error) {
	if perNiche <= 0 {
		return client.FetchTrendingPosts(ctx, niches, period, minLikes, limit)
	}

	nicheStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	seen := make(map[string]bool)
	var all []models.TrendingPost

	for _, niche := range niches {
		fmt.Printf("  %s\n", nicheStyle.Render(fmt.Sprintf("Searching: %s", niche)))
		posts, err := client.FetchTrendingPosts(ctx, []string{niche}, period, minLikes, perNiche)
		if err != nil {
			fmt.Printf("  Warning: search for %q failed: %v\n", niche, err)
			continue
		}
		for i := range posts {
			if !seen[posts[i].PlatformPostID] {
				seen[posts[i].PlatformPostID] = true
				all = append(all, posts[i])
			}
		}
	}

	sort.Slice(all, func(i, j int) bool {
		ei := all[i].Likes + all[i].Reposts + all[i].Comments
		ej := all[j].Likes + all[j].Reposts + all[j].Comments
		return ei > ej
	})

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}

	return all, nil
}

func displayTrendingPost(num int, tp *models.TrendingPost) {
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Width(72)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	metricsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	header := headerStyle.Render(fmt.Sprintf("#%d %s (@%s) [%s]", num, tp.AuthorName, tp.AuthorUsername, tp.Platform))
	metrics := metricsStyle.Render(fmt.Sprintf("Likes: %d  Reposts: %d  Comments: %d", tp.Likes, tp.Reposts, tp.Comments))

	content := tp.Content
	if len(content) > 280 {
		content = content[:277] + "..."
	}

	nicheTags := ""
	if len(tp.NicheTags) > 0 {
		nicheTags = metaStyle.Render(fmt.Sprintf("Niches: %v", tp.NicheTags))
	}

	mediaInfo := ""
	if len(tp.Media) > 0 {
		counts := make(map[string]int)
		for _, m := range tp.Media {
			counts[m.Type]++
		}
		var parts []string
		if n := counts["photo"]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d photo(s)", n))
		}
		if n := counts["video"]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d video(s)", n))
		}
		if n := counts["animated_gif"]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d GIF(s)", n))
		}
		if len(parts) > 0 {
			mediaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
			mediaInfo = mediaStyle.Render(fmt.Sprintf("Media: %s", strings.Join(parts, ", ")))
		}
	}

	body := fmt.Sprintf("%s\n%s\n\n%s\n%s", header, metrics, content, nicheTags)
	if mediaInfo != "" {
		body += "\n" + mediaInfo
	}
	fmt.Println(cardStyle.Render(body))
	fmt.Println()
}
