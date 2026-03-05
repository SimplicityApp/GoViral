package x

import (
	"context"
	"fmt"
	"testing"

	"github.com/shuhao/goviral/pkg/models"
)

// mockFetcher implements the fetcher interface for testing.
type mockFetcher struct {
	posts         []models.Post
	trendingPosts []models.TrendingPost
	err           error
}

func (m *mockFetcher) FetchMyPosts(_ context.Context, _ int) ([]models.Post, error) {
	return m.posts, m.err
}

func (m *mockFetcher) FetchTrendingPosts(_ context.Context, _ []string, _ string, _ int, _ int) ([]models.TrendingPost, error) {
	return m.trendingPosts, m.err
}

func TestFallbackClient_PrimarySucceeds_TwikitNotCalled(t *testing.T) {
	expectedPosts := []models.Post{{PlatformPostID: "primary-1", Platform: "x"}}
	primary := &mockFetcher{posts: expectedPosts}
	twikit := &mockFetcher{err: fmt.Errorf("should not be called")}

	fc := &FallbackClient{primary: primary, twikit: twikit}
	posts, err := fc.FetchMyPosts(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 || posts[0].PlatformPostID != "primary-1" {
		t.Errorf("expected primary posts, got %v", posts)
	}
}

func TestFallbackClient_PrimaryFails_TwikitSucceeds(t *testing.T) {
	primary := &mockFetcher{err: fmt.Errorf("API error")}
	twikitPosts := []models.Post{{PlatformPostID: "twikit-1", Platform: "x"}}
	twikit := &mockFetcher{posts: twikitPosts}

	fc := &FallbackClient{primary: primary, twikit: twikit}
	posts, err := fc.FetchMyPosts(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 || posts[0].PlatformPostID != "twikit-1" {
		t.Errorf("expected twikit posts, got %v", posts)
	}
}

func TestFallbackClient_BothFail(t *testing.T) {
	primary := &mockFetcher{err: fmt.Errorf("primary error")}
	twikit := &mockFetcher{err: fmt.Errorf("twikit error")}

	fc := &FallbackClient{primary: primary, twikit: twikit}
	_, err := fc.FetchMyPosts(context.Background(), 10)
	if err == nil {
		t.Fatal("expected error when both fail, got nil")
	}
	errMsg := err.Error()
	if errMsg != "primary API failed: primary error; twikit fallback also failed: twikit error" {
		t.Errorf("unexpected error message: %s", errMsg)
	}
}

func TestFallbackClient_TwikitNil(t *testing.T) {
	primary := &mockFetcher{err: fmt.Errorf("primary error")}

	fc := &FallbackClient{primary: primary, twikit: nil}
	_, err := fc.FetchMyPosts(context.Background(), 10)
	if err == nil {
		t.Fatal("expected error when twikit is nil, got nil")
	}
	errMsg := err.Error()
	if errMsg != "X cookies not configured — sync your X cookies via the browser extension or provide them manually in Settings" {
		t.Errorf("unexpected error message: %s", errMsg)
	}
}

func TestFallbackClient_FetchTrendingPosts_UsesPrimary(t *testing.T) {
	expectedPosts := []models.TrendingPost{{PlatformPostID: "trending-1", Platform: "x"}}
	primary := &mockFetcher{trendingPosts: expectedPosts}
	twikit := &mockFetcher{err: fmt.Errorf("should not be called")}

	fc := &FallbackClient{primary: primary, twikit: twikit}
	posts, err := fc.FetchTrendingPosts(context.Background(), []string{"tech"}, "day", 100, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 || posts[0].PlatformPostID != "trending-1" {
		t.Errorf("expected primary trending posts, got %v", posts)
	}
}

func TestFallbackClient_FetchTrendingPosts_PrimaryFails_TwikitSucceeds(t *testing.T) {
	primary := &mockFetcher{err: fmt.Errorf("API error")}
	twikitPosts := []models.TrendingPost{{PlatformPostID: "twikit-1", Platform: "x"}}
	twikit := &mockFetcher{trendingPosts: twikitPosts}

	fc := &FallbackClient{primary: primary, twikit: twikit}
	posts, err := fc.FetchTrendingPosts(context.Background(), []string{"tech"}, "day", 100, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 || posts[0].PlatformPostID != "twikit-1" {
		t.Errorf("expected twikit trending posts, got %v", posts)
	}
}

func TestFallbackClient_FetchTrendingPosts_TwikitNil(t *testing.T) {
	primary := &mockFetcher{err: fmt.Errorf("API error")}

	fc := &FallbackClient{primary: primary, twikit: nil}
	_, err := fc.FetchTrendingPosts(context.Background(), []string{"tech"}, "day", 100, 10)
	if err == nil {
		t.Fatal("expected error when twikit is nil")
	}
}

func TestFallbackClient_PrimaryDisabled_SkipsPrimary(t *testing.T) {
	primary := &mockFetcher{err: fmt.Errorf("X API returned status 402: CreditsDepleted")}
	twikitPosts := []models.Post{{PlatformPostID: "twikit-1", Platform: "x"}}
	twikit := &mockFetcher{posts: twikitPosts}

	fc := &FallbackClient{primary: primary, twikit: twikit}

	// First call: primary fails with 402, falls back to twikit, disables primary.
	posts, err := fc.FetchMyPosts(context.Background(), 10)
	if err != nil {
		t.Fatalf("first call unexpected error: %v", err)
	}
	if posts[0].PlatformPostID != "twikit-1" {
		t.Errorf("expected twikit posts on first call")
	}
	if !fc.primaryDisabled {
		t.Fatal("expected primaryDisabled to be true after 402")
	}

	// Second call: should go straight to twikit without touching primary.
	primary.err = fmt.Errorf("should not be called")
	posts, err = fc.FetchMyPosts(context.Background(), 10)
	if err != nil {
		t.Fatalf("second call unexpected error: %v", err)
	}
	if posts[0].PlatformPostID != "twikit-1" {
		t.Errorf("expected twikit posts on second call")
	}
}

func TestFallbackClient_NonAccountError_DoesNotDisable(t *testing.T) {
	primary := &mockFetcher{err: fmt.Errorf("network timeout")}
	twikitPosts := []models.Post{{PlatformPostID: "twikit-1", Platform: "x"}}
	twikit := &mockFetcher{posts: twikitPosts}

	fc := &FallbackClient{primary: primary, twikit: twikit}
	_, _ = fc.FetchMyPosts(context.Background(), 10)

	if fc.primaryDisabled {
		t.Fatal("expected primaryDisabled to be false for non-account errors")
	}
}
