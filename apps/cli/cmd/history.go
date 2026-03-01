package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
)

var (
	historyStatus string
	historyID     int64
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "View past generated content",
	RunE:  runHistory,
}

func init() {
	historyCmd.Flags().StringVar(&historyStatus, "status", "", "Filter by status (draft, approved, posted)")
	historyCmd.Flags().Int64Var(&historyID, "id", 0, "Show specific generation by ID")
	rootCmd.AddCommand(historyCmd)
}

func runHistory(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))

	// Show single item by ID
	if historyID > 0 {
		gc, err := database.GetGeneratedContentByID(historyID)
		if err != nil {
			return fmt.Errorf("fetching generated content: %w", err)
		}
		if gc == nil {
			return fmt.Errorf("generated content #%d not found", historyID)
		}

		fmt.Println(titleStyle.Render(fmt.Sprintf("Generated Content #%d", gc.ID)))
		displayHistoryItem(gc.ID, gc.TargetPlatform, gc.Status, gc.OriginalContent, gc.GeneratedContent, gc.CreatedAt.Format("2006-01-02 15:04:05"))
		return nil
	}

	// List items
	items, err := database.GetGeneratedContent(historyStatus, "", 50)
	if err != nil {
		return fmt.Errorf("fetching history: %w", err)
	}

	if len(items) == 0 {
		fmt.Println("No generated content found. Run 'goviral generate' first.")
		return nil
	}

	fmt.Println(titleStyle.Render(fmt.Sprintf("Generated Content History (%d items)", len(items))))
	fmt.Println()

	for _, gc := range items {
		displayHistoryItem(gc.ID, gc.TargetPlatform, gc.Status, gc.OriginalContent, gc.GeneratedContent, gc.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func displayHistoryItem(id int64, platform, status, original, generated, createdAt string) {
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Width(72)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	statusStyle := lipgloss.NewStyle().Bold(true)

	switch status {
	case "approved":
		statusStyle = statusStyle.Foreground(lipgloss.Color("10"))
	case "posted":
		statusStyle = statusStyle.Foreground(lipgloss.Color("13"))
	default:
		statusStyle = statusStyle.Foreground(lipgloss.Color("11"))
	}

	header := headerStyle.Render(fmt.Sprintf("#%d [%s]", id, platform))
	statusStr := statusStyle.Render(status)
	meta := metaStyle.Render(createdAt)

	origPreview := original
	if len(origPreview) > 100 {
		origPreview = origPreview[:97] + "..."
	}

	genPreview := generated
	if len(genPreview) > 280 {
		genPreview = genPreview[:277] + "..."
	}

	body := fmt.Sprintf("%s  %s  %s\n\nOriginal: %s\n\nGenerated:\n%s", header, statusStr, meta, metaStyle.Render(origPreview), genPreview)
	fmt.Println(cardStyle.Render(body))
	fmt.Println()
}
