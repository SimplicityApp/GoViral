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

	// Self-description for persona building
	SelfDescription string `json:"self_description,omitempty"`
}

// MergedXConfig returns an XConfig with app-level credentials from global config
// and user-level credentials exclusively from the per-user config.
// This prevents new users from inheriting the server operator's identity.
func (uc *UserConfig) MergedXConfig(global Config) XConfig {
	return XConfig{
		// App-level: from global config
		APIKey:       global.X.APIKey,
		APISecret:    global.X.APISecret,
		BearerToken:  global.X.BearerToken,
		ClientID:     global.X.ClientID,
		ClientSecret: global.X.ClientSecret,
		// User-level: from per-user config only (never global)
		AccessToken:       uc.XAccessToken,
		AccessTokenSecret: uc.XAccessTokenSecret,
		RefreshToken:      uc.XRefreshToken,
		TokenExpiry:       uc.XTokenExpiry,
		Username:          uc.XUsername,
	}
}

// MergedLinkedInConfig returns a LinkedInConfig with app-level credentials from global config
// and user-level credentials exclusively from the per-user config.
func (uc *UserConfig) MergedLinkedInConfig(global Config) LinkedInConfig {
	return LinkedInConfig{
		// App-level: from global config
		ClientID:     global.LinkedIn.ClientID,
		ClientSecret: global.LinkedIn.ClientSecret,
		// User-level: from per-user config only
		AccessToken: uc.LinkedInAccessToken,
		PersonURN:   uc.LinkedInPersonURN,
	}
}

// MergedYouTubeConfig returns a YouTubeConfig with app-level credentials from global config
// and user-level credentials exclusively from the per-user config.
func (uc *UserConfig) MergedYouTubeConfig(global Config) YouTubeConfig {
	return YouTubeConfig{
		// App-level: from global config
		ClientID:     global.YouTube.ClientID,
		ClientSecret: global.YouTube.ClientSecret,
		// User-level: from per-user config only
		AccessToken:  uc.YouTubeAccessToken,
		RefreshToken: uc.YouTubeRefreshToken,
		TokenExpiry:  uc.YouTubeTokenExpiry,
		ChannelID:    uc.YouTubeChannelID,
	}
}

// MergedTikTokConfig returns a TikTokConfig with app-level credentials from global config
// and user-level credentials exclusively from the per-user config.
func (uc *UserConfig) MergedTikTokConfig(global Config) TikTokConfig {
	return TikTokConfig{
		// App-level: from global config
		ClientKey:    global.TikTok.ClientKey,
		ClientSecret: global.TikTok.ClientSecret,
		// User-level: from per-user config only
		AccessToken:  uc.TikTokAccessToken,
		RefreshToken: uc.TikTokRefreshToken,
		TokenExpiry:  uc.TikTokTokenExpiry,
		Username:     uc.TikTokUsername,
	}
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

// MergedGitHubToken returns the user's OAuth token. No global fallback to prevent credential leaking.
func (uc *UserConfig) MergedGitHubToken(global Config) string {
	return uc.GitHubAccessToken
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
