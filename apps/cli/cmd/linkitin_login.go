package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	linkedinplatform "github.com/shuhao/goviral/internal/platform/linkedin"
)

var linkitinLoginCmd = &cobra.Command{
	Use:   "linkitin-login",
	Short: "Extract LinkedIn cookies from Chrome for the linkitin fallback",
	Long: `Extracts your LinkedIn session cookies from Chrome and saves them
to ~/.goviral/linkitin_cookies.json. These cookies are used as a fallback
when the LinkedIn official API is unavailable or unconfigured.

Prerequisites: you must be logged into LinkedIn in Chrome.
You only need to run this once (or again if cookies expire).`,
	RunE: runLinkitinLogin,
}

func init() {
	rootCmd.AddCommand(linkitinLoginCmd)
}

func runLinkitinLogin(cmd *cobra.Command, args []string) error {
	lc, err := linkedinplatform.NewLinkitinClient()
	if err != nil {
		return fmt.Errorf("setting up linkitin: %w", err)
	}

	fmt.Println("Extracting LinkedIn cookies from Chrome...")
	if err := lc.ExtractCookies(context.Background()); err != nil {
		return fmt.Errorf("cookie extraction failed: %w", err)
	}

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render("\nCookies extracted successfully!"))
	fmt.Println("Saved to ~/.goviral/linkitin_cookies.json")
	fmt.Println("The linkitin fallback will now be used when the LinkedIn API is unavailable.")
	return nil
}
