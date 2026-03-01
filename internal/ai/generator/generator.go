package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/shuhao/goviral/internal/ai/prompts"
	"github.com/shuhao/goviral/pkg/models"

	"github.com/shuhao/goviral/internal/ai/claude"
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

	response, err := g.client.SendMessageJSON(ctx, prompt, userMessage, prompts.GenerateResultsSchema())
	if err != nil {
		return nil, fmt.Errorf("generating content: %w", err)
	}

	results, err := parseResults(response)
	if err != nil {
		return nil, fmt.Errorf("generating content: %w", err)
	}

	return results, nil
}

// GenerateRepoPost creates viral post variations from a GitHub commit.
func (g *Generator) GenerateRepoPost(ctx context.Context, req models.RepoPostRequest) ([]models.GenerateResult, error) {
	userMessage := prompts.BuildRepoUserMessage(req)

	platform := prompts.Platform(req.TargetPlatform)
	prompt := prompts.RepoPostPrompt(platform)

	if req.IncludeCodeImage {
		response, err := g.client.SendMessageJSON(ctx, prompt, userMessage, prompts.RepoGenerateResultsSchema())
		if err != nil {
			return nil, fmt.Errorf("generating repo post: %w", err)
		}

		results, err := parseRepoResults(response)
		if err != nil {
			return nil, fmt.Errorf("generating repo post: %w", err)
		}
		return results, nil
	}

	response, err := g.client.SendMessageJSON(ctx, prompt, userMessage, prompts.GenerateResultsSchema())
	if err != nil {
		return nil, fmt.Errorf("generating repo post: %w", err)
	}

	results, err := parseResults(response)
	if err != nil {
		return nil, fmt.Errorf("generating repo post: %w", err)
	}

	return results, nil
}

// GenerateComment creates comment variations for a trending post.
func (g *Generator) GenerateComment(ctx context.Context, req models.GenerateCommentRequest) ([]models.GenerateResult, error) {
	userMessage, err := buildCommentUserMessage(req)
	if err != nil {
		return nil, fmt.Errorf("generating comment: %w", err)
	}

	platform := prompts.Platform(req.TargetPlatform)
	prompt := prompts.CommentPrompt(platform)

	response, err := g.client.SendMessageJSON(ctx, prompt, userMessage, prompts.GenerateResultsSchema())
	if err != nil {
		return nil, fmt.Errorf("generating comment: %w", err)
	}

	results, err := parseResults(response)
	if err != nil {
		return nil, fmt.Errorf("generating comment: %w", err)
	}

	return results, nil
}

// ClassifyPost classifies a single trending post as rewrite or repost.
func (g *Generator) ClassifyPost(ctx context.Context, post models.TrendingPost) (*models.ClassifyResult, error) {
	userMessage := formatPostForClassification(post)

	response, err := g.client.SendMessageJSON(ctx, prompts.ClassifyPrompt(), userMessage, prompts.ClassifySingleSchema())
	if err != nil {
		return nil, fmt.Errorf("classifying post: %w", err)
	}

	var result models.ClassifyResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
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

	response, err := g.client.SendMessageJSON(ctx, prompts.ClassifyPrompt(), userMessage, prompts.ClassifyBatchSchema())
	if err != nil {
		return nil, fmt.Errorf("batch classifying posts: %w", err)
	}

	var wrapper struct {
		Results []models.ClassifyResult `json:"results"`
	}
	if err := json.Unmarshal([]byte(response), &wrapper); err != nil {
		// Fallback: classify individually
		results := make([]models.ClassifyResult, 0, len(posts))
		for _, post := range posts {
			r, err := g.ClassifyPost(ctx, post)
			if err != nil {
				return nil, fmt.Errorf("fallback classifying post %s: %w", post.PlatformPostID, err)
			}
			results = append(results, *r)
		}
		return results, nil
	}

	return wrapper.Results, nil
}

// SelectActions decides the optimal action (post, repost, or comment) for each trending post.
// On parse failure, falls back to individual calls.
func (g *Generator) SelectActions(ctx context.Context, posts []models.TrendingPost, platform string) ([]models.ActionSelectResult, error) {
	if len(posts) == 0 {
		return nil, nil
	}

	userMessage := formatPostsForActionSelection(posts, platform)

	response, err := g.client.SendMessageJSON(ctx, prompts.ActionSelectPrompt(), userMessage, prompts.ActionSelectBatchSchema())
	if err != nil {
		return nil, fmt.Errorf("selecting actions: %w", err)
	}

	var wrapper struct {
		Results []models.ActionSelectResult `json:"results"`
	}
	if err := json.Unmarshal([]byte(response), &wrapper); err != nil {
		return nil, fmt.Errorf("parsing action select results: %w", err)
	}

	return wrapper.Results, nil
}

// CompeteContent ranks a set of generated content entries by viral potential and returns the
// top maxWinners results sorted by rank ascending. If maxWinners <= 0 all results are returned.
func (g *Generator) CompeteContent(ctx context.Context, entries []models.CompeteEntry, maxWinners int, platform string) ([]models.CompeteResult, error) {
	if len(entries) == 0 {
		return nil, nil
	}

	userMessage := buildCompeteUserMessage(entries, platform, 1, maxWinners)

	response, err := g.client.SendMessageJSON(ctx, prompts.CompetePrompt(), userMessage, prompts.CompeteResultsSchema())
	if err != nil {
		return nil, fmt.Errorf("competing content: %w", err)
	}

	var wrapper struct {
		Rankings []models.CompeteResult `json:"rankings"`
	}
	if err := json.Unmarshal([]byte(response), &wrapper); err != nil {
		return nil, fmt.Errorf("parsing compete results: %w", err)
	}

	results := wrapper.Rankings
	sort.Slice(results, func(i, j int) bool {
		return results[i].Rank < results[j].Rank
	})

	// Enforce minimum 1 winner: if AI returned nothing, take the first entry as fallback
	if len(results) == 0 && len(entries) > 0 {
		slog.Warn("compete: AI returned 0 winners, forcing 1", "entries", len(entries))
		results = []models.CompeteResult{{
			ContentID: entries[0].GeneratedContent.ID,
			Rank:      1,
			Score:     0,
			Reasoning: "fallback: no winners selected by competition",
		}}
	}

	if maxWinners > 0 && len(results) > maxWinners {
		results = results[:maxWinners]
	}

	return results, nil
}

// DecideImage decides whether an image should accompany the given content.
func (g *Generator) DecideImage(ctx context.Context, content string, platform string) (*models.ImageDecision, error) {
	prompt := prompts.ImageDecisionPrompt(prompts.Platform(platform))
	userMessage := fmt.Sprintf("Content to evaluate:\n%s", content)

	response, err := g.client.SendMessageJSON(ctx, prompt, userMessage, prompts.ImageDecisionSchema())
	if err != nil {
		return nil, fmt.Errorf("deciding image: %w", err)
	}

	var decision models.ImageDecision
	if err := json.Unmarshal([]byte(response), &decision); err != nil {
		return nil, fmt.Errorf("parsing image decision: %w", err)
	}

	return &decision, nil
}

// GenerateImagePrompt generates a Gemini-optimized image prompt for the given content.
func (g *Generator) GenerateImagePrompt(ctx context.Context, content string, platform string) (string, error) {
	prompt := prompts.ImageGenerationPrompt(prompts.Platform(platform))
	userMessage := fmt.Sprintf("Content to create an image for:\n%s", content)

	response, err := g.client.SendMessageJSON(ctx, prompt, userMessage, prompts.ImagePromptSchema())
	if err != nil {
		return "", fmt.Errorf("generating image prompt: %w", err)
	}

	var result struct {
		ImagePrompt string `json:"image_prompt"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
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

func formatPostsForActionSelection(posts []models.TrendingPost, platform string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Platform: %s\n\n", platform)
	fmt.Fprintf(&b, "Select the optimal action for each of the following %d posts:\n\n", len(posts))
	for i, post := range posts {
		fmt.Fprintf(&b, "--- Post %d ---\n", i+1)
		fmt.Fprintf(&b, "Author: %s (@%s)\n", post.AuthorName, post.AuthorUsername)
		fmt.Fprintf(&b, "Platform: %s\n", post.Platform)
		fmt.Fprintf(&b, "Engagement: Likes %d, Reposts %d, Comments %d, Impressions %d\n", post.Likes, post.Reposts, post.Comments, post.Impressions)
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

	if req.StyleDirection != "" {
		styleTarget := "rewrite"
		if req.IsRepost {
			styleTarget = "commentary"
		}
		fmt.Fprintf(&b, "\nStyle direction from the user: %s\nIncorporate this tone/style preference into your %s.\n", req.StyleDirection, styleTarget)
	}

	return b.String(), nil
}

func buildCommentUserMessage(req models.GenerateCommentRequest) (string, error) {
	personaJSON, err := json.Marshal(req.Persona.Profile)
	if err != nil {
		return "", fmt.Errorf("marshaling persona profile: %w", err)
	}

	tp := req.TrendingPost

	var b strings.Builder
	fmt.Fprintf(&b, "## My Persona Profile\n%s\n\n", string(personaJSON))
	fmt.Fprintf(&b, "## Target Platform\n%s\n\n", req.TargetPlatform)
	fmt.Fprintf(&b, "## Post to Comment On\n")
	fmt.Fprintf(&b, "Author: %s (@%s)\n", tp.AuthorName, tp.AuthorUsername)
	fmt.Fprintf(&b, "Platform: %s\n", tp.Platform)
	fmt.Fprintf(&b, "Engagement: Likes %d, Reposts %d, Comments %d, Impressions %d\n", tp.Likes, tp.Reposts, tp.Comments, tp.Impressions)
	fmt.Fprintf(&b, "Content:\n%s\n\n", tp.Content)

	fmt.Fprintf(&b, "## Instructions\n")
	fmt.Fprintf(&b, "1. Read the post carefully and identify the key insight or topic\n")
	fmt.Fprintf(&b, "2. Write a comment that adds genuine value in the persona voice above\n")
	fmt.Fprintf(&b, "3. Keep it to 1-3 sentences — concise and impactful\n")
	fmt.Fprintf(&b, "4. Reference something specific from the post\n")
	fmt.Fprintf(&b, "5. Optimize for %s platform engagement\n\n", req.TargetPlatform)
	fmt.Fprintf(&b, "Generate %d comment variations.\n", req.Count)

	if req.StyleDirection != "" {
		fmt.Fprintf(&b, "\nStyle direction from the user: %s\nIncorporate this tone/style preference into your comment.\n", req.StyleDirection)
	}

	return b.String(), nil
}

func buildCompeteUserMessage(entries []models.CompeteEntry, platform string, minWinners, maxWinners int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Platform: %s\n\n", platform)
	fmt.Fprintf(&b, "Select the best content items from the following %d candidates:\n\n", len(entries))
	for _, e := range entries {
		tp := e.TrendingPost
		gc := e.GeneratedContent
		fmt.Fprintf(&b, "--- Content ID %d ---\n", gc.ID)
		fmt.Fprintf(&b, "Source trending post:\n")
		fmt.Fprintf(&b, "  Author: %s (@%s)\n", tp.AuthorName, tp.AuthorUsername)
		fmt.Fprintf(&b, "  Engagement: Likes %d, Reposts %d, Comments %d\n", tp.Likes, tp.Reposts, tp.Comments)
		fmt.Fprintf(&b, "  Content: %s\n", tp.Content)
		fmt.Fprintf(&b, "Generated content:\n%s\n\n", gc.GeneratedContent)
	}
	fmt.Fprintf(&b, "Selection range: return at least %d and at most %d items.\n", minWinners, maxWinners)
	return b.String()
}

type rawResult struct {
	Content         string `json:"content"`
	ViralMechanic   string `json:"viral_mechanic"`
	ConfidenceScore int    `json:"confidence_score"`
}

type rawRepoResult struct {
	Content         string           `json:"content"`
	ViralMechanic   string           `json:"viral_mechanic"`
	ConfidenceScore int              `json:"confidence_score"`
	CodeSnippet     *rawCodeSnippet  `json:"code_snippet"`
}

type rawCodeSnippet struct {
	Filename         string `json:"filename"`
	StartLine        int    `json:"start_line"`
	EndLine          int    `json:"end_line"`
	ImageDescription string `json:"image_description"`
}

func parseRepoResults(response string) ([]models.GenerateResult, error) {
	var wrapper struct {
		Results []rawRepoResult `json:"results"`
	}
	if err := json.Unmarshal([]byte(response), &wrapper); err != nil {
		return nil, fmt.Errorf("parsing repo results JSON: %w", err)
	}

	results := make([]models.GenerateResult, len(wrapper.Results))
	for i, r := range wrapper.Results {
		results[i] = models.GenerateResult{
			Content:         r.Content,
			ViralMechanic:   r.ViralMechanic,
			ConfidenceScore: r.ConfidenceScore,
		}
		if r.CodeSnippet != nil && r.CodeSnippet.Filename != "" &&
			r.CodeSnippet.StartLine > 0 && r.CodeSnippet.EndLine >= r.CodeSnippet.StartLine {
			results[i].CodeSnippet = &models.CodeSnippet{
				Filename:    r.CodeSnippet.Filename,
				StartLine:   r.CodeSnippet.StartLine,
				EndLine:     r.CodeSnippet.EndLine,
				Description: r.CodeSnippet.ImageDescription,
			}
		}
	}

	return results, nil
}

func parseResults(response string) ([]models.GenerateResult, error) {
	var wrapper struct {
		Results []rawResult `json:"results"`
	}
	if err := json.Unmarshal([]byte(response), &wrapper); err != nil {
		return nil, fmt.Errorf("parsing generated results JSON: %w", err)
	}

	results := make([]models.GenerateResult, len(wrapper.Results))
	for i, r := range wrapper.Results {
		results[i] = models.GenerateResult{
			Content:         r.Content,
			ViralMechanic:   r.ViralMechanic,
			ConfidenceScore: r.ConfidenceScore,
		}
	}

	return results, nil
}
