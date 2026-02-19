package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	linkedinplatform "github.com/shuhao/goviral/internal/platform/linkedin"
)

var likitLoginCmd = &cobra.Command{
	Use:   "likit-login",
	Short: "Extract LinkedIn cookies from Chrome for the likit fallback",
	Long: `Extracts your LinkedIn session cookies from Chrome and saves them
to ~/.goviral/likit_cookies.json. These cookies are used as a fallback
when the LinkedIn official API is unavailable or unconfigured.

Prerequisites: you must be logged into LinkedIn in Chrome.
You only need to run this once (or again if cookies expire).`,
	RunE: runLikitLogin,
}

func init() {
	rootCmd.AddCommand(likitLoginCmd)
}

func runLikitLogin(cmd *cobra.Command, args []string) error {
	lc, err := linkedinplatform.NewLikitClient()
	if err != nil {
		return fmt.Errorf("setting up likit: %w", err)
	}

	fmt.Println("Extracting LinkedIn cookies from Chrome...")
	if err := lc.ExtractCookies(context.Background()); err != nil {
		return fmt.Errorf("cookie extraction failed: %w", err)
	}

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render("\nCookies extracted successfully!"))
	fmt.Println("Saved to ~/.goviral/likit_cookies.json")
	fmt.Println("The likit fallback will now be used when the LinkedIn API is unavailable.")
	return nil
}
