package models

import "time"

// MediaAttachment represents a media item attached to a post.
type MediaAttachment struct {
	Type       string // "photo", "video", "animated_gif"
	URL        string
	PreviewURL string
	AltText    string
}

// TrendingPost represents a trending post discovered from a platform.
type TrendingPost struct {
	ID             int64
	Platform       string
	PlatformPostID string
	AuthorUsername string
	AuthorName     string
	Content        string
	Likes          int
	Reposts        int
	Comments       int
	Impressions    int
	NicheTags      []string
	Media          []MediaAttachment
	PostedAt       time.Time
	FetchedAt      time.Time
}
