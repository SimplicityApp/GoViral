package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/pkg/models"
)

const systemPrompt = `You are a viral content ghostwriter. Your job is to take a trending post and rewrite it to match a specific person's voice and style while keeping the viral potential.

Respond ONLY with valid JSON array, no markdown formatting. Each element should have:
- "content": the rewritten post (ready to copy-paste)
- "viral_mechanic": brief note on what viral mechanic you preserved
- "confidence_score": number 1-10 on viral potential
- "suggest_image": boolean — true if an accompanying image would significantly boost engagement for this post
- "image_prompt": if suggest_image is true, provide a detailed image generation prompt describing the ideal image to pair with this post. The prompt should describe composition, style, colors, and subject matter. If suggest_image is false, leave this as an empty string.`

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

	response, err := g.client.SendMessage(ctx, systemPrompt, userMessage)
	if err != nil {
		return nil, fmt.Errorf("generating content: %w", err)
	}

	results, err := parseResults(response)
	if err != nil {
		return nil, fmt.Errorf("generating content: %w", err)
	}

	return results, nil
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
	fmt.Fprintf(&b, "1. Identify why this post went viral\n")
	fmt.Fprintf(&b, "2. Rewrite it matching the persona voice above\n")
	fmt.Fprintf(&b, "3. Adapt to these niches: %s\n", strings.Join(req.Niches, ", "))
	fmt.Fprintf(&b, "4. Keep the viral mechanics intact\n")
	fmt.Fprintf(&b, "5. Optimize for %s platform\n\n", req.TargetPlatform)
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
	SuggestImage    bool   `json:"suggest_image"`
	ImagePrompt     string `json:"image_prompt"`
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
			SuggestImage:    r.SuggestImage,
			ImagePrompt:     r.ImagePrompt,
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
