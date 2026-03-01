package models

import (
	"context"
	"time"
)

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

// PlatformScheduler defines the interface for scheduling content on a platform natively.
type PlatformScheduler interface {
	ScheduleTweet(ctx context.Context, text string, scheduledAtUnix int64) (string, error)
}

// QuotePoster defines the interface for posting quote tweets (reposts with commentary).
type QuotePoster interface {
	PostQuoteTweet(ctx context.Context, text string, quoteTweetID string) (string, error)
}

// QuoteScheduler defines the interface for natively scheduling a quote tweet.
type QuoteScheduler interface {
	ScheduleQuoteTweet(ctx context.Context, text string, quoteTweetID string, scheduledAtUnix int64) (string, error)
}

// LinkedInPoster defines the interface for posting content to LinkedIn.
type LinkedInPoster interface {
	CreatePost(ctx context.Context, text string) (string, error)
	UploadImage(ctx context.Context, imageData []byte, filename string) (string, error)
	CreatePostWithImage(ctx context.Context, text string, imageData []byte, filename string) (string, error)
	CreateScheduledPost(ctx context.Context, text string, scheduledAt time.Time) (string, error)
	CreateScheduledPostWithImage(ctx context.Context, text string, imageData []byte, filename string, scheduledAt time.Time) (string, error)
}

// LinkedInReposter defines the interface for reposting LinkedIn content.
type LinkedInReposter interface {
	Repost(ctx context.Context, postURN string, text string) (string, error)
}

// LinkedInCommenter defines the interface for commenting on LinkedIn posts.
type LinkedInCommenter interface {
	// CreateComment posts a comment on a LinkedIn post.
	// threadURN is the optional urn:li:ugcPost:N URN for the LinkedIn comment API's
	// threadUrn field; pass "" when unknown and linkitin will derive it from postURN.
	CreateComment(ctx context.Context, postURN string, threadURN string, text string) (string, error)
}

// YouTubePoster uploads short videos to YouTube (auto-detected as Shorts).
type YouTubePoster interface {
	UploadVideo(ctx context.Context, videoPath string, title string, description string, tags []string) (string, error)
	UploadVideoWithThumbnail(ctx context.Context, videoPath string, thumbnailPath string, title string, description string, tags []string) (string, error)
}

// TikTokPoster uploads videos to TikTok.
type TikTokPoster interface {
	UploadVideo(ctx context.Context, videoPath string, description string, tags []string) (string, error)
	ScheduleVideo(ctx context.Context, videoPath string, description string, tags []string, scheduledAt time.Time) (string, error)
}

// VideoFetcher fetches trending video content for discovery.
type VideoFetcher interface {
	FetchTrendingVideos(ctx context.Context, niches []string, limit int) ([]TrendingPost, error)
}
