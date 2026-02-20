package models

import "context"

// PersonaAnalyzer defines the interface for building persona profiles.
type PersonaAnalyzer interface {
	BuildProfile(ctx context.Context, posts []Post, platform string) (*PersonaProfile, error)
}

// ContentGenerator defines the interface for generating viral content.
type ContentGenerator interface {
	Generate(ctx context.Context, req GenerateRequest) ([]GenerateResult, error)
	ClassifyPost(ctx context.Context, post TrendingPost) (*ClassifyResult, error)
	ClassifyPosts(ctx context.Context, posts []TrendingPost) ([]ClassifyResult, error)
	DecideImage(ctx context.Context, content string, platform string) (*ImageDecision, error)
	GenerateImagePrompt(ctx context.Context, content string, platform string) (string, error)
}

// GenerateRequest contains parameters for content generation.
type GenerateRequest struct {
	TrendingPost   TrendingPost
	Persona        Persona
	TargetPlatform string
	Niches         []string
	Count          int
	MaxChars       int  // Maximum character length for generated content (0 = no limit)
	IsRepost       bool // When true, generate quote tweet commentary instead of full rewrites
}

// GenerateResult contains a single generated content variation.
type GenerateResult struct {
	Content         string
	ViralMechanic   string
	ConfidenceScore int
}

// ClassifyResult contains the classification of a trending post as rewrite or repost.
type ClassifyResult struct {
	Decision   string `json:"decision"`   // "rewrite" or "repost"
	Reasoning  string `json:"reasoning"`
	Confidence int    `json:"confidence"`
}

// ImageDecision contains the decision about whether to include an image.
type ImageDecision struct {
	SuggestImage bool   `json:"suggest_image"`
	Reasoning    string `json:"reasoning"`
}
