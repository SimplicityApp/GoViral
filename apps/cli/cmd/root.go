package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goviral",
	Short: "GoViral - Create viral content for X and LinkedIn",
	Long: `GoViral analyzes your existing posts to build a persona profile,
discovers trending posts in your niches, and uses AI to rewrite
trending content to match your voice and style.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
