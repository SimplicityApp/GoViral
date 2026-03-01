package linkedin

import (
	"errors"
	"testing"
	"time"
)

func TestParseLinkitinPosts_ValidData(t *testing.T) {
	result := map[string]interface{}{
		"posts": []interface{}{
			map[string]interface{}{
				"urn":         "urn:li:activity:123",
				"text":        "Hello world",
				"likes":       42,
				"comments":    5,
				"reposts":     3,
				"impressions": 1000,
				"created_at":  "2025-01-15T10:30:00Z",
				"author": map[string]interface{}{
					"urn":        "urn:li:person:456",
					"first_name": "John",
					"last_name":  "Doe",
					"headline":   "Software Engineer",
				},
			},
			map[string]interface{}{
				"urn":         "urn:li:activity:789",
				"text":        "Another post",
				"likes":       10,
				"comments":    2,
				"reposts":     1,
				"impressions": 500,
				"created_at":  "",
			},
		},
	}

	posts, err := parseLinkitinPosts(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}

	p := posts[0]
	if p.PlatformPostID != "urn:li:activity:123" {
		t.Errorf("expected urn:li:activity:123, got %s", p.PlatformPostID)
	}
	if p.Content != "Hello world" {
		t.Errorf("expected 'Hello world', got %s", p.Content)
	}
	if p.Likes != 42 {
		t.Errorf("expected 42 likes, got %d", p.Likes)
	}
	if p.Comments != 5 {
		t.Errorf("expected 5 comments, got %d", p.Comments)
	}
	if p.Reposts != 3 {
		t.Errorf("expected 3 reposts, got %d", p.Reposts)
	}
	if p.Impressions != 1000 {
		t.Errorf("expected 1000 impressions, got %d", p.Impressions)
	}
	if p.Platform != "linkedin" {
		t.Errorf("expected linkedin platform, got %s", p.Platform)
	}
	expectedTime, _ := time.Parse(time.RFC3339, "2025-01-15T10:30:00Z")
	if !p.PostedAt.Equal(expectedTime) {
		t.Errorf("expected posted at %v, got %v", expectedTime, p.PostedAt)
	}

	// Second post: no created_at should result in zero time.
	p2 := posts[1]
	if p2.PlatformPostID != "urn:li:activity:789" {
		t.Errorf("expected urn:li:activity:789, got %s", p2.PlatformPostID)
	}
	if !p2.PostedAt.IsZero() {
		t.Errorf("expected zero time for empty created_at, got %v", p2.PostedAt)
	}
}

func TestIsLinkitinAuthError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"unrelated error", errors.New("network timeout"), false},
		{"jsessionid not set", errors.New("JSESSIONID not set - call set_cookies() or login first"), true},
		{"not logged in", errors.New("linkitin: not logged in"), true},
		{"login first", errors.New("please login first"), true},
		{"session expired", errors.New("session expired, re-authenticate"), true},
		{"cookies are expired", errors.New("cookies are expired"), true},
		{"status 401", errors.New("HTTP status code: 401"), true},
		{"status 403", errors.New("HTTP status code: 403"), true},
		{"wrapped auth error", errors.New("creating LinkedIn post: JSESSIONID not set"), true},
		{"case insensitive", errors.New("SESSION EXPIRED"), true},
		{"status 500 not auth", errors.New("status code: 500"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLinkitinAuthError(tt.err)
			if got != tt.want {
				t.Errorf("IsLinkitinAuthError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestParseLinkitinPosts_MissingPostsField(t *testing.T) {
	result := map[string]interface{}{
		"status": "ok",
	}

	_, err := parseLinkitinPosts(result)
	if err == nil {
		t.Fatal("expected error for missing posts field, got nil")
	}
}

func TestParseLinkitinPosts_EmptyPosts(t *testing.T) {
	result := map[string]interface{}{
		"posts": []interface{}{},
	}

	posts, err := parseLinkitinPosts(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(posts))
	}
}
