package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/shuhao/goviral/pkg/models"
)

// InsertDaemonBatch inserts a new daemon batch and returns its ID.
func (db *DB) InsertDaemonBatch(b *models.DaemonBatch) (int64, error) {
	result, err := db.conn.Exec(
		`INSERT INTO daemon_batches (platform, status, content_ids, trending_ids, telegram_message_id, approval_source, reply_text, parsed_intent, error_message)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.Platform, b.Status, b.ContentIDs, b.TrendingIDs, b.TelegramMessageID,
		b.ApprovalSource, b.ReplyText, b.ParsedIntent, b.ErrorMessage,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting daemon batch: %w", err)
	}
	return result.LastInsertId()
}

// GetDaemonBatch returns a daemon batch by ID.
func (db *DB) GetDaemonBatch(id int64) (*models.DaemonBatch, error) {
	var b models.DaemonBatch
	err := db.conn.QueryRow(
		`SELECT id, platform, status, content_ids, trending_ids, telegram_message_id, approval_source, reply_text, parsed_intent, error_message, created_at, updated_at, notified_at, resolved_at
		 FROM daemon_batches WHERE id = ?`, id,
	).Scan(&b.ID, &b.Platform, &b.Status, &b.ContentIDs, &b.TrendingIDs, &b.TelegramMessageID,
		&b.ApprovalSource, &b.ReplyText, &b.ParsedIntent, &b.ErrorMessage,
		&b.CreatedAt, &b.UpdatedAt, &b.NotifiedAt, &b.ResolvedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting daemon batch %d: %w", id, err)
	}
	return &b, nil
}

// GetDaemonBatches returns daemon batches with optional platform and status filters.
func (db *DB) GetDaemonBatches(platform, status string, limit int) ([]models.DaemonBatch, error) {
	query := `SELECT id, platform, status, content_ids, trending_ids, telegram_message_id, approval_source, reply_text, parsed_intent, error_message, created_at, updated_at, notified_at, resolved_at
		FROM daemon_batches WHERE 1=1`
	var args []interface{}

	if platform != "" {
		query += " AND platform = ?"
		args = append(args, platform)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying daemon batches: %w", err)
	}
	defer rows.Close()
	return scanDaemonBatches(rows)
}

// UpdateDaemonBatchStatus updates the status and related fields of a daemon batch.
func (db *DB) UpdateDaemonBatchStatus(id int64, status string, extras map[string]interface{}) error {
	setClauses := "status = ?, updated_at = CURRENT_TIMESTAMP"
	args := []interface{}{status}

	if v, ok := extras["telegram_message_id"]; ok {
		setClauses += ", telegram_message_id = ?"
		args = append(args, v)
	}
	if v, ok := extras["approval_source"]; ok {
		setClauses += ", approval_source = ?"
		args = append(args, v)
	}
	if v, ok := extras["reply_text"]; ok {
		setClauses += ", reply_text = ?"
		args = append(args, v)
	}
	if v, ok := extras["parsed_intent"]; ok {
		setClauses += ", parsed_intent = ?"
		args = append(args, v)
	}
	if v, ok := extras["error_message"]; ok {
		setClauses += ", error_message = ?"
		args = append(args, v)
	}
	if v, ok := extras["notified_at"]; ok {
		setClauses += ", notified_at = ?"
		args = append(args, v)
	}
	if v, ok := extras["resolved_at"]; ok {
		setClauses += ", resolved_at = ?"
		args = append(args, v)
	}

	args = append(args, id)
	_, err := db.conn.Exec(
		fmt.Sprintf("UPDATE daemon_batches SET %s WHERE id = ?", setClauses),
		args...,
	)
	if err != nil {
		return fmt.Errorf("updating daemon batch %d status: %w", id, err)
	}
	return nil
}

// GetDaemonBatchByTelegramMsgID returns a daemon batch by its Telegram message ID.
func (db *DB) GetDaemonBatchByTelegramMsgID(msgID int64) (*models.DaemonBatch, error) {
	var b models.DaemonBatch
	err := db.conn.QueryRow(
		`SELECT id, platform, status, content_ids, trending_ids, telegram_message_id, approval_source, reply_text, parsed_intent, error_message, created_at, updated_at, notified_at, resolved_at
		 FROM daemon_batches WHERE telegram_message_id = ?`, msgID,
	).Scan(&b.ID, &b.Platform, &b.Status, &b.ContentIDs, &b.TrendingIDs, &b.TelegramMessageID,
		&b.ApprovalSource, &b.ReplyText, &b.ParsedIntent, &b.ErrorMessage,
		&b.CreatedAt, &b.UpdatedAt, &b.NotifiedAt, &b.ResolvedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting daemon batch by telegram msg %d: %w", msgID, err)
	}
	return &b, nil
}

// GetLatestDaemonBatch returns the most recent daemon batch for a platform.
func (db *DB) GetLatestDaemonBatch(platform string) (*models.DaemonBatch, error) {
	var b models.DaemonBatch
	err := db.conn.QueryRow(
		`SELECT id, platform, status, content_ids, trending_ids, telegram_message_id, approval_source, reply_text, parsed_intent, error_message, created_at, updated_at, notified_at, resolved_at
		 FROM daemon_batches WHERE platform = ? ORDER BY created_at DESC LIMIT 1`, platform,
	).Scan(&b.ID, &b.Platform, &b.Status, &b.ContentIDs, &b.TrendingIDs, &b.TelegramMessageID,
		&b.ApprovalSource, &b.ReplyText, &b.ParsedIntent, &b.ErrorMessage,
		&b.CreatedAt, &b.UpdatedAt, &b.NotifiedAt, &b.ResolvedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting latest daemon batch for %s: %w", platform, err)
	}
	return &b, nil
}

// GetGeneratedContentByBatchID returns generated content linked to a daemon batch.
func (db *DB) GetGeneratedContentByBatchID(batchID int64) ([]models.GeneratedContent, error) {
	rows, err := db.conn.Query(
		`SELECT id, source_trending_id, target_platform, original_content, generated_content, persona_id, prompt_used, created_at, status, COALESCE(platform_post_ids, ''), posted_at, COALESCE(image_prompt, ''), COALESCE(image_path, ''), COALESCE(is_repost, 0), COALESCE(quote_tweet_id, '')
		 FROM generated_content WHERE daemon_batch_id = ? ORDER BY id ASC`, batchID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying content for batch %d: %w", batchID, err)
	}
	defer rows.Close()

	var contents []models.GeneratedContent
	for rows.Next() {
		var gc models.GeneratedContent
		if err := rows.Scan(&gc.ID, &gc.SourceTrendingID, &gc.TargetPlatform, &gc.OriginalContent, &gc.GeneratedContent, &gc.PersonaID, &gc.PromptUsed, &gc.CreatedAt, &gc.Status, &gc.PlatformPostIDs, &gc.PostedAt, &gc.ImagePrompt, &gc.ImagePath, &gc.IsRepost, &gc.QuoteTweetID); err != nil {
			return nil, fmt.Errorf("scanning generated content row: %w", err)
		}
		contents = append(contents, gc)
	}
	return contents, rows.Err()
}

func scanDaemonBatches(rows *sql.Rows) ([]models.DaemonBatch, error) {
	var batches []models.DaemonBatch
	for rows.Next() {
		var b models.DaemonBatch
		if err := rows.Scan(&b.ID, &b.Platform, &b.Status, &b.ContentIDs, &b.TrendingIDs, &b.TelegramMessageID,
			&b.ApprovalSource, &b.ReplyText, &b.ParsedIntent, &b.ErrorMessage,
			&b.CreatedAt, &b.UpdatedAt, &b.NotifiedAt, &b.ResolvedAt); err != nil {
			return nil, fmt.Errorf("scanning daemon batch row: %w", err)
		}
		batches = append(batches, b)
	}
	return batches, rows.Err()
}

// SetGeneratedContentBatchID sets the daemon_batch_id on a generated content record.
func (db *DB) SetGeneratedContentBatchID(contentID, batchID int64) error {
	_, err := db.conn.Exec("UPDATE generated_content SET daemon_batch_id = ? WHERE id = ?", batchID, contentID)
	if err != nil {
		return fmt.Errorf("setting batch ID on content %d: %w", contentID, err)
	}
	return nil
}

// GetPendingDaemonBatches returns batches in notified/awaiting_reply status older than the given duration.
func (db *DB) GetPendingDaemonBatches(olderThan time.Duration) ([]models.DaemonBatch, error) {
	cutoff := time.Now().Add(-olderThan)
	rows, err := db.conn.Query(
		`SELECT id, platform, status, content_ids, trending_ids, telegram_message_id, approval_source, reply_text, parsed_intent, error_message, created_at, updated_at, notified_at, resolved_at
		 FROM daemon_batches WHERE status IN ('notified', 'awaiting_reply') AND created_at < ? ORDER BY created_at ASC`, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("querying pending daemon batches: %w", err)
	}
	defer rows.Close()
	return scanDaemonBatches(rows)
}
