package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	xplatform "github.com/shuhao/goviral/internal/platform/x"
)

var twikitLoginCmd = &cobra.Command{
	Use:   "twikit-login",
	Short: "Extract X cookies from Chrome for the twikit fallback",
	Long: `Extracts your X/Twitter session cookies from Chrome and saves them
to ~/.goviral/twikit_cookies.json. These cookies are used as a fallback
when the X API bearer token is unavailable or exhausted.

Prerequisites: you must be logged into X in Chrome.
You only need to run this once (or again if cookies expire).`,
	RunE: runTwikitLogin,
}

func init() {
	rootCmd.AddCommand(twikitLoginCmd)
}

func runTwikitLogin(cmd *cobra.Command, args []string) error {
	tc, err := xplatform.NewTwikitClient("")
	if err != nil {
		return fmt.Errorf("setting up twikit: %w", err)
	}

	fmt.Println("Extracting X cookies from Chrome...")
	if err := tc.ExtractCookies(context.Background()); err != nil {
		return fmt.Errorf("cookie extraction failed: %w", err)
	}

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render("\nCookies extracted successfully!"))
	fmt.Println("Saved to ~/.goviral/twikit_cookies.json")
	fmt.Println("The twikit fallback will now be used when the X API is unavailable.")
	return nil
}
