package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/internal/ai/generator"
	"github.com/shuhao/goviral/internal/codeimg"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	ghclient "github.com/shuhao/goviral/internal/platform/github"
	"github.com/shuhao/goviral/pkg/models"
)

// RepoService handles GitHub repo operations.
type RepoService struct {
	db  *db.DB
	cfg *config.Config

	// In-memory cache for available repos, keyed by token
	cachedRepos      map[string][]models.GitHubRepo
	cachedReposAt    map[string]time.Time
}

// NewRepoService creates a new RepoService.
func NewRepoService(database *db.DB, cfg *config.Config) *RepoService {
	return &RepoService{
		db:            database,
		cfg:           cfg,
		cachedRepos:   make(map[string][]models.GitHubRepo),
		cachedReposAt: make(map[string]time.Time),
	}
}

// githubToken returns the provided user token if non-empty, otherwise falls back to the global PAT.
func (s *RepoService) githubToken(userToken string) string {
	if userToken != "" {
		return userToken
	}
	return s.cfg.GitHub.PersonalAccessToken
}

// ListAvailableRepos returns all GitHub repos accessible to the authenticated
// user (personal + org). Results are cached in memory for 5 minutes per token.
func (s *RepoService) ListAvailableRepos(ctx context.Context, userToken string) ([]models.GitHubRepo, error) {
	token := s.githubToken(userToken)
	if token == "" {
		return nil, fmt.Errorf("GitHub token not configured — connect via OAuth or set a PAT")
	}

	if cached, ok := s.cachedRepos[token]; ok && time.Since(s.cachedReposAt[token]) < 5*time.Minute {
		return cached, nil
	}

	client := ghclient.NewClient(token)
	repos, err := client.ListUserRepos(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing user repos: %w", err)
	}

	s.cachedRepos[token] = repos
	s.cachedReposAt[token] = time.Now()

	return repos, nil
}

// AddRepo validates a GitHub repo via the API and upserts it in the DB.
func (s *RepoService) AddRepo(ctx context.Context, userID string, owner, name, userToken string) (*models.GitHubRepo, error) {
	client := ghclient.NewClient(s.githubToken(userToken))

	repo, err := client.GetRepo(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("fetching repo %s/%s from GitHub: %w", owner, name, err)
	}

	if err := s.db.UpsertGitHubRepo(userID, repo); err != nil {
		return nil, fmt.Errorf("saving repo %s/%s: %w", owner, name, err)
	}

	return repo, nil
}

// ListRepos returns all tracked repositories.
func (s *RepoService) ListRepos(ctx context.Context, userID string) ([]models.GitHubRepo, error) {
	repos, err := s.db.ListGitHubRepos(userID)
	if err != nil {
		return nil, fmt.Errorf("listing repos: %w", err)
	}
	return repos, nil
}

// DeleteRepo removes a tracked repository by ID.
func (s *RepoService) DeleteRepo(ctx context.Context, userID string, id int64) error {
	if err := s.db.DeleteGitHubRepo(userID, id); err != nil {
		return fmt.Errorf("deleting repo %d: %w", id, err)
	}
	return nil
}

// GetRepo returns a single tracked repository by ID.
func (s *RepoService) GetRepo(ctx context.Context, userID string, id int64) (*models.GitHubRepo, error) {
	repo, err := s.db.GetGitHubRepo(userID, id)
	if err != nil {
		return nil, fmt.Errorf("getting repo %d: %w", id, err)
	}
	return repo, nil
}

// UpdateRepoSettings updates the target audience and links for a repo.
func (s *RepoService) UpdateRepoSettings(ctx context.Context, userID string, id int64, targetAudience string, links []models.RepoLink) (*models.GitHubRepo, error) {
	if err := s.db.UpdateRepoSettings(userID, id, targetAudience, links); err != nil {
		return nil, fmt.Errorf("updating repo %d settings: %w", id, err)
	}
	return s.db.GetGitHubRepo(userID, id)
}

// GetContentByID returns a generated content record by ID.
func (s *RepoService) GetContentByID(ctx context.Context, userID string, id int64) (*models.GeneratedContent, error) {
	return s.db.GetGeneratedContentByID(userID, id)
}

// FetchCommits fetches commits from GitHub for the given repo, upserts them in
// the DB, and sends progress events. The progress channel is always closed when
// this function returns.
//
// If sinceStr is empty, the method auto-detects the latest stored commit date
// and uses it as the "since" filter (incremental fetch). For repos with no
// stored commits yet, all commits are fetched. When limit > 0, the total number
// of fetched commits is capped; otherwise all available commits are returned.
func (s *RepoService) FetchCommits(ctx context.Context, userID string, repoID int64, limit int, sinceStr string, userToken string, progress chan<- dto.ProgressEvent) ([]models.RepoCommitRecord, error) {
	defer close(progress)

	repo, err := s.db.GetGitHubRepo(userID, repoID)
	if err != nil {
		return nil, fmt.Errorf("getting repo %d: %w", repoID, err)
	}
	if repo == nil {
		return nil, fmt.Errorf("repo %d not found", repoID)
	}

	progress <- dto.ProgressEvent{
		Type:       "progress",
		Message:    fmt.Sprintf("Fetching commits for %s/%s...", repo.Owner, repo.Name),
		Percentage: 5,
	}

	// Build list options. Only use since when the caller explicitly provides it;
	// otherwise fetch the full commit list and skip detail-fetching for commits
	// already in the DB. This avoids the problem where an incomplete initial
	// import permanently misses older commits via auto-since detection.
	opts := models.CommitListOptions{
		Limit: limit,
	}

	if sinceStr != "" {
		t, err := time.Parse(time.RFC3339, sinceStr)
		if err == nil {
			opts.Since = &t
		}
	}

	client := ghclient.NewClient(s.githubToken(userToken))

	commits, err := client.ListAllCommits(ctx, repo.Owner, repo.Name, opts, func(page int, count int) {
		pct := 5 + min(25, page*5) // 5% → 30% during pagination
		progress <- dto.ProgressEvent{
			Type:       "progress",
			Message:    fmt.Sprintf("Fetching commits page %d... (%d commits so far)", page, count),
			Percentage: pct,
		}
	})
	if err != nil {
		return nil, fmt.Errorf("listing commits for %s/%s: %w", repo.Owner, repo.Name, err)
	}

	if len(commits) == 0 {
		progress <- dto.ProgressEvent{
			Type:       "complete",
			Message:    fmt.Sprintf("0 new commits for %s/%s", repo.Owner, repo.Name),
			Percentage: 100,
			Data:       []models.RepoCommitRecord{},
		}
		return nil, nil
	}

	progress <- dto.ProgressEvent{
		Type:       "progress",
		Message:    fmt.Sprintf("Found %d commits, fetching details...", len(commits)),
		Percentage: 30,
	}

	var records []models.RepoCommitRecord
	var newCount int
	total := len(commits)

	for i, c := range commits {
		select {
		case <-ctx.Done():
			return records, ctx.Err()
		default:
		}

		pct := 30 + ((i+1)*65)/total

		// Skip detail fetch for commits already stored in DB
		existing, _ := s.db.GetRepoCommit(repoID, c.SHA)
		if existing != nil {
			records = append(records, *existing)
			progress <- dto.ProgressEvent{
				Type:       "progress",
				Message:    fmt.Sprintf("Commit %d/%d already stored: %.7s", i+1, total, c.SHA),
				Percentage: pct,
			}
			continue
		}

		newCount++
		progress <- dto.ProgressEvent{
			Type:       "progress",
			Message:    fmt.Sprintf("Fetching commit %d/%d: %.7s", i+1, total, c.SHA),
			Percentage: pct,
		}

		// Fetch full commit detail including diff
		detail, err := client.GetCommit(ctx, repo.Owner, repo.Name, c.SHA)
		if err != nil {
			slog.Error("fetching commit detail", "sha", c.SHA, "error", err)
			// Use the summary-only commit on failure
			detail = &c
		}

		filesJSON, err := json.Marshal(detail.Files)
		if err != nil {
			filesJSON = []byte("[]")
		}

		rc := models.RepoCommitRecord{
			RepoID:       repoID,
			SHA:          detail.SHA,
			Message:      detail.Message,
			AuthorName:   detail.AuthorName,
			AuthorEmail:  detail.AuthorEmail,
			CommittedAt:  detail.CommittedAt,
			Additions:    detail.Additions,
			Deletions:    detail.Deletions,
			FilesChanged: detail.FilesChanged,
			DiffSummary:  ghclient.SummarizeDiff(*detail),
			DiffPatch:    detail.DiffPatch,
			FilesJSON:    string(filesJSON),
		}

		if err := s.db.UpsertRepoCommit(&rc); err != nil {
			slog.Error("upserting commit", "sha", c.SHA, "error", err)
			continue
		}

		records = append(records, rc)
	}

	progress <- dto.ProgressEvent{
		Type:       "complete",
		Message:    fmt.Sprintf("Fetched %d new commits (%d total) for %s/%s", newCount, len(records), repo.Owner, repo.Name),
		Percentage: 100,
		Data:       records,
	}

	return records, nil
}

// ListCommits returns stored commits for a repo, after validating user ownership.
func (s *RepoService) ListCommits(ctx context.Context, userID string, repoID int64, limit int) ([]models.RepoCommitRecord, error) {
	repo, err := s.db.GetGitHubRepo(userID, repoID)
	if err != nil {
		return nil, fmt.Errorf("getting repo %d: %w", repoID, err)
	}
	if repo == nil {
		return nil, fmt.Errorf("repo %d not found", repoID)
	}

	commits, err := s.db.ListRepoCommits(repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("listing commits for repo %d: %w", repoID, err)
	}
	return commits, nil
}

// GenerateFromCommits loads commits, loads persona, calls the generator, optionally
// renders a code image, and stores the results as GeneratedContent records with
// source_type='commit'. The progress channel is always closed when this function returns.
func (s *RepoService) GenerateFromCommits(ctx context.Context, userID string, req dto.GenerateRepoPostRequest, progress chan<- dto.ProgressEvent) ([]models.GeneratedContent, error) {
	defer close(progress)

	if s.cfg.Claude.APIKey == "" {
		return nil, fmt.Errorf("Claude API key not configured")
	}

	// Normalize fields — frontend may send "platform" or "target_platform",
	// and "include_code_images" (plural) or "include_code_image" (singular).
	targetPlatform := req.TargetPlatform
	if targetPlatform == "" {
		targetPlatform = req.Platform
	}
	if targetPlatform == "" {
		targetPlatform = "x"
	}
	includeCodeImage := req.IncludeCodeImage || req.IncludeCodeImages
	codeImageTemplate := req.CodeImageTemplate
	codeImageTheme := req.CodeImageTheme
	count := req.Count
	if count <= 0 {
		count = 3
	}

	progress <- dto.ProgressEvent{
		Type:       "progress",
		Message:    "Loading persona...",
		Percentage: 5,
	}

	persona, err := s.db.GetPersona(userID, targetPlatform)
	if err != nil {
		return nil, fmt.Errorf("getting persona for %s: %w", targetPlatform, err)
	}
	if persona == nil {
		return nil, fmt.Errorf("no persona found for platform %s; build persona first", targetPlatform)
	}

	client := claude.NewClient(s.cfg.Claude.APIKey, s.cfg.Claude.Model)
	gen := generator.NewGenerator(client)

	total := len(req.CommitIDs)
	var allContent []models.GeneratedContent

	for i, commitID := range req.CommitIDs {
		select {
		case <-ctx.Done():
			return allContent, ctx.Err()
		default:
		}

		pct := 10 + ((i+1)*80)/total
		progress <- dto.ProgressEvent{
			Type:       "progress",
			Message:    fmt.Sprintf("Generating posts for commit %d/%d (ID: %d)...", i+1, total, commitID),
			Percentage: pct,
		}

		rc, err := s.db.GetRepoCommitByIDForUser(userID, commitID)
		if err != nil {
			return nil, fmt.Errorf("getting commit %d: %w", commitID, err)
		}
		if rc == nil {
			return nil, fmt.Errorf("commit %d not found", commitID)
		}

		repo, err := s.db.GetGitHubRepo(userID, rc.RepoID)
		if err != nil {
			return nil, fmt.Errorf("getting repo %d for commit %d: %w", rc.RepoID, commitID, err)
		}
		if repo == nil {
			return nil, fmt.Errorf("repo %d not found", rc.RepoID)
		}

		// Reconstruct a GitHubCommit from the stored record
		var files []models.GitHubFileChange
		if rc.FilesJSON != "" && rc.FilesJSON != "[]" {
			if err := json.Unmarshal([]byte(rc.FilesJSON), &files); err != nil {
				slog.Error("parsing commit files JSON", "commit_id", commitID, "error", err)
			}
		}

		ghCommit := models.GitHubCommit{
			SHA:          rc.SHA,
			Message:      rc.Message,
			AuthorName:   rc.AuthorName,
			CommittedAt:  rc.CommittedAt,
			Additions:    rc.Additions,
			Deletions:    rc.Deletions,
			FilesChanged: rc.FilesChanged,
			DiffPatch:    rc.DiffPatch,
			Files:        files,
		}

		maxChars := 0
		if targetPlatform == "linkedin" {
			maxChars = 2000
		}

		results, err := gen.GenerateRepoPost(ctx, models.RepoPostRequest{
			Commit:           ghCommit,
			Repo:             *repo,
			Persona:          *persona,
			TargetPlatform:   targetPlatform,
			Count:            count,
			MaxChars:         maxChars,
			IncludeCodeImage: includeCodeImage,
			StyleDirection:   req.StyleDirection,
			TargetAudience:   repo.TargetAudience,
			Links:            repo.Links,
		})
		if err != nil {
			return nil, fmt.Errorf("generating post for commit %d: %w", commitID, err)
		}

		// Ensure links are present — the AI sometimes omits them despite the prompt.
		if len(repo.Links) > 0 {
			for ri := range results {
				missing := false
				for _, link := range repo.Links {
					if !strings.Contains(results[ri].Content, link.URL) {
						missing = true
						break
					}
				}
				if missing {
					var linkBlock strings.Builder
					linkBlock.WriteString("\n\n")
					for _, link := range repo.Links {
						if !strings.Contains(results[ri].Content, link.URL) {
							fmt.Fprintf(&linkBlock, "%s: %s\n", link.Label, link.URL)
						}
					}
					results[ri].Content = strings.TrimRight(results[ri].Content, "\n") + linkBlock.String()
				}
			}
		}

		for ri, r := range results {
			var codeImagePath string
			if includeCodeImage {
				progress <- dto.ProgressEvent{
					Type:    "progress",
					Message: fmt.Sprintf("Rendering code image for commit %d variation %d...", commitID, ri+1),
				}

				if r.CodeSnippet != nil {
					slog.Info("AI selected code snippet", "commit_id", commitID, "variation", ri+1,
						"file", r.CodeSnippet.Filename, "start", r.CodeSnippet.StartLine, "end", r.CodeSnippet.EndLine)
					_, savedPath, err := s.RenderCodeImageFromSnippet(ctx, *r.CodeSnippet, files, commitID, rc.SHA, codeImageTemplate, codeImageTheme, r.CodeSnippet.Description)
					if err != nil {
						slog.Error("rendering AI-selected code image", "commit_id", commitID, "error", err)
						// Fallback to heuristic
						_, savedPath, err = s.RenderCodeImage(ctx, userID, commitID, codeImageTemplate, codeImageTheme)
						if err != nil {
							slog.Error("rendering fallback code image", "commit_id", commitID, "error", err)
						} else {
							codeImagePath = savedPath
						}
					} else {
						codeImagePath = savedPath
					}
				} else {
					// No AI snippet — use heuristic fallback
					slog.Warn("no AI code snippet returned, using heuristic", "commit_id", commitID, "variation", ri+1)
					_, savedPath, err := s.RenderCodeImage(ctx, userID, commitID, codeImageTemplate, codeImageTheme)
					if err != nil {
						slog.Error("rendering code image", "commit_id", commitID, "error", err)
					} else {
						codeImagePath = savedPath
					}
				}
			}

			var codeImageDescription string
			if r.CodeSnippet != nil {
				codeImageDescription = r.CodeSnippet.Description
			}

			gc := models.GeneratedContent{
				TargetPlatform:       targetPlatform,
				OriginalContent:      rc.Message,
				GeneratedContent:     r.Content,
				PersonaID:            persona.ID,
				PromptUsed:           fmt.Sprintf("repo-%s", targetPlatform),
				Status:               "draft",
				SourceType:           "commit",
				SourceCommitID:       rc.ID,
				CodeImagePath:        codeImagePath,
				CodeImageDescription: codeImageDescription,
			}

			id, err := s.db.InsertGeneratedContent(userID, &gc)
			if err != nil {
				return nil, fmt.Errorf("saving generated content for commit %d: %w", commitID, err)
			}
			gc.ID = id
			allContent = append(allContent, gc)
		}
	}

	progress <- dto.ProgressEvent{
		Type:       "complete",
		Message:    fmt.Sprintf("Generated %d content items from %d commits", len(allContent), total),
		Percentage: 100,
		Data:       allContent,
	}

	return allContent, nil
}

// RenderCodeImage renders a diff image for the given commit, saves it to
// ~/.goviral/images/, and returns the PNG bytes, the saved path, and any error.
// templateName and themeName are optional; empty strings use defaults.
func (s *RepoService) RenderCodeImage(ctx context.Context, userID string, commitID int64, templateName, themeName string) ([]byte, string, error) {
	rc, err := s.db.GetRepoCommitByIDForUser(userID, commitID)
	if err != nil {
		return nil, "", fmt.Errorf("getting commit %d: %w", commitID, err)
	}
	if rc == nil {
		return nil, "", fmt.Errorf("commit %d not found", commitID)
	}

	var files []models.GitHubFileChange
	if rc.FilesJSON != "" && rc.FilesJSON != "[]" {
		if err := json.Unmarshal([]byte(rc.FilesJSON), &files); err != nil {
			return nil, "", fmt.Errorf("parsing commit files JSON: %w", err)
		}
	}

	bestFile := ghclient.SelectBestFile(files)
	if bestFile == nil || bestFile.Patch == "" {
		return nil, "", fmt.Errorf("no renderable file found in commit %d", commitID)
	}

	renderer, err := codeimg.NewRenderer()
	if err != nil {
		return nil, "", fmt.Errorf("creating code image renderer: %w", err)
	}
	defer renderer.Close()

	pngBytes, err := renderer.RenderDiff(bestFile.Patch, bestFile.Filename, models.RenderOptions{
		Theme:           themeName,
		Template:        templateName,
		ShowLineNumbers: true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("rendering diff for commit %d: %w", commitID, err)
	}

	// Save to ~/.goviral/images/
	imagesDir := filepath.Join(config.DefaultConfigDir(), "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return nil, "", fmt.Errorf("creating images directory: %w", err)
	}

	filename := fmt.Sprintf("commit_%d_%s_%d.png", commitID, rc.SHA[:7], time.Now().Unix())
	savedPath := filepath.Join(imagesDir, filename)
	if err := os.WriteFile(savedPath, pngBytes, 0644); err != nil {
		return nil, "", fmt.Errorf("saving code image: %w", err)
	}

	return pngBytes, savedPath, nil
}

// RenderCodeImageFromSnippet renders a code diff image using an AI-selected snippet.
// It finds the matching file, computes the global line offset, and renders only
// the selected range using FormatDiffForTemplateRange.
func (s *RepoService) RenderCodeImageFromSnippet(ctx context.Context, snippet models.CodeSnippet, files []models.GitHubFileChange, commitID int64, sha string, templateName, themeName string, description string) ([]byte, string, error) {
	// Find the file and compute its global offset using the same ordering
	// and filtering as NumberedDiffBlock in the prompts package.
	var targetFile *models.GitHubFileChange
	globalOffset := 0

	for i := range files {
		f := &files[i]
		if models.IsLockfile(f.Filename) || f.Patch == "" {
			continue
		}
		if f.Filename == snippet.Filename {
			targetFile = f
			break
		}
		// Count the number of numbered lines in this file's patch
		// (same logic as NumberedDiffBlock: skip git header lines, count everything else)
		for _, raw := range strings.Split(f.Patch, "\n") {
			if strings.HasPrefix(raw, "diff ") || strings.HasPrefix(raw, "index ") ||
				strings.HasPrefix(raw, "--- ") || strings.HasPrefix(raw, "+++ ") ||
				strings.HasPrefix(raw, "new file") || strings.HasPrefix(raw, "deleted file") ||
				strings.HasPrefix(raw, "rename ") || strings.HasPrefix(raw, "similarity ") {
				continue
			}
			globalOffset++
		}
	}

	if targetFile == nil {
		return nil, "", fmt.Errorf("file %q not found in commit files", snippet.Filename)
	}

	diffData := codeimg.FormatDiffForTemplateRange(targetFile.Patch, globalOffset, snippet.StartLine, snippet.EndLine)
	diffData.Filename = snippet.Filename
	diffData.Language = codeimg.DetectLanguage(snippet.Filename)
	diffData.Description = description

	if len(diffData.Lines) == 0 {
		return nil, "", fmt.Errorf("no lines in range L%d-L%d for %s", snippet.StartLine, snippet.EndLine, snippet.Filename)
	}

	renderer, err := codeimg.NewRenderer()
	if err != nil {
		return nil, "", fmt.Errorf("creating code image renderer: %w", err)
	}
	defer renderer.Close()

	pngBytes, err := renderer.RenderDiffData(diffData, models.RenderOptions{
		Theme:           themeName,
		Template:        templateName,
		ShowLineNumbers: true,
		Description:     description,
	})
	if err != nil {
		return nil, "", fmt.Errorf("rendering snippet for commit %d: %w", commitID, err)
	}

	// Save to ~/.goviral/images/
	imagesDir := filepath.Join(config.DefaultConfigDir(), "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return nil, "", fmt.Errorf("creating images directory: %w", err)
	}

	shortSHA := sha
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}
	filename := fmt.Sprintf("commit_%d_%s_snippet_%d.png", commitID, shortSHA, time.Now().Unix())
	savedPath := filepath.Join(imagesDir, filename)
	if err := os.WriteFile(savedPath, pngBytes, 0644); err != nil {
		return nil, "", fmt.Errorf("saving code image: %w", err)
	}

	return pngBytes, savedPath, nil
}

// ReRenderCodeImage re-renders the code image for an existing content item
// using the current CodeImageDescription stored in the DB. It overwrites
// the existing PNG file and returns the updated path.
func (s *RepoService) ReRenderCodeImage(ctx context.Context, userID string, contentID int64) (string, error) {
	gc, err := s.db.GetGeneratedContentByID(userID, contentID)
	if err != nil {
		return "", fmt.Errorf("getting content %d: %w", contentID, err)
	}
	if gc == nil {
		return "", fmt.Errorf("content %d not found", contentID)
	}
	if gc.SourceCommitID == 0 {
		return "", fmt.Errorf("content %d has no source commit", contentID)
	}

	rc, err := s.db.GetRepoCommitByIDForUser(userID, gc.SourceCommitID)
	if err != nil {
		return "", fmt.Errorf("getting commit %d: %w", gc.SourceCommitID, err)
	}
	if rc == nil {
		return "", fmt.Errorf("commit %d not found", gc.SourceCommitID)
	}

	var files []models.GitHubFileChange
	if rc.FilesJSON != "" && rc.FilesJSON != "[]" {
		if err := json.Unmarshal([]byte(rc.FilesJSON), &files); err != nil {
			return "", fmt.Errorf("parsing commit files JSON: %w", err)
		}
	}

	renderer, err := codeimg.NewRenderer()
	if err != nil {
		return "", fmt.Errorf("creating code image renderer: %w", err)
	}
	defer renderer.Close()

	bestFile := ghclient.SelectBestFile(files)
	if bestFile == nil || bestFile.Patch == "" {
		return "", fmt.Errorf("no renderable file found in commit %d", gc.SourceCommitID)
	}

	diffData := codeimg.FormatDiffForTemplate(bestFile.Patch, 0)
	diffData.Filename = bestFile.Filename
	diffData.Language = codeimg.DetectLanguage(bestFile.Filename)
	diffData.Description = gc.CodeImageDescription

	pngBytes, err := renderer.RenderDiffData(diffData, models.RenderOptions{
		ShowLineNumbers: true,
		Description:     gc.CodeImageDescription,
	})
	if err != nil {
		return "", fmt.Errorf("rendering code image: %w", err)
	}

	// Overwrite existing file or create new one
	savedPath := gc.CodeImagePath
	if savedPath == "" {
		imagesDir := filepath.Join(config.DefaultConfigDir(), "images")
		if err := os.MkdirAll(imagesDir, 0755); err != nil {
			return "", fmt.Errorf("creating images directory: %w", err)
		}
		shortSHA := rc.SHA
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}
		savedPath = filepath.Join(imagesDir, fmt.Sprintf("commit_%d_%s_rerender_%d.png", gc.SourceCommitID, shortSHA, time.Now().Unix()))
	}

	if err := os.WriteFile(savedPath, pngBytes, 0644); err != nil {
		return "", fmt.Errorf("saving code image: %w", err)
	}

	return savedPath, nil
}
