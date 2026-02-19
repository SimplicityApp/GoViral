package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/internal/platform/linkedin"
	xplatform "github.com/shuhao/goviral/internal/platform/x"
	"github.com/shuhao/goviral/pkg/models"
)

var (
	fetchPlatform string
	fetchLimit    int
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch your posts from X and/or LinkedIn",
	RunE:  runFetch,
}

func init() {
	fetchCmd.Flags().StringVarP(&fetchPlatform, "platform", "p", "all", "Platform to fetch from (x, linkedin, all)")
	fetchCmd.Flags().IntVarP(&fetchLimit, "limit", "l", 50, "Number of posts to fetch")
	rootCmd.AddCommand(fetchCmd)
}

func runFetch(cmd *cobra.Command, args []string) error {
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

	if fetchPlatform == "x" || fetchPlatform == "all" {
		fmt.Println(titleStyle.Render("Fetching posts from X..."))
		if err := fetchFromX(ctx, cfg, database); err != nil {
			fmt.Printf("  Warning: X fetch failed: %v\n", err)
			if fetchPlatform == "x" {
				return err
			}
		}
	}

	if fetchPlatform == "linkedin" || fetchPlatform == "all" {
		fmt.Println(titleStyle.Render("Fetching posts from LinkedIn..."))
		if err := fetchFromLinkedIn(ctx, cfg, database); err != nil {
			fmt.Printf("  Warning: LinkedIn fetch failed: %v\n", err)
			if fetchPlatform == "linkedin" {
				return err
			}
		}
	}

	return nil
}

func fetchFromX(ctx context.Context, cfg *config.Config, database *db.DB) error {
	client := xplatform.NewFallbackClient(cfg.X)
	posts, err := client.FetchMyPosts(ctx, fetchLimit)
	if err != nil {
		return err
	}

	return storePosts(database, posts, "X")
}

func fetchFromLinkedIn(ctx context.Context, cfg *config.Config, database *db.DB) error {
	client := linkedin.NewFallbackClient(cfg.LinkedIn, nil)
	posts, err := client.FetchMyPosts(ctx, fetchLimit)
	if err != nil {
		return err
	}

	return storePosts(database, posts, "LinkedIn")
}

func storePosts(database *db.DB, posts []models.Post, platform string) error {
	for i := range posts {
		if err := database.UpsertPost(&posts[i]); err != nil {
			return fmt.Errorf("storing %s post: %w", platform, err)
		}
	}

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render(fmt.Sprintf("  Fetched and stored %d %s posts", len(posts), platform)))
	return nil
}
