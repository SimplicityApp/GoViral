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
func (a *Analyzer) BuildProfile(ctx context.Context, posts []models.Post, platform string, selfDescription string) (*models.PersonaProfile, error) {
	if len(posts) == 0 && selfDescription == "" {
		return nil, fmt.Errorf("building persona profile: no posts or self-description provided")
	}

	var userMessage string
	if selfDescription != "" {
		userMessage = fmt.Sprintf("The user describes themselves as:\n%s\n\n", selfDescription)
	}
	if len(posts) > 0 {
		userMessage += formatPosts(posts)
	}
	if len(posts) == 0 {
		userMessage += "No posts are available. Build the persona entirely from the self-description above."
	}
	systemPrompt := prompts.PersonaPrompt(prompts.Platform(platform))

	response, err := a.client.SendMessageJSON(ctx, systemPrompt, userMessage, prompts.PersonaProfileSchema())
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
	var profile models.PersonaProfile
	if err := json.Unmarshal([]byte(response), &profile); err != nil {
		return nil, fmt.Errorf("parsing persona JSON: %w", err)
	}

	return &profile, nil
}
