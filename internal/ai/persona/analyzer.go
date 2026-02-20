package persona

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/internal/ai/prompts"
	"github.com/shuhao/goviral/pkg/models"
)

// Analyzer implements models.PersonaAnalyzer using a Claude MessageSender.
type Analyzer struct {
	client claude.MessageSender
}

// NewAnalyzer creates a new persona Analyzer.
func NewAnalyzer(client claude.MessageSender) *Analyzer {
	return &Analyzer{client: client}
}

// BuildProfile analyzes posts and builds a PersonaProfile.
func (a *Analyzer) BuildProfile(ctx context.Context, posts []models.Post, platform string) (*models.PersonaProfile, error) {
	if len(posts) == 0 {
		return nil, fmt.Errorf("building persona profile: no posts provided")
	}

	userMessage := formatPosts(posts)
	systemPrompt := prompts.PersonaPrompt(prompts.Platform(platform))

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
