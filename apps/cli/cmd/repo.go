package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/internal/ai/generator"
	"github.com/shuhao/goviral/internal/codeimg"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	ghclient "github.com/shuhao/goviral/internal/platform/github"
	"github.com/shuhao/goviral/pkg/models"
)

var (
	repoFetchLimit    int
	repoFetchRepo     string
	repoGenRepo       string
	repoGenPlatform   string
	repoGenCount      int
	repoGenImages     bool
	repoGenStyle      string
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage GitHub repos and generate posts from commits",
	Long: `Connect GitHub repositories, fetch recent commits, and generate
viral "build in public" posts from your code changes.

Examples:
  goviral repo add shuhao/goviral
  goviral repo list
  goviral repo fetch --repo shuhao/goviral --limit 10
  goviral repo generate --repo shuhao/goviral --platform x --count 3`,
}

var repoAddCmd = &cobra.Command{
	Use:   "add <owner/repo>",
	Short: "Add a GitHub repository to track",
	Args:  cobra.ExactArgs(1),
	RunE:  runRepoAdd,
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tracked GitHub repositories",
	RunE:  runRepoList,
}

var repoRemoveCmd = &cobra.Command{
	Use:   "remove <owner/repo>",
	Short: "Remove a tracked repository",
	Args:  cobra.ExactArgs(1),
	RunE:  runRepoRemove,
}

var repoFetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch recent commits from a tracked repo",
	RunE:  runRepoFetch,
}

var repoGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate viral posts from fetched commits",
	RunE:  runRepoGenerate,
}

func init() {
	repoFetchCmd.Flags().StringVar(&repoFetchRepo, "repo", "", "Repository (owner/repo)")
	repoFetchCmd.Flags().IntVar(&repoFetchLimit, "limit", 10, "Number of commits to fetch")

	repoGenerateCmd.Flags().StringVar(&repoGenRepo, "repo", "", "Repository (owner/repo)")
	repoGenerateCmd.Flags().StringVarP(&repoGenPlatform, "platform", "p", "x", "Target platform (x, linkedin)")
	repoGenerateCmd.Flags().IntVarP(&repoGenCount, "count", "c", 3, "Number of variations per commit")
	repoGenerateCmd.Flags().BoolVar(&repoGenImages, "images", false, "Generate code diff images")
	repoGenerateCmd.Flags().StringVar(&repoGenStyle, "style", "", "Style direction for generation")

	repoCmd.AddCommand(repoAddCmd)
	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoRemoveCmd)
	repoCmd.AddCommand(repoFetchCmd)
	repoCmd.AddCommand(repoGenerateCmd)
	rootCmd.AddCommand(repoCmd)
}

func runRepoAdd(cmd *cobra.Command, args []string) error {
	parts := strings.SplitN(args[0], "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid format: use owner/repo (e.g. shuhao/goviral)")
	}
	owner, name := parts[0], parts[1]

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.GitHub.PersonalAccessToken == "" {
		return fmt.Errorf("GitHub personal access token not configured; add it to ~/.goviral/config.yaml under github.personal_access_token")
	}

	database, err := db.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	fmt.Println(titleStyle.Render(fmt.Sprintf("Adding %s/%s...", owner, name)))

	client := ghclient.NewClient(cfg.GitHub.PersonalAccessToken)
	repo, err := client.GetRepo(context.Background(), owner, name)
	if err != nil {
		return fmt.Errorf("fetching repo from GitHub: %w", err)
	}

	if err := database.UpsertGitHubRepo("", repo); err != nil {
		return fmt.Errorf("saving repo: %w", err)
	}

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render(fmt.Sprintf("Added %s", repo.FullName)))
	if repo.Description != "" {
		fmt.Printf("  %s\n", repo.Description)
	}
	if repo.Language != "" {
		fmt.Printf("  Language: %s  Branch: %s\n", repo.Language, repo.DefaultBranch)
	}

	return nil
}

func runRepoList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	repos, err := database.ListGitHubRepos("")
	if err != nil {
		return fmt.Errorf("listing repos: %w", err)
	}

	if len(repos) == 0 {
		fmt.Println("No tracked repos. Use 'goviral repo add owner/repo' to add one.")
		return nil
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	fmt.Println(titleStyle.Render(fmt.Sprintf("Tracked repos (%d):", len(repos))))

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(0, 1).
		Width(60)

	for _, r := range repos {
		lang := ""
		if r.Language != "" {
			lang = fmt.Sprintf("  [%s]", r.Language)
		}
		header := lipgloss.NewStyle().Bold(true).Render(r.FullName) + lang
		desc := ""
		if r.Description != "" {
			desc = "\n" + r.Description
		}
		fmt.Println(cardStyle.Render(header + desc))
	}

	return nil
}

func runRepoRemove(cmd *cobra.Command, args []string) error {
	fullName := args[0]

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	database, err := db.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	repo, err := database.GetGitHubRepoByFullName("", fullName)
	if err != nil {
		return fmt.Errorf("looking up repo: %w", err)
	}
	if repo == nil {
		return fmt.Errorf("repo %s not found", fullName)
	}

	if err := database.DeleteGitHubRepo("", repo.ID); err != nil {
		return fmt.Errorf("deleting repo: %w", err)
	}

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render(fmt.Sprintf("Removed %s", fullName)))
	return nil
}

func runRepoFetch(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.GitHub.PersonalAccessToken == "" {
		return fmt.Errorf("GitHub personal access token not configured")
	}

	database, err := db.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	repoName := repoFetchRepo
	if repoName == "" {
		repoName = cfg.GitHub.DefaultOwner + "/" + cfg.GitHub.DefaultRepo
	}
	if repoName == "/" || repoName == "" {
		return fmt.Errorf("--repo flag required or set default_owner/default_repo in config")
	}

	repo, err := database.GetGitHubRepoByFullName("", repoName)
	if err != nil {
		return fmt.Errorf("looking up repo: %w", err)
	}
	if repo == nil {
		return fmt.Errorf("repo %s not tracked; use 'goviral repo add %s' first", repoName, repoName)
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	fmt.Println(titleStyle.Render(fmt.Sprintf("Fetching %d commits from %s...", repoFetchLimit, repoName)))

	client := ghclient.NewClient(cfg.GitHub.PersonalAccessToken)
	commits, err := client.ListCommits(context.Background(), repo.Owner, repo.Name, models.CommitListOptions{
		Limit: repoFetchLimit,
	})
	if err != nil {
		return fmt.Errorf("fetching commits: %w", err)
	}

	// Fetch full details for each commit
	saved := 0
	for i, c := range commits {
		fmt.Printf("  [%d/%d] %s %.50s\n", i+1, len(commits), c.SHA[:7], firstLine(c.Message))

		full, err := client.GetCommit(context.Background(), repo.Owner, repo.Name, c.SHA)
		if err != nil {
			fmt.Printf("    Warning: could not fetch full details: %v\n", err)
			full = &c
		}

		filesJSON, _ := json.Marshal(full.Files)
		rc := &models.RepoCommitRecord{
			RepoID:       repo.ID,
			SHA:          full.SHA,
			Message:      full.Message,
			AuthorName:   full.AuthorName,
			AuthorEmail:  full.AuthorEmail,
			CommittedAt:  full.CommittedAt,
			Additions:    full.Additions,
			Deletions:    full.Deletions,
			FilesChanged: full.FilesChanged,
			DiffSummary:  ghclient.SummarizeDiff(*full),
			DiffPatch:    full.DiffPatch,
			FilesJSON:    string(filesJSON),
		}
		if err := database.UpsertRepoCommit(rc); err != nil {
			fmt.Printf("    Warning: save failed: %v\n", err)
			continue
		}
		saved++
	}

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render(fmt.Sprintf("\nFetched and saved %d commits", saved)))
	return nil
}

func runRepoGenerate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Claude.APIKey == "" {
		return fmt.Errorf("Claude API key not configured")
	}

	database, err := db.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	repoName := repoGenRepo
	if repoName == "" {
		repoName = cfg.GitHub.DefaultOwner + "/" + cfg.GitHub.DefaultRepo
	}
	if repoName == "/" || repoName == "" {
		return fmt.Errorf("--repo flag required or set default_owner/default_repo in config")
	}

	repo, err := database.GetGitHubRepoByFullName("", repoName)
	if err != nil {
		return fmt.Errorf("looking up repo: %w", err)
	}
	if repo == nil {
		return fmt.Errorf("repo %s not tracked", repoName)
	}

	// Get persona for target platform
	persona, err := database.GetPersona("", repoGenPlatform)
	if err != nil {
		return fmt.Errorf("getting persona: %w", err)
	}
	if persona == nil {
		return fmt.Errorf("no persona found for %s; run 'goviral profile build' first", repoGenPlatform)
	}

	// Get latest commits
	commits, err := database.ListRepoCommits(repo.ID, 5)
	if err != nil {
		return fmt.Errorf("listing commits: %w", err)
	}
	if len(commits) == 0 {
		return fmt.Errorf("no commits fetched; run 'goviral repo fetch' first")
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	fmt.Println(titleStyle.Render(fmt.Sprintf("Generating %s posts from %d commits...\n", repoGenPlatform, len(commits))))

	claudeClient := claude.NewClient(cfg.Claude.APIKey, cfg.Claude.Model)
	gen := generator.NewGenerator(claudeClient)

	var renderer *codeimg.Renderer
	if repoGenImages {
		renderer, err = codeimg.NewRenderer()
		if err != nil {
			fmt.Printf("Warning: could not start code image renderer: %v\n", err)
		} else {
			defer renderer.Close()
		}
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("10")).
		Padding(0, 1).
		Width(72)

	totalGenerated := 0
	for _, rc := range commits {
		fmt.Printf("Commit %s: %s\n", rc.SHA[:7], firstLine(rc.Message))

		var files []models.GitHubFileChange
		if rc.FilesJSON != "" {
			_ = json.Unmarshal([]byte(rc.FilesJSON), &files)
		}

		commit := models.GitHubCommit{
			SHA:          rc.SHA,
			Message:      rc.Message,
			AuthorName:   rc.AuthorName,
			AuthorEmail:  rc.AuthorEmail,
			CommittedAt:  rc.CommittedAt,
			Additions:    rc.Additions,
			Deletions:    rc.Deletions,
			FilesChanged: rc.FilesChanged,
			DiffPatch:    rc.DiffPatch,
			Files:        files,
		}

		results, err := gen.GenerateRepoPost(context.Background(), models.RepoPostRequest{
			Commit:         commit,
			Repo:           *repo,
			Persona:        *persona,
			TargetPlatform: repoGenPlatform,
			Count:          repoGenCount,
			StyleDirection: repoGenStyle,
		})
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}

		// Optionally render code image
		var codeImagePath string
		if renderer != nil && len(files) > 0 {
			bestFile := ghclient.SelectBestFile(files)
			if bestFile != nil && bestFile.Patch != "" {
				pngBytes, err := renderer.RenderDiff(bestFile.Patch, bestFile.Filename, models.RenderOptions{})
				if err != nil {
					fmt.Printf("  Warning: code image render failed: %v\n", err)
				} else {
					imgPath, err := saveCodeImage(cfg, rc.SHA[:7], pngBytes)
					if err != nil {
						fmt.Printf("  Warning: saving code image failed: %v\n", err)
					} else {
						codeImagePath = imgPath
					}
				}
			}
		}

		for i, r := range results {
			gc := models.GeneratedContent{
				TargetPlatform:   repoGenPlatform,
				OriginalContent:  rc.Message,
				GeneratedContent: r.Content,
				PersonaID:        persona.ID,
				PromptUsed:       fmt.Sprintf("repo-%s", repoGenPlatform),
				Status:           "draft",
				SourceType:       "commit",
				SourceCommitID:   rc.ID,
				CodeImagePath:    codeImagePath,
			}

			contentID, err := database.InsertGeneratedContent("", &gc)
			if err != nil {
				fmt.Printf("  Warning: save failed: %v\n", err)
				continue
			}

			header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render(fmt.Sprintf("Variation #%d", i+1))
			score := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11")).Render(fmt.Sprintf("Confidence: %d/10", r.ConfidenceScore))
			meta := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(fmt.Sprintf("Mechanic: %s  |  ID: %d", r.ViralMechanic, contentID))

			body := fmt.Sprintf("%s  %s\n%s\n\n%s", header, score, meta, r.Content)
			fmt.Println(cardStyle.Render(body))
			fmt.Println()
			totalGenerated++
		}
	}

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render(fmt.Sprintf("Generated %d posts from %d commits", totalGenerated, len(commits))))
	return nil
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

func saveCodeImage(cfg *config.Config, shortSHA string, pngBytes []byte) (string, error) {
	imagesDir := config.DefaultConfigDir() + "/images"
	if err := ensureDir(imagesDir); err != nil {
		return "", fmt.Errorf("creating images directory: %w", err)
	}

	filename := fmt.Sprintf("diff-%s.png", shortSHA)
	path := imagesDir + "/" + filename

	if err := writeFile(path, pngBytes); err != nil {
		return "", fmt.Errorf("writing code image: %w", err)
	}

	return path, nil
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}
