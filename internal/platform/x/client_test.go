package x

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
)

// rewriteTransport redirects all HTTP requests to the test server.
type rewriteTransport struct {
	serverURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	u, _ := url.Parse(t.serverURL)
	req.URL.Scheme = u.Scheme
	req.URL.Host = u.Host
	return http.DefaultTransport.RoundTrip(req)
}

func mustReadFile(tb testing.TB, path string) []byte {
	tb.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("reading file %s: %v", path, err)
	}
	return data
}

func newTestClient(serverURL string) *Client {
	return &Client{
		bearerToken:   "test-bearer-token",
		username:      "testuser",
		httpClient:    &http.Client{Transport: &rewriteTransport{serverURL: serverURL}},
		windowStarted: time.Now(),
	}
}

func TestFetchMyPosts_Success(t *testing.T) {
	userResp := mustReadFile(t, "../../../testdata/x_user_response.json")
	tweetsResp := mustReadFile(t, "../../../testdata/x_tweets_response.json")

	mux := http.NewServeMux()
	mux.HandleFunc("/2/users/by/username/testuser", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-bearer-token" {
			t.Errorf("Authorization header = %q, want 'Bearer test-bearer-token'", got)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(userResp)
	})
	mux.HandleFunc("/2/users/12345/tweets", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("tweet.fields"); got != "public_metrics,created_at,attachments" {
			t.Errorf("tweet.fields = %q, want 'public_metrics,created_at,attachments'", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(tweetsResp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(server.URL)
	posts, err := client.FetchMyPosts(context.Background(), 10)
	if err != nil {
		t.Fatalf("FetchMyPosts() error = %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("FetchMyPosts() returned %d posts, want 2", len(posts))
	}

	// Verify first post
	p := posts[0]
	if p.Platform != "x" {
		t.Errorf("posts[0].Platform = %q, want 'x'", p.Platform)
	}
	if p.PlatformPostID != "1001" {
		t.Errorf("posts[0].PlatformPostID = %q, want '1001'", p.PlatformPostID)
	}
	if p.Content != "Hello world! This is my first test tweet." {
		t.Errorf("posts[0].Content = %q, want expected content", p.Content)
	}
	if p.Likes != 42 {
		t.Errorf("posts[0].Likes = %d, want 42", p.Likes)
	}
	if p.Reposts != 10 {
		t.Errorf("posts[0].Reposts = %d, want 10", p.Reposts)
	}
	if p.Comments != 5 {
		t.Errorf("posts[0].Comments = %d, want 5", p.Comments)
	}
	if p.Impressions != 1000 {
		t.Errorf("posts[0].Impressions = %d, want 1000", p.Impressions)
	}

	// Verify second post
	if posts[1].PlatformPostID != "1002" {
		t.Errorf("posts[1].PlatformPostID = %q, want '1002'", posts[1].PlatformPostID)
	}
	if posts[1].Likes != 100 {
		t.Errorf("posts[1].Likes = %d, want 100", posts[1].Likes)
	}
}

func TestFetchMyPosts_LimitEnforced(t *testing.T) {
	userResp := mustReadFile(t, "../../../testdata/x_user_response.json")
	tweetsResp := mustReadFile(t, "../../../testdata/x_tweets_response.json")

	mux := http.NewServeMux()
	mux.HandleFunc("/2/users/by/username/testuser", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(userResp)
	})
	mux.HandleFunc("/2/users/12345/tweets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(tweetsResp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(server.URL)
	posts, err := client.FetchMyPosts(context.Background(), 1)
	if err != nil {
		t.Fatalf("FetchMyPosts() error = %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("FetchMyPosts(limit=1) returned %d posts, want 1", len(posts))
	}
}

func TestFetchTrendingPosts_Success(t *testing.T) {
	searchResp := mustReadFile(t, "../../../testdata/x_search_response.json")

	mux := http.NewServeMux()
	mux.HandleFunc("/2/tweets/search/recent", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("query")
		if q == "" {
			t.Error("expected non-empty query parameter")
		}
		if st := r.URL.Query().Get("start_time"); st == "" {
			t.Error("expected start_time parameter to be set")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(searchResp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(server.URL)
	posts, err := client.FetchTrendingPosts(context.Background(), []string{"tech"}, "week", 100, 10)
	if err != nil {
		t.Fatalf("FetchTrendingPosts() error = %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("FetchTrendingPosts() returned %d posts, want 2", len(posts))
	}

	// Sorted by engagement: post 2001 = 5000+1200+300 = 6500, post 2002 = 3000+800+200 = 4000
	if posts[0].PlatformPostID != "2001" {
		t.Errorf("posts[0].PlatformPostID = %q, want '2001' (highest engagement)", posts[0].PlatformPostID)
	}
	if posts[0].AuthorUsername != "techguru" {
		t.Errorf("posts[0].AuthorUsername = %q, want 'techguru'", posts[0].AuthorUsername)
	}
	if posts[0].AuthorName != "Tech Guru" {
		t.Errorf("posts[0].AuthorName = %q, want 'Tech Guru'", posts[0].AuthorName)
	}
	if posts[0].Likes != 5000 {
		t.Errorf("posts[0].Likes = %d, want 5000", posts[0].Likes)
	}
	if posts[0].Platform != "x" {
		t.Errorf("posts[0].Platform = %q, want 'x'", posts[0].Platform)
	}
	if len(posts[0].NicheTags) != 1 || posts[0].NicheTags[0] != "tech" {
		t.Errorf("posts[0].NicheTags = %v, want [tech]", posts[0].NicheTags)
	}
}

func TestFetchTrendingPosts_DeduplicatesAcrossNiches(t *testing.T) {
	searchResp := mustReadFile(t, "../../../testdata/x_search_response.json")

	mux := http.NewServeMux()
	mux.HandleFunc("/2/tweets/search/recent", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(searchResp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(server.URL)
	// Two niches that return the same posts — should be deduplicated
	posts, err := client.FetchTrendingPosts(context.Background(), []string{"tech", "AI"}, "week", 100, 10)
	if err != nil {
		t.Fatalf("FetchTrendingPosts() error = %v", err)
	}
	if len(posts) != 2 {
		t.Errorf("FetchTrendingPosts() returned %d posts, want 2 (deduplicated)", len(posts))
	}
}

func TestFetchMyPosts_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.FetchMyPosts(context.Background(), 10)
	if err == nil {
		t.Fatal("FetchMyPosts() expected error for 500 response, got nil")
	}
}

func TestFetchMyPosts_RateLimit429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	client := newTestClient(server.URL)
	_, err := client.FetchMyPosts(ctx, 10)
	if err == nil {
		t.Fatal("FetchMyPosts() expected error for 429 response, got nil")
	}
}

func TestFetchMyPosts_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := newTestClient(server.URL)
	_, err := client.FetchMyPosts(ctx, 10)
	if err == nil {
		t.Fatal("FetchMyPosts() expected error for cancelled context, got nil")
	}
}
