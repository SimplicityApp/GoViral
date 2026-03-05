package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize GoViral configuration and database",
	Long:  "Interactive setup wizard that prompts for API keys and creates the config file and database.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println("Welcome to GoViral! Let's set up your configuration.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	cfg := &config.Config{
		Claude: config.ClaudeConfig{
			Model: "claude-sonnet-4-20250514",
		},
		DBPath: config.DefaultDBPath(),
		Niches: []string{},
	}

	// X/Twitter credentials
	fmt.Println("\n--- X (Twitter) Configuration ---")
	fmt.Print("X Bearer Token: ")
	cfg.X.BearerToken = readLine(reader)
	fmt.Print("X Username (without @): ")
	cfg.X.Username = readLine(reader)
	fmt.Print("X API Key (optional): ")
	cfg.X.APIKey = readLine(reader)
	fmt.Print("X API Secret (optional): ")
	cfg.X.APISecret = readLine(reader)
	fmt.Print("X Access Token (optional): ")
	cfg.X.AccessToken = readLine(reader)
	fmt.Print("X Access Token Secret (optional): ")
	cfg.X.AccessTokenSecret = readLine(reader)

	// LinkedIn credentials
	fmt.Println("\n--- LinkedIn Configuration ---")
	fmt.Print("LinkedIn Client ID: ")
	cfg.LinkedIn.ClientID = readLine(reader)
	fmt.Print("LinkedIn Client Secret: ")
	cfg.LinkedIn.ClientSecret = readLine(reader)
	fmt.Print("LinkedIn Access Token: ")
	cfg.LinkedIn.AccessToken = readLine(reader)

	// Niches
	fmt.Println("\n--- Content Niches ---")
	fmt.Println("Enter your niches (comma-separated):")
	fmt.Print("> ")
	nichesInput := readLine(reader)
	for _, n := range strings.Split(nichesInput, ",") {
		n = strings.TrimSpace(n)
		if n != "" {
			cfg.Niches = append(cfg.Niches, n)
		}
	}

	// Save config
	if err := config.EnsureConfigDir(); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	configPath := config.DefaultConfigPath()
	if err := config.Save(cfg, configPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Printf("\nConfig saved to %s\n", configPath)

	// Create database
	database, err := db.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("creating database: %w", err)
	}
	database.Close()
	fmt.Printf("Database created at %s\n", cfg.DBPath)

	fmt.Println("\nGoViral is ready! Try running:")
	fmt.Println("  goviral fetch --platform x")
	fmt.Println("  goviral profile build")
	fmt.Println("  goviral trending --platform x")
	fmt.Println("  goviral generate --platform x")

	return nil
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}
