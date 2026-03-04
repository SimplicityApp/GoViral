package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/pkg/models"
)

var (
	postsPlatform string
	postsLimit    int
)

var postsCmd = &cobra.Command{
	Use:   "posts",
	Short: "View your fetched posts",
	RunE:  runPosts,
}

func init() {
	postsCmd.Flags().StringVarP(&postsPlatform, "platform", "p", "all", "Platform to show (x, linkedin, all)")
	postsCmd.Flags().IntVarP(&postsLimit, "limit", "l", 20, "Number of posts to show")
	rootCmd.AddCommand(postsCmd)
}

func runPosts(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	var posts []models.Post
	if postsPlatform == "all" {
		posts, err = database.GetAllPosts("")
	} else {
		posts, err = database.GetPostsByPlatform("", postsPlatform)
	}
	if err != nil {
		return fmt.Errorf("fetching posts: %w", err)
	}

	if len(posts) == 0 {
		fmt.Println("No posts found. Run 'goviral fetch' first.")
		return nil
	}

	if postsLimit > 0 && len(posts) > postsLimit {
		posts = posts[:postsLimit]
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	fmt.Println(titleStyle.Render(fmt.Sprintf("Your Posts (%d shown)", len(posts))))
	fmt.Println()

	for i, p := range posts {
		displayPost(i+1, &p)
	}

	return nil
}

func displayPost(num int, p *models.Post) {
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Width(72)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	metricsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	platform := strings.ToUpper(p.Platform)
	header := headerStyle.Render(fmt.Sprintf("#%d [%s]", num, platform))
	date := metaStyle.Render(p.PostedAt.Format("2006-01-02 15:04"))

	metrics := metricsStyle.Render(fmt.Sprintf(
		"Likes: %d  Reposts: %d  Comments: %d  Views: %d",
		p.Likes, p.Reposts, p.Comments, p.Impressions,
	))

	content := p.Content
	if len(content) > 280 {
		content = content[:277] + "..."
	}

	body := fmt.Sprintf("%s  %s\n%s\n\n%s", header, date, metrics, content)
	fmt.Println(cardStyle.Render(body))
	fmt.Println()
}
