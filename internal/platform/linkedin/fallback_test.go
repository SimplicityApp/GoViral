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

func (m *mockLinkedinPoster) CreateComment(_ context.Context, _ string, _ string) (string, error) {
	return m.postID, m.err
}

func (m *mockLinkedinPoster) CreateScheduledPost(_ context.Context, _ string, _ time.Time) (string, error) {
	return m.postID, m.err
}

func (m *mockLinkedinPoster) CreateScheduledPostWithImage(_ context.Context, _ string, _ []byte, _ string, _ time.Time) (string, error) {
	return m.postID, m.err
}

func TestFallbackClient_PrimarySucceeds_LinkitinNotCalled(t *testing.T) {
	expectedPosts := []models.Post{{PlatformPostID: "primary-1", Platform: "linkedin"}}
	primary := &mockLinkedinFetcher{posts: expectedPosts}
	linkitin := &mockLinkedinFetcher{err: fmt.Errorf("should not be called")}

	fc := &FallbackClient{primary: primary, linkitin: linkitin}
	posts, err := fc.FetchMyPosts(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 || posts[0].PlatformPostID != "primary-1" {
		t.Errorf("expected primary posts, got %v", posts)
	}
}

func TestFallbackClient_PrimaryFails_LinkitinSucceeds(t *testing.T) {
	primary := &mockLinkedinFetcher{err: fmt.Errorf("API error")}
	linkitinPosts := []models.Post{{PlatformPostID: "linkitin-1", Platform: "linkedin"}}
	linkitin := &mockLinkedinFetcher{posts: linkitinPosts}

	fc := &FallbackClient{primary: primary, linkitin: linkitin}
	posts, err := fc.FetchMyPosts(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 || posts[0].PlatformPostID != "linkitin-1" {
		t.Errorf("expected linkitin posts, got %v", posts)
	}
}

func TestFallbackClient_BothFail(t *testing.T) {
	primary := &mockLinkedinFetcher{err: fmt.Errorf("primary error")}
	linkitin := &mockLinkedinFetcher{err: fmt.Errorf("linkitin error")}

	fc := &FallbackClient{primary: primary, linkitin: linkitin}
	_, err := fc.FetchMyPosts(context.Background(), 10)
	if err == nil {
		t.Fatal("expected error when both fail, got nil")
	}
	errMsg := err.Error()
	if errMsg != "official API failed: primary error; linkitin fallback also failed: linkitin error" {
		t.Errorf("unexpected error message: %s", errMsg)
	}
}

func TestFallbackClient_LinkitinNil(t *testing.T) {
	primary := &mockLinkedinFetcher{err: fmt.Errorf("primary error")}

	fc := &FallbackClient{primary: primary, linkitin: nil}
	_, err := fc.FetchMyPosts(context.Background(), 10)
	if err == nil {
		t.Fatal("expected error when linkitin is nil, got nil")
	}
	errMsg := err.Error()
	if errMsg != "official LinkedIn API failed: primary error (linkitin fallback unavailable)" {
		t.Errorf("unexpected error message: %s", errMsg)
	}
}

func TestFallbackClient_FetchTrendingPosts_UsesPrimary(t *testing.T) {
	expectedPosts := []models.TrendingPost{{PlatformPostID: "trending-1", Platform: "linkedin"}}
	primary := &mockLinkedinFetcher{trendingPosts: expectedPosts}
	linkitin := &mockLinkedinFetcher{err: fmt.Errorf("should not be called")}

	fc := &FallbackClient{primary: primary, linkitin: linkitin}
	posts, err := fc.FetchTrendingPosts(context.Background(), []string{"tech"}, "day", 100, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 || posts[0].PlatformPostID != "trending-1" {
		t.Errorf("expected primary trending posts, got %v", posts)
	}
}

func TestFallbackClient_FetchTrendingPosts_PrimaryFails_LinkitinSucceeds(t *testing.T) {
	primary := &mockLinkedinFetcher{err: fmt.Errorf("API error")}
	linkitinPosts := []models.TrendingPost{{PlatformPostID: "linkitin-1", Platform: "linkedin"}}
	linkitin := &mockLinkedinFetcher{trendingPosts: linkitinPosts}

	fc := &FallbackClient{primary: primary, linkitin: linkitin}
	posts, err := fc.FetchTrendingPosts(context.Background(), []string{"tech"}, "day", 100, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 || posts[0].PlatformPostID != "linkitin-1" {
		t.Errorf("expected linkitin trending posts, got %v", posts)
	}
}

func TestFallbackClient_FetchTrendingPosts_LinkitinNil(t *testing.T) {
	primary := &mockLinkedinFetcher{err: fmt.Errorf("API error")}

	fc := &FallbackClient{primary: primary, linkitin: nil}
	_, err := fc.FetchTrendingPosts(context.Background(), []string{"tech"}, "day", 100, 10)
	if err == nil {
		t.Fatal("expected error when linkitin is nil")
	}
}

func TestFallbackClient_PrimaryDisabled_SkipsPrimary(t *testing.T) {
	primary := &mockLinkedinFetcher{err: fmt.Errorf("status 401: Unauthorized")}
	linkitinPosts := []models.Post{{PlatformPostID: "linkitin-1", Platform: "linkedin"}}
	linkitin := &mockLinkedinFetcher{posts: linkitinPosts}

	fc := &FallbackClient{primary: primary, linkitin: linkitin}

	// First call: primary fails with 401, falls back to linkitin, disables primary.
	posts, err := fc.FetchMyPosts(context.Background(), 10)
	if err != nil {
		t.Fatalf("first call unexpected error: %v", err)
	}
	if posts[0].PlatformPostID != "linkitin-1" {
		t.Errorf("expected linkitin posts on first call")
	}
	if !fc.primaryDisabled {
		t.Fatal("expected primaryDisabled to be true after 401")
	}

	// Second call: should go straight to linkitin without touching primary.
	primary.err = fmt.Errorf("should not be called")
	posts, err = fc.FetchMyPosts(context.Background(), 10)
	if err != nil {
		t.Fatalf("second call unexpected error: %v", err)
	}
	if posts[0].PlatformPostID != "linkitin-1" {
		t.Errorf("expected linkitin posts on second call")
	}
}

func TestFallbackClient_NonAccountError_DoesNotDisable(t *testing.T) {
	primary := &mockLinkedinFetcher{err: fmt.Errorf("network timeout")}
	linkitinPosts := []models.Post{{PlatformPostID: "linkitin-1", Platform: "linkedin"}}
	linkitin := &mockLinkedinFetcher{posts: linkitinPosts}

	fc := &FallbackClient{primary: primary, linkitin: linkitin}
	_, _ = fc.FetchMyPosts(context.Background(), 10)

	if fc.primaryDisabled {
		t.Fatal("expected primaryDisabled to be false for non-account errors")
	}
}

func TestFallbackClient_CreatePost_Success(t *testing.T) {
	poster := &mockLinkedinPoster{postID: "urn:li:share:123"}
	fc := &FallbackClient{
		primary:        &mockLinkedinFetcher{},
		linkitinPoster: poster,
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
		primary:        &mockLinkedinFetcher{},
		linkitinPoster: nil,
	}

	_, err := fc.CreatePost(context.Background(), "Hello LinkedIn!")
	if err == nil {
		t.Fatal("expected error when poster is nil, got nil")
	}
}

func TestFallbackClient_UploadImage_Success(t *testing.T) {
	poster := &mockLinkedinPoster{postID: "urn:li:image:456"}
	fc := &FallbackClient{
		primary:        &mockLinkedinFetcher{},
		linkitinPoster: poster,
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
		primary:        &mockLinkedinFetcher{},
		linkitinPoster: nil,
	}

	_, err := fc.UploadImage(context.Background(), []byte("image-data"), "test.png")
	if err == nil {
		t.Fatal("expected error when poster is nil, got nil")
	}
}

func TestFallbackClient_CreatePostWithImage_Success(t *testing.T) {
	poster := &mockLinkedinPoster{postID: "urn:li:share:789"}
	fc := &FallbackClient{
		primary:        &mockLinkedinFetcher{},
		linkitinPoster: poster,
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
		primary:        &mockLinkedinFetcher{},
		linkitinPoster: nil,
	}

	_, err := fc.CreatePostWithImage(context.Background(), "Post with image", []byte("img"), "photo.jpg")
	if err == nil {
		t.Fatal("expected error when poster is nil, got nil")
	}
}
