package generator

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/shuhao/goviral/pkg/models"
)

// mockMessageSender implements claude.MessageSender for testing.
type mockMessageSender struct {
	response    string
	err         error
	lastSystem  string
	lastMessage string
}

func (m *mockMessageSender) SendMessage(ctx context.Context, systemPrompt string, userMessage string) (string, error) {
	m.lastSystem = systemPrompt
	m.lastMessage = userMessage
	return m.response, m.err
}

func sampleRequest() models.GenerateRequest {
	return models.GenerateRequest{
		TrendingPost: models.TrendingPost{
			Platform:       "x",
			PlatformPostID: "2001",
			AuthorUsername: "techguru",
			AuthorName:     "Tech Guru",
			Content:        "AI is changing everything!",
			Likes:          5000,
			Reposts:        1200,
			Comments:       300,
			Impressions:    100000,
		},
		Persona: models.Persona{
			Platform: "x",
			Profile: models.PersonaProfile{
				WritingTone:  "professional",
				VoiceSummary: "Thoughtful tech leader.",
			},
		},
		TargetPlatform: "x",
		Niches:         []string{"tech", "AI"},
		Count:          2,
	}
}

func TestGenerate_Success(t *testing.T) {
	generatedJSON := `[
		{
			"content": "Hot take: AI isn't replacing devs, it's making the best ones even better.",
			"viral_mechanic": "contrarian hook",
			"confidence_score": 8
		},
		{
			"content": "Everyone talks about AI. Few talk about the humans behind it.",
			"viral_mechanic": "curiosity gap",
			"confidence_score": 7
		}
	]`

	mock := &mockMessageSender{response: generatedJSON}
	gen := NewGenerator(mock)

	results, err := gen.Generate(context.Background(), sampleRequest())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Generate() returned %d results, want 2", len(results))
	}
	if results[0].ConfidenceScore != 8 {
		t.Errorf("results[0].ConfidenceScore = %d, want 8", results[0].ConfidenceScore)
	}
	if results[0].ViralMechanic != "contrarian hook" {
		t.Errorf("results[0].ViralMechanic = %q, want 'contrarian hook'", results[0].ViralMechanic)
	}
	if results[0].Content == "" {
		t.Error("results[0].Content should not be empty")
	}
	if results[1].ConfidenceScore != 7 {
		t.Errorf("results[1].ConfidenceScore = %d, want 7", results[1].ConfidenceScore)
	}
}

func TestGenerate_MarkdownWrappedJSON(t *testing.T) {
	generatedJSON := "```json\n" + `[{"content": "test content", "viral_mechanic": "hook", "confidence_score": 5}]` + "\n```"

	mock := &mockMessageSender{response: generatedJSON}
	gen := NewGenerator(mock)

	results, err := gen.Generate(context.Background(), sampleRequest())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Generate() returned %d results, want 1", len(results))
	}
	if results[0].Content != "test content" {
		t.Errorf("results[0].Content = %q, want 'test content'", results[0].Content)
	}
}

func TestGenerate_GenericCodeBlock(t *testing.T) {
	generatedJSON := "```\n" + `[{"content": "abc", "viral_mechanic": "def", "confidence_score": 6}]` + "\n```"

	mock := &mockMessageSender{response: generatedJSON}
	gen := NewGenerator(mock)

	results, err := gen.Generate(context.Background(), sampleRequest())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Generate() returned %d results, want 1", len(results))
	}
	if results[0].ConfidenceScore != 6 {
		t.Errorf("results[0].ConfidenceScore = %d, want 6", results[0].ConfidenceScore)
	}
}

func TestGenerate_PromptConstruction(t *testing.T) {
	mock := &mockMessageSender{response: `[{"content":"x","viral_mechanic":"y","confidence_score":5}]`}
	gen := NewGenerator(mock)

	req := sampleRequest()
	_, err := gen.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify system prompt is the generator's system prompt
	if mock.lastSystem == "" {
		t.Error("system prompt should not be empty")
	}

	// Verify user message contains expected elements
	if !strings.Contains(mock.lastMessage, "techguru") {
		t.Error("user message should contain author username")
	}
	if !strings.Contains(mock.lastMessage, "Tech Guru") {
		t.Error("user message should contain author name")
	}
	if !strings.Contains(mock.lastMessage, "AI is changing everything!") {
		t.Error("user message should contain trending post content")
	}
	if !strings.Contains(mock.lastMessage, "Generate 2 variations") {
		t.Error("user message should contain variation count")
	}
	if !strings.Contains(mock.lastMessage, "tech") {
		t.Error("user message should contain niche")
	}
	if !strings.Contains(mock.lastMessage, "Persona Profile") {
		t.Error("user message should contain persona section")
	}
}

func TestGenerate_InvalidJSON(t *testing.T) {
	mock := &mockMessageSender{response: "this is not valid json"}
	gen := NewGenerator(mock)

	_, err := gen.Generate(context.Background(), sampleRequest())
	if err == nil {
		t.Fatal("Generate() expected error for invalid JSON, got nil")
	}
}

func TestGenerate_ClientError(t *testing.T) {
	mock := &mockMessageSender{err: errors.New("API connection error")}
	gen := NewGenerator(mock)

	_, err := gen.Generate(context.Background(), sampleRequest())
	if err == nil {
		t.Fatal("Generate() expected error when client fails, got nil")
	}
	if !strings.Contains(err.Error(), "API connection error") {
		t.Errorf("expected error to contain 'API connection error', got: %v", err)
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
			input: `[{"key": "value"}]`,
			want:  `[{"key": "value"}]`,
		},
		{
			name:  "json code block",
			input: "```json\n[{\"key\": \"value\"}]\n```",
			want:  `[{"key": "value"}]`,
		},
		{
			name:  "generic code block",
			input: "```\n[{\"key\": \"value\"}]\n```",
			want:  `[{"key": "value"}]`,
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
