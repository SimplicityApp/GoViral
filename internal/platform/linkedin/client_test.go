package linkedin

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

func newTestClient(serverURL string, influencerURNs []string) *Client {
	return &Client{
		accessToken:    "test-access-token",
		authorID:       "ABC123",
		influencerURNs: influencerURNs,
		httpClient:     &http.Client{Transport: &rewriteTransport{serverURL: serverURL}},
		windowStarted:  time.Now(),
	}
}

func TestFetchMyPosts_Success(t *testing.T) {
	postsResp := mustReadFile(t, "../../../testdata/linkedin_posts_response.json")

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/ugcPosts", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token" {
			t.Errorf("Authorization = %q, want 'Bearer test-access-token'", got)
		}
		if got := r.Header.Get("X-Restli-Protocol-Version"); got != "2.0.0" {
			t.Errorf("X-Restli-Protocol-Version = %q, want '2.0.0'", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(postsResp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(server.URL, nil)
	posts, err := client.FetchMyPosts(context.Background(), 10)
	if err != nil {
		t.Fatalf("FetchMyPosts() error = %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("FetchMyPosts() returned %d posts, want 2", len(posts))
	}

	p := posts[0]
	if p.Platform != "linkedin" {
		t.Errorf("posts[0].Platform = %q, want 'linkedin'", p.Platform)
	}
	if p.PlatformPostID != "urn:li:ugcPost:100001" {
		t.Errorf("posts[0].PlatformPostID = %q, want 'urn:li:ugcPost:100001'", p.PlatformPostID)
	}
	if p.Content != "Excited to share my thoughts on leadership in tech!" {
		t.Errorf("posts[0].Content = %q", p.Content)
	}
	if p.Likes != 150 {
		t.Errorf("posts[0].Likes = %d, want 150", p.Likes)
	}
	if p.Reposts != 20 {
		t.Errorf("posts[0].Reposts = %d, want 20", p.Reposts)
	}
	if p.Comments != 30 {
		t.Errorf("posts[0].Comments = %d, want 30", p.Comments)
	}
}

func TestFetchMyPosts_LimitEnforced(t *testing.T) {
	postsResp := mustReadFile(t, "../../../testdata/linkedin_posts_response.json")

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/ugcPosts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(postsResp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(server.URL, nil)
	posts, err := client.FetchMyPosts(context.Background(), 1)
	if err != nil {
		t.Fatalf("FetchMyPosts() error = %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("FetchMyPosts(limit=1) returned %d posts, want 1", len(posts))
	}
}

func TestFetchTrendingPosts_Success(t *testing.T) {
	postsResp := mustReadFile(t, "../../../testdata/linkedin_posts_response.json")

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/ugcPosts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(postsResp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(server.URL, []string{"urn:li:person:INFLUENCER1"})
	posts, err := client.FetchTrendingPosts(context.Background(), []string{"tech"}, "week", 100, 10)
	if err != nil {
		t.Fatalf("FetchTrendingPosts() error = %v", err)
	}
	// Both posts have >= 100 likes (150 and 250), so both pass the filter
	if len(posts) != 2 {
		t.Fatalf("FetchTrendingPosts() returned %d posts, want 2", len(posts))
	}

	// Sorted by engagement: post 100002 = 250+35+45 = 330, post 100001 = 150+20+30 = 200
	if posts[0].PlatformPostID != "urn:li:ugcPost:100002" {
		t.Errorf("posts[0].PlatformPostID = %q, want 'urn:li:ugcPost:100002' (highest engagement)", posts[0].PlatformPostID)
	}
	if posts[0].Platform != "linkedin" {
		t.Errorf("posts[0].Platform = %q, want 'linkedin'", posts[0].Platform)
	}
	if len(posts[0].NicheTags) != 1 || posts[0].NicheTags[0] != "tech" {
		t.Errorf("posts[0].NicheTags = %v, want [tech]", posts[0].NicheTags)
	}
}

func TestFetchTrendingPosts_MinLikesFilter(t *testing.T) {
	postsResp := mustReadFile(t, "../../../testdata/linkedin_posts_response.json")

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/ugcPosts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(postsResp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(server.URL, []string{"urn:li:person:INFLUENCER1"})
	// minLikes=200 filters out post with 150 likes
	posts, err := client.FetchTrendingPosts(context.Background(), []string{"tech"}, "week", 200, 10)
	if err != nil {
		t.Fatalf("FetchTrendingPosts() error = %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("FetchTrendingPosts(minLikes=200) returned %d posts, want 1", len(posts))
	}
	if posts[0].Likes != 250 {
		t.Errorf("posts[0].Likes = %d, want 250", posts[0].Likes)
	}
}

func TestFetchTrendingPosts_NoInfluencers(t *testing.T) {
	client := newTestClient("http://unused", nil)
	_, err := client.FetchTrendingPosts(context.Background(), []string{"tech"}, "week", 100, 10)
	if err == nil {
		t.Fatal("FetchTrendingPosts() expected error for no influencer URNs, got nil")
	}
}

func TestFetchTrendingPosts_EmptyInfluencers(t *testing.T) {
	client := newTestClient("http://unused", []string{})
	_, err := client.FetchTrendingPosts(context.Background(), []string{"tech"}, "week", 100, 10)
	if err == nil {
		t.Fatal("FetchTrendingPosts() expected error for empty influencer URNs, got nil")
	}
}

func TestFetchMyPosts_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestClient(server.URL, nil)
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

	client := newTestClient(server.URL, nil)
	_, err := client.FetchMyPosts(ctx, 10)
	if err == nil {
		t.Fatal("FetchMyPosts() expected error for 429 response, got nil")
	}
}
