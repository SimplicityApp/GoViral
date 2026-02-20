package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/internal/ai/prompts"
	"github.com/shuhao/goviral/pkg/models"
)

// Generator implements models.ContentGenerator using a Claude MessageSender.
type Generator struct {
	client claude.MessageSender
}

// NewGenerator creates a new content Generator.
func NewGenerator(client claude.MessageSender) *Generator {
	return &Generator{client: client}
}

// Generate creates viral content variations based on the request.
func (g *Generator) Generate(ctx context.Context, req models.GenerateRequest) ([]models.GenerateResult, error) {
	userMessage, err := buildUserMessage(req)
	if err != nil {
		return nil, fmt.Errorf("generating content: %w", err)
	}

	platform := prompts.Platform(req.TargetPlatform)
	prompt := prompts.GeneratePrompt(platform, req.IsRepost)

	response, err := g.client.SendMessage(ctx, prompt, userMessage)
	if err != nil {
		return nil, fmt.Errorf("generating content: %w", err)
	}

	results, err := parseResults(response)
	if err != nil {
		return nil, fmt.Errorf("generating content: %w", err)
	}

	return results, nil
}

// ClassifyPost classifies a single trending post as rewrite or repost.
func (g *Generator) ClassifyPost(ctx context.Context, post models.TrendingPost) (*models.ClassifyResult, error) {
	userMessage := formatPostForClassification(post)

	response, err := g.client.SendMessage(ctx, prompts.ClassifyPrompt(), userMessage)
	if err != nil {
		return nil, fmt.Errorf("classifying post: %w", err)
	}

	cleaned := stripMarkdownJSON(response)
	var result models.ClassifyResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("parsing classification result: %w", err)
	}

	return &result, nil
}

// ClassifyPosts classifies multiple trending posts in a single batch call.
// On parse failure, falls back to individual ClassifyPost calls.
func (g *Generator) ClassifyPosts(ctx context.Context, posts []models.TrendingPost) ([]models.ClassifyResult, error) {
	if len(posts) == 0 {
		return nil, nil
	}
	if len(posts) == 1 {
		r, err := g.ClassifyPost(ctx, posts[0])
		if err != nil {
			return nil, err
		}
		return []models.ClassifyResult{*r}, nil
	}

	userMessage := formatPostsForClassification(posts)

	response, err := g.client.SendMessage(ctx, prompts.ClassifyPrompt(), userMessage)
	if err != nil {
		return nil, fmt.Errorf("batch classifying posts: %w", err)
	}

	cleaned := stripMarkdownJSON(response)
	var results []models.ClassifyResult
	if err := json.Unmarshal([]byte(cleaned), &results); err != nil {
		// Fallback: classify individually
		results = make([]models.ClassifyResult, 0, len(posts))
		for _, post := range posts {
			r, err := g.ClassifyPost(ctx, post)
			if err != nil {
				return nil, fmt.Errorf("fallback classifying post %s: %w", post.PlatformPostID, err)
			}
			results = append(results, *r)
		}
	}

	return results, nil
}

// DecideImage decides whether an image should accompany the given content.
func (g *Generator) DecideImage(ctx context.Context, content string, platform string) (*models.ImageDecision, error) {
	prompt := prompts.ImageDecisionPrompt(prompts.Platform(platform))
	userMessage := fmt.Sprintf("Content to evaluate:\n%s", content)

	response, err := g.client.SendMessage(ctx, prompt, userMessage)
	if err != nil {
		return nil, fmt.Errorf("deciding image: %w", err)
	}

	cleaned := stripMarkdownJSON(response)
	var decision models.ImageDecision
	if err := json.Unmarshal([]byte(cleaned), &decision); err != nil {
		return nil, fmt.Errorf("parsing image decision: %w", err)
	}

	return &decision, nil
}

// GenerateImagePrompt generates a Gemini-optimized image prompt for the given content.
func (g *Generator) GenerateImagePrompt(ctx context.Context, content string, platform string) (string, error) {
	prompt := prompts.ImageGenerationPrompt(prompts.Platform(platform))
	userMessage := fmt.Sprintf("Content to create an image for:\n%s", content)

	response, err := g.client.SendMessage(ctx, prompt, userMessage)
	if err != nil {
		return "", fmt.Errorf("generating image prompt: %w", err)
	}

	cleaned := stripMarkdownJSON(response)
	var result struct {
		ImagePrompt string `json:"image_prompt"`
	}
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return "", fmt.Errorf("parsing image prompt result: %w", err)
	}

	return result.ImagePrompt, nil
}

func formatPostForClassification(post models.TrendingPost) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Author: %s (@%s)\n", post.AuthorName, post.AuthorUsername)
	fmt.Fprintf(&b, "Platform: %s\n", post.Platform)
	fmt.Fprintf(&b, "Engagement: Likes %d, Reposts %d, Comments %d\n", post.Likes, post.Reposts, post.Comments)
	fmt.Fprintf(&b, "Content:\n%s\n", post.Content)
	return b.String()
}

func formatPostsForClassification(posts []models.TrendingPost) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Classify the following %d posts:\n\n", len(posts))
	for i, post := range posts {
		fmt.Fprintf(&b, "--- Post %d ---\n", i+1)
		fmt.Fprintf(&b, "Author: %s (@%s)\n", post.AuthorName, post.AuthorUsername)
		fmt.Fprintf(&b, "Platform: %s\n", post.Platform)
		fmt.Fprintf(&b, "Engagement: Likes %d, Reposts %d, Comments %d\n", post.Likes, post.Reposts, post.Comments)
		fmt.Fprintf(&b, "Content:\n%s\n\n", post.Content)
	}
	return b.String()
}

func buildUserMessage(req models.GenerateRequest) (string, error) {
	personaJSON, err := json.Marshal(req.Persona.Profile)
	if err != nil {
		return "", fmt.Errorf("marshaling persona profile: %w", err)
	}

	tp := req.TrendingPost

	var b strings.Builder
	fmt.Fprintf(&b, "## My Persona Profile\n%s\n\n", string(personaJSON))
	fmt.Fprintf(&b, "## Target Platform\n%s\n\n", req.TargetPlatform)
	fmt.Fprintf(&b, "## Trending Post\n")
	fmt.Fprintf(&b, "Author: %s (@%s)\n", tp.AuthorName, tp.AuthorUsername)
	fmt.Fprintf(&b, "Platform: %s\n", tp.Platform)
	fmt.Fprintf(&b, "Engagement: Likes %d, Reposts %d, Comments %d, Impressions %d\n", tp.Likes, tp.Reposts, tp.Comments, tp.Impressions)
	fmt.Fprintf(&b, "Content:\n%s\n\n", tp.Content)

	if len(tp.Media) > 0 {
		fmt.Fprintf(&b, "Attached Media:\n")
		for _, m := range tp.Media {
			fmt.Fprintf(&b, "- Type: %s", m.Type)
			if m.URL != "" {
				fmt.Fprintf(&b, ", URL: %s", m.URL)
			}
			if m.AltText != "" {
				fmt.Fprintf(&b, ", Alt: %s", m.AltText)
			}
			fmt.Fprintf(&b, "\n")
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "## Instructions\n")
	if req.IsRepost {
		fmt.Fprintf(&b, "1. Identify why this post went viral and what makes it shareable\n")
		fmt.Fprintf(&b, "2. Write short quote tweet commentary (1-3 sentences) in the persona voice above\n")
		fmt.Fprintf(&b, "3. Add value with a hot take, amplification, personal anecdote, or contrarian view\n")
		fmt.Fprintf(&b, "4. Keep it punchy — the original post will be embedded below your commentary\n")
		fmt.Fprintf(&b, "5. Optimize for %s platform\n\n", req.TargetPlatform)
	} else {
		fmt.Fprintf(&b, "1. Identify why this post went viral\n")
		fmt.Fprintf(&b, "2. Rewrite it matching the persona voice above\n")
		fmt.Fprintf(&b, "3. Adapt to these niches: %s\n", strings.Join(req.Niches, ", "))
		fmt.Fprintf(&b, "4. Keep the viral mechanics intact\n")
		fmt.Fprintf(&b, "5. Optimize for %s platform\n\n", req.TargetPlatform)
	}
	if req.MaxChars > 0 {
		fmt.Fprintf(&b, "6. Each post MUST be %d characters or fewer\n\n", req.MaxChars)
	}
	fmt.Fprintf(&b, "Generate %d variations.\n", req.Count)

	return b.String(), nil
}

type rawResult struct {
	Content         string `json:"content"`
	ViralMechanic   string `json:"viral_mechanic"`
	ConfidenceScore int    `json:"confidence_score"`
}

func parseResults(response string) ([]models.GenerateResult, error) {
	cleaned := stripMarkdownJSON(response)

	var raw []rawResult
	if err := json.Unmarshal([]byte(cleaned), &raw); err != nil {
		return nil, fmt.Errorf("parsing generated results JSON: %w", err)
	}

	results := make([]models.GenerateResult, len(raw))
	for i, r := range raw {
		results[i] = models.GenerateResult{
			Content:         r.Content,
			ViralMechanic:   r.ViralMechanic,
			ConfidenceScore: r.ConfidenceScore,
		}
	}

	return results, nil
}

func stripMarkdownJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	return s
}
