package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/pkg/models"
)

const testUserID = "test-user-1"

// --- helpers ---

func setupTestService(t *testing.T) (*PublishService, *db.DB) {
	t.Helper()
	testDB, err := db.New(":memory:")
	if err != nil {
		t.Fatalf("creating test db: %v", err)
	}
	t.Cleanup(func() { testDB.Close() })
	cfg := &config.Config{}
	return NewPublishService(testDB, cfg), testDB
}

func swapFactory[T any](t *testing.T, target *T, mock T) {
	t.Helper()
	orig := *target
	*target = mock
	t.Cleanup(func() { *target = orig })
}

func insertContent(t *testing.T, testDB *db.DB, gc *models.GeneratedContent) int64 {
	t.Helper()
	if gc.Status == "" {
		gc.Status = "approved"
	}
	if gc.GeneratedContent == "" {
		gc.GeneratedContent = "test content"
	}
	id, err := testDB.InsertGeneratedContent(testUserID, gc)
	if err != nil {
		t.Fatalf("inserting test content: %v", err)
	}
	return id
}

// --- mocks ---

type mockXPoster struct {
	calls     []string
	returnID  string
	returnErr error
}

func (m *mockXPoster) PostTweet(_ context.Context, _ string) (string, error) {
	m.calls = append(m.calls, "PostTweet")
	return m.returnID, m.returnErr
}

func (m *mockXPoster) PostReply(_ context.Context, _ string, _ string) (string, error) {
	m.calls = append(m.calls, "PostReply")
	return m.returnID, m.returnErr
}

type mockLinkedInPoster struct {
	calls     []string
	returnID  string
	returnErr error
}

func (m *mockLinkedInPoster) CreatePost(_ context.Context, _ string) (string, error) {
	m.calls = append(m.calls, "CreatePost")
	return m.returnID, m.returnErr
}

func (m *mockLinkedInPoster) UploadImage(_ context.Context, _ []byte, _ string) (string, error) {
	m.calls = append(m.calls, "UploadImage")
	return m.returnID, m.returnErr
}

func (m *mockLinkedInPoster) CreatePostWithImage(_ context.Context, _ string, _ []byte, _ string) (string, error) {
	m.calls = append(m.calls, "CreatePostWithImage")
	return m.returnID, m.returnErr
}

func (m *mockLinkedInPoster) CreateScheduledPost(_ context.Context, _ string, _ time.Time) (string, error) {
	m.calls = append(m.calls, "CreateScheduledPost")
	return m.returnID, m.returnErr
}

func (m *mockLinkedInPoster) CreateScheduledPostWithImage(_ context.Context, _ string, _ []byte, _ string, _ time.Time) (string, error) {
	m.calls = append(m.calls, "CreateScheduledPostWithImage")
	return m.returnID, m.returnErr
}

type mockLinkedInCommenter struct {
	calls     []string
	returnID  string
	returnErr error
}

func (m *mockLinkedInCommenter) CreateComment(_ context.Context, _ string, _ string, _ string) (string, error) {
	m.calls = append(m.calls, "CreateComment")
	return m.returnID, m.returnErr
}

type mockLinkedInReposter struct {
	calls     []string
	returnID  string
	returnErr error
}

func (m *mockLinkedInReposter) Repost(_ context.Context, _ string, _ string) (string, error) {
	m.calls = append(m.calls, "Repost")
	return m.returnID, m.returnErr
}

type mockXQuotePoster struct {
	calls     []string
	returnID  string
	returnErr error
}

func (m *mockXQuotePoster) PostQuoteTweet(_ context.Context, _ string, _ string) (string, error) {
	m.calls = append(m.calls, "PostQuoteTweet")
	return m.returnID, m.returnErr
}

// --- tests: comment routing ---

func TestPublishX_Comment(t *testing.T) {
	svc, testDB := setupTestService(t)
	id := insertContent(t, testDB, &models.GeneratedContent{
		TargetPlatform: "x",
		IsComment:      true,
		QuoteTweetID:   "tweet-123",
	})

	mock := &mockXPoster{returnID: "reply-456"}
	swapFactory(t, &newXPoster, func(_ config.XConfig) models.PlatformPoster { return mock })

	ids, parts, err := svc.PublishX(context.Background(), testUserID, id, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.calls) != 1 || mock.calls[0] != "PostReply" {
		t.Fatalf("expected [PostReply], got %v", mock.calls)
	}
	if len(ids) != 1 || ids[0] != "reply-456" {
		t.Fatalf("expected [reply-456], got %v", ids)
	}
	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}

	// Verify DB status updated
	gc, _ := testDB.GetGeneratedContentByID(testUserID, id)
	if gc.Status != "posted" {
		t.Fatalf("expected status 'posted', got %q", gc.Status)
	}
}

func TestPublishX_RegularPost(t *testing.T) {
	svc, testDB := setupTestService(t)
	id := insertContent(t, testDB, &models.GeneratedContent{
		TargetPlatform:   "x",
		IsComment:        false,
		GeneratedContent: "hello world",
	})

	mock := &mockXPoster{returnID: "tweet-789"}
	swapFactory(t, &newXPoster, func(_ config.XConfig) models.PlatformPoster { return mock })

	ids, _, err := svc.PublishX(context.Background(), testUserID, id, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.calls) != 1 || mock.calls[0] != "PostTweet" {
		t.Fatalf("expected [PostTweet], got %v", mock.calls)
	}
	if ids[0] != "tweet-789" {
		t.Fatalf("expected tweet-789, got %v", ids[0])
	}
}

func TestPublishLinkedIn_Comment(t *testing.T) {
	svc, testDB := setupTestService(t)
	id := insertContent(t, testDB, &models.GeneratedContent{
		TargetPlatform: "linkedin",
		IsComment:      true,
		QuoteTweetID:   "urn:li:activity:123",
	})

	mock := &mockLinkedInCommenter{returnID: "urn:li:comment:456"}
	swapFactory(t, &newLinkedInCommenter, func(_ config.LinkedInConfig) models.LinkedInCommenter { return mock })

	ids, parts, err := svc.PublishLinkedIn(context.Background(), testUserID, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.calls) != 1 || mock.calls[0] != "CreateComment" {
		t.Fatalf("expected [CreateComment], got %v", mock.calls)
	}
	if ids[0] != "urn:li:comment:456" {
		t.Fatalf("expected urn:li:comment:456, got %v", ids[0])
	}
	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}

	gc, _ := testDB.GetGeneratedContentByID(testUserID, id)
	if gc.Status != "posted" {
		t.Fatalf("expected status 'posted', got %q", gc.Status)
	}
}

func TestPublishLinkedIn_RegularPost(t *testing.T) {
	svc, testDB := setupTestService(t)
	id := insertContent(t, testDB, &models.GeneratedContent{
		TargetPlatform:   "linkedin",
		IsComment:        false,
		GeneratedContent: "linkedin post",
	})

	mock := &mockLinkedInPoster{returnID: "urn:li:share:789"}
	swapFactory(t, &newLinkedInPoster, func(_ config.LinkedInConfig) models.LinkedInPoster { return mock })

	ids, _, err := svc.PublishLinkedIn(context.Background(), testUserID, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.calls) != 1 || mock.calls[0] != "CreatePost" {
		t.Fatalf("expected [CreatePost], got %v", mock.calls)
	}
	if ids[0] != "urn:li:share:789" {
		t.Fatalf("expected urn:li:share:789, got %v", ids[0])
	}
}

// --- tests: Publish() dispatcher routes comments ---

func TestPublish_DispatchesCommentToCorrectPlatform(t *testing.T) {
	t.Run("x comment", func(t *testing.T) {
		svc, testDB := setupTestService(t)
		id := insertContent(t, testDB, &models.GeneratedContent{
			TargetPlatform: "x",
			IsComment:      true,
			QuoteTweetID:   "tweet-parent",
		})

		mock := &mockXPoster{returnID: "reply-x"}
		swapFactory(t, &newXPoster, func(_ config.XConfig) models.PlatformPoster { return mock })

		ids, _, err := svc.Publish(context.Background(), testUserID, id, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mock.calls) != 1 || mock.calls[0] != "PostReply" {
			t.Fatalf("expected [PostReply], got %v", mock.calls)
		}
		if ids[0] != "reply-x" {
			t.Fatalf("expected reply-x, got %v", ids[0])
		}
	})

	t.Run("linkedin comment", func(t *testing.T) {
		svc, testDB := setupTestService(t)
		id := insertContent(t, testDB, &models.GeneratedContent{
			TargetPlatform: "linkedin",
			IsComment:      true,
			QuoteTweetID:   "urn:li:activity:parent",
		})

		mock := &mockLinkedInCommenter{returnID: "urn:li:comment:new"}
		swapFactory(t, &newLinkedInCommenter, func(_ config.LinkedInConfig) models.LinkedInCommenter { return mock })

		ids, _, err := svc.Publish(context.Background(), testUserID, id, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mock.calls) != 1 || mock.calls[0] != "CreateComment" {
			t.Fatalf("expected [CreateComment], got %v", mock.calls)
		}
		if ids[0] != "urn:li:comment:new" {
			t.Fatalf("expected urn:li:comment:new, got %v", ids[0])
		}
	})
}

// --- tests: quote tweet / repost ---

func TestPublishX_QuoteTweet(t *testing.T) {
	svc, testDB := setupTestService(t)
	id := insertContent(t, testDB, &models.GeneratedContent{
		TargetPlatform: "x",
		IsRepost:       true,
		QuoteTweetID:   "tweet-to-quote",
	})

	mock := &mockXQuotePoster{returnID: "qt-001"}
	swapFactory(t, &newXQuotePoster, func(_ config.XConfig) models.QuotePoster { return mock })

	ids, _, err := svc.PublishX(context.Background(), testUserID, id, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.calls) != 1 || mock.calls[0] != "PostQuoteTweet" {
		t.Fatalf("expected [PostQuoteTweet], got %v", mock.calls)
	}
	if ids[0] != "qt-001" {
		t.Fatalf("expected qt-001, got %v", ids[0])
	}
}

func TestPublishLinkedIn_Repost(t *testing.T) {
	svc, testDB := setupTestService(t)
	id := insertContent(t, testDB, &models.GeneratedContent{
		TargetPlatform: "linkedin",
		IsRepost:       true,
		QuoteTweetID:   "urn:li:activity:to-repost",
	})

	mock := &mockLinkedInReposter{returnID: "urn:li:share:reposted"}
	swapFactory(t, &newLinkedInReposter, func(_ config.LinkedInConfig) models.LinkedInReposter { return mock })

	ids, _, err := svc.PublishLinkedIn(context.Background(), testUserID, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.calls) != 1 || mock.calls[0] != "Repost" {
		t.Fatalf("expected [Repost], got %v", mock.calls)
	}
	if ids[0] != "urn:li:share:reposted" {
		t.Fatalf("expected urn:li:share:reposted, got %v", ids[0])
	}
}

// --- tests: error & edge cases ---

func TestPublishX_ContentNotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	_, _, err := svc.PublishX(context.Background(), testUserID, 9999, false)
	if err == nil {
		t.Fatal("expected error for missing content")
	}
}

func TestPublishX_AlreadyPosted(t *testing.T) {
	svc, testDB := setupTestService(t)
	id := insertContent(t, testDB, &models.GeneratedContent{
		TargetPlatform: "x",
		Status:         "posted",
	})

	// Need to set status to posted via the DB since insertContent uses the given status
	_ = testDB.UpdateGeneratedContentPosted(testUserID, id, "some-id")

	_, _, err := svc.PublishX(context.Background(), testUserID, id, false)
	if err == nil {
		t.Fatal("expected error for already-posted content")
	}
}

func TestPublishX_WrongPlatform(t *testing.T) {
	svc, testDB := setupTestService(t)
	id := insertContent(t, testDB, &models.GeneratedContent{
		TargetPlatform: "linkedin",
	})

	_, _, err := svc.PublishX(context.Background(), testUserID, id, false)
	if err == nil {
		t.Fatal("expected error for wrong platform")
	}
}

func TestPublishLinkedIn_ContentNotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	_, _, err := svc.PublishLinkedIn(context.Background(), testUserID, 9999)
	if err == nil {
		t.Fatal("expected error for missing content")
	}
}

func TestPublishLinkedIn_AlreadyPosted(t *testing.T) {
	svc, testDB := setupTestService(t)
	id := insertContent(t, testDB, &models.GeneratedContent{
		TargetPlatform: "linkedin",
	})
	_ = testDB.UpdateGeneratedContentPosted(testUserID, id, "some-id")

	_, _, err := svc.PublishLinkedIn(context.Background(), testUserID, id)
	if err == nil {
		t.Fatal("expected error for already-posted content")
	}
}

func TestPublishLinkedIn_WrongPlatform(t *testing.T) {
	svc, testDB := setupTestService(t)
	id := insertContent(t, testDB, &models.GeneratedContent{
		TargetPlatform: "x",
	})

	_, _, err := svc.PublishLinkedIn(context.Background(), testUserID, id)
	if err == nil {
		t.Fatal("expected error for wrong platform")
	}
}

func TestPublish_UnsupportedPlatform(t *testing.T) {
	svc, testDB := setupTestService(t)
	id := insertContent(t, testDB, &models.GeneratedContent{
		TargetPlatform: "mastodon",
	})

	_, _, err := svc.Publish(context.Background(), testUserID, id, false)
	if err == nil {
		t.Fatal("expected error for unsupported platform")
	}
}

func TestPublishX_CommentError(t *testing.T) {
	svc, testDB := setupTestService(t)
	id := insertContent(t, testDB, &models.GeneratedContent{
		TargetPlatform: "x",
		IsComment:      true,
		QuoteTweetID:   "tweet-123",
	})

	mock := &mockXPoster{returnErr: fmt.Errorf("network error")}
	swapFactory(t, &newXPoster, func(_ config.XConfig) models.PlatformPoster { return mock })

	_, _, err := svc.PublishX(context.Background(), testUserID, id, false)
	if err == nil {
		t.Fatal("expected error from failed comment")
	}
}

func TestPublishLinkedIn_CommentError(t *testing.T) {
	svc, testDB := setupTestService(t)
	id := insertContent(t, testDB, &models.GeneratedContent{
		TargetPlatform: "linkedin",
		IsComment:      true,
		QuoteTweetID:   "urn:li:activity:123",
	})

	mock := &mockLinkedInCommenter{returnErr: fmt.Errorf("api error")}
	swapFactory(t, &newLinkedInCommenter, func(_ config.LinkedInConfig) models.LinkedInCommenter { return mock })

	_, _, err := svc.PublishLinkedIn(context.Background(), testUserID, id)
	if err == nil {
		t.Fatal("expected error from failed comment")
	}
}
