package x

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"log/slog"
	"sync"
	"time"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/pkg/models"
)

// Compile-time interface compliance checks.
var _ models.PlatformClient = (*Client)(nil)
var _ models.PlatformPoster = (*Client)(nil)
var _ models.MediaPoster = (*Client)(nil)
var _ models.QuotePoster = (*Client)(nil)

const (
	baseURL           = "https://api.twitter.com/2"
	rateLimitRequests = 300
	rateLimitWindow   = 15 * time.Minute
	maxBackoffRetries = 3
)

// Client implements models.PlatformClient and models.PlatformPoster for X/Twitter API v2.
type Client struct {
	bearerToken string
	accessToken string
	username    string
	httpClient  *http.Client

	mu            sync.Mutex
	requestCount  int
	windowStarted time.Time
}

// NewClient creates a new X API client.
func NewClient(cfg config.XConfig) *Client {
	return &Client{
		bearerToken:   cfg.BearerToken,
		accessToken:   cfg.AccessToken,
		username:      cfg.Username,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		windowStarted: time.Now(),
	}
}

// PostTweet posts a new tweet and returns the tweet ID.
func (c *Client) PostTweet(ctx context.Context, text string) (string, error) {
	payload := map[string]interface{}{
		"text": text,
	}
	return c.doPostTweet(ctx, payload)
}

// PostQuoteTweet posts a quote tweet and returns the new tweet ID.
func (c *Client) PostQuoteTweet(ctx context.Context, text string, quoteTweetID string) (string, error) {
	payload := map[string]interface{}{
		"text":           text,
		"quote_tweet_id": quoteTweetID,
	}
	return c.doPostTweet(ctx, payload)
}

// PostReply posts a reply to an existing tweet and returns the new tweet ID.
func (c *Client) PostReply(ctx context.Context, text string, inReplyToID string) (string, error) {
	payload := map[string]interface{}{
		"text": text,
		"reply": map[string]string{
			"in_reply_to_tweet_id": inReplyToID,
		},
	}
	return c.doPostTweet(ctx, payload)
}

// doPostTweet sends a POST request to create a tweet.
func (c *Client) doPostTweet(ctx context.Context, payload map[string]interface{}) (string, error) {
	if c.accessToken == "" {
		return "", fmt.Errorf("no access token configured; run 'goviral auth x' first")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling tweet payload: %w", err)
	}

	endpoint := baseURL + "/tweets"
	var lastErr error
	for attempt := 0; attempt <= maxBackoffRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return "", fmt.Errorf("creating post request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("executing post request: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", fmt.Errorf("reading post response: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = &rateLimitError{statusCode: resp.StatusCode}
			if attempt == maxBackoffRetries {
				return "", lastErr
			}
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
			continue
		}

		if resp.StatusCode != http.StatusCreated {
			return "", fmt.Errorf("X API returned status %d: %s", resp.StatusCode, string(respBody))
		}

		var tweetResp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(respBody, &tweetResp); err != nil {
			return "", fmt.Errorf("parsing tweet response: %w", err)
		}
		return tweetResp.Data.ID, nil
	}
	return "", lastErr
}

// UploadMedia uploads image data to X and returns the media ID.
func (c *Client) UploadMedia(ctx context.Context, imageData []byte, mimeType string) (string, error) {
	if c.accessToken == "" {
		return "", fmt.Errorf("no access token configured; run 'goviral auth x' first")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("media", "image.png")
	if err != nil {
		return "", fmt.Errorf("creating multipart form: %w", err)
	}
	if _, err := part.Write(imageData); err != nil {
		return "", fmt.Errorf("writing image data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("closing multipart writer: %w", err)
	}

	endpoint := "https://upload.twitter.com/1.1/media/upload.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", fmt.Errorf("creating upload request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing upload request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading upload response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("media upload returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var uploadResp struct {
		MediaIDString string `json:"media_id_string"`
	}
	if err := json.Unmarshal(respBody, &uploadResp); err != nil {
		return "", fmt.Errorf("parsing upload response: %w", err)
	}
	if uploadResp.MediaIDString == "" {
		return "", fmt.Errorf("upload returned empty media_id_string")
	}
	return uploadResp.MediaIDString, nil
}

// PostTweetWithMedia posts a tweet with media attachments.
func (c *Client) PostTweetWithMedia(ctx context.Context, text string, mediaIDs []string) (string, error) {
	payload := map[string]interface{}{
		"text": text,
		"media": map[string]interface{}{
			"media_ids": mediaIDs,
		},
	}
	return c.doPostTweet(ctx, payload)
}

// PostReplyWithMedia posts a reply with media attachments.
func (c *Client) PostReplyWithMedia(ctx context.Context, text string, inReplyToID string, mediaIDs []string) (string, error) {
	payload := map[string]interface{}{
		"text": text,
		"reply": map[string]string{
			"in_reply_to_tweet_id": inReplyToID,
		},
		"media": map[string]interface{}{
			"media_ids": mediaIDs,
		},
	}
	return c.doPostTweet(ctx, payload)
}

// FetchMyPosts retrieves the authenticated user's recent tweets.
func (c *Client) FetchMyPosts(ctx context.Context, limit int) ([]models.Post, error) {
	userID, err := c.resolveUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolving user ID for @%s: %w", c.username, err)
	}

	endpoint := fmt.Sprintf("%s/users/%s/tweets", baseURL, userID)
	params := url.Values{}
	params.Set("tweet.fields", "public_metrics,created_at,attachments")
	params.Set("expansions", "attachments.media_keys")
	params.Set("media.fields", "url,preview_image_url,alt_text,type")
	if limit > 0 && limit <= 100 {
		params.Set("max_results", strconv.Itoa(limit))
	} else {
		params.Set("max_results", "100")
	}

	body, err := c.doGetWithRetry(ctx, endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("fetching tweets for user %s: %w", userID, err)
	}

	var resp tweetsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing tweets response: %w", err)
	}

	now := time.Now()
	posts := make([]models.Post, 0, len(resp.Data))
	for _, t := range resp.Data {
		postedAt, _ := time.Parse(time.RFC3339, t.CreatedAt)
		posts = append(posts, models.Post{
			Platform:       "x",
			PlatformPostID: t.ID,
			Content:        t.Text,
			Likes:          t.PublicMetrics.LikeCount,
			Reposts:        t.PublicMetrics.RetweetCount,
			Comments:       t.PublicMetrics.ReplyCount,
			Impressions:    t.PublicMetrics.ImpressionCount,
			PostedAt:       postedAt,
			FetchedAt:      now,
		})
	}

	if limit > 0 && len(posts) > limit {
		posts = posts[:limit]
	}
	return posts, nil
}

// FetchTrendingPosts searches for trending/high-engagement posts matching the given niches.
func (c *Client) FetchTrendingPosts(ctx context.Context, niches []string, period string, minLikes int, limit int) ([]models.TrendingPost, error) {
	seen := make(map[string]bool)
	var allPosts []models.TrendingPost
	var lastErr error

	for _, niche := range niches {
		query := fmt.Sprintf("\"%s\" min_faves:%d lang:en -is:retweet", niche, minLikes)
		endpoint := fmt.Sprintf("%s/tweets/search/recent", baseURL)
		params := url.Values{}
		params.Set("query", query)
		params.Set("tweet.fields", "public_metrics,created_at,author_id,attachments")
		params.Set("expansions", "author_id,attachments.media_keys")
		params.Set("user.fields", "username,name")
		params.Set("media.fields", "url,preview_image_url,alt_text,type")
		params.Set("max_results", "100")

		cutoff, err := models.PeriodCutoff(period, time.Now())
		if err != nil {
			return nil, fmt.Errorf("computing period cutoff: %w", err)
		}
		// Twitter /tweets/search/recent has a 7-day hard limit.
		sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
		if cutoff.Before(sevenDaysAgo) {
			cutoff = sevenDaysAgo
		}
		params.Set("start_time", cutoff.Format(time.RFC3339))

		body, err := c.doGetWithRetry(ctx, endpoint, params)
		if err != nil {
			slog.Warn("skipping niche due to API error", "niche", niche, "error", err)
			lastErr = err
			continue
		}

		var resp searchResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			slog.Warn("skipping niche due to parse error", "niche", niche, "error", err)
			lastErr = err
			continue
		}

		userMap := make(map[string]userObject)
		for _, u := range resp.Includes.Users {
			userMap[u.ID] = u
		}

		mediaMap := make(map[string]mediaObject)
		for _, m := range resp.Includes.Media {
			mediaMap[m.MediaKey] = m
		}

		now := time.Now()
		for _, t := range resp.Data {
			if seen[t.ID] {
				continue
			}
			seen[t.ID] = true

			postedAt, _ := time.Parse(time.RFC3339, t.CreatedAt)
			author := userMap[t.AuthorID]

			var media []models.MediaAttachment
			for _, key := range t.Attachments.MediaKeys {
				if m, ok := mediaMap[key]; ok {
					media = append(media, models.MediaAttachment{
						Type:       m.Type,
						URL:        m.URL,
						PreviewURL: m.PreviewImageURL,
						AltText:    m.AltText,
					})
				}
			}

			allPosts = append(allPosts, models.TrendingPost{
				Platform:       "x",
				PlatformPostID: t.ID,
				AuthorUsername: author.Username,
				AuthorName:     author.Name,
				Content:        t.Text,
				Likes:          t.PublicMetrics.LikeCount,
				Reposts:        t.PublicMetrics.RetweetCount,
				Comments:       t.PublicMetrics.ReplyCount,
				Impressions:    t.PublicMetrics.ImpressionCount,
				NicheTags:      []string{niche},
				Media:          media,
				PostedAt:       postedAt,
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
	// If every niche failed and we have nothing to show, propagate the last
	// error so the FallbackClient knows to try twikit.
	if len(allPosts) == 0 && lastErr != nil {
		return nil, lastErr
	}
	return allPosts, nil
}

func engagement(p models.TrendingPost) int {
	return p.Likes + p.Reposts + p.Comments
}

// resolveUserID looks up the user ID for the configured username.
func (c *Client) resolveUserID(ctx context.Context) (string, error) {
	endpoint := fmt.Sprintf("%s/users/by/username/%s", baseURL, c.username)
	body, err := c.doGetWithRetry(ctx, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("looking up user by username: %w", err)
	}

	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parsing user lookup response: %w", err)
	}
	if resp.Data.ID == "" {
		return "", fmt.Errorf("user @%s not found", c.username)
	}
	return resp.Data.ID, nil
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
	req.Header.Set("Authorization", "Bearer "+c.bearerToken)

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
		return nil, fmt.Errorf("X API returned status %d: %s", resp.StatusCode, string(body))
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

type tweetObject struct {
	ID            string        `json:"id"`
	Text          string        `json:"text"`
	CreatedAt     string        `json:"created_at"`
	AuthorID      string        `json:"author_id"`
	PublicMetrics publicMetrics `json:"public_metrics"`
	Attachments   struct {
		MediaKeys []string `json:"media_keys"`
	} `json:"attachments"`
}

type mediaObject struct {
	MediaKey       string `json:"media_key"`
	Type           string `json:"type"`
	URL            string `json:"url"`
	PreviewImageURL string `json:"preview_image_url"`
	AltText        string `json:"alt_text"`
}

type publicMetrics struct {
	LikeCount       int `json:"like_count"`
	RetweetCount    int `json:"retweet_count"`
	ReplyCount      int `json:"reply_count"`
	ImpressionCount int `json:"impression_count"`
}

type userObject struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type tweetsResponse struct {
	Data []tweetObject `json:"data"`
}

type searchResponse struct {
	Data     []tweetObject `json:"data"`
	Includes struct {
		Users []userObject  `json:"users"`
		Media []mediaObject `json:"media"`
	} `json:"includes"`
}
