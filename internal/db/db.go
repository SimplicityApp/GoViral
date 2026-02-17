package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
		platform_post_id TEXT UNIQUE NOT NULL,
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

	return nil
}

// --- my_posts CRUD ---

// UpsertPost inserts or updates a post by platform_post_id.
func (db *DB) UpsertPost(p *models.Post) error {
	query := `
	INSERT INTO my_posts (platform, platform_post_id, content, likes, reposts, comments, impressions, posted_at, fetched_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(platform_post_id) DO UPDATE SET
		content=excluded.content, likes=excluded.likes, reposts=excluded.reposts,
		comments=excluded.comments, impressions=excluded.impressions, fetched_at=excluded.fetched_at
	`
	_, err := db.conn.Exec(query, p.Platform, p.PlatformPostID, p.Content, p.Likes, p.Reposts, p.Comments, p.Impressions, p.PostedAt, time.Now())
	if err != nil {
		return fmt.Errorf("upserting post: %w", err)
	}
	return nil
}

// GetPostsByPlatform returns all posts for a given platform.
func (db *DB) GetPostsByPlatform(platform string) ([]models.Post, error) {
	rows, err := db.conn.Query(
		"SELECT id, platform, platform_post_id, content, likes, reposts, comments, impressions, posted_at, fetched_at FROM my_posts WHERE platform = ? ORDER BY posted_at DESC",
		platform,
	)
	if err != nil {
		return nil, fmt.Errorf("querying posts: %w", err)
	}
	defer rows.Close()
	return scanPosts(rows)
}

// GetAllPosts returns all posts across platforms.
func (db *DB) GetAllPosts() ([]models.Post, error) {
	rows, err := db.conn.Query(
		"SELECT id, platform, platform_post_id, content, likes, reposts, comments, impressions, posted_at, fetched_at FROM my_posts ORDER BY posted_at DESC",
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

// UpsertPersona inserts or updates a persona profile for a platform.
func (db *DB) UpsertPersona(p *models.Persona) error {
	profileJSON, err := json.Marshal(p.Profile)
	if err != nil {
		return fmt.Errorf("marshaling persona profile: %w", err)
	}

	// Check if persona exists for this platform
	var existingID int64
	err = db.conn.QueryRow("SELECT id FROM persona WHERE platform = ?", p.Platform).Scan(&existingID)
	if err == sql.ErrNoRows {
		_, err = db.conn.Exec(
			"INSERT INTO persona (platform, profile_json) VALUES (?, ?)",
			p.Platform, string(profileJSON),
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

// GetPersona returns the persona for a given platform.
func (db *DB) GetPersona(platform string) (*models.Persona, error) {
	var p models.Persona
	var profileJSON string
	err := db.conn.QueryRow(
		"SELECT id, platform, profile_json, created_at, updated_at FROM persona WHERE platform = ?",
		platform,
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

	query := `
	INSERT INTO trending_posts (platform, platform_post_id, author_username, author_name, content, likes, reposts, comments, impressions, niche_tags, media_json, posted_at, fetched_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(platform_post_id) DO UPDATE SET
		content=excluded.content, likes=excluded.likes, reposts=excluded.reposts,
		comments=excluded.comments, impressions=excluded.impressions, niche_tags=excluded.niche_tags, media_json=excluded.media_json, fetched_at=excluded.fetched_at
	`
	_, err = db.conn.Exec(query, tp.Platform, tp.PlatformPostID, tp.AuthorUsername, tp.AuthorName, tp.Content, tp.Likes, tp.Reposts, tp.Comments, tp.Impressions, string(nicheTags), string(mediaJSON), tp.PostedAt, time.Now())
	if err != nil {
		return fmt.Errorf("upserting trending post: %w", err)
	}
	return nil
}

// GetTrendingPosts returns trending posts from the most recent fetch session,
// with optional platform and limit filters. Only posts fetched within 5 minutes
// of the latest fetched_at are returned, so stale results from previous runs
// are excluded.
func (db *DB) GetTrendingPosts(platform string, limit int) ([]models.TrendingPost, error) {
	query := "SELECT id, platform, platform_post_id, author_username, author_name, content, likes, reposts, comments, impressions, niche_tags, COALESCE(media_json, '[]'), posted_at, fetched_at FROM trending_posts"
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
		if err := rows.Scan(&tp.ID, &tp.Platform, &tp.PlatformPostID, &tp.AuthorUsername, &tp.AuthorName, &tp.Content, &tp.Likes, &tp.Reposts, &tp.Comments, &tp.Impressions, &nicheTagsJSON, &mediaJSON, &tp.PostedAt, &tp.FetchedAt); err != nil {
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
		"SELECT id, platform, platform_post_id, author_username, author_name, content, likes, reposts, comments, impressions, niche_tags, COALESCE(media_json, '[]'), posted_at, fetched_at FROM trending_posts WHERE id = ?",
		id,
	).Scan(&tp.ID, &tp.Platform, &tp.PlatformPostID, &tp.AuthorUsername, &tp.AuthorName, &tp.Content, &tp.Likes, &tp.Reposts, &tp.Comments, &tp.Impressions, &nicheTagsJSON, &mediaJSON, &tp.PostedAt, &tp.FetchedAt)
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
func (db *DB) InsertGeneratedContent(gc *models.GeneratedContent) (int64, error) {
	result, err := db.conn.Exec(
		"INSERT INTO generated_content (source_trending_id, target_platform, original_content, generated_content, persona_id, prompt_used, status, image_prompt, image_path) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		gc.SourceTrendingID, gc.TargetPlatform, gc.OriginalContent, gc.GeneratedContent, gc.PersonaID, gc.PromptUsed, gc.Status, gc.ImagePrompt, gc.ImagePath,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting generated content: %w", err)
	}
	return result.LastInsertId()
}

// GetGeneratedContent returns generated content with optional status filter.
func (db *DB) GetGeneratedContent(status string, limit int) ([]models.GeneratedContent, error) {
	query := "SELECT id, source_trending_id, target_platform, original_content, generated_content, persona_id, prompt_used, created_at, status, COALESCE(platform_post_ids, ''), posted_at, COALESCE(image_prompt, ''), COALESCE(image_path, '') FROM generated_content"
	var args []interface{}

	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
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
		if err := rows.Scan(&gc.ID, &gc.SourceTrendingID, &gc.TargetPlatform, &gc.OriginalContent, &gc.GeneratedContent, &gc.PersonaID, &gc.PromptUsed, &gc.CreatedAt, &gc.Status, &gc.PlatformPostIDs, &gc.PostedAt, &gc.ImagePrompt, &gc.ImagePath); err != nil {
			return nil, fmt.Errorf("scanning generated content row: %w", err)
		}
		contents = append(contents, gc)
	}
	return contents, rows.Err()
}

// GetGeneratedContentByID returns a single generated content record by ID.
func (db *DB) GetGeneratedContentByID(id int64) (*models.GeneratedContent, error) {
	var gc models.GeneratedContent
	err := db.conn.QueryRow(
		"SELECT id, source_trending_id, target_platform, original_content, generated_content, persona_id, prompt_used, created_at, status, COALESCE(platform_post_ids, ''), posted_at, COALESCE(image_prompt, ''), COALESCE(image_path, '') FROM generated_content WHERE id = ?",
		id,
	).Scan(&gc.ID, &gc.SourceTrendingID, &gc.TargetPlatform, &gc.OriginalContent, &gc.GeneratedContent, &gc.PersonaID, &gc.PromptUsed, &gc.CreatedAt, &gc.Status, &gc.PlatformPostIDs, &gc.PostedAt, &gc.ImagePrompt, &gc.ImagePath)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying generated content: %w", err)
	}
	return &gc, nil
}

// UpdateGeneratedContentStatus updates the status of a generated content record.
func (db *DB) UpdateGeneratedContentStatus(id int64, status string) error {
	_, err := db.conn.Exec("UPDATE generated_content SET status = ? WHERE id = ?", status, id)
	if err != nil {
		return fmt.Errorf("updating generated content status: %w", err)
	}
	return nil
}

// UpdateGeneratedContentPosted marks content as posted and stores the tweet IDs.
func (db *DB) UpdateGeneratedContentPosted(id int64, platformPostIDs string) error {
	_, err := db.conn.Exec(
		"UPDATE generated_content SET status = 'posted', platform_post_ids = ?, posted_at = CURRENT_TIMESTAMP WHERE id = ?",
		platformPostIDs, id,
	)
	if err != nil {
		return fmt.Errorf("updating generated content as posted: %w", err)
	}
	return nil
}

// --- scheduled_posts CRUD ---

// InsertScheduledPost schedules a generated content item for future posting.
func (db *DB) InsertScheduledPost(contentID int64, scheduledAt time.Time) (int64, error) {
	result, err := db.conn.Exec(
		"INSERT INTO scheduled_posts (generated_content_id, scheduled_at) VALUES (?, ?)",
		contentID, scheduledAt,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting scheduled post: %w", err)
	}
	return result.LastInsertId()
}

// GetPendingScheduledPosts returns scheduled posts that are due and still pending.
func (db *DB) GetPendingScheduledPosts() ([]models.ScheduledPost, error) {
	rows, err := db.conn.Query(
		"SELECT id, generated_content_id, scheduled_at, status, COALESCE(error_message, ''), created_at FROM scheduled_posts WHERE status = 'pending' AND scheduled_at <= CURRENT_TIMESTAMP ORDER BY scheduled_at ASC",
	)
	if err != nil {
		return nil, fmt.Errorf("querying pending scheduled posts: %w", err)
	}
	defer rows.Close()
	return scanScheduledPosts(rows)
}

// GetScheduledPosts returns scheduled posts with optional status filter and limit.
func (db *DB) GetScheduledPosts(status string, limit int) ([]models.ScheduledPost, error) {
	query := "SELECT id, generated_content_id, scheduled_at, status, COALESCE(error_message, ''), created_at FROM scheduled_posts"
	var args []interface{}

	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
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

func scanScheduledPosts(rows *sql.Rows) ([]models.ScheduledPost, error) {
	var posts []models.ScheduledPost
	for rows.Next() {
		var sp models.ScheduledPost
		if err := rows.Scan(&sp.ID, &sp.GeneratedContentID, &sp.ScheduledAt, &sp.Status, &sp.ErrorMessage, &sp.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning scheduled post row: %w", err)
		}
		posts = append(posts, sp)
	}
	return posts, rows.Err()
}
