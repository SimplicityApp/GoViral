package persona

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/shuhao/goviral/pkg/models"
)

// mockMessageSender implements claude.MessageSender for testing.
type mockMessageSender struct {
	response string
	err      error
}

func (m *mockMessageSender) SendMessage(ctx context.Context, systemPrompt string, userMessage string) (string, error) {
	return m.response, m.err
}

func samplePosts() []models.Post {
	return []models.Post{
		{
			Platform:    "x",
			Content:     "Hot take: Go is the best language for building APIs. Fight me.",
			Likes:       500,
			Reposts:     100,
			Comments:    75,
			Impressions: 10000,
		},
		{
			Platform:    "x",
			Content:     "Just shipped a new feature. Here's what I learned about clean architecture...",
			Likes:       300,
			Reposts:     50,
			Comments:    30,
			Impressions: 5000,
		},
	}
}

func TestBuildProfile_Success(t *testing.T) {
	personaJSON := `{
		"writing_tone": "professional yet approachable",
		"typical_length": "medium (100-200 words)",
		"common_themes": ["technology", "startups", "leadership"],
		"vocabulary_level": "advanced",
		"engagement_patterns": "questions drive engagement",
		"structural_patterns": ["short paragraphs"],
		"emoji_usage": "minimal",
		"hashtag_usage": "1-2 per post",
		"call_to_action_style": "open-ended questions",
		"unique_quirks": ["bold statements"],
		"voice_summary": "A thoughtful tech leader."
	}`

	mock := &mockMessageSender{response: personaJSON}
	analyzer := NewAnalyzer(mock)

	profile, err := analyzer.BuildProfile(context.Background(), samplePosts())
	if err != nil {
		t.Fatalf("BuildProfile() error = %v", err)
	}
	if profile.WritingTone != "professional yet approachable" {
		t.Errorf("WritingTone = %q, want 'professional yet approachable'", profile.WritingTone)
	}
	if len(profile.CommonThemes) != 3 {
		t.Errorf("CommonThemes has %d items, want 3", len(profile.CommonThemes))
	}
	if profile.VocabularyLevel != "advanced" {
		t.Errorf("VocabularyLevel = %q, want 'advanced'", profile.VocabularyLevel)
	}
	if profile.VoiceSummary != "A thoughtful tech leader." {
		t.Errorf("VoiceSummary = %q, want 'A thoughtful tech leader.'", profile.VoiceSummary)
	}
}

func TestBuildProfile_MarkdownWrappedJSON(t *testing.T) {
	personaJSON := "```json\n" + `{
		"writing_tone": "casual",
		"typical_length": "short",
		"common_themes": ["tech"],
		"vocabulary_level": "simple",
		"engagement_patterns": "asks questions",
		"structural_patterns": ["single posts"],
		"emoji_usage": "heavy",
		"hashtag_usage": "none",
		"call_to_action_style": "direct",
		"unique_quirks": ["uses slang"],
		"voice_summary": "Casual tech voice."
	}` + "\n```"

	mock := &mockMessageSender{response: personaJSON}
	analyzer := NewAnalyzer(mock)

	profile, err := analyzer.BuildProfile(context.Background(), samplePosts())
	if err != nil {
		t.Fatalf("BuildProfile() error = %v", err)
	}
	if profile.WritingTone != "casual" {
		t.Errorf("WritingTone = %q, want 'casual'", profile.WritingTone)
	}
}

func TestBuildProfile_MarkdownWrappedGenericCodeBlock(t *testing.T) {
	personaJSON := "```\n" + `{
		"writing_tone": "witty",
		"typical_length": "medium",
		"common_themes": ["humor"],
		"vocabulary_level": "moderate",
		"engagement_patterns": "viral hooks",
		"structural_patterns": ["threads"],
		"emoji_usage": "moderate",
		"hashtag_usage": "sparse",
		"call_to_action_style": "indirect",
		"unique_quirks": ["puns"],
		"voice_summary": "Witty and humorous."
	}` + "\n```"

	mock := &mockMessageSender{response: personaJSON}
	analyzer := NewAnalyzer(mock)

	profile, err := analyzer.BuildProfile(context.Background(), samplePosts())
	if err != nil {
		t.Fatalf("BuildProfile() error = %v", err)
	}
	if profile.WritingTone != "witty" {
		t.Errorf("WritingTone = %q, want 'witty'", profile.WritingTone)
	}
}

func TestBuildProfile_EmptyPosts(t *testing.T) {
	mock := &mockMessageSender{response: "{}"}
	analyzer := NewAnalyzer(mock)

	_, err := analyzer.BuildProfile(context.Background(), nil)
	if err == nil {
		t.Fatal("BuildProfile() expected error for empty posts, got nil")
	}
	if !strings.Contains(err.Error(), "no posts provided") {
		t.Errorf("expected error about no posts, got: %v", err)
	}
}

func TestBuildProfile_EmptyPostSlice(t *testing.T) {
	mock := &mockMessageSender{response: "{}"}
	analyzer := NewAnalyzer(mock)

	_, err := analyzer.BuildProfile(context.Background(), []models.Post{})
	if err == nil {
		t.Fatal("BuildProfile() expected error for empty post slice, got nil")
	}
}

func TestBuildProfile_InvalidJSON(t *testing.T) {
	mock := &mockMessageSender{response: "this is not valid json at all"}
	analyzer := NewAnalyzer(mock)

	_, err := analyzer.BuildProfile(context.Background(), samplePosts())
	if err == nil {
		t.Fatal("BuildProfile() expected error for invalid JSON response, got nil")
	}
}

func TestBuildProfile_ClientError(t *testing.T) {
	mock := &mockMessageSender{err: errors.New("API error")}
	analyzer := NewAnalyzer(mock)

	_, err := analyzer.BuildProfile(context.Background(), samplePosts())
	if err == nil {
		t.Fatal("BuildProfile() expected error when client fails, got nil")
	}
	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("expected error to contain 'API error', got: %v", err)
	}
}

func TestFormatPosts(t *testing.T) {
	posts := samplePosts()
	result := formatPosts(posts)

	if !strings.Contains(result, "Analyze the following posts") {
		t.Error("formatPosts() should contain intro text")
	}
	if !strings.Contains(result, posts[0].Content) {
		t.Error("formatPosts() should contain first post content")
	}
	if !strings.Contains(result, posts[1].Content) {
		t.Error("formatPosts() should contain second post content")
	}
	if !strings.Contains(result, "Likes: 500") {
		t.Error("formatPosts() should contain first post metrics")
	}
	if !strings.Contains(result, "[x]") {
		t.Error("formatPosts() should contain platform tag")
	}
}

func TestStripMarkdownJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain JSON",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "json code block",
			input: "```json\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:  "generic code block",
			input: "```\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:  "with whitespace",
			input: "  \n{\"key\": \"value\"}\n  ",
			want:  `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMarkdownJSON(tt.input)
			if got != tt.want {
				t.Errorf("stripMarkdownJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}
