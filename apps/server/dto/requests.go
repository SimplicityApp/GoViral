package dto

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
	Status           string  `json:"status,omitempty"`            // "draft", "approved", "posted"
	GeneratedContent *string `json:"generated_content,omitempty"` // optional content text update
}

type UpdateConfigRequest struct {
	Claude   *ClaudeConfigUpdate   `json:"claude,omitempty"`
	Gemini   *GeminiConfigUpdate   `json:"gemini,omitempty"`
	X        *XConfigUpdate        `json:"x,omitempty"`
	LinkedIn *LinkedInConfigUpdate `json:"linkedin,omitempty"`
	Niches         *[]string `json:"niches,omitempty"`
	LinkedInNiches *[]string `json:"linkedin_niches,omitempty"`
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
	APIKey       *string `json:"api_key,omitempty"`
	APISecret    *string `json:"api_secret,omitempty"`
	BearerToken  *string `json:"bearer_token,omitempty"`
	ClientID     *string `json:"client_id,omitempty"`
	ClientSecret *string `json:"client_secret,omitempty"`
	Username     *string `json:"username,omitempty"`
}

type LinkedInConfigUpdate struct {
	ClientID     *string `json:"client_id,omitempty"`
	ClientSecret *string `json:"client_secret,omitempty"`
	PersonURN    *string `json:"person_urn,omitempty"`
}

// --- Daemon requests ---

type BatchActionRequest struct {
	Action     string           `json:"action"`      // "approve", "reject", "edit"
	ContentIDs []int64          `json:"content_ids,omitempty"`
	Edits      map[int64]string `json:"edits,omitempty"`
	ScheduleAt string           `json:"schedule_at,omitempty"` // RFC3339
}

type DaemonRunNowRequest struct {
	Platform string `json:"platform"`
}

type DaemonConfigUpdateRequest struct {
	Enabled       *bool              `json:"enabled,omitempty"`
	Schedules     map[string]string  `json:"schedules,omitempty"`
	MaxPerBatch   *int               `json:"max_per_batch,omitempty"`
	AutoSkipAfter *string            `json:"auto_skip_after,omitempty"`
	TrendingLimit *int               `json:"trending_limit,omitempty"`
	MinLikes      *int               `json:"min_likes,omitempty"`
	Period        *string            `json:"period,omitempty"`
	BotToken      *string            `json:"bot_token,omitempty"`
	ChatID        *int64             `json:"chat_id,omitempty"`
	WebhookURL    *string            `json:"webhook_url,omitempty"`
}
