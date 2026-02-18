package models

import "context"

// PersonaAnalyzer defines the interface for building persona profiles.
type PersonaAnalyzer interface {
	BuildProfile(ctx context.Context, posts []Post) (*PersonaProfile, error)
}

// ContentGenerator defines the interface for generating viral content.
type ContentGenerator interface {
	Generate(ctx context.Context, req GenerateRequest) ([]GenerateResult, error)
}

// GenerateRequest contains parameters for content generation.
type GenerateRequest struct {
	TrendingPost   TrendingPost
	Persona        Persona
	TargetPlatform string
	Niches         []string
	Count          int
	MaxChars       int  // Maximum character length for generated content (0 = no limit)
	ForceImage     bool // When true, always generate an image prompt
	IsRepost       bool // When true, generate quote tweet commentary instead of full rewrites
}

// GenerateResult contains a single generated content variation.
type GenerateResult struct {
	Content         string
	ViralMechanic   string
	ConfidenceScore int
	SuggestImage    bool
	ImagePrompt     string
}
