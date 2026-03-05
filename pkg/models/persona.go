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

// DefaultPersonaProfile returns a generic, neutral persona profile
// used as a fallback when no user-specific persona has been built.
func DefaultPersonaProfile(platform string) PersonaProfile {
	return PersonaProfile{
		WritingTone:        "casual and authentic",
		TypicalLength:      "medium",
		CommonThemes:       []string{"general interest"},
		VocabularyLevel:    "conversational",
		EngagementPatterns: "balanced mix of original thoughts and reactions",
		StructuralPatterns: []string{"concise statements", "occasional questions"},
		EmojiUsage:         "minimal",
		HashtagUsage:       "sparingly when relevant",
		CallToActionStyle:  "subtle and natural",
		UniqueQuirks:       []string{},
		VoiceSummary:       "A straightforward, approachable voice that communicates clearly and naturally.",
	}
}
