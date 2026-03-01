// Package github provides a GitHub REST API v3 client for fetching repository
// and commit data to drive content generation workflows.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shuhao/goviral/pkg/models"
)

const defaultBaseURL = "https://api.github.com"

// Compile-time interface compliance check.
var _ models.GitHubClient = (*Client)(nil)

// Client implements models.GitHubClient using the GitHub REST API v3.
// Authenticate with a personal access token or a fine-grained token that has
// read access to the target repositories.
type Client struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

// NewClient creates a Client authenticated with the provided GitHub token.
// The token is sent as a Bearer token on every request.
func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      token,
		baseURL:    defaultBaseURL,
	}
}

// GetRepo fetches repository metadata for the given owner/name pair.
func (c *Client) GetRepo(ctx context.Context, owner, name string) (*models.GitHubRepo, error) {
	path := fmt.Sprintf("/repos/%s/%s", owner, name)
	body, err := c.doGet(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching repo %s/%s: %w", owner, name, err)
	}

	var raw githubRepoResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing repo response for %s/%s: %w", owner, name, err)
	}

	return &models.GitHubRepo{
		ID:            raw.ID,
		Owner:         owner,
		Name:          name,
		FullName:      raw.FullName,
		Description:   raw.Description,
		DefaultBranch: raw.DefaultBranch,
		Language:      raw.Language,
		Private:       raw.Private,
		AddedAt:       time.Now(),
	}, nil
}

// ListCommits returns commits for the given repository, honoring the options in
// opts. When opts.Limit is 0 or negative it defaults to 30 (GitHub's default
// per_page). The maximum accepted by the GitHub API per page is 100.
func (c *Client) ListCommits(ctx context.Context, owner, repo string, opts models.CommitListOptions) ([]models.GitHubCommit, error) {
	params := url.Values{}
	if opts.Since != nil {
		params.Set("since", opts.Since.Format(time.RFC3339))
	}
	if opts.Until != nil {
		params.Set("until", opts.Until.Format(time.RFC3339))
	}
	if opts.Branch != "" {
		params.Set("sha", opts.Branch)
	}

	perPage := opts.Limit
	if perPage <= 0 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}
	params.Set("per_page", strconv.Itoa(perPage))

	path := fmt.Sprintf("/repos/%s/%s/commits", owner, repo)
	body, err := c.doGet(ctx, path, params)
	if err != nil {
		return nil, fmt.Errorf("listing commits for %s/%s: %w", owner, repo, err)
	}

	var raw []githubCommitSummary
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing commits list for %s/%s: %w", owner, repo, err)
	}

	commits := make([]models.GitHubCommit, 0, len(raw))
	for _, r := range raw {
		committedAt, _ := time.Parse(time.RFC3339, r.Commit.Author.Date)
		commits = append(commits, models.GitHubCommit{
			SHA:         r.SHA,
			Message:     r.Commit.Message,
			AuthorName:  r.Commit.Author.Name,
			AuthorEmail: r.Commit.Author.Email,
			CommittedAt: committedAt,
		})
	}

	return commits, nil
}

// ListAllCommits paginates through all commit pages for the given repository.
// It uses per_page=100 (GitHub max) and iterates until a page returns fewer
// results or the context is cancelled. The optional onPage callback is invoked
// after each page with the 1-based page number and total commits fetched so far.
// If opts.Limit > 0 the total number of returned commits is capped to that value.
func (c *Client) ListAllCommits(ctx context.Context, owner, repo string, opts models.CommitListOptions, onPage func(page int, count int)) ([]models.GitHubCommit, error) {
	const perPage = 100
	var all []models.GitHubCommit

	for page := 1; ; page++ {
		select {
		case <-ctx.Done():
			return all, ctx.Err()
		default:
		}

		params := url.Values{}
		if opts.Since != nil {
			params.Set("since", opts.Since.Format(time.RFC3339))
		}
		if opts.Until != nil {
			params.Set("until", opts.Until.Format(time.RFC3339))
		}
		if opts.Branch != "" {
			params.Set("sha", opts.Branch)
		}
		params.Set("per_page", strconv.Itoa(perPage))
		params.Set("page", strconv.Itoa(page))

		path := fmt.Sprintf("/repos/%s/%s/commits", owner, repo)
		body, err := c.doGet(ctx, path, params)
		if err != nil {
			return all, fmt.Errorf("listing commits page %d for %s/%s: %w", page, owner, repo, err)
		}

		var raw []githubCommitSummary
		if err := json.Unmarshal(body, &raw); err != nil {
			return all, fmt.Errorf("parsing commits page %d for %s/%s: %w", page, owner, repo, err)
		}

		for _, r := range raw {
			committedAt, _ := time.Parse(time.RFC3339, r.Commit.Author.Date)
			all = append(all, models.GitHubCommit{
				SHA:         r.SHA,
				Message:     r.Commit.Message,
				AuthorName:  r.Commit.Author.Name,
				AuthorEmail: r.Commit.Author.Email,
				CommittedAt: committedAt,
			})
		}

		if onPage != nil {
			onPage(page, len(all))
		}

		// Stop if we got fewer than a full page (last page)
		if len(raw) < perPage {
			break
		}

		// Stop if we've hit the requested limit
		if opts.Limit > 0 && len(all) >= opts.Limit {
			all = all[:opts.Limit]
			break
		}
	}

	return all, nil
}

// GetCommit fetches full commit detail including the diff patch and per-file
// change statistics for the given SHA.
func (c *Client) GetCommit(ctx context.Context, owner, repo, sha string) (*models.GitHubCommit, error) {
	path := fmt.Sprintf("/repos/%s/%s/commits/%s", owner, repo, sha)
	body, err := c.doGet(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching commit %s in %s/%s: %w", sha, owner, repo, err)
	}

	var raw githubCommitDetail
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing commit detail for %s in %s/%s: %w", sha, owner, repo, err)
	}

	committedAt, _ := time.Parse(time.RFC3339, raw.Commit.Author.Date)

	files := make([]models.GitHubFileChange, 0, len(raw.Files))
	var patchParts []string
	for _, f := range raw.Files {
		files = append(files, models.GitHubFileChange{
			Filename:  f.Filename,
			Status:    f.Status,
			Additions: f.Additions,
			Deletions: f.Deletions,
			Patch:     f.Patch,
		})
		if f.Patch != "" {
			patchParts = append(patchParts, fmt.Sprintf("--- a/%s\n+++ b/%s\n%s", f.Filename, f.Filename, f.Patch))
		}
	}

	return &models.GitHubCommit{
		SHA:          raw.SHA,
		Message:      raw.Commit.Message,
		AuthorName:   raw.Commit.Author.Name,
		AuthorEmail:  raw.Commit.Author.Email,
		CommittedAt:  committedAt,
		Additions:    raw.Stats.Additions,
		Deletions:    raw.Stats.Deletions,
		FilesChanged: len(raw.Files),
		DiffPatch:    strings.Join(patchParts, "\n"),
		Files:        files,
	}, nil
}

// ListUserRepos fetches all repositories accessible to the authenticated user
// (personal, collaborator, and organization member repos). It paginates through
// all pages using per_page=100.
func (c *Client) ListUserRepos(ctx context.Context) ([]models.GitHubRepo, error) {
	const perPage = 100
	var all []models.GitHubRepo

	for page := 1; ; page++ {
		select {
		case <-ctx.Done():
			return all, ctx.Err()
		default:
		}

		params := url.Values{}
		params.Set("affiliation", "owner,collaborator,organization_member")
		params.Set("sort", "updated")
		params.Set("per_page", strconv.Itoa(perPage))
		params.Set("page", strconv.Itoa(page))

		body, err := c.doGet(ctx, "/user/repos", params)
		if err != nil {
			return all, fmt.Errorf("listing user repos page %d: %w", page, err)
		}

		var raw []githubUserRepoResponse
		if err := json.Unmarshal(body, &raw); err != nil {
			return all, fmt.Errorf("parsing user repos page %d: %w", page, err)
		}

		for _, r := range raw {
			all = append(all, models.GitHubRepo{
				ID:            r.ID,
				Owner:         r.Owner.Login,
				Name:          r.Name,
				FullName:      r.FullName,
				Description:   r.Description,
				DefaultBranch: r.DefaultBranch,
				Language:      r.Language,
				Private:       r.Private,
			})
		}

		if len(raw) < perPage {
			break
		}
	}

	return all, nil
}

// doGet executes an authenticated GET request against the GitHub API.
// It checks X-RateLimit-Remaining and returns a descriptive error when the
// caller has exhausted its quota before attempting the underlying HTTP call.
func (c *Client) doGet(ctx context.Context, path string, params url.Values) ([]byte, error) {
	reqURL := c.baseURL + path
	if len(params) > 0 {
		reqURL = reqURL + "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", path, err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request to %s: %w", path, err)
	}
	defer resp.Body.Close()

	// Check rate limit header before reading the body — a remaining count of 0
	// means the window is exhausted and subsequent calls will all fail.
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining == "0" {
		resetHeader := resp.Header.Get("X-RateLimit-Reset")
		resetUnix, _ := strconv.ParseInt(resetHeader, 10, 64)
		var resetInfo string
		if resetUnix > 0 {
			resetAt := time.Unix(resetUnix, 0)
			resetInfo = fmt.Sprintf("; resets at %s (in %s)",
				resetAt.Format(time.RFC3339),
				time.Until(resetAt).Round(time.Second))
		}
		return nil, fmt.Errorf("github rate limit exhausted%s", resetInfo)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body from %s: %w", path, err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("github authentication failed (HTTP 401): check your token")
	}
	if resp.StatusCode == http.StatusForbidden {
		// Forbidden can also mean rate-limited at the secondary rate limit.
		var apiErr githubErrorResponse
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
			return nil, fmt.Errorf("github API forbidden: %s", apiErr.Message)
		}
		return nil, fmt.Errorf("github API forbidden (HTTP 403)")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("github resource not found at %s (HTTP 404)", path)
	}
	if resp.StatusCode != http.StatusOK {
		var apiErr githubErrorResponse
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
			return nil, fmt.Errorf("github API error (HTTP %d): %s", resp.StatusCode, apiErr.Message)
		}
		return nil, fmt.Errorf("github API returned HTTP %d for %s", resp.StatusCode, path)
	}

	return body, nil
}

// ---- GitHub API response types ----

// githubErrorResponse is the standard GitHub API error envelope.
type githubErrorResponse struct {
	Message string `json:"message"`
}

// githubRepoResponse maps the fields we use from GET /repos/{owner}/{repo}.
type githubRepoResponse struct {
	ID            int64  `json:"id"`
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	DefaultBranch string `json:"default_branch"`
	Language      string `json:"language"`
	Private       bool   `json:"private"`
}

// githubUserRepoResponse maps fields from GET /user/repos.
type githubUserRepoResponse struct {
	ID            int64  `json:"id"`
	FullName      string `json:"full_name"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	DefaultBranch string `json:"default_branch"`
	Language      string `json:"language"`
	Private       bool   `json:"private"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
}

// githubCommitSummary maps the subset of fields returned by the commits list
// endpoint (GET /repos/{owner}/{repo}/commits).
type githubCommitSummary struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Date  string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

// githubCommitDetail maps the full response from GET /repos/{owner}/{repo}/commits/{sha}.
type githubCommitDetail struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Date  string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
	Stats struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
		Total     int `json:"total"`
	} `json:"stats"`
	Files []struct {
		Filename  string `json:"filename"`
		Status    string `json:"status"`
		Additions int    `json:"additions"`
		Deletions int    `json:"deletions"`
		Patch     string `json:"patch"`
	} `json:"files"`
}
