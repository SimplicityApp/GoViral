package models

import "context"

// PersonaAnalyzer defines the interface for building persona profiles.
type PersonaAnalyzer interface {
	BuildProfile(ctx context.Context, posts []Post, platform string) (*PersonaProfile, error)
}

// ContentGenerator defines the interface for generating viral content.
type ContentGenerator interface {
	Generate(ctx context.Context, req GenerateRequest) ([]GenerateResult, error)
	GenerateComment(ctx context.Context, req GenerateCommentRequest) ([]GenerateResult, error)
	GenerateRepoPost(ctx context.Context, req RepoPostRequest) ([]GenerateResult, error)
	ClassifyPost(ctx context.Context, post TrendingPost) (*ClassifyResult, error)
	ClassifyPosts(ctx context.Context, posts []TrendingPost) ([]ClassifyResult, error)
	SelectActions(ctx context.Context, posts []TrendingPost, platform string) ([]ActionSelectResult, error)
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
	MaxChars       int    // Maximum character length for generated content (0 = no limit)
	IsRepost       bool   // When true, generate quote tweet commentary instead of full rewrites
	StyleDirection string // Optional tone/style instruction for rewrites (e.g., "more casual")
}

// CodeSnippet identifies a specific range of diff lines for a code image.
type CodeSnippet struct {
	Filename    string `json:"filename"`
	StartLine   int    `json:"start_line"` // 1-based global diff line number
	EndLine     int    `json:"end_line"`   // inclusive
	Description string `json:"image_description"`
}

// GenerateResult contains a single generated content variation.
type GenerateResult struct {
	Content         string
	ViralMechanic   string
	ConfidenceScore int
	CodeSnippet     *CodeSnippet
}

// ClassifyResult contains the classification of a trending post as rewrite or repost.
type ClassifyResult struct {
	Decision   string `json:"decision"`   // "rewrite" or "repost"
	Reasoning  string `json:"reasoning"`
	Confidence int    `json:"confidence"`
}

// GenerateCommentRequest contains parameters for comment generation.
type GenerateCommentRequest struct {
	TrendingPost   TrendingPost
	Persona        Persona
	TargetPlatform string
	Count          int
	StyleDirection string
}

// ImageDecision contains the decision about whether to include an image.
type ImageDecision struct {
	SuggestImage bool   `json:"suggest_image"`
	Reasoning    string `json:"reasoning"`
}

// ActionSelectResult contains the AI's action selection for a trending post.
type ActionSelectResult struct {
	Action     string `json:"action"`     // "post", "repost", "comment"
	Reasoning  string `json:"reasoning"`
	Confidence int    `json:"confidence"`
}

// CompeteEntry pairs a trending post with its generated content for competition ranking.
type CompeteEntry struct {
	TrendingPost     TrendingPost
	GeneratedContent GeneratedContent
}

// CompeteResult contains the ranking result for a single content item.
type CompeteResult struct {
	ContentID int64   `json:"content_id"`
	Rank      int     `json:"rank"`
	Score     float64 `json:"score"`
	Reasoning string  `json:"reasoning"`
}
