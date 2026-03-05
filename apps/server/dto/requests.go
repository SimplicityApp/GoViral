package dto

type IngestPostsRequest struct {
	Platform string       `json:"platform"`
	Posts    []IngestPost `json:"posts"`
}

type IngestPost struct {
	PlatformPostID string `json:"platform_post_id"`
	Content        string `json:"content"`
	Likes          int    `json:"likes"`
	Reposts        int    `json:"reposts"`
	Comments       int    `json:"comments"`
	Impressions    int    `json:"impressions"`
	PostedAt       string `json:"posted_at"`
}

type IngestTrendingRequest struct {
	Platform string               `json:"platform"`
	Posts    []IngestTrendingPost `json:"posts"`
}

type IngestTrendingPost struct {
	PlatformPostID string   `json:"platform_post_id"`
	AuthorUsername string   `json:"author_username"`
	AuthorName     string   `json:"author_name"`
	Content        string   `json:"content"`
	Likes          int      `json:"likes"`
	Reposts        int      `json:"reposts"`
	Comments       int      `json:"comments"`
	Impressions    int      `json:"impressions"`
	NicheTags      []string `json:"niche_tags"`
	PostedAt       string   `json:"posted_at"`
}

type IngestResponse struct {
	Count int `json:"count"`
}

type FetchPostsRequest struct {
	Platform string `json:"platform"` // "x", "linkedin", or "" for both
}

type DiscoverTrendingRequest struct {
	Platform string `json:"platform"`
	Period   string `json:"period"`   // "day", "week", "month"
	MinLikes int    `json:"min_likes"`
	Limit    int    `json:"limit"`
}

type BuildPersonaRequest struct {
	Platform string `json:"platform"`
}

type GenerateRequest struct {
	TrendingPostIDs []int64 `json:"trending_post_ids"`
	TargetPlatform  string  `json:"target_platform"`
	Count           int     `json:"count"`
	MaxChars        int     `json:"max_chars"`
	ForceImage      bool    `json:"force_image"`
	IsRepost        bool    `json:"is_repost"`
}

type PublishRequest struct {
	ContentID int64 `json:"content_id"`
	Numbered  bool  `json:"numbered"`
}

type ScheduleRequest struct {
	ContentID   int64  `json:"content_id"`
	ScheduledAt string `json:"scheduled_at"` // RFC3339
}

type UpdateStatusRequest struct {
	Status               string  `json:"status,omitempty"`                // "draft", "approved", "posted"
	GeneratedContent     *string `json:"generated_content,omitempty"`     // optional content text update
	CodeImageDescription *string `json:"code_image_description,omitempty"` // optional code image description update
}

type UpdateConfigRequest struct {
	Claude         *ClaudeConfigUpdate   `json:"claude,omitempty"`
	Gemini         *GeminiConfigUpdate   `json:"gemini,omitempty"`
	X              *XConfigUpdate        `json:"x,omitempty"`
	LinkedIn       *LinkedInConfigUpdate `json:"linkedin,omitempty"`
	YouTube        *YouTubeConfigUpdate  `json:"youtube,omitempty"`
	TikTok         *TikTokConfigUpdate   `json:"tiktok,omitempty"`
	Niches         *[]string             `json:"niches,omitempty"`
	LinkedInNiches *[]string             `json:"linkedin_niches,omitempty"`
}

type ClaudeConfigUpdate struct {
	APIKey *string `json:"api_key,omitempty"`
	Model  *string `json:"model,omitempty"`
}

type GeminiConfigUpdate struct {
	APIKey *string `json:"api_key,omitempty"`
	Model  *string `json:"model,omitempty"`
}

type XConfigUpdate struct {
	Username *string `json:"username"`
}

type LinkedInConfigUpdate struct {
	PersonURN *string `json:"person_urn"`
}

type YouTubeConfigUpdate struct {
	ChannelID *string `json:"channel_id"`
}

type TikTokConfigUpdate struct {
	Username *string `json:"username"`
}

// --- Daemon requests ---

type BatchActionRequest struct {
	Action     string           `json:"action"`      // "approve", "reject", "edit"
	ContentIDs []int64          `json:"content_ids,omitempty"`
	Edits      map[int64]string `json:"edits,omitempty"`
	ScheduleAt string           `json:"schedule_at,omitempty"` // RFC3339
}

type GenerateCommentRequest struct {
	TrendingPostID int64  `json:"trending_post_id"`
	Platform       string `json:"platform"`
	Count          int    `json:"count"`
}

type PostCommentRequest struct {
	ContentID int64 `json:"content_id"`
}

// --- Repo-to-post requests ---

type RepoLinkDTO struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type UpdateRepoSettingsRequest struct {
	TargetAudience string        `json:"target_audience"`
	Links          []RepoLinkDTO `json:"links"`
}

type AddRepoRequest struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
}

type FetchCommitsRequest struct {
	Limit int    `json:"limit"`
	Since string `json:"since"` // RFC3339, optional
}

type GenerateRepoPostRequest struct {
	CommitIDs         []int64 `json:"commit_ids"`
	TargetPlatform    string  `json:"target_platform,omitempty"`
	Platform          string  `json:"platform,omitempty"`
	Count             int     `json:"count"`
	IncludeCodeImage  bool    `json:"include_code_image"`
	IncludeCodeImages bool    `json:"include_code_images"`
	StyleDirection    string  `json:"style_direction"`
	CodeImageTemplate string  `json:"code_image_template"` // e.g. "github", "macos", "vscode"
	CodeImageTheme    string  `json:"code_image_theme"`    // e.g. "github-dark", "dracula", "nord"
}

type RenderCodeImageRequest struct {
	CommitID int64  `json:"commit_id"`
	Template string `json:"template"` // e.g. "github", "macos", "vscode"
	Theme    string `json:"theme"`    // e.g. "github-dark", "dracula", "nord"
}

type VideoUploadRequest struct {
	ContentID     int64  `json:"content_id"`
	VideoPath     string `json:"video_path"`
	ThumbnailPath string `json:"thumbnail_path,omitempty"`
	Title         string `json:"title,omitempty"`
	Description   string `json:"description"`
	Tags          []string `json:"tags,omitempty"`
}

type DaemonRunNowRequest struct {
	Platform string `json:"platform"`
}

type DaemonConfigUpdateRequest struct {
	Enabled        *bool              `json:"enabled,omitempty"`
	Schedules      map[string]string  `json:"schedules,omitempty"`
	MaxPerBatch    *int               `json:"max_per_batch,omitempty"`
	AutoSkipAfter  *string            `json:"auto_skip_after,omitempty"`
	TrendingLimit  *int               `json:"trending_limit,omitempty"`
	MinLikes       *int               `json:"min_likes,omitempty"`
	Period         *string            `json:"period,omitempty"`
	DigestMode          *bool              `json:"digest_mode,omitempty"`
	DigestSchedule      *string            `json:"digest_schedule,omitempty"`
	DigestMaxPosts      *int               `json:"digest_max_posts,omitempty"`
	AutoPublish         *bool              `json:"auto_publish,omitempty"`
	AutoPublishMaxPosts *int               `json:"auto_publish_max_posts,omitempty"`
	BotToken            *string            `json:"bot_token,omitempty"`
	ChatID         *int64             `json:"chat_id,omitempty"`
	WebhookURL     *string            `json:"webhook_url,omitempty"`
}
