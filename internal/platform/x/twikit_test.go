package x

import (
	"os"
	"testing"
	"time"
)

func TestParseTwikitOutput_ValidJSON(t *testing.T) {
	data, err := os.ReadFile("testdata/twikit_response.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	posts, err := parseTwikitOutput(data)
	if err != nil {
		t.Fatalf("parseTwikitOutput returned error: %v", err)
	}

	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}

	p := posts[0]
	if p.PlatformPostID != "1234567890" {
		t.Errorf("expected ID 1234567890, got %s", p.PlatformPostID)
	}
	if p.Content != "Hello world! This is a test tweet." {
		t.Errorf("unexpected content: %s", p.Content)
	}
	if p.Platform != "x" {
		t.Errorf("expected platform x, got %s", p.Platform)
	}
	if p.Likes != 42 {
		t.Errorf("expected 42 likes, got %d", p.Likes)
	}
	if p.Reposts != 10 {
		t.Errorf("expected 10 reposts, got %d", p.Reposts)
	}
	if p.Comments != 5 {
		t.Errorf("expected 5 comments, got %d", p.Comments)
	}
	if p.Impressions != 1500 {
		t.Errorf("expected 1500 impressions, got %d", p.Impressions)
	}

	// Verify date parsing: "Wed Jan 15 14:30:00 +0000 2025"
	expected := time.Date(2025, time.January, 15, 14, 30, 0, 0, time.UTC)
	if !p.PostedAt.Equal(expected) {
		t.Errorf("expected PostedAt %v, got %v", expected, p.PostedAt)
	}

	if p.FetchedAt.IsZero() {
		t.Error("FetchedAt should not be zero")
	}
}

func TestParseTwikitOutput_ErrorJSON(t *testing.T) {
	data := []byte(`{"error": "user not found"}`)
	_, err := parseTwikitOutput(data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "twikit: user not found" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseTwikitOutput_InvalidJSON(t *testing.T) {
	data := []byte(`not json at all`)
	_, err := parseTwikitOutput(data)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseTwikitOutput_EmptyTweets(t *testing.T) {
	data := []byte(`{"tweets": []}`)
	posts, err := parseTwikitOutput(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(posts))
	}
}

func TestParseTwikitTrendingOutput_ValidJSON(t *testing.T) {
	data, err := os.ReadFile("testdata/twikit_trending_response.json")
	if err != nil {
		t.Fatalf("reading test fixture: %v", err)
	}

	posts, err := parseTwikitTrendingOutput(data)
	if err != nil {
		t.Fatalf("parseTwikitTrendingOutput returned error: %v", err)
	}

	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}

	p := posts[0]
	if p.PlatformPostID != "9876543210" {
		t.Errorf("expected ID 9876543210, got %s", p.PlatformPostID)
	}
	if p.AuthorUsername != "techguru" {
		t.Errorf("expected author techguru, got %s", p.AuthorUsername)
	}
	if p.AuthorName != "Tech Guru" {
		t.Errorf("expected author name Tech Guru, got %s", p.AuthorName)
	}
	if p.Likes != 5000 {
		t.Errorf("expected 5000 likes, got %d", p.Likes)
	}
	if p.Impressions != 250000 {
		t.Errorf("expected 250000 impressions, got %d", p.Impressions)
	}
	if len(p.NicheTags) != 1 || p.NicheTags[0] != "AI" {
		t.Errorf("expected niche tags [AI], got %v", p.NicheTags)
	}
	if p.Platform != "x" {
		t.Errorf("expected platform x, got %s", p.Platform)
	}
}

func TestParseTwikitTrendingOutput_ErrorJSON(t *testing.T) {
	data := []byte(`{"error": "search failed"}`)
	_, err := parseTwikitTrendingOutput(data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseTwikitTrendingOutput_EmptyResults(t *testing.T) {
	data := []byte(`{"trending": []}`)
	posts, err := parseTwikitTrendingOutput(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(posts))
	}
}
