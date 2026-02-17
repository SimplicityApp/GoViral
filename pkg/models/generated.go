package models

import "time"

// GeneratedContent represents AI-generated content based on a trending post.
type GeneratedContent struct {
	ID               int64
	SourceTrendingID int64
	TargetPlatform   string
	OriginalContent  string
	GeneratedContent string
	PersonaID        int64
	PromptUsed       string
	CreatedAt        time.Time
	Status           string // "draft", "approved", "posted"
	PlatformPostIDs  string
	PostedAt         *time.Time
	ImagePrompt      string
	ImagePath        string
}

// ScheduledPost represents a post scheduled for future publishing.
type ScheduledPost struct {
	ID                 int64
	GeneratedContentID int64
	ScheduledAt        time.Time
	Status             string // "pending", "posted", "failed"
	ErrorMessage       string
	CreatedAt          time.Time
}
