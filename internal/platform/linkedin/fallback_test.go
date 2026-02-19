package linkedin

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shuhao/goviral/pkg/models"
)

// mockLinkedinFetcher implements the linkedinFetcher interface for testing.
type mockLinkedinFetcher struct {
	posts         []models.Post
	trendingPosts []models.TrendingPost
	err           error
}

func (m *mockLinkedinFetcher) FetchMyPosts(_ context.Context, _ int) ([]models.Post, error) {
	return m.posts, m.err
}

func (m *mockLinkedinFetcher) FetchTrendingPosts(_ context.Context, _ []string, _ string, _ int, _ int) ([]models.TrendingPost, error) {
	return m.trendingPosts, m.err
}

// mockLinkedinPoster implements the linkedinPoster interface for testing.
type mockLinkedinPoster struct {
	postID string
	err    error
}

func (m *mockLinkedinPoster) CreatePost(_ context.Context, _ string) (string, error) {
	return m.postID, m.err
}

func (m *mockLinkedinPoster) UploadImage(_ context.Context, _ []byte, _ string) (string, error) {
	return m.postID, m.err
}

func (m *mockLinkedinPoster) CreatePostWithImage(_ context.Context, _ string, _ []byte, _ string) (string, error) {
	return m.postID, m.err
}

func (m *mockLinkedinPoster) Repost(_ context.Context, _ string, _ string) (string, error) {
	return m.postID, m.err
}

func (m *mockLinkedinPoster) CreateScheduledPost(_ context.Context, _ string, _ time.Time) (string, error) {
	return m.postID, m.err
}

func (m *mockLinkedinPoster) CreateScheduledPostWithImage(_ context.Context, _ string, _ []byte, _ string, _ time.Time) (string, error) {
	return m.postID, m.err
}

func TestFallbackClient_PrimarySucceeds_LikitNotCalled(t *testing.T) {
	expectedPosts := []models.Post{{PlatformPostID: "primary-1", Platform: "linkedin"}}
	primary := &mockLinkedinFetcher{posts: expectedPosts}
	likit := &mockLinkedinFetcher{err: fmt.Errorf("should not be called")}

	fc := &FallbackClient{primary: primary, likit: likit}
	posts, err := fc.FetchMyPosts(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 || posts[0].PlatformPostID != "primary-1" {
		t.Errorf("expected primary posts, got %v", posts)
	}
}

func TestFallbackClient_PrimaryFails_LikitSucceeds(t *testing.T) {
	primary := &mockLinkedinFetcher{err: fmt.Errorf("API error")}
	likitPosts := []models.Post{{PlatformPostID: "likit-1", Platform: "linkedin"}}
	likit := &mockLinkedinFetcher{posts: likitPosts}

	fc := &FallbackClient{primary: primary, likit: likit}
	posts, err := fc.FetchMyPosts(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 || posts[0].PlatformPostID != "likit-1" {
		t.Errorf("expected likit posts, got %v", posts)
	}
}

func TestFallbackClient_BothFail(t *testing.T) {
	primary := &mockLinkedinFetcher{err: fmt.Errorf("primary error")}
	likit := &mockLinkedinFetcher{err: fmt.Errorf("likit error")}

	fc := &FallbackClient{primary: primary, likit: likit}
	_, err := fc.FetchMyPosts(context.Background(), 10)
	if err == nil {
		t.Fatal("expected error when both fail, got nil")
	}
	errMsg := err.Error()
	if errMsg != "official API failed: primary error; likit fallback also failed: likit error" {
		t.Errorf("unexpected error message: %s", errMsg)
	}
}

func TestFallbackClient_LikitNil(t *testing.T) {
	primary := &mockLinkedinFetcher{err: fmt.Errorf("primary error")}

	fc := &FallbackClient{primary: primary, likit: nil}
	_, err := fc.FetchMyPosts(context.Background(), 10)
	if err == nil {
		t.Fatal("expected error when likit is nil, got nil")
	}
	errMsg := err.Error()
	if errMsg != "official LinkedIn API failed: primary error (likit fallback unavailable)" {
		t.Errorf("unexpected error message: %s", errMsg)
	}
}

func TestFallbackClient_FetchTrendingPosts_UsesPrimary(t *testing.T) {
	expectedPosts := []models.TrendingPost{{PlatformPostID: "trending-1", Platform: "linkedin"}}
	primary := &mockLinkedinFetcher{trendingPosts: expectedPosts}
	likit := &mockLinkedinFetcher{err: fmt.Errorf("should not be called")}

	fc := &FallbackClient{primary: primary, likit: likit}
	posts, err := fc.FetchTrendingPosts(context.Background(), []string{"tech"}, "day", 100, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 || posts[0].PlatformPostID != "trending-1" {
		t.Errorf("expected primary trending posts, got %v", posts)
	}
}

func TestFallbackClient_FetchTrendingPosts_PrimaryFails_LikitSucceeds(t *testing.T) {
	primary := &mockLinkedinFetcher{err: fmt.Errorf("API error")}
	likitPosts := []models.TrendingPost{{PlatformPostID: "likit-1", Platform: "linkedin"}}
	likit := &mockLinkedinFetcher{trendingPosts: likitPosts}

	fc := &FallbackClient{primary: primary, likit: likit}
	posts, err := fc.FetchTrendingPosts(context.Background(), []string{"tech"}, "day", 100, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 || posts[0].PlatformPostID != "likit-1" {
		t.Errorf("expected likit trending posts, got %v", posts)
	}
}

func TestFallbackClient_FetchTrendingPosts_LikitNil(t *testing.T) {
	primary := &mockLinkedinFetcher{err: fmt.Errorf("API error")}

	fc := &FallbackClient{primary: primary, likit: nil}
	_, err := fc.FetchTrendingPosts(context.Background(), []string{"tech"}, "day", 100, 10)
	if err == nil {
		t.Fatal("expected error when likit is nil")
	}
}

func TestFallbackClient_PrimaryDisabled_SkipsPrimary(t *testing.T) {
	primary := &mockLinkedinFetcher{err: fmt.Errorf("status 401: Unauthorized")}
	likitPosts := []models.Post{{PlatformPostID: "likit-1", Platform: "linkedin"}}
	likit := &mockLinkedinFetcher{posts: likitPosts}

	fc := &FallbackClient{primary: primary, likit: likit}

	// First call: primary fails with 401, falls back to likit, disables primary.
	posts, err := fc.FetchMyPosts(context.Background(), 10)
	if err != nil {
		t.Fatalf("first call unexpected error: %v", err)
	}
	if posts[0].PlatformPostID != "likit-1" {
		t.Errorf("expected likit posts on first call")
	}
	if !fc.primaryDisabled {
		t.Fatal("expected primaryDisabled to be true after 401")
	}

	// Second call: should go straight to likit without touching primary.
	primary.err = fmt.Errorf("should not be called")
	posts, err = fc.FetchMyPosts(context.Background(), 10)
	if err != nil {
		t.Fatalf("second call unexpected error: %v", err)
	}
	if posts[0].PlatformPostID != "likit-1" {
		t.Errorf("expected likit posts on second call")
	}
}

func TestFallbackClient_NonAccountError_DoesNotDisable(t *testing.T) {
	primary := &mockLinkedinFetcher{err: fmt.Errorf("network timeout")}
	likitPosts := []models.Post{{PlatformPostID: "likit-1", Platform: "linkedin"}}
	likit := &mockLinkedinFetcher{posts: likitPosts}

	fc := &FallbackClient{primary: primary, likit: likit}
	_, _ = fc.FetchMyPosts(context.Background(), 10)

	if fc.primaryDisabled {
		t.Fatal("expected primaryDisabled to be false for non-account errors")
	}
}

func TestFallbackClient_CreatePost_Success(t *testing.T) {
	poster := &mockLinkedinPoster{postID: "urn:li:share:123"}
	fc := &FallbackClient{
		primary:     &mockLinkedinFetcher{},
		likitPoster: poster,
	}

	urn, err := fc.CreatePost(context.Background(), "Hello LinkedIn!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if urn != "urn:li:share:123" {
		t.Errorf("expected urn:li:share:123, got %s", urn)
	}
}

func TestFallbackClient_CreatePost_PosterUnavailable(t *testing.T) {
	fc := &FallbackClient{
		primary:     &mockLinkedinFetcher{},
		likitPoster: nil,
	}

	_, err := fc.CreatePost(context.Background(), "Hello LinkedIn!")
	if err == nil {
		t.Fatal("expected error when poster is nil, got nil")
	}
}

func TestFallbackClient_UploadImage_Success(t *testing.T) {
	poster := &mockLinkedinPoster{postID: "urn:li:image:456"}
	fc := &FallbackClient{
		primary:     &mockLinkedinFetcher{},
		likitPoster: poster,
	}

	mediaURN, err := fc.UploadImage(context.Background(), []byte("image-data"), "test.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mediaURN != "urn:li:image:456" {
		t.Errorf("expected urn:li:image:456, got %s", mediaURN)
	}
}

func TestFallbackClient_UploadImage_PosterUnavailable(t *testing.T) {
	fc := &FallbackClient{
		primary:     &mockLinkedinFetcher{},
		likitPoster: nil,
	}

	_, err := fc.UploadImage(context.Background(), []byte("image-data"), "test.png")
	if err == nil {
		t.Fatal("expected error when poster is nil, got nil")
	}
}

func TestFallbackClient_CreatePostWithImage_Success(t *testing.T) {
	poster := &mockLinkedinPoster{postID: "urn:li:share:789"}
	fc := &FallbackClient{
		primary:     &mockLinkedinFetcher{},
		likitPoster: poster,
	}

	urn, err := fc.CreatePostWithImage(context.Background(), "Post with image", []byte("img"), "photo.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if urn != "urn:li:share:789" {
		t.Errorf("expected urn:li:share:789, got %s", urn)
	}
}

func TestFallbackClient_CreatePostWithImage_PosterUnavailable(t *testing.T) {
	fc := &FallbackClient{
		primary:     &mockLinkedinFetcher{},
		likitPoster: nil,
	}

	_, err := fc.CreatePostWithImage(context.Background(), "Post with image", []byte("img"), "photo.jpg")
	if err == nil {
		t.Fatal("expected error when poster is nil, got nil")
	}
}
