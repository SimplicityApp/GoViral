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
	UserID         string
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
	// ThreadURN is the LinkedIn ugcPost URN (urn:li:ugcPost:N) required by the
	// comment API's threadUrn field. Only set for LinkedIn feed posts; empty for
	// posts fetched via search or trending discovery.
	ThreadURN string
	VideoURL  string // URL to the original video
	ViewCount int    // video view count
	Duration  int    // video duration in seconds
	IsVideo   bool   // distinguishes video from text posts
}
