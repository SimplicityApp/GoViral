package models

import (
	"context"
	"time"
)

// RepoLink represents a custom link associated with a repository (e.g. GitHub URL, PyPI link).
type RepoLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// GitHubRepo represents a tracked GitHub repository.
type GitHubRepo struct {
	ID             int64
	Owner          string
	Name           string
	FullName       string
	Description    string
	DefaultBranch  string
	Language       string
	Private        bool
	TargetAudience string
	Links          []RepoLink
	AddedAt        time.Time
}

// GitHubCommit represents a single commit from the GitHub API.
type GitHubCommit struct {
	SHA          string
	Message      string
	AuthorName   string
	AuthorEmail  string
	CommittedAt  time.Time
	Additions    int
	Deletions    int
	FilesChanged int
	DiffPatch    string             // Full unified diff for the commit
	Files        []GitHubFileChange // Per-file changes
}

// RepoCommitRecord is the DB row for a fetched commit.
type RepoCommitRecord struct {
	ID           int64
	RepoID       int64
	SHA          string
	Message      string
	AuthorName   string
	AuthorEmail  string
	CommittedAt  time.Time
	Additions    int
	Deletions    int
	FilesChanged int
	DiffSummary  string
	DiffPatch    string
	FilesJSON    string // JSON-encoded []GitHubFileChange
	FetchedAt    time.Time
}

// GitHubFileChange represents a single file changed in a commit.
type GitHubFileChange struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"` // "added", "modified", "removed", "renamed"
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch"`
}

// CommitListOptions configures how commits are fetched.
type CommitListOptions struct {
	Since  *time.Time
	Until  *time.Time
	Limit  int
	Branch string
}

// GitHubClient defines the interface for interacting with the GitHub API.
type GitHubClient interface {
	GetRepo(ctx context.Context, owner, name string) (*GitHubRepo, error)
	ListCommits(ctx context.Context, owner, repo string, opts CommitListOptions) ([]GitHubCommit, error)
	GetCommit(ctx context.Context, owner, repo, sha string) (*GitHubCommit, error)
	ListUserRepos(ctx context.Context) ([]GitHubRepo, error)
}

// CodeImageRenderer renders code diffs as PNG images.
type CodeImageRenderer interface {
	RenderDiff(diff, filename string, opts RenderOptions) ([]byte, error)
	Close()
}

// RenderOptions configures how a diff image is rendered.
type RenderOptions struct {
	Theme           string // Theme name (e.g. "github-dark", "dracula", "nord")
	Template        string // Template name (e.g. "github", "macos", "vscode", "minimal", "terminal", "card")
	MaxLines        int
	Width           int
	FontSize        int
	ShowLineNumbers bool
	Description     string // Optional description text (used by templates that support it)
	RepoName        string // Optional "owner/repo" for display
}

// RepoPostRequest contains parameters for generating posts from commits.
type RepoPostRequest struct {
	Commit           GitHubCommit
	Repo             GitHubRepo
	Persona          Persona
	TargetPlatform   string
	Count            int
	MaxChars         int
	IncludeCodeImage bool
	StyleDirection   string
	TargetAudience   string
	Links            []RepoLink
}
