package models

import "time"

// Persona represents a user's social media persona profile.
type Persona struct {
	ID        int64
	Platform  string
	Profile   PersonaProfile
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PersonaProfile contains the detailed style analysis of a user's writing.
type PersonaProfile struct {
	WritingTone        string   `json:"writing_tone"`
	TypicalLength      string   `json:"typical_length"`
	CommonThemes       []string `json:"common_themes"`
	VocabularyLevel    string   `json:"vocabulary_level"`
	EngagementPatterns string   `json:"engagement_patterns"`
	StructuralPatterns []string `json:"structural_patterns"`
	EmojiUsage         string   `json:"emoji_usage"`
	HashtagUsage       string   `json:"hashtag_usage"`
	CallToActionStyle  string   `json:"call_to_action_style"`
	UniqueQuirks       []string `json:"unique_quirks"`
	VoiceSummary       string   `json:"voice_summary"`
}
