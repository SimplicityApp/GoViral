package models

import "time"

// Post represents a user's post from X or LinkedIn.
type Post struct {
	ID             int64
	Platform       string // "x" or "linkedin"
	PlatformPostID string
	Content        string
	Likes          int
	Reposts        int
	Comments       int
	Impressions    int
	PostedAt       time.Time
	FetchedAt      time.Time
}
