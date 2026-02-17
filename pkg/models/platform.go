package models

import "context"

// PlatformClient defines the interface for interacting with social media platforms.
type PlatformClient interface {
	FetchMyPosts(ctx context.Context, limit int) ([]Post, error)
	FetchTrendingPosts(ctx context.Context, niches []string, period string, minLikes int, limit int) ([]TrendingPost, error)
}

// PlatformPoster defines the interface for posting content to social media platforms.
type PlatformPoster interface {
	PostTweet(ctx context.Context, text string) (string, error)
	PostReply(ctx context.Context, text string, inReplyToID string) (string, error)
}

// MediaPoster extends PlatformPoster with media upload and media-aware posting.
type MediaPoster interface {
	UploadMedia(ctx context.Context, imageData []byte, mimeType string) (string, error)
	PostTweetWithMedia(ctx context.Context, text string, mediaIDs []string) (string, error)
	PostReplyWithMedia(ctx context.Context, text string, inReplyToID string, mediaIDs []string) (string, error)
}
