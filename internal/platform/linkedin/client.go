package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/pkg/models"
)

// Compile-time interface compliance check.
var _ models.PlatformClient = (*Client)(nil)

const (
	baseURL           = "https://api.linkedin.com/v2"
	rateLimitRequests = 100
	rateLimitWindow   = 1 * time.Minute
	maxBackoffRetries = 3
)

// Client implements models.PlatformClient for the LinkedIn API.
type Client struct {
	accessToken    string
	authorID       string
	influencerURNs []string
	httpClient     *http.Client

	mu            sync.Mutex
	requestCount  int
	windowStarted time.Time
}

// NewClient creates a new LinkedIn API client.
// influencerURNs is a fallback list of LinkedIn member URNs for trending discovery.
func NewClient(cfg config.LinkedInConfig, influencerURNs []string) *Client {
	return &Client{
		accessToken:    cfg.AccessToken,
		authorID:       cfg.PersonURN,
		influencerURNs: influencerURNs,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		windowStarted:  time.Now(),
	}
}

// FetchMyPosts retrieves the authenticated user's UGC posts from LinkedIn.
func (c *Client) FetchMyPosts(ctx context.Context, limit int) ([]models.Post, error) {
	if c.authorID == "" {
		return nil, fmt.Errorf("no access token: person URN not configured")
	}
	authorURN := fmt.Sprintf("urn:li:person:%s", c.authorID)
	endpoint := fmt.Sprintf("%s/ugcPosts", baseURL)
	params := url.Values{}
	params.Set("q", "authors")
	params.Set("authors", fmt.Sprintf("List(%s)", authorURN))
	params.Set("count", fmt.Sprintf("%d", clampLimit(limit, 100)))

	body, err := c.doGetWithRetry(ctx, endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("fetching LinkedIn posts for %s: %w", authorURN, err)
	}

	var resp ugcPostsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing LinkedIn posts response: %w", err)
	}

	now := time.Now()
	posts := make([]models.Post, 0, len(resp.Elements))
	for _, elem := range resp.Elements {
		posts = append(posts, models.Post{
			Platform:       "linkedin",
			PlatformPostID: elem.ID,
			Content:        extractText(elem),
			Likes:          elem.SocialDetail.TotalLikes,
			Reposts:        elem.SocialDetail.TotalShares,
			Comments:       elem.SocialDetail.TotalComments,
			Impressions:    0,
			PostedAt:       time.UnixMilli(elem.Created.Time),
			FetchedAt:      now,
		})
	}

	if limit > 0 && len(posts) > limit {
		posts = posts[:limit]
	}
	return posts, nil
}

// FetchTrendingPosts discovers trending posts from influencer URNs.
func (c *Client) FetchTrendingPosts(ctx context.Context, niches []string, period string, minLikes int, limit int) ([]models.TrendingPost, error) {
	if len(c.influencerURNs) == 0 {
		return nil, fmt.Errorf("no influencer URNs configured for LinkedIn trending discovery")
	}

	seen := make(map[string]bool)
	var allPosts []models.TrendingPost
	now := time.Now()

	cutoff, err := models.PeriodCutoff(period, now)
	if err != nil {
		return nil, fmt.Errorf("computing period cutoff: %w", err)
	}

	for _, urn := range c.influencerURNs {
		endpoint := fmt.Sprintf("%s/ugcPosts", baseURL)
		params := url.Values{}
		params.Set("q", "authors")
		params.Set("authors", fmt.Sprintf("List(%s)", urn))
		params.Set("count", "50")

		body, err := c.doGetWithRetry(ctx, endpoint, params)
		if err != nil {
			continue
		}

		var resp ugcPostsResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			continue
		}

		for _, elem := range resp.Elements {
			if seen[elem.ID] {
				continue
			}
			if time.UnixMilli(elem.Created.Time).Before(cutoff) {
				continue
			}
			if elem.SocialDetail.TotalLikes < minLikes {
				continue
			}
			seen[elem.ID] = true

			allPosts = append(allPosts, models.TrendingPost{
				Platform:       "linkedin",
				PlatformPostID: elem.ID,
				AuthorUsername: elem.Author,
				AuthorName:     elem.Author,
				Content:        extractText(elem),
				Likes:          elem.SocialDetail.TotalLikes,
				Reposts:        elem.SocialDetail.TotalShares,
				Comments:       elem.SocialDetail.TotalComments,
				Impressions:    0,
				NicheTags:      niches,
				PostedAt:       time.UnixMilli(elem.Created.Time),
				FetchedAt:      now,
			})
		}
	}

	sort.Slice(allPosts, func(i, j int) bool {
		return engagement(allPosts[i]) > engagement(allPosts[j])
	})

	if limit > 0 && len(allPosts) > limit {
		allPosts = allPosts[:limit]
	}
	return allPosts, nil
}

func engagement(p models.TrendingPost) int {
	return p.Likes + p.Reposts + p.Comments
}

func extractText(elem ugcPostElement) string {
	if elem.SpecificContent.ShareContent.ShareCommentary.Text != "" {
		return elem.SpecificContent.ShareContent.ShareCommentary.Text
	}
	return ""
}

func clampLimit(limit, max int) int {
	if limit <= 0 || limit > max {
		return max
	}
	return limit
}

// doGetWithRetry performs an HTTP GET with rate limiting and exponential backoff on 429.
func (c *Client) doGetWithRetry(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= maxBackoffRetries; attempt++ {
		if err := c.waitForRateLimit(ctx); err != nil {
			return nil, fmt.Errorf("waiting for rate limit: %w", err)
		}

		body, err := c.doGet(ctx, endpoint, params)
		if err == nil {
			return body, nil
		}

		if !isRateLimitError(err) || attempt == maxBackoffRetries {
			return nil, err
		}
		lastErr = err

		backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return nil, lastErr
}

// doGet performs a single authenticated HTTP GET request.
func (c *Client) doGet(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	reqURL := endpoint
	if len(params) > 0 {
		reqURL = endpoint + "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, &rateLimitError{statusCode: resp.StatusCode}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LinkedIn API returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// waitForRateLimit blocks if the rate limit window is exhausted.
func (c *Client) waitForRateLimit(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if now.Sub(c.windowStarted) >= rateLimitWindow {
		c.requestCount = 0
		c.windowStarted = now
	}

	if c.requestCount >= rateLimitRequests {
		waitTime := rateLimitWindow - now.Sub(c.windowStarted)
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			c.mu.Lock()
			return ctx.Err()
		case <-time.After(waitTime):
		}
		c.mu.Lock()
		c.requestCount = 0
		c.windowStarted = time.Now()
	}

	c.requestCount++
	return nil
}

type rateLimitError struct {
	statusCode int
}

func (e *rateLimitError) Error() string {
	return fmt.Sprintf("rate limited (HTTP %d)", e.statusCode)
}

func isRateLimitError(err error) bool {
	_, ok := err.(*rateLimitError)
	return ok
}

// API response types

type ugcPostsResponse struct {
	Elements []ugcPostElement `json:"elements"`
}

type ugcPostElement struct {
	ID              string          `json:"id"`
	Author          string          `json:"author"`
	Created         createdInfo     `json:"created"`
	SpecificContent specificContent `json:"specificContent"`
	SocialDetail    socialDetail    `json:"socialDetail"`
}

type createdInfo struct {
	Time int64 `json:"time"`
}

type specificContent struct {
	ShareContent shareContent `json:"com.linkedin.ugc.ShareContent"`
}

type shareContent struct {
	ShareCommentary shareCommentary `json:"shareCommentary"`
}

type shareCommentary struct {
	Text string `json:"text"`
}

type socialDetail struct {
	TotalLikes    int `json:"totalSocialActivityCounts.numLikes"`
	TotalComments int `json:"totalSocialActivityCounts.numComments"`
	TotalShares   int `json:"totalSocialActivityCounts.numShares"`
}
