package dto

import "time"

type HealthResponse struct {
	Status    string            `json:"status"`
	Platforms map[string]string `json:"platforms"`
}

type PostResponse struct {
	ID             int64     `json:"id"`
	Platform       string    `json:"platform"`
	PlatformPostID string    `json:"platform_post_id"`
	Content        string    `json:"content"`
	Likes          int       `json:"likes"`
	Reposts        int       `json:"reposts"`
	Comments       int       `json:"comments"`
	Impressions    int       `json:"impressions"`
	PostedAt       time.Time `json:"posted_at"`
	FetchedAt      time.Time `json:"fetched_at"`
}

type TrendingPostResponse struct {
	ID             int64     `json:"id"`
	Platform       string    `json:"platform"`
	PlatformPostID string    `json:"platform_post_id"`
	AuthorUsername string    `json:"author_username"`
	AuthorName     string    `json:"author_name"`
	Content        string    `json:"content"`
	Likes          int       `json:"likes"`
	Reposts        int       `json:"reposts"`
	Comments       int       `json:"comments"`
	Impressions    int       `json:"impressions"`
	NicheTags      []string  `json:"niche_tags"`
	PostedAt       time.Time `json:"posted_at"`
	FetchedAt      time.Time `json:"fetched_at"`
	IsActionable   bool      `json:"is_actionable"`
}

type PersonaResponse struct {
	ID        int64                  `json:"id"`
	Platform  string                 `json:"platform"`
	Profile   map[string]interface{} `json:"profile"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type GeneratedContentResponse struct {
	ID               int64      `json:"id"`
	SourceTrendingID int64      `json:"source_trending_id"`
	TargetPlatform   string     `json:"target_platform"`
	OriginalContent  string     `json:"original_content"`
	GeneratedContent string     `json:"generated_content"`
	PersonaID        int64      `json:"persona_id"`
	PromptUsed       string     `json:"prompt_used"`
	CreatedAt        time.Time  `json:"created_at"`
	Status           string     `json:"status"`
	PlatformPostIDs  string     `json:"platform_post_ids,omitempty"`
	PostedAt         *time.Time `json:"posted_at,omitempty"`
	ImagePrompt      string     `json:"image_prompt,omitempty"`
	ImagePath        string     `json:"image_path,omitempty"`
	IsRepost         bool       `json:"is_repost"`
	QuoteTweetID     string     `json:"quote_tweet_id,omitempty"`
	IsComment        bool       `json:"is_comment"`
	SourceType       string     `json:"source_type,omitempty"`
	SourceCommitID   int64      `json:"source_commit_id,omitempty"`
	CodeImagePath        string `json:"code_image_path,omitempty"`
	CodeImageDescription string `json:"code_image_description,omitempty"`
	VideoPath            string `json:"video_path,omitempty"`
	ThumbnailPath        string `json:"thumbnail_path,omitempty"`
	VideoDuration        int    `json:"video_duration,omitempty"`
	VideoTitle           string `json:"video_title,omitempty"`
}

type ScheduledPostResponse struct {
	ID                 int64     `json:"id"`
	GeneratedContentID int64     `json:"generated_content_id"`
	ScheduledAt        time.Time `json:"scheduled_at"`
	Status             string    `json:"status"`
	ErrorMessage       string    `json:"error_message,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	ContentPreview     string    `json:"content_preview,omitempty"`
	TargetPlatform     string    `json:"target_platform,omitempty"`
	PlatformScheduleID string    `json:"platform_schedule_id,omitempty"`
}

type ConfigResponse struct {
	Claude         ConfigClaudeResponse   `json:"claude"`
	Gemini         ConfigGeminiResponse   `json:"gemini"`
	X              ConfigXResponse        `json:"x"`
	LinkedIn       ConfigLinkedInResponse `json:"linkedin"`
	GitHub         ConfigGitHubResponse   `json:"github"`
	YouTube        ConfigYouTubeResponse  `json:"youtube"`
	TikTok         ConfigTikTokResponse   `json:"tiktok"`
	Niches         []string               `json:"niches"`
	LinkedInNiches []string               `json:"linkedin_niches"`
}

type ConfigClaudeResponse struct {
	APIKey string `json:"api_key"`
	Model  string `json:"model"`
}

type ConfigGeminiResponse struct {
	APIKey string `json:"api_key"`
	Model  string `json:"model"`
}

type ConfigXResponse struct {
	APIKey       string `json:"api_key"`
	APISecret    string `json:"api_secret"`
	BearerToken  string `json:"bearer_token"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Username     string `json:"username"`
	HasAuth      bool   `json:"has_auth"`
	HasTwikitAuth bool  `json:"has_twikit_auth"`
}

type ConfigLinkedInResponse struct {
	ClientID        string `json:"client_id"`
	ClientSecret    string `json:"client_secret"`
	HasAuth         bool   `json:"has_auth"`
	HasLinkitinAuth bool   `json:"has_linkitin_auth"`
}

type ConfigGitHubResponse struct {
	PersonalAccessToken string `json:"personal_access_token"`
	DefaultOwner        string `json:"default_owner"`
	DefaultRepo         string `json:"default_repo"`
}

type ConfigYouTubeResponse struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	ChannelID    string `json:"channel_id"`
	HasAuth      bool   `json:"has_auth"`
}

type ConfigTikTokResponse struct {
	ClientKey    string `json:"client_key"`
	ClientSecret string `json:"client_secret"`
	Username     string `json:"username"`
	HasAuth      bool   `json:"has_auth"`
}

type VideoUploadResponse struct {
	VideoID  string `json:"video_id"`
	Platform string `json:"platform"`
	URL      string `json:"url,omitempty"`
}

// --- Repo-to-post responses ---

type RepoResponse struct {
	ID             int64          `json:"id"`
	Owner          string         `json:"owner"`
	Name           string         `json:"name"`
	FullName       string         `json:"full_name"`
	Description    string         `json:"description"`
	DefaultBranch  string         `json:"default_branch"`
	Language       string         `json:"language"`
	AddedAt        string         `json:"added_at"`
	TargetAudience string         `json:"target_audience"`
	Links          []RepoLinkDTO `json:"links"`
}

type RepoCommitResponse struct {
	ID           int64              `json:"id"`
	RepoID       int64              `json:"repo_id"`
	SHA          string             `json:"sha"`
	Message      string             `json:"message"`
	AuthorName   string             `json:"author_name"`
	CommittedAt  string             `json:"committed_at"`
	Additions    int                `json:"additions"`
	Deletions    int                `json:"deletions"`
	FilesChanged int                `json:"files_changed"`
	DiffSummary  string             `json:"diff_summary"`
	Files        []RepoFileResponse `json:"files"`
}

type RepoFileResponse struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

type AvailableRepoResponse struct {
	FullName    string `json:"full_name"`
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Private     bool   `json:"private"`
}

type CommentResponse struct {
	CommentURN string                   `json:"comment_urn"`
	Content    GeneratedContentResponse `json:"content"`
}

type PublishResponse struct {
	PostIDs     []string `json:"post_ids"`
	ThreadParts []string `json:"thread_parts,omitempty"`
}

type OperationResponse struct {
	ID     string      `json:"id"`
	Status string      `json:"status"` // "running", "completed", "failed"
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type ProgressEvent struct {
	Type       string      `json:"type"` // "progress", "complete", "error"
	Message    string      `json:"message"`
	Percentage int         `json:"percentage"`
	Data       interface{} `json:"data,omitempty"`
}

// --- Daemon responses ---

type DaemonStatusResponse struct {
	Running   bool                          `json:"running"`
	Platforms map[string]PlatformDaemonInfo `json:"platforms"`
}

type PlatformDaemonInfo struct {
	Schedule    string  `json:"schedule"`
	NextRun     *string `json:"next_run,omitempty"`    // RFC3339
	LastRun     *string `json:"last_run,omitempty"`    // RFC3339
	LastBatchID *int64  `json:"last_batch_id,omitempty"`
	NextDigest  *string `json:"next_digest,omitempty"` // RFC3339
	Paused      bool    `json:"paused,omitempty"`
	PausedAt    *string `json:"paused_at,omitempty"`    // RFC3339
	PauseReason string  `json:"pause_reason,omitempty"`
}

type DaemonBatchResponse struct {
	ID                int64                      `json:"id"`
	Platform          string                     `json:"platform"`
	Status            string                     `json:"status"`
	ContentIDs        []int64                    `json:"content_ids"`
	TrendingIDs       []int64                    `json:"trending_ids"`
	TelegramMessageID int64                      `json:"telegram_message_id"`
	ApprovalSource    string                     `json:"approval_source"`
	ReplyText         string                     `json:"reply_text"`
	ErrorMessage      string                     `json:"error_message"`
	BatchType         string                     `json:"batch_type"`
	CreatedAt         string                     `json:"created_at"`
	UpdatedAt         string                     `json:"updated_at"`
	NotifiedAt        *string                    `json:"notified_at,omitempty"`
	ResolvedAt        *string                    `json:"resolved_at,omitempty"`
	Contents          []GeneratedContentResponse `json:"contents,omitempty"`
}

type DaemonConfigResponse struct {
	Daemon   DaemonSettingsResponse   `json:"daemon"`
	Telegram TelegramSettingsResponse `json:"telegram"`
}

type DaemonSettingsResponse struct {
	Enabled        bool              `json:"enabled"`
	Schedules      map[string]string `json:"schedules"`
	MaxPerBatch    int               `json:"max_per_batch"`
	AutoSkipAfter  string            `json:"auto_skip_after"`
	TrendingLimit  int               `json:"trending_limit"`
	MinLikes       int               `json:"min_likes"`
	Period         string            `json:"period"`
	DigestMode          bool              `json:"digest_mode"`
	DigestSchedule      string            `json:"digest_schedule"`
	DigestMaxPosts      int               `json:"digest_max_posts"`
	AutoPublish         bool              `json:"auto_publish"`
	AutoPublishMaxPosts int               `json:"auto_publish_max_posts"`
}

type TelegramSettingsResponse struct {
	BotToken   string `json:"bot_token"`
	ChatID     int64  `json:"chat_id"`
	WebhookURL string `json:"webhook_url"`
	Connected  bool   `json:"connected"`
}

// --- Code image template responses ---

type CodeImageTemplateResponse struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	SupportsDescription bool   `json:"supports_description"`
}

type CodeImageThemeResponse struct {
	Name string `json:"name"`
}

type CodeImageOptionsResponse struct {
	Templates []CodeImageTemplateResponse `json:"templates"`
	Themes    []CodeImageThemeResponse    `json:"themes"`
}

type CodeImagePreviewsResponse struct {
	Theme    string            `json:"theme"`
	Previews map[string]string `json:"previews"` // template_name → full HTML
}
