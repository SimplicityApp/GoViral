package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shuhao/goviral/pkg/models"
	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
}

// New opens a SQLite database at the given path and runs migrations.
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better concurrent access
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS my_posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		platform TEXT NOT NULL,
		platform_post_id TEXT NOT NULL,
		content TEXT NOT NULL,
		likes INTEGER DEFAULT 0,
		reposts INTEGER DEFAULT 0,
		comments INTEGER DEFAULT 0,
		impressions INTEGER DEFAULT 0,
		posted_at DATETIME,
		fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS persona (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		platform TEXT NOT NULL,
		profile_json TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS trending_posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		platform TEXT NOT NULL,
		platform_post_id TEXT UNIQUE NOT NULL,
		author_username TEXT,
		author_name TEXT,
		content TEXT NOT NULL,
		likes INTEGER DEFAULT 0,
		reposts INTEGER DEFAULT 0,
		comments INTEGER DEFAULT 0,
		impressions INTEGER DEFAULT 0,
		niche_tags TEXT,
		posted_at DATETIME,
		fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS generated_content (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_trending_id INTEGER REFERENCES trending_posts(id),
		target_platform TEXT NOT NULL,
		original_content TEXT NOT NULL,
		generated_content TEXT NOT NULL,
		persona_id INTEGER REFERENCES persona(id),
		prompt_used TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		status TEXT DEFAULT 'draft'
	);

	CREATE TABLE IF NOT EXISTS scheduled_posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		generated_content_id INTEGER NOT NULL REFERENCES generated_content(id),
		scheduled_at DATETIME NOT NULL,
		status TEXT DEFAULT 'pending',
		error_message TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.conn.Exec(schema); err != nil {
		return err
	}

	// Add new columns to generated_content (ignore errors for already-existing columns)
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN platform_post_ids TEXT DEFAULT ''")
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN posted_at DATETIME")
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN image_prompt TEXT DEFAULT ''")
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN image_path TEXT DEFAULT ''")

	// Add media_json column to trending_posts
	db.conn.Exec("ALTER TABLE trending_posts ADD COLUMN media_json TEXT DEFAULT '[]'")

	// Add repost columns to generated_content
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN is_repost INTEGER DEFAULT 0")
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN quote_tweet_id TEXT DEFAULT ''")

	// Add daemon_batch_id to generated_content
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN daemon_batch_id INTEGER DEFAULT 0")

	// Add comment support
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN is_comment INTEGER DEFAULT 0")

	// Daemon batches table (batch_type added later via ALTER)
	db.conn.Exec(`CREATE TABLE IF NOT EXISTS daemon_batches (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		platform TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		content_ids TEXT DEFAULT '[]',
		trending_ids TEXT DEFAULT '[]',
		telegram_message_id INTEGER DEFAULT 0,
		approval_source TEXT DEFAULT '',
		reply_text TEXT DEFAULT '',
		parsed_intent TEXT DEFAULT '',
		error_message TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		notified_at DATETIME,
		resolved_at DATETIME
	)`)

	// Add batch_type to daemon_batches for comment batches
	db.conn.Exec("ALTER TABLE daemon_batches ADD COLUMN batch_type TEXT DEFAULT 'content'")

	// GitHub repos table
	db.conn.Exec(`CREATE TABLE IF NOT EXISTS github_repos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		owner TEXT NOT NULL,
		name TEXT NOT NULL,
		full_name TEXT NOT NULL,
		description TEXT DEFAULT '',
		default_branch TEXT DEFAULT 'main',
		language TEXT DEFAULT '',
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	// Repo commits table
	db.conn.Exec(`CREATE TABLE IF NOT EXISTS repo_commits (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_id INTEGER NOT NULL REFERENCES github_repos(id),
		sha TEXT NOT NULL,
		message TEXT NOT NULL,
		author_name TEXT DEFAULT '',
		author_email TEXT DEFAULT '',
		committed_at DATETIME,
		additions INTEGER DEFAULT 0,
		deletions INTEGER DEFAULT 0,
		files_changed INTEGER DEFAULT 0,
		diff_summary TEXT DEFAULT '',
		diff_patch TEXT DEFAULT '',
		files_json TEXT DEFAULT '[]',
		fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(repo_id, sha)
	)`)

	// Add per-repo custom settings
	db.conn.Exec("ALTER TABLE github_repos ADD COLUMN target_audience TEXT DEFAULT ''")
	db.conn.Exec("ALTER TABLE github_repos ADD COLUMN links_json TEXT DEFAULT '[]'")

	// Add commit-sourced content columns to generated_content
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN source_type TEXT DEFAULT 'trending'")
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN source_commit_id INTEGER DEFAULT 0")
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN code_image_path TEXT DEFAULT ''")

	// Add thread_urn to trending_posts for LinkedIn ugcPost comment threading
	db.conn.Exec("ALTER TABLE trending_posts ADD COLUMN thread_urn TEXT DEFAULT ''")

	// Add code image description for AI-generated overlay text
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN code_image_description TEXT DEFAULT ''")

	// Video support columns for YouTube Shorts & TikTok
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN video_path TEXT DEFAULT ''")
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN thumbnail_path TEXT DEFAULT ''")
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN video_duration INTEGER DEFAULT 0")
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN video_title TEXT DEFAULT ''")

	// Video fields for trending posts
	db.conn.Exec("ALTER TABLE trending_posts ADD COLUMN video_url TEXT DEFAULT ''")
	db.conn.Exec("ALTER TABLE trending_posts ADD COLUMN view_count INTEGER DEFAULT 0")
	db.conn.Exec("ALTER TABLE trending_posts ADD COLUMN duration INTEGER DEFAULT 0")
	db.conn.Exec("ALTER TABLE trending_posts ADD COLUMN is_video INTEGER DEFAULT 0")

	// Archive existing cross-platform content (source trending post platform ≠ target_platform).
	// Idempotent: status NOT IN ('posted', 'archived') prevents re-archiving.
	db.conn.Exec(`
		UPDATE generated_content SET status = 'archived'
		WHERE status NOT IN ('posted', 'archived')
		  AND source_trending_id != 0
		  AND id IN (
		      SELECT gc.id FROM generated_content gc
		      JOIN trending_posts tp ON gc.source_trending_id = tp.id
		      WHERE gc.target_platform != tp.platform
		  )
	`)

	// --- Multi-tenancy: users table and user_id columns ---
	db.conn.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	// Add user_id column to user-scoped tables (idempotent ALTERs)
	db.conn.Exec("ALTER TABLE my_posts ADD COLUMN user_id TEXT NOT NULL DEFAULT ''")
	db.conn.Exec("ALTER TABLE persona ADD COLUMN user_id TEXT NOT NULL DEFAULT ''")
	db.conn.Exec("ALTER TABLE generated_content ADD COLUMN user_id TEXT NOT NULL DEFAULT ''")
	db.conn.Exec("ALTER TABLE scheduled_posts ADD COLUMN user_id TEXT NOT NULL DEFAULT ''")
	db.conn.Exec("ALTER TABLE github_repos ADD COLUMN user_id TEXT NOT NULL DEFAULT ''")

	// --- Migration: drop column-level UNIQUE from my_posts and github_repos ---
	// SQLite column-level UNIQUE creates sqlite_autoindex_<table>_N entries.
	// These conflict with composite unique indexes for multi-tenant upserts.
	// We rebuild the tables to remove the column-level constraint.
	if err := db.dropColumnUnique("my_posts", `CREATE TABLE my_posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		platform TEXT NOT NULL,
		platform_post_id TEXT NOT NULL,
		content TEXT NOT NULL,
		likes INTEGER DEFAULT 0,
		reposts INTEGER DEFAULT 0,
		comments INTEGER DEFAULT 0,
		impressions INTEGER DEFAULT 0,
		posted_at DATETIME,
		fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		user_id TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		return fmt.Errorf("migrating my_posts unique constraint: %w", err)
	}

	if err := db.dropColumnUnique("github_repos", `CREATE TABLE github_repos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		owner TEXT NOT NULL,
		name TEXT NOT NULL,
		full_name TEXT NOT NULL,
		description TEXT DEFAULT '',
		default_branch TEXT DEFAULT 'main',
		language TEXT DEFAULT '',
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		target_audience TEXT DEFAULT '',
		links_json TEXT DEFAULT '[]',
		user_id TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		return fmt.Errorf("migrating github_repos unique constraint: %w", err)
	}

	// Composite unique indexes for user-scoped data
	// NOTE: These must run AFTER dropColumnUnique table rebuilds, which destroy all indexes.
	db.conn.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_my_posts_user_platform_post ON my_posts(user_id, platform_post_id)")
	db.conn.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_persona_user_platform ON persona(user_id, platform)")
	db.conn.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_github_repos_user_fullname ON github_repos(user_id, full_name)")

	return nil
}

// dropColumnUnique checks if a table has a sqlite_autoindex (column-level UNIQUE)
// and rebuilds the table without it. Idempotent: no-ops if no autoindex exists.
func (db *DB) dropColumnUnique(tableName, createSQL string) error {
	rows, err := db.conn.Query(fmt.Sprintf("PRAGMA index_list(%s)", tableName))
	if err != nil {
		return fmt.Errorf("querying index list for %s: %w", tableName, err)
	}
	defer rows.Close()

	hasAutoIndex := false
	for rows.Next() {
		var seq int
		var name, origin string
		var unique, partial int
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return fmt.Errorf("scanning index row for %s: %w", tableName, err)
		}
		if strings.HasPrefix(name, "sqlite_autoindex_"+tableName+"_") {
			hasAutoIndex = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating indexes for %s: %w", tableName, err)
	}

	if !hasAutoIndex {
		return nil
	}

	// Rebuild the table inside a transaction to remove column-level UNIQUE
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction for %s rebuild: %w", tableName, err)
	}
	defer tx.Rollback()

	tmpName := tableName + "_old"
	if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", tableName, tmpName)); err != nil {
		return fmt.Errorf("renaming %s to %s: %w", tableName, tmpName, err)
	}
	if _, err := tx.Exec(createSQL); err != nil {
		return fmt.Errorf("creating new %s: %w", tableName, err)
	}
	if _, err := tx.Exec(fmt.Sprintf("INSERT INTO %s SELECT * FROM %s", tableName, tmpName)); err != nil {
		return fmt.Errorf("copying data from %s into %s: %w", tmpName, tableName, err)
	}
	if _, err := tx.Exec(fmt.Sprintf("DROP TABLE %s", tmpName)); err != nil {
		return fmt.Errorf("dropping %s: %w", tmpName, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing %s rebuild: %w", tableName, err)
	}

	return nil
}

// GetOrCreateUser upserts a user by UUID, updating last_seen_at on conflict.
func (db *DB) GetOrCreateUser(uuid string) error {
	_, err := db.conn.Exec(
		"INSERT INTO users (uuid) VALUES (?) ON CONFLICT(uuid) DO UPDATE SET last_seen_at = CURRENT_TIMESTAMP",
		uuid,
	)
	if err != nil {
		return fmt.Errorf("upserting user %s: %w", uuid, err)
	}
	return nil
}

// --- my_posts CRUD ---

// UpsertPost inserts or updates a post by (user_id, platform_post_id).
func (db *DB) UpsertPost(userID string, p *models.Post) error {
	query := `
	INSERT INTO my_posts (user_id, platform, platform_post_id, content, likes, reposts, comments, impressions, posted_at, fetched_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(user_id, platform_post_id) DO UPDATE SET
		content=excluded.content, likes=excluded.likes, reposts=excluded.reposts,
		comments=excluded.comments, impressions=excluded.impressions, fetched_at=excluded.fetched_at
	`
	_, err := db.conn.Exec(query, userID, p.Platform, p.PlatformPostID, p.Content, p.Likes, p.Reposts, p.Comments, p.Impressions, p.PostedAt, time.Now())
	if err != nil {
		return fmt.Errorf("upserting post: %w", err)
	}
	return nil
}

// GetPostsByPlatform returns all posts for a given platform and user.
func (db *DB) GetPostsByPlatform(userID string, platform string) ([]models.Post, error) {
	rows, err := db.conn.Query(
		"SELECT id, platform, platform_post_id, content, likes, reposts, comments, impressions, posted_at, fetched_at FROM my_posts WHERE user_id = ? AND platform = ? ORDER BY CASE WHEN posted_at > '2000-01-01' THEN posted_at ELSE fetched_at END DESC",
		userID, platform,
	)
	if err != nil {
		return nil, fmt.Errorf("querying posts: %w", err)
	}
	defer rows.Close()
	return scanPosts(rows)
}

// GetAllPosts returns all posts across platforms for a user.
func (db *DB) GetAllPosts(userID string) ([]models.Post, error) {
	rows, err := db.conn.Query(
		"SELECT id, platform, platform_post_id, content, likes, reposts, comments, impressions, posted_at, fetched_at FROM my_posts WHERE user_id = ? ORDER BY CASE WHEN posted_at > '2000-01-01' THEN posted_at ELSE fetched_at END DESC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying all posts: %w", err)
	}
	defer rows.Close()
	return scanPosts(rows)
}

func scanPosts(rows *sql.Rows) ([]models.Post, error) {
	var posts []models.Post
	for rows.Next() {
		var p models.Post
		if err := rows.Scan(&p.ID, &p.Platform, &p.PlatformPostID, &p.Content, &p.Likes, &p.Reposts, &p.Comments, &p.Impressions, &p.PostedAt, &p.FetchedAt); err != nil {
			return nil, fmt.Errorf("scanning post row: %w", err)
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

// --- persona CRUD ---

// UpsertPersona inserts or updates a persona profile for a platform and user.
func (db *DB) UpsertPersona(userID string, p *models.Persona) error {
	profileJSON, err := json.Marshal(p.Profile)
	if err != nil {
		return fmt.Errorf("marshaling persona profile: %w", err)
	}

	// Check if persona exists for this platform and user
	var existingID int64
	err = db.conn.QueryRow("SELECT id FROM persona WHERE platform = ? AND user_id = ?", p.Platform, userID).Scan(&existingID)
	if err == sql.ErrNoRows {
		_, err = db.conn.Exec(
			"INSERT INTO persona (platform, profile_json, user_id) VALUES (?, ?, ?)",
			p.Platform, string(profileJSON), userID,
		)
		if err != nil {
			return fmt.Errorf("inserting persona: %w", err)
		}
	} else if err == nil {
		_, err = db.conn.Exec(
			"UPDATE persona SET profile_json = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			string(profileJSON), existingID,
		)
		if err != nil {
			return fmt.Errorf("updating persona: %w", err)
		}
	} else {
		return fmt.Errorf("checking existing persona: %w", err)
	}
	return nil
}

// GetPersona returns the persona for a given platform and user.
func (db *DB) GetPersona(userID string, platform string) (*models.Persona, error) {
	var p models.Persona
	var profileJSON string
	err := db.conn.QueryRow(
		"SELECT id, platform, profile_json, created_at, updated_at FROM persona WHERE platform = ? AND user_id = ?",
		platform, userID,
	).Scan(&p.ID, &p.Platform, &profileJSON, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying persona: %w", err)
	}
	if err := json.Unmarshal([]byte(profileJSON), &p.Profile); err != nil {
		return nil, fmt.Errorf("parsing persona profile JSON: %w", err)
	}
	return &p, nil
}

// --- trending_posts CRUD ---

// UpsertTrendingPost inserts or updates a trending post.
func (db *DB) UpsertTrendingPost(tp *models.TrendingPost) error {
	nicheTags, err := json.Marshal(tp.NicheTags)
	if err != nil {
		return fmt.Errorf("marshaling niche tags: %w", err)
	}

	mediaJSON, err := json.Marshal(tp.Media)
	if err != nil {
		return fmt.Errorf("marshaling media: %w", err)
	}

	isVideo := 0
	if tp.IsVideo {
		isVideo = 1
	}
	query := `
	INSERT INTO trending_posts (platform, platform_post_id, author_username, author_name, content, likes, reposts, comments, impressions, niche_tags, media_json, posted_at, fetched_at, thread_urn, video_url, view_count, duration, is_video)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(platform_post_id) DO UPDATE SET
		content=excluded.content, likes=excluded.likes, reposts=excluded.reposts,
		comments=excluded.comments, impressions=excluded.impressions, niche_tags=excluded.niche_tags, media_json=excluded.media_json, fetched_at=excluded.fetched_at, thread_urn=excluded.thread_urn, video_url=excluded.video_url, view_count=excluded.view_count, duration=excluded.duration, is_video=excluded.is_video
	RETURNING id
	`
	row := db.conn.QueryRow(query, tp.Platform, tp.PlatformPostID, tp.AuthorUsername, tp.AuthorName, tp.Content, tp.Likes, tp.Reposts, tp.Comments, tp.Impressions, string(nicheTags), string(mediaJSON), tp.PostedAt, time.Now(), tp.ThreadURN, tp.VideoURL, tp.ViewCount, tp.Duration, isVideo)
	if err := row.Scan(&tp.ID); err != nil {
		return fmt.Errorf("upserting trending post: %w", err)
	}
	return nil
}

// GetTrendingPosts returns trending posts from the most recent fetch session,
// with optional platform and limit filters. Only posts fetched within 5 minutes
// of the latest fetched_at are returned, so stale results from previous runs
// are excluded.
func (db *DB) GetTrendingPosts(platform string, limit int) ([]models.TrendingPost, error) {
	query := "SELECT id, platform, platform_post_id, author_username, author_name, content, likes, reposts, comments, impressions, niche_tags, COALESCE(media_json, '[]'), posted_at, fetched_at, COALESCE(thread_urn, ''), COALESCE(video_url, ''), COALESCE(view_count, 0), COALESCE(duration, 0), COALESCE(is_video, 0) FROM trending_posts"
	var args []interface{}

	if platform != "" {
		query += " WHERE platform = ?"
		args = append(args, platform)
	}
	query += " ORDER BY likes DESC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying trending posts: %w", err)
	}
	defer rows.Close()

	var allPosts []models.TrendingPost
	for rows.Next() {
		var tp models.TrendingPost
		var nicheTagsJSON, mediaJSON string
		if err := rows.Scan(&tp.ID, &tp.Platform, &tp.PlatformPostID, &tp.AuthorUsername, &tp.AuthorName, &tp.Content, &tp.Likes, &tp.Reposts, &tp.Comments, &tp.Impressions, &nicheTagsJSON, &mediaJSON, &tp.PostedAt, &tp.FetchedAt, &tp.ThreadURN, &tp.VideoURL, &tp.ViewCount, &tp.Duration, &tp.IsVideo); err != nil {
			return nil, fmt.Errorf("scanning trending post row: %w", err)
		}
		if nicheTagsJSON != "" {
			if err := json.Unmarshal([]byte(nicheTagsJSON), &tp.NicheTags); err != nil {
				return nil, fmt.Errorf("parsing niche tags JSON: %w", err)
			}
		}
		if mediaJSON != "" && mediaJSON != "[]" {
			if err := json.Unmarshal([]byte(mediaJSON), &tp.Media); err != nil {
				return nil, fmt.Errorf("parsing media JSON: %w", err)
			}
		}
		allPosts = append(allPosts, tp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Filter to only include posts from the most recent fetch session
	// (within 5 minutes of the latest fetched_at).
	if len(allPosts) > 0 {
		var latestFetch time.Time
		for _, p := range allPosts {
			if p.FetchedAt.After(latestFetch) {
				latestFetch = p.FetchedAt
			}
		}
		cutoff := latestFetch.Add(-5 * time.Minute)
		var filtered []models.TrendingPost
		for _, p := range allPosts {
			if !p.FetchedAt.Before(cutoff) {
				filtered = append(filtered, p)
			}
		}
		allPosts = filtered
	}

	if limit > 0 && len(allPosts) > limit {
		allPosts = allPosts[:limit]
	}
	return allPosts, nil
}

// GetTrendingPostByID returns a single trending post by ID.
func (db *DB) GetTrendingPostByID(id int64) (*models.TrendingPost, error) {
	var tp models.TrendingPost
	var nicheTagsJSON, mediaJSON string
	err := db.conn.QueryRow(
		"SELECT id, platform, platform_post_id, author_username, author_name, content, likes, reposts, comments, impressions, niche_tags, COALESCE(media_json, '[]'), posted_at, fetched_at, COALESCE(thread_urn, ''), COALESCE(video_url, ''), COALESCE(view_count, 0), COALESCE(duration, 0), COALESCE(is_video, 0) FROM trending_posts WHERE id = ?",
		id,
	).Scan(&tp.ID, &tp.Platform, &tp.PlatformPostID, &tp.AuthorUsername, &tp.AuthorName, &tp.Content, &tp.Likes, &tp.Reposts, &tp.Comments, &tp.Impressions, &nicheTagsJSON, &mediaJSON, &tp.PostedAt, &tp.FetchedAt, &tp.ThreadURN, &tp.VideoURL, &tp.ViewCount, &tp.Duration, &tp.IsVideo)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying trending post: %w", err)
	}
	if nicheTagsJSON != "" {
		if err := json.Unmarshal([]byte(nicheTagsJSON), &tp.NicheTags); err != nil {
			return nil, fmt.Errorf("parsing niche tags JSON: %w", err)
		}
	}
	if mediaJSON != "" && mediaJSON != "[]" {
		if err := json.Unmarshal([]byte(mediaJSON), &tp.Media); err != nil {
			return nil, fmt.Errorf("parsing media JSON: %w", err)
		}
	}
	return &tp, nil
}

// --- generated_content CRUD ---

// InsertGeneratedContent inserts a new generated content record.
func (db *DB) InsertGeneratedContent(userID string, gc *models.GeneratedContent) (int64, error) {
	sourceType := gc.SourceType
	if sourceType == "" {
		sourceType = "trending"
	}
	result, err := db.conn.Exec(
		"INSERT INTO generated_content (user_id, source_trending_id, target_platform, original_content, generated_content, persona_id, prompt_used, status, image_prompt, image_path, is_repost, quote_tweet_id, is_comment, source_type, source_commit_id, code_image_path, code_image_description, video_path, thumbnail_path, video_duration, video_title) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		userID, gc.SourceTrendingID, gc.TargetPlatform, gc.OriginalContent, gc.GeneratedContent, gc.PersonaID, gc.PromptUsed, gc.Status, gc.ImagePrompt, gc.ImagePath, gc.IsRepost, gc.QuoteTweetID, gc.IsComment, sourceType, gc.SourceCommitID, gc.CodeImagePath, gc.CodeImageDescription, gc.VideoPath, gc.ThumbnailPath, gc.VideoDuration, gc.VideoTitle,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting generated content: %w", err)
	}
	return result.LastInsertId()
}

// GetGeneratedContent returns generated content with optional status and platform filters for a user.
func (db *DB) GetGeneratedContent(userID string, status string, platform string, limit int) ([]models.GeneratedContent, error) {
	query := "SELECT id, source_trending_id, target_platform, original_content, generated_content, persona_id, prompt_used, created_at, status, COALESCE(platform_post_ids, ''), posted_at, COALESCE(image_prompt, ''), COALESCE(image_path, ''), COALESCE(is_repost, 0), COALESCE(quote_tweet_id, ''), COALESCE(is_comment, 0), COALESCE(source_type, 'trending'), COALESCE(source_commit_id, 0), COALESCE(code_image_path, ''), COALESCE(code_image_description, ''), COALESCE(video_path, ''), COALESCE(thumbnail_path, ''), COALESCE(video_duration, 0), COALESCE(video_title, '') FROM generated_content"
	var args []interface{}
	var conditions []string

	conditions = append(conditions, "user_id = ?")
	args = append(args, userID)

	if status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	} else {
		// When no status filter is given, exclude archived records (soft-deleted)
		conditions = append(conditions, "status != 'archived'")
	}
	if platform != "" {
		conditions = append(conditions, "target_platform = ?")
		args = append(args, platform)
	}
	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			query += " AND " + c
		}
	}
	query += " ORDER BY created_at DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying generated content: %w", err)
	}
	defer rows.Close()

	var contents []models.GeneratedContent
	for rows.Next() {
		var gc models.GeneratedContent
		if err := rows.Scan(&gc.ID, &gc.SourceTrendingID, &gc.TargetPlatform, &gc.OriginalContent, &gc.GeneratedContent, &gc.PersonaID, &gc.PromptUsed, &gc.CreatedAt, &gc.Status, &gc.PlatformPostIDs, &gc.PostedAt, &gc.ImagePrompt, &gc.ImagePath, &gc.IsRepost, &gc.QuoteTweetID, &gc.IsComment, &gc.SourceType, &gc.SourceCommitID, &gc.CodeImagePath, &gc.CodeImageDescription, &gc.VideoPath, &gc.ThumbnailPath, &gc.VideoDuration, &gc.VideoTitle); err != nil {
			return nil, fmt.Errorf("scanning generated content row: %w", err)
		}
		contents = append(contents, gc)
	}
	return contents, rows.Err()
}

// GetGeneratedContentByID returns a single generated content record by ID, scoped to user.
func (db *DB) GetGeneratedContentByID(userID string, id int64) (*models.GeneratedContent, error) {
	var gc models.GeneratedContent
	err := db.conn.QueryRow(
		"SELECT id, source_trending_id, target_platform, original_content, generated_content, persona_id, prompt_used, created_at, status, COALESCE(platform_post_ids, ''), posted_at, COALESCE(image_prompt, ''), COALESCE(image_path, ''), COALESCE(is_repost, 0), COALESCE(quote_tweet_id, ''), COALESCE(is_comment, 0), COALESCE(source_type, 'trending'), COALESCE(source_commit_id, 0), COALESCE(code_image_path, ''), COALESCE(code_image_description, ''), COALESCE(video_path, ''), COALESCE(thumbnail_path, ''), COALESCE(video_duration, 0), COALESCE(video_title, '') FROM generated_content WHERE id = ? AND user_id = ?",
		id, userID,
	).Scan(&gc.ID, &gc.SourceTrendingID, &gc.TargetPlatform, &gc.OriginalContent, &gc.GeneratedContent, &gc.PersonaID, &gc.PromptUsed, &gc.CreatedAt, &gc.Status, &gc.PlatformPostIDs, &gc.PostedAt, &gc.ImagePrompt, &gc.ImagePath, &gc.IsRepost, &gc.QuoteTweetID, &gc.IsComment, &gc.SourceType, &gc.SourceCommitID, &gc.CodeImagePath, &gc.CodeImageDescription, &gc.VideoPath, &gc.ThumbnailPath, &gc.VideoDuration, &gc.VideoTitle)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying generated content: %w", err)
	}
	return &gc, nil
}

// UpdateGeneratedContentStatus updates the status of a generated content record for a user.
func (db *DB) UpdateGeneratedContentStatus(userID string, id int64, status string) error {
	_, err := db.conn.Exec("UPDATE generated_content SET status = ? WHERE id = ? AND user_id = ?", status, id, userID)
	if err != nil {
		return fmt.Errorf("updating generated content status: %w", err)
	}
	return nil
}

// UpdateGeneratedContentText updates the generated_content text of a record for a user.
func (db *DB) UpdateGeneratedContentText(userID string, id int64, content string) error {
	_, err := db.conn.Exec("UPDATE generated_content SET generated_content = ? WHERE id = ? AND user_id = ?", content, id, userID)
	if err != nil {
		return fmt.Errorf("updating generated content text: %w", err)
	}
	return nil
}

// UpdateGeneratedContentIsRepost updates the is_repost flag of a generated content record for a user.
func (db *DB) UpdateGeneratedContentIsRepost(userID string, id int64, isRepost bool) error {
	val := 0
	if isRepost {
		val = 1
	}
	_, err := db.conn.Exec("UPDATE generated_content SET is_repost = ? WHERE id = ? AND user_id = ?", val, id, userID)
	if err != nil {
		return fmt.Errorf("updating generated content is_repost: %w", err)
	}
	return nil
}

// UpdateGeneratedContentQuoteTweetID updates the quote_tweet_id of a generated content record for a user.
func (db *DB) UpdateGeneratedContentQuoteTweetID(userID string, id int64, quoteTweetID string) error {
	_, err := db.conn.Exec("UPDATE generated_content SET quote_tweet_id = ? WHERE id = ? AND user_id = ?", quoteTweetID, id, userID)
	if err != nil {
		return fmt.Errorf("updating generated content quote_tweet_id: %w", err)
	}
	return nil
}

// UpdateCodeImageDescription updates the code image description of a generated content record for a user.
func (db *DB) UpdateCodeImageDescription(userID string, id int64, description string) error {
	_, err := db.conn.Exec("UPDATE generated_content SET code_image_description = ? WHERE id = ? AND user_id = ?", description, id, userID)
	if err != nil {
		return fmt.Errorf("updating code image description: %w", err)
	}
	return nil
}

// DeleteGeneratedContent removes a generated content record by ID for a user.
func (db *DB) DeleteGeneratedContent(userID string, id int64) error {
	_, err := db.conn.Exec("DELETE FROM generated_content WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return fmt.Errorf("deleting generated content: %w", err)
	}
	return nil
}

// UpdateGeneratedContentPosted marks content as posted and stores the platform post IDs for a user.
func (db *DB) UpdateGeneratedContentPosted(userID string, id int64, platformPostIDs string) error {
	_, err := db.conn.Exec(
		"UPDATE generated_content SET status = 'posted', platform_post_ids = ?, posted_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?",
		platformPostIDs, id, userID,
	)
	if err != nil {
		return fmt.Errorf("updating generated content as posted: %w", err)
	}
	return nil
}

// --- scheduled_posts CRUD ---

// InsertScheduledPost schedules a generated content item for future posting.
func (db *DB) InsertScheduledPost(userID string, contentID int64, scheduledAt time.Time) (int64, error) {
	result, err := db.conn.Exec(
		"INSERT INTO scheduled_posts (user_id, generated_content_id, scheduled_at) VALUES (?, ?, ?)",
		userID, contentID, scheduledAt,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting scheduled post: %w", err)
	}
	return result.LastInsertId()
}

// GetPendingScheduledPosts returns scheduled posts that are due and still pending (all users, for daemon).
func (db *DB) GetPendingScheduledPosts() ([]models.ScheduledPost, error) {
	rows, err := db.conn.Query(
		"SELECT id, generated_content_id, scheduled_at, status, COALESCE(error_message, ''), created_at, COALESCE(user_id, '') FROM scheduled_posts WHERE status = 'pending' AND scheduled_at <= CURRENT_TIMESTAMP ORDER BY scheduled_at ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("querying pending scheduled posts: %w", err)
	}
	defer rows.Close()
	return scanScheduledPosts(rows)
}

// GetScheduledPosts returns scheduled posts with optional status filter and limit for a user.
func (db *DB) GetScheduledPosts(userID string, status string, limit int) ([]models.ScheduledPost, error) {
	query := "SELECT id, generated_content_id, scheduled_at, status, COALESCE(error_message, ''), created_at, COALESCE(user_id, '') FROM scheduled_posts"
	var args []interface{}
	var conditions []string

	conditions = append(conditions, "user_id = ?")
	args = append(args, userID)

	if status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}
	query += " WHERE " + conditions[0]
	for _, c := range conditions[1:] {
		query += " AND " + c
	}
	query += " ORDER BY scheduled_at ASC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying scheduled posts: %w", err)
	}
	defer rows.Close()
	return scanScheduledPosts(rows)
}

// UpdateScheduledPostStatus updates the status and optional error message of a scheduled post.
func (db *DB) UpdateScheduledPostStatus(id int64, status string, errMsg string) error {
	_, err := db.conn.Exec(
		"UPDATE scheduled_posts SET status = ?, error_message = ? WHERE id = ?",
		status, errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("updating scheduled post status: %w", err)
	}
	return nil
}

// DeleteScheduledPost deletes a scheduled post by ID for a user.
func (db *DB) DeleteScheduledPost(userID string, id int64) error {
	result, err := db.conn.Exec("DELETE FROM scheduled_posts WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return fmt.Errorf("deleting scheduled post: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("scheduled post %d not found", id)
	}
	return nil
}

func scanScheduledPosts(rows *sql.Rows) ([]models.ScheduledPost, error) {
	var posts []models.ScheduledPost
	for rows.Next() {
		var sp models.ScheduledPost
		if err := rows.Scan(&sp.ID, &sp.GeneratedContentID, &sp.ScheduledAt, &sp.Status, &sp.ErrorMessage, &sp.CreatedAt, &sp.UserID); err != nil {
			return nil, fmt.Errorf("scanning scheduled post row: %w", err)
		}
		posts = append(posts, sp)
	}
	return posts, rows.Err()
}
