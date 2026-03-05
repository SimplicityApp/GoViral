package config

// UserConfig holds per-user settings stored in the database.
// These override or supplement global config on a per-user basis.
type UserConfig struct {
	// AI BYOK (Bring Your Own Key) — optional, overrides global key
	ClaudeAPIKey string `json:"claude_api_key,omitempty"`
	ClaudeModel  string `json:"claude_model,omitempty"`
	GeminiAPIKey string `json:"gemini_api_key,omitempty"`
	GeminiModel  string `json:"gemini_model,omitempty"`

	// X/Twitter per-user credentials
	XUsername          string `json:"x_username,omitempty"`
	XAccessToken       string `json:"x_access_token,omitempty"`
	XAccessTokenSecret string `json:"x_access_token_secret,omitempty"`
	XRefreshToken      string `json:"x_refresh_token,omitempty"`
	XTokenExpiry       string `json:"x_token_expiry,omitempty"`

	// LinkedIn per-user credentials
	LinkedInAccessToken string `json:"linkedin_access_token,omitempty"`
	LinkedInPersonURN   string `json:"linkedin_person_urn,omitempty"`

	// YouTube per-user credentials
	YouTubeAccessToken  string `json:"youtube_access_token,omitempty"`
	YouTubeRefreshToken string `json:"youtube_refresh_token,omitempty"`
	YouTubeTokenExpiry  string `json:"youtube_token_expiry,omitempty"`
	YouTubeChannelID    string `json:"youtube_channel_id,omitempty"`

	// TikTok per-user credentials
	TikTokAccessToken  string `json:"tiktok_access_token,omitempty"`
	TikTokRefreshToken string `json:"tiktok_refresh_token,omitempty"`
	TikTokTokenExpiry  string `json:"tiktok_token_expiry,omitempty"`
	TikTokUsername     string `json:"tiktok_username,omitempty"`

	// GitHub per-user credentials
	GitHubAccessToken string `json:"github_access_token,omitempty"`

	// Cookie data (JSON strings)
	TwikitCookiesJSON   string `json:"twikit_cookies_json,omitempty"`
	LinkitinCookiesJSON string `json:"linkitin_cookies_json,omitempty"`

	// Niches
	Niches         []string `json:"niches,omitempty"`
	LinkedInNiches []string `json:"linkedin_niches,omitempty"`
}

// MergedXConfig returns an XConfig starting from global app credentials,
// then overriding with the user's per-user fields where non-empty.
func (uc *UserConfig) MergedXConfig(global Config) XConfig {
	cfg := global.X
	if uc.XUsername != "" {
		cfg.Username = uc.XUsername
	}
	if uc.XAccessToken != "" {
		cfg.AccessToken = uc.XAccessToken
	}
	if uc.XAccessTokenSecret != "" {
		cfg.AccessTokenSecret = uc.XAccessTokenSecret
	}
	if uc.XRefreshToken != "" {
		cfg.RefreshToken = uc.XRefreshToken
	}
	if uc.XTokenExpiry != "" {
		cfg.TokenExpiry = uc.XTokenExpiry
	}
	return cfg
}

// MergedLinkedInConfig returns a LinkedInConfig starting from global app credentials,
// then overriding with user's per-user fields where non-empty.
func (uc *UserConfig) MergedLinkedInConfig(global Config) LinkedInConfig {
	cfg := global.LinkedIn
	if uc.LinkedInAccessToken != "" {
		cfg.AccessToken = uc.LinkedInAccessToken
	}
	if uc.LinkedInPersonURN != "" {
		cfg.PersonURN = uc.LinkedInPersonURN
	}
	return cfg
}

// MergedYouTubeConfig returns a YouTubeConfig starting from global app credentials,
// then overriding with user's per-user fields where non-empty.
func (uc *UserConfig) MergedYouTubeConfig(global Config) YouTubeConfig {
	cfg := global.YouTube
	if uc.YouTubeAccessToken != "" {
		cfg.AccessToken = uc.YouTubeAccessToken
	}
	if uc.YouTubeRefreshToken != "" {
		cfg.RefreshToken = uc.YouTubeRefreshToken
	}
	if uc.YouTubeTokenExpiry != "" {
		cfg.TokenExpiry = uc.YouTubeTokenExpiry
	}
	if uc.YouTubeChannelID != "" {
		cfg.ChannelID = uc.YouTubeChannelID
	}
	return cfg
}

// MergedTikTokConfig returns a TikTokConfig starting from global app credentials,
// then overriding with user's per-user fields where non-empty.
func (uc *UserConfig) MergedTikTokConfig(global Config) TikTokConfig {
	cfg := global.TikTok
	if uc.TikTokAccessToken != "" {
		cfg.AccessToken = uc.TikTokAccessToken
	}
	if uc.TikTokRefreshToken != "" {
		cfg.RefreshToken = uc.TikTokRefreshToken
	}
	if uc.TikTokTokenExpiry != "" {
		cfg.TokenExpiry = uc.TikTokTokenExpiry
	}
	if uc.TikTokUsername != "" {
		cfg.Username = uc.TikTokUsername
	}
	return cfg
}

// ResolvedClaudeConfig returns the effective ClaudeConfig for this user.
// If the user has their own API key, it takes precedence over the global key.
func (uc *UserConfig) ResolvedClaudeConfig(global Config) ClaudeConfig {
	if uc.ClaudeAPIKey == "" {
		return global.Claude
	}
	model := uc.ClaudeModel
	if model == "" {
		model = global.Claude.Model
	}
	return ClaudeConfig{
		APIKey:     uc.ClaudeAPIKey,
		Model:      model,
		DailyLimit: global.Claude.DailyLimit,
	}
}

// ResolvedGeminiConfig returns the effective GeminiConfig for this user.
// If the user has their own API key, it takes precedence over the global key.
func (uc *UserConfig) ResolvedGeminiConfig(global Config) GeminiConfig {
	if uc.GeminiAPIKey == "" {
		return global.Gemini
	}
	model := uc.GeminiModel
	if model == "" {
		model = global.Gemini.Model
	}
	return GeminiConfig{
		APIKey:     uc.GeminiAPIKey,
		Model:      model,
		DailyLimit: global.Gemini.DailyLimit,
	}
}

// UsingOwnClaudeKey reports whether the user has configured their own Claude API key.
func (uc *UserConfig) UsingOwnClaudeKey() bool {
	return uc.ClaudeAPIKey != ""
}

// UsingOwnGeminiKey reports whether the user has configured their own Gemini API key.
func (uc *UserConfig) UsingOwnGeminiKey() bool {
	return uc.GeminiAPIKey != ""
}

// MergedGitHubToken returns the user's OAuth token if set, otherwise falls back to the global PAT.
func (uc *UserConfig) MergedGitHubToken(global Config) string {
	if uc.GitHubAccessToken != "" {
		return uc.GitHubAccessToken
	}
	return global.GitHub.PersonalAccessToken
}

// MergedNiches returns the user's X/Twitter niches if set, otherwise falls back to global niches.
func (uc *UserConfig) MergedNiches(global Config) []string {
	if len(uc.Niches) > 0 {
		return uc.Niches
	}
	return global.Niches
}

// MergedLinkedInNiches returns the user's LinkedIn niches if set, otherwise falls back to global niches.
func (uc *UserConfig) MergedLinkedInNiches(global Config) []string {
	if len(uc.LinkedInNiches) > 0 {
		return uc.LinkedInNiches
	}
	return global.LinkedInNiches
}
