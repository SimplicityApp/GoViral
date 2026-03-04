package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shuhao/goviral/pkg/models"
)

// --- github_repos CRUD ---

// UpsertGitHubRepo inserts or updates a GitHub repo by (user_id, full_name).
func (db *DB) UpsertGitHubRepo(userID string, repo *models.GitHubRepo) error {
	linksJSON, err := json.Marshal(repo.Links)
	if err != nil {
		linksJSON = []byte("[]")
	}

	query := `
	INSERT INTO github_repos (user_id, owner, name, full_name, description, default_branch, language, target_audience, links_json, added_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(user_id, full_name) DO UPDATE SET
		description=excluded.description, default_branch=excluded.default_branch, language=excluded.language,
		target_audience=excluded.target_audience, links_json=excluded.links_json
	RETURNING id
	`
	row := db.conn.QueryRow(query, userID, repo.Owner, repo.Name, repo.FullName, repo.Description, repo.DefaultBranch, repo.Language, repo.TargetAudience, string(linksJSON), time.Now())
	if err := row.Scan(&repo.ID); err != nil {
		return fmt.Errorf("upserting github repo: %w", err)
	}
	return nil
}

// GetGitHubRepo returns a repo by ID, scoped to user.
func (db *DB) GetGitHubRepo(userID string, id int64) (*models.GitHubRepo, error) {
	var r models.GitHubRepo
	var linksJSON string
	err := db.conn.QueryRow(
		"SELECT id, owner, name, full_name, description, default_branch, language, COALESCE(target_audience, ''), COALESCE(links_json, '[]'), added_at FROM github_repos WHERE id = ? AND user_id = ?",
		id, userID,
	).Scan(&r.ID, &r.Owner, &r.Name, &r.FullName, &r.Description, &r.DefaultBranch, &r.Language, &r.TargetAudience, &linksJSON, &r.AddedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting github repo: %w", err)
	}
	if linksJSON != "" && linksJSON != "[]" {
		if err := json.Unmarshal([]byte(linksJSON), &r.Links); err != nil {
			return nil, fmt.Errorf("parsing repo links JSON: %w", err)
		}
	}
	return &r, nil
}

// GetGitHubRepoByFullName returns a repo by its full_name, scoped to user.
func (db *DB) GetGitHubRepoByFullName(userID string, fullName string) (*models.GitHubRepo, error) {
	var r models.GitHubRepo
	var linksJSON string
	err := db.conn.QueryRow(
		"SELECT id, owner, name, full_name, description, default_branch, language, COALESCE(target_audience, ''), COALESCE(links_json, '[]'), added_at FROM github_repos WHERE full_name = ? AND user_id = ?",
		fullName, userID,
	).Scan(&r.ID, &r.Owner, &r.Name, &r.FullName, &r.Description, &r.DefaultBranch, &r.Language, &r.TargetAudience, &linksJSON, &r.AddedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting github repo by full name: %w", err)
	}
	if linksJSON != "" && linksJSON != "[]" {
		if err := json.Unmarshal([]byte(linksJSON), &r.Links); err != nil {
			return nil, fmt.Errorf("parsing repo links JSON: %w", err)
		}
	}
	return &r, nil
}

// ListGitHubRepos returns all tracked repos for a user.
func (db *DB) ListGitHubRepos(userID string) ([]models.GitHubRepo, error) {
	rows, err := db.conn.Query(
		"SELECT id, owner, name, full_name, description, default_branch, language, COALESCE(target_audience, ''), COALESCE(links_json, '[]'), added_at FROM github_repos WHERE user_id = ? ORDER BY added_at DESC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing github repos: %w", err)
	}
	defer rows.Close()

	var repos []models.GitHubRepo
	for rows.Next() {
		var r models.GitHubRepo
		var linksJSON string
		if err := rows.Scan(&r.ID, &r.Owner, &r.Name, &r.FullName, &r.Description, &r.DefaultBranch, &r.Language, &r.TargetAudience, &linksJSON, &r.AddedAt); err != nil {
			return nil, fmt.Errorf("scanning github repo row: %w", err)
		}
		if linksJSON != "" && linksJSON != "[]" {
			if err := json.Unmarshal([]byte(linksJSON), &r.Links); err != nil {
				return nil, fmt.Errorf("parsing repo links JSON: %w", err)
			}
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

// UpdateRepoSettings updates the target_audience and links for a repo, scoped to user.
func (db *DB) UpdateRepoSettings(userID string, id int64, targetAudience string, links []models.RepoLink) error {
	linksJSON, err := json.Marshal(links)
	if err != nil {
		return fmt.Errorf("marshaling repo links: %w", err)
	}
	_, err = db.conn.Exec(
		"UPDATE github_repos SET target_audience = ?, links_json = ? WHERE id = ? AND user_id = ?",
		targetAudience, string(linksJSON), id, userID,
	)
	if err != nil {
		return fmt.Errorf("updating repo settings: %w", err)
	}
	return nil
}

// DeleteGitHubRepo removes a tracked repo by ID, scoped to user.
func (db *DB) DeleteGitHubRepo(userID string, id int64) error {
	_, err := db.conn.Exec("DELETE FROM github_repos WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return fmt.Errorf("deleting github repo: %w", err)
	}
	return nil
}

// GetLatestCommitDate returns the most recent committed_at timestamp for the
// given repo, or nil if no commits exist yet.
func (db *DB) GetLatestCommitDate(repoID int64) (*time.Time, error) {
	var raw sql.NullString
	err := db.conn.QueryRow(
		"SELECT MAX(committed_at) FROM repo_commits WHERE repo_id = ?",
		repoID,
	).Scan(&raw)
	if err != nil {
		return nil, fmt.Errorf("getting latest commit date for repo %d: %w", repoID, err)
	}
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw.String)
	if err != nil {
		// Try the SQLite datetime format as fallback
		t, err = time.Parse("2006-01-02T15:04:05Z", raw.String)
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05", raw.String)
			if err != nil {
				// Go's time.String() format used by modernc.org/sqlite driver
				t, err = time.Parse("2006-01-02 15:04:05 +0000 UTC", raw.String)
				if err != nil {
					t, err = time.Parse("2006-01-02 15:04:05 -0700 MST", raw.String)
					if err != nil {
						return nil, fmt.Errorf("parsing latest commit date %q for repo %d: %w", raw.String, repoID, err)
					}
				}
			}
		}
	}
	return &t, nil
}

// --- repo_commits CRUD ---

// UpsertRepoCommit inserts or updates a repo commit.
func (db *DB) UpsertRepoCommit(rc *models.RepoCommitRecord) error {
	query := `
	INSERT INTO repo_commits (repo_id, sha, message, author_name, author_email, committed_at, additions, deletions, files_changed, diff_summary, diff_patch, files_json, fetched_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(repo_id, sha) DO UPDATE SET
		message=excluded.message, author_name=excluded.author_name, author_email=excluded.author_email,
		additions=excluded.additions, deletions=excluded.deletions, files_changed=excluded.files_changed,
		diff_summary=excluded.diff_summary, diff_patch=excluded.diff_patch, files_json=excluded.files_json,
		fetched_at=excluded.fetched_at
	RETURNING id
	`
	row := db.conn.QueryRow(query,
		rc.RepoID, rc.SHA, rc.Message, rc.AuthorName, rc.AuthorEmail, rc.CommittedAt,
		rc.Additions, rc.Deletions, rc.FilesChanged, rc.DiffSummary, rc.DiffPatch, rc.FilesJSON,
		time.Now(),
	)
	if err := row.Scan(&rc.ID); err != nil {
		return fmt.Errorf("upserting repo commit: %w", err)
	}
	return nil
}

// GetRepoCommit returns a commit by repo_id and SHA.
func (db *DB) GetRepoCommit(repoID int64, sha string) (*models.RepoCommitRecord, error) {
	var rc models.RepoCommitRecord
	err := db.conn.QueryRow(
		"SELECT id, repo_id, sha, message, author_name, author_email, committed_at, additions, deletions, files_changed, diff_summary, diff_patch, files_json, fetched_at FROM repo_commits WHERE repo_id = ? AND sha = ?",
		repoID, sha,
	).Scan(&rc.ID, &rc.RepoID, &rc.SHA, &rc.Message, &rc.AuthorName, &rc.AuthorEmail, &rc.CommittedAt, &rc.Additions, &rc.Deletions, &rc.FilesChanged, &rc.DiffSummary, &rc.DiffPatch, &rc.FilesJSON, &rc.FetchedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting repo commit: %w", err)
	}
	return &rc, nil
}

// GetRepoCommitByID returns a commit by its ID.
func (db *DB) GetRepoCommitByID(id int64) (*models.RepoCommitRecord, error) {
	var rc models.RepoCommitRecord
	err := db.conn.QueryRow(
		"SELECT id, repo_id, sha, message, author_name, author_email, committed_at, additions, deletions, files_changed, diff_summary, diff_patch, files_json, fetched_at FROM repo_commits WHERE id = ?",
		id,
	).Scan(&rc.ID, &rc.RepoID, &rc.SHA, &rc.Message, &rc.AuthorName, &rc.AuthorEmail, &rc.CommittedAt, &rc.Additions, &rc.Deletions, &rc.FilesChanged, &rc.DiffSummary, &rc.DiffPatch, &rc.FilesJSON, &rc.FetchedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting repo commit by id: %w", err)
	}
	return &rc, nil
}

// ListRepoCommits returns commits for a repo, ordered by committed_at desc.
func (db *DB) ListRepoCommits(repoID int64, limit int) ([]models.RepoCommitRecord, error) {
	query := "SELECT id, repo_id, sha, message, author_name, author_email, committed_at, additions, deletions, files_changed, diff_summary, diff_patch, files_json, fetched_at FROM repo_commits WHERE repo_id = ? ORDER BY committed_at DESC"
	var args []interface{}
	args = append(args, repoID)
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing repo commits: %w", err)
	}
	defer rows.Close()

	var commits []models.RepoCommitRecord
	for rows.Next() {
		var rc models.RepoCommitRecord
		if err := rows.Scan(&rc.ID, &rc.RepoID, &rc.SHA, &rc.Message, &rc.AuthorName, &rc.AuthorEmail, &rc.CommittedAt, &rc.Additions, &rc.Deletions, &rc.FilesChanged, &rc.DiffSummary, &rc.DiffPatch, &rc.FilesJSON, &rc.FetchedAt); err != nil {
			return nil, fmt.Errorf("scanning repo commit row: %w", err)
		}
		commits = append(commits, rc)
	}
	return commits, rows.Err()
}

// ParseCommitFiles parses the FilesJSON field of a RepoCommitRecord.
func ParseCommitFiles(filesJSON string) ([]models.GitHubFileChange, error) {
	var files []models.GitHubFileChange
	if filesJSON == "" || filesJSON == "[]" {
		return files, nil
	}
	if err := json.Unmarshal([]byte(filesJSON), &files); err != nil {
		return nil, fmt.Errorf("parsing commit files JSON: %w", err)
	}
	return files, nil
}
