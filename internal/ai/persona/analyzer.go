package persona

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/pkg/models"
)

const systemPrompt = `You are a social media style analyst. Analyze the following posts from a user and produce a detailed persona profile in JSON format. Include:
- writing_tone: (e.g., casual, professional, witty, provocative)
- typical_length: average post length range
- common_themes: recurring topics (as array of strings)
- vocabulary_level: (simple, moderate, advanced, technical)
- engagement_patterns: what types of posts get the most engagement
- structural_patterns: (uses threads, single posts, questions, lists, stories)
- emoji_usage: frequency and types
- hashtag_usage: frequency and common ones
- call_to_action_style: how they engage audience
- unique_quirks: any distinctive writing habits (as array of strings)
- voice_summary: a 2-3 sentence summary of their voice

Respond ONLY with valid JSON, no markdown formatting or code blocks.`

// Analyzer implements models.PersonaAnalyzer using a Claude MessageSender.
type Analyzer struct {
	client claude.MessageSender
}

// NewAnalyzer creates a new persona Analyzer.
func NewAnalyzer(client claude.MessageSender) *Analyzer {
	return &Analyzer{client: client}
}

// BuildProfile analyzes posts and builds a PersonaProfile.
func (a *Analyzer) BuildProfile(ctx context.Context, posts []models.Post) (*models.PersonaProfile, error) {
	if len(posts) == 0 {
		return nil, fmt.Errorf("building persona profile: no posts provided")
	}

	userMessage := formatPosts(posts)

	response, err := a.client.SendMessage(ctx, systemPrompt, userMessage)
	if err != nil {
		return nil, fmt.Errorf("building persona profile: %w", err)
	}

	profile, err := parseProfile(response)
	if err != nil {
		return nil, fmt.Errorf("building persona profile: %w", err)
	}

	return profile, nil
}

func formatPosts(posts []models.Post) string {
	var b strings.Builder
	b.WriteString("Analyze the following posts:\n\n")
	for i, p := range posts {
		fmt.Fprintf(&b, "%d. [%s] (Likes: %d, Reposts: %d, Comments: %d, Impressions: %d)\n%s\n\n",
			i+1, p.Platform, p.Likes, p.Reposts, p.Comments, p.Impressions, p.Content)
	}
	return b.String()
}

func parseProfile(response string) (*models.PersonaProfile, error) {
	cleaned := stripMarkdownJSON(response)

	var profile models.PersonaProfile
	if err := json.Unmarshal([]byte(cleaned), &profile); err != nil {
		return nil, fmt.Errorf("parsing persona JSON: %w", err)
	}

	return &profile, nil
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
