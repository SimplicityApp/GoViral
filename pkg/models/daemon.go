package models

import "time"

// Daemon batch status constants.
const (
	BatchStatusPending       = "pending"
	BatchStatusNotified      = "notified"
	BatchStatusAwaitingReply = "awaiting_reply"
	BatchStatusApproved      = "approved"
	BatchStatusRejected      = "rejected"
	BatchStatusPosted        = "posted"
	BatchStatusScheduled     = "scheduled"
	BatchStatusArchived      = "archived"
	BatchStatusFailed        = "failed"
)

// DaemonBatch represents a batch of generated content from the autopilot daemon.
type DaemonBatch struct {
	ID                int64     `json:"id"`
	Platform          string    `json:"platform"`
	Status            string    `json:"status"`
	ContentIDs        string    `json:"content_ids"`         // JSON array of content IDs
	TrendingIDs       string    `json:"trending_ids"`        // JSON array of trending post IDs
	TelegramMessageID int64     `json:"telegram_message_id"`
	ApprovalSource    string    `json:"approval_source"`     // "telegram" or "web"
	ReplyText         string    `json:"reply_text"`
	ParsedIntent      string    `json:"parsed_intent"`       // JSON of DaemonIntent
	ErrorMessage      string    `json:"error_message"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	NotifiedAt        *time.Time `json:"notified_at,omitempty"`
	ResolvedAt        *time.Time `json:"resolved_at,omitempty"`
}

// DaemonIntent represents a parsed user intent from a Telegram reply.
type DaemonIntent struct {
	Action     string            `json:"action"`      // "approve", "reject", "edit", "schedule"
	ContentIDs []int64           `json:"content_ids,omitempty"`
	Edits      map[int64]string  `json:"edits,omitempty"`
	ScheduleAt *time.Time        `json:"schedule_at,omitempty"`
	Message    string            `json:"message"`
}
