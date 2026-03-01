package linkedin

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/pkg/models"
)

//go:embed scripts/linkitin_bridge.py
var linkitinScript []byte

// LinkitinClient interacts with LinkedIn via a Python linkitin subprocess using cookie-based auth.
// Requires a one-time login via ExtractCookies() which saves cookies to ~/.goviral/linkitin_cookies.json.
type LinkitinClient struct {
	pythonPath string
	scriptPath string
}

// NewLinkitinClient creates a LinkitinClient. Returns an error if python3/python
// is not on PATH or the embedded script cannot be written to disk.
// It reuses the shared virtualenv at ~/.goviral/venv/.
func NewLinkitinClient() (*LinkitinClient, error) {
	pythonPath, err := ensureLinkitinVenv()
	if err != nil {
		return nil, fmt.Errorf("setting up python venv for linkitin: %w", err)
	}

	scriptPath, err := ensureLinkitinScript()
	if err != nil {
		return nil, fmt.Errorf("writing linkitin script: %w", err)
	}

	return &LinkitinClient{
		pythonPath: pythonPath,
		scriptPath: scriptPath,
	}, nil
}

// ExtractCookies extracts LinkedIn session cookies from Chrome and saves them
// to ~/.goviral/linkitin_cookies.json. The user must be logged into LinkedIn in Chrome.
func (c *LinkitinClient) ExtractCookies(ctx context.Context) error {
	result, err := c.runCommand(ctx, linkitinCommand{Action: "login_browser"})
	if err != nil {
		return fmt.Errorf("extracting LinkedIn cookies: %w", err)
	}
	if errMsg := result["error"]; errMsg != nil {
		return fmt.Errorf("extracting LinkedIn cookies: %s", errMsg)
	}
	return nil
}

// LoginWithCookies authenticates with manually provided cookies.
func (c *LinkitinClient) LoginWithCookies(ctx context.Context, liAt string, jsessionID string) error {
	result, err := c.runCommand(ctx, linkitinCommand{
		Action:     "login",
		LiAt:       liAt,
		JSessionID: jsessionID,
	})
	if err != nil {
		return fmt.Errorf("logging in with cookies: %w", err)
	}
	if errMsg := result["error"]; errMsg != nil {
		return fmt.Errorf("logging in with cookies: %s", errMsg)
	}
	return nil
}

// IsLinkitinAuthError checks whether the error indicates an expired or missing LinkedIn session.
func IsLinkitinAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	authPatterns := []string{
		"jsessionid not set",
		"not logged in",
		"login first",
		"session expired",
		"cookies are expired",
		"session likely expired",
		"status code: 401",
		"status code: 403",
	}
	for _, p := range authPatterns {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}

// CookieFilePath returns the path to the linkitin cookie file (~/.goviral/linkitin_cookies.json).
func CookieFilePath() string {
	return filepath.Join(config.DefaultConfigDir(), "linkitin_cookies.json")
}

// retryOnAuthError runs fn, and if it returns an auth error, re-extracts cookies
// from Chrome and retries fn exactly once.
func (c *LinkitinClient) retryOnAuthError(ctx context.Context, opName string, fn func() error) error {
	err := fn()
	if err == nil || !IsLinkitinAuthError(err) {
		return err
	}

	slog.Warn("linkitin auth error, re-extracting cookies", "op", opName, "original_error", err)
	if extractErr := c.ExtractCookies(ctx); extractErr != nil {
		slog.Warn("cookie re-extraction failed, returning original error", "op", opName, "extract_error", extractErr)
		return err
	}

	slog.Info("cookies re-extracted, retrying operation", "op", opName)
	retryErr := fn()
	if retryErr != nil {
		slog.Error("linkitin op failed after cookie retry", "op", opName, "error", retryErr)
	}
	return retryErr
}

// FetchMyPosts fetches the user's LinkedIn posts via the linkitin subprocess.
func (c *LinkitinClient) FetchMyPosts(ctx context.Context, limit int) ([]models.Post, error) {
	var posts []models.Post
	err := c.retryOnAuthError(ctx, "FetchMyPosts", func() error {
		result, err := c.runCommand(ctx, linkitinCommand{
			Action: "get_my_posts",
			Limit:  limit,
		})
		if err != nil {
			return fmt.Errorf("fetching LinkedIn posts: %w", err)
		}
		if errMsg := result["error"]; errMsg != nil {
			return fmt.Errorf("fetching LinkedIn posts: %s", errMsg)
		}
		posts, err = parseLinkitinPosts(result)
		return err
	})
	return posts, err
}

// FetchTrendingPosts fetches trending LinkedIn posts. It first tries the dedicated
// get_trending_posts action (which uses linkitin's trending API), falling back to
// search_posts with niche keywords if trending fails.
//
// Fetch order (highest quality first):
//  1. Home feed  — real URNs, algorithmically ranked for the authenticated user
//  2. Topic-less followed-trending — broadly trending posts from followed people
//  3. Per-niche trending — fills remaining slots up to limit
func (c *LinkitinClient) FetchTrendingPosts(ctx context.Context, niches []string, period string, minLikes int, limit int) ([]models.TrendingPost, error) {
	var allPosts []models.TrendingPost
	seen := make(map[string]bool)
	now := time.Now()
	nicheErrors := 0

	// ── Priority 1: home feed (real URNs, algorithmically relevant) ──
	slog.Info("linkedin fetching feed")
	feedPosts, feedErr := c.fetchFeedPosts(ctx, limit*2)
	if feedErr != nil {
		slog.Warn("linkedin feed fetch failed", "error", feedErr)
	} else {
		feedAdded := 0
		for _, p := range feedPosts {
			if seen[p.PlatformPostID] || p.Likes < minLikes {
				continue
			}
			seen[p.PlatformPostID] = true
			allPosts = append(allPosts, p)
			feedAdded++
		}
		slog.Info("linkedin feed complete", "posts", feedAdded)
	}

	// ── Priority 2: topic-less trending from followed (real URNs) ──
	slog.Info("linkedin fetching followed trending")
	followedPosts, followedErr := c.fetchFollowedTrending(ctx, period, limit*2)
	if followedErr != nil {
		slog.Warn("linkedin followed trending fetch failed", "error", followedErr)
	} else {
		followedAdded := 0
		for _, p := range followedPosts {
			if seen[p.PlatformPostID] || p.Likes < minLikes {
				continue
			}
			seen[p.PlatformPostID] = true
			allPosts = append(allPosts, p)
			followedAdded++
		}
		slog.Info("linkedin followed trending complete", "posts", followedAdded)
	}

	// ── Priority 3: per-niche trending (fills remaining slots) ──
	for _, niche := range niches {
		slog.Info("linkedin fetching trending", "niche", niche)
		trendingPosts, err := c.fetchTrendingForNiche(ctx, niche, period, limit)
		if err != nil {
			slog.Warn("linkedin trending fetch failed for niche, trying search fallback", "niche", niche, "error", err)
			// Trending failed, fall back to search_posts.
			rawPosts, searchErr := c.searchPostsForNiche(ctx, niche, limit)
			if searchErr != nil {
				slog.Warn("skipping linkedin niche due to search error", "niche", niche, "error", searchErr)
				nicheErrors++
				continue
			}
			added := 0
			for _, p := range rawPosts {
				if seen[p.PlatformPostID] || p.Likes < minLikes {
					continue
				}
				seen[p.PlatformPostID] = true
				allPosts = append(allPosts, models.TrendingPost{
					Platform:       "linkedin",
					PlatformPostID: p.PlatformPostID,
					Content:        p.Content,
					Likes:          p.Likes,
					Reposts:        p.Reposts,
					Comments:       p.Comments,
					Impressions:    p.Impressions,
					NicheTags:      []string{niche},
					PostedAt:       p.PostedAt,
					FetchedAt:      now,
				})
				added++
			}
			slog.Info("linkedin niche complete", "niche", niche, "posts", added, "via", "search_fallback")
			continue
		}

		added := 0
		for _, p := range trendingPosts {
			if seen[p.PlatformPostID] || p.Likes < minLikes {
				continue
			}
			seen[p.PlatformPostID] = true
			allPosts = append(allPosts, p)
			added++
		}
		slog.Info("linkedin niche complete", "niche", niche, "posts", added)
	}

	// If every niche failed AND the feed also failed, surface an error so the
	// caller (and the daemon) can alert the user — mirrors twikit's behaviour.
	if len(allPosts) == 0 && nicheErrors == len(niches) && feedErr != nil {
		return nil, fmt.Errorf("all linkedin fetches failed (session likely expired) — re-run 'goviral linkitin-login'")
	}

	if limit > 0 && len(allPosts) > limit {
		allPosts = allPosts[:limit]
	}
	return allPosts, nil
}

// fetchTrendingForNiche tries the dedicated get_trending_posts action for a single niche.
func (c *LinkitinClient) fetchTrendingForNiche(ctx context.Context, niche string, period string, limit int) ([]models.TrendingPost, error) {
	candidateLimit := limit * 3
	if candidateLimit < 60 {
		candidateLimit = 60
	}
	var posts []models.TrendingPost
	err := c.retryOnAuthError(ctx, "fetchTrendingForNiche", func() error {
		result, err := c.runCommand(ctx, linkitinCommand{
			Action:       "get_trending_posts",
			Topic:        niche,
			Period:       mapPeriodToLinkitin(period),
			Limit:        candidateLimit,
			FromFollowed: true,
			Scrolls:      7,
		})
		if err != nil {
			return err
		}
		if errMsg := result["error"]; errMsg != nil {
			return fmt.Errorf("%s", errMsg)
		}
		posts, err = parseLinkitinTrendingPosts(result, niche, time.Now())
		return err
	})
	return posts, err
}

// fetchFeedPosts fetches the authenticated user's home feed (posts from followed people).
func (c *LinkitinClient) fetchFeedPosts(ctx context.Context, limit int) ([]models.TrendingPost, error) {
	var posts []models.TrendingPost
	err := c.retryOnAuthError(ctx, "fetchFeedPosts", func() error {
		result, err := c.runCommand(ctx, linkitinCommand{
			Action: "get_feed",
			Limit:  limit,
		})
		if err != nil {
			return err
		}
		if errMsg := result["error"]; errMsg != nil {
			return fmt.Errorf("%s", errMsg)
		}
		posts, err = parseLinkitinTrendingPosts(result, "feed", time.Now())
		return err
	})
	return posts, err
}

// fetchFollowedTrending fetches trending posts from followed people without a
// topic filter (equivalent to test_fetch.py step 2: get_trending_posts with no keyword).
func (c *LinkitinClient) fetchFollowedTrending(ctx context.Context, period string, limit int) ([]models.TrendingPost, error) {
	var posts []models.TrendingPost
	err := c.retryOnAuthError(ctx, "fetchFollowedTrending", func() error {
		result, err := c.runCommand(ctx, linkitinCommand{
			Action:       "get_trending_posts",
			Topic:        "", // no keyword filter — broadly trending from followed people
			Period:       mapPeriodToLinkitin(period),
			Limit:        limit,
			FromFollowed: true,
			Scrolls:      5,
		})
		if err != nil {
			return err
		}
		if errMsg := result["error"]; errMsg != nil {
			return fmt.Errorf("%s", errMsg)
		}
		posts, err = parseLinkitinTrendingPosts(result, "followed", time.Now())
		return err
	})
	return posts, err
}

// searchPostsForNiche falls back to search_posts for a single niche.
func (c *LinkitinClient) searchPostsForNiche(ctx context.Context, niche string, limit int) ([]models.Post, error) {
	var posts []models.Post
	err := c.retryOnAuthError(ctx, "searchPostsForNiche", func() error {
		result, err := c.runCommand(ctx, linkitinCommand{
			Action:   "search_posts",
			Keywords: niche,
			Limit:    limit,
		})
		if err != nil {
			return err
		}
		if errMsg := result["error"]; errMsg != nil {
			return fmt.Errorf("%s", errMsg)
		}
		posts, err = parseLinkitinPosts(result)
		return err
	})
	return posts, err
}

// CreatePost creates a new LinkedIn post.
func (c *LinkitinClient) CreatePost(ctx context.Context, text string) (string, error) {
	var urn string
	err := c.retryOnAuthError(ctx, "CreatePost", func() error {
		result, err := c.runCommand(ctx, linkitinCommand{
			Action: "create_post",
			Text:   text,
		})
		if err != nil {
			return fmt.Errorf("creating LinkedIn post: %w", err)
		}
		if errMsg := result["error"]; errMsg != nil {
			return fmt.Errorf("creating LinkedIn post: %s", errMsg)
		}
		var ok bool
		urn, ok = result["urn"].(string)
		if !ok || urn == "" {
			return fmt.Errorf("linkitin returned empty URN for created post")
		}
		return nil
	})
	return urn, err
}

// SearchPosts searches for LinkedIn posts matching keywords.
func (c *LinkitinClient) SearchPosts(ctx context.Context, keywords string, limit int) ([]models.Post, error) {
	var posts []models.Post
	err := c.retryOnAuthError(ctx, "SearchPosts", func() error {
		result, err := c.runCommand(ctx, linkitinCommand{
			Action:   "search_posts",
			Keywords: keywords,
			Limit:    limit,
		})
		if err != nil {
			return fmt.Errorf("searching LinkedIn posts: %w", err)
		}
		if errMsg := result["error"]; errMsg != nil {
			return fmt.Errorf("searching LinkedIn posts: %s", errMsg)
		}
		posts, err = parseLinkitinPosts(result)
		return err
	})
	return posts, err
}

// UploadImage uploads an image and returns the media URN.
func (c *LinkitinClient) UploadImage(ctx context.Context, imageData []byte, filename string) (string, error) {
	encoded := base64.StdEncoding.EncodeToString(imageData)
	var mediaURN string
	err := c.retryOnAuthError(ctx, "UploadImage", func() error {
		result, err := c.runCommand(ctx, linkitinCommand{
			Action:    "upload_image",
			ImageData: encoded,
			Filename:  filename,
		})
		if err != nil {
			return fmt.Errorf("uploading image to LinkedIn: %w", err)
		}
		if errMsg := result["error"]; errMsg != nil {
			return fmt.Errorf("uploading image to LinkedIn: %s", errMsg)
		}
		var ok bool
		mediaURN, ok = result["media_urn"].(string)
		if !ok || mediaURN == "" {
			return fmt.Errorf("linkitin returned empty media URN")
		}
		return nil
	})
	return mediaURN, err
}

// Repost reshares an existing LinkedIn post, optionally with commentary text.
func (c *LinkitinClient) Repost(ctx context.Context, postURN string, text string) (string, error) {
	postURN = resolveLinkedInShareURN(postURN)
	var urn string
	err := c.retryOnAuthError(ctx, "Repost", func() error {
		result, err := c.runCommand(ctx, linkitinCommand{
			Action:  "repost",
			PostURN: postURN,
			Text:    text,
		})
		if err != nil {
			return fmt.Errorf("reposting LinkedIn post: %w", err)
		}
		if errMsg := result["error"]; errMsg != nil {
			return fmt.Errorf("reposting LinkedIn post: %s", errMsg)
		}
		var ok bool
		urn, ok = result["urn"].(string)
		if !ok || urn == "" {
			return fmt.Errorf("linkitin returned empty URN for repost")
		}
		return nil
	})
	return urn, err
}

// CreateComment posts a comment on an existing LinkedIn post.
// threadURN is the optional urn:li:ugcPost:N for the LinkedIn comment API's threadUrn field;
// pass "" to let linkitin derive it from postURN via _build_thread_urn.
func (c *LinkitinClient) CreateComment(ctx context.Context, postURN string, threadURN string, text string) (string, error) {
	postURN = resolveLinkedInCommentURN(postURN)
	var urn string
	err := c.retryOnAuthError(ctx, "CreateComment", func() error {
		result, err := c.runCommand(ctx, linkitinCommand{
			Action:    "comment_post",
			PostURN:   postURN,
			ThreadURN: threadURN,
			Text:      text,
		})
		if err != nil {
			return fmt.Errorf("commenting on LinkedIn post: %w", err)
		}
		if errMsg := result["error"]; errMsg != nil {
			return fmt.Errorf("commenting on LinkedIn post: %s", errMsg)
		}
		urn, _ = result["urn"].(string)
		return nil
	})
	return urn, err
}

// CreatePostWithImage creates a post with an attached image.
func (c *LinkitinClient) CreatePostWithImage(ctx context.Context, text string, imageData []byte, filename string) (string, error) {
	encoded := base64.StdEncoding.EncodeToString(imageData)
	var urn string
	err := c.retryOnAuthError(ctx, "CreatePostWithImage", func() error {
		result, err := c.runCommand(ctx, linkitinCommand{
			Action:    "create_post_with_image",
			Text:      text,
			ImageData: encoded,
			Filename:  filename,
		})
		if err != nil {
			return fmt.Errorf("creating LinkedIn post with image: %w", err)
		}
		if errMsg := result["error"]; errMsg != nil {
			return fmt.Errorf("creating LinkedIn post with image: %s", errMsg)
		}
		var ok bool
		urn, ok = result["urn"].(string)
		if !ok || urn == "" {
			return fmt.Errorf("linkitin returned empty URN for created post with image")
		}
		return nil
	})
	return urn, err
}

// CreateScheduledPost schedules a LinkedIn post for future publishing.
// Returns the post URN if available, or an empty string if the post was scheduled but URN is unavailable.
func (c *LinkitinClient) CreateScheduledPost(ctx context.Context, text string, scheduledAt time.Time) (string, error) {
	var urn string
	err := c.retryOnAuthError(ctx, "CreateScheduledPost", func() error {
		result, err := c.runCommand(ctx, linkitinCommand{
			Action:      "create_scheduled_post",
			Text:        text,
			ScheduledAt: scheduledAt.Format(time.RFC3339),
		})
		if err != nil {
			return fmt.Errorf("scheduling LinkedIn post: %w", err)
		}
		if errMsg := result["error"]; errMsg != nil {
			return fmt.Errorf("scheduling LinkedIn post: %s", errMsg)
		}
		if u, ok := result["urn"].(string); ok && u != "" {
			urn = u
		} else {
			urn = "scheduled"
		}
		return nil
	})
	return urn, err
}

// CreateScheduledPostWithImage schedules a LinkedIn post with an image for future publishing.
// Returns the post URN if available, or a placeholder string if the post was scheduled but URN is unavailable.
func (c *LinkitinClient) CreateScheduledPostWithImage(ctx context.Context, text string, imageData []byte, filename string, scheduledAt time.Time) (string, error) {
	encoded := base64.StdEncoding.EncodeToString(imageData)
	var urn string
	err := c.retryOnAuthError(ctx, "CreateScheduledPostWithImage", func() error {
		result, err := c.runCommand(ctx, linkitinCommand{
			Action:      "create_scheduled_post_with_image",
			Text:        text,
			ImageData:   encoded,
			Filename:    filename,
			ScheduledAt: scheduledAt.Format(time.RFC3339),
		})
		if err != nil {
			return fmt.Errorf("scheduling LinkedIn post with image: %w", err)
		}
		if errMsg := result["error"]; errMsg != nil {
			return fmt.Errorf("scheduling LinkedIn post with image: %s", errMsg)
		}
		if u, ok := result["urn"].(string); ok && u != "" {
			urn = u
		} else {
			urn = "scheduled"
		}
		return nil
	})
	return urn, err
}

// runCommand executes a single command against the linkitin bridge script.
// Each invocation spawns the script, sends the JSON command via stdin, and reads the response.
// A 60-second timeout prevents hung Python processes from blocking the pipeline.
func (c *LinkitinClient) runCommand(ctx context.Context, cmd linkitinCommand) (map[string]interface{}, error) {
	cmdJSON, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("marshaling linkitin command: %w", err)
	}
	// Append newline so the bridge reads the line.
	cmdJSON = append(cmdJSON, '\n')

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var stdout, stderr bytes.Buffer
	proc := exec.CommandContext(ctx, c.pythonPath, c.scriptPath)
	proc.Stdin = bytes.NewReader(cmdJSON)
	proc.Stdout = &stdout
	proc.Stderr = &stderr

	if err := proc.Run(); err != nil {
		if stdout.Len() > 0 {
			var errResp map[string]interface{}
			if jsonErr := json.Unmarshal(stdout.Bytes(), &errResp); jsonErr == nil {
				if errMsg, ok := errResp["error"]; ok {
					return nil, fmt.Errorf("linkitin: %s", errMsg)
				}
			}
		}
		stderrMsg := stderr.String()
		if stderrMsg != "" {
			return nil, fmt.Errorf("running linkitin subprocess: %w (stderr: %s)", err, stderrMsg)
		}
		return nil, fmt.Errorf("running linkitin subprocess: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("parsing linkitin output: %w (raw: %s)", err, stdout.String())
	}
	return result, nil
}

// linkitinCommand represents a JSON command sent to the linkitin bridge script.
type linkitinCommand struct {
	Action       string `json:"action"`
	LiAt         string `json:"li_at,omitempty"`
	JSessionID   string `json:"jsessionid,omitempty"`
	Text         string `json:"text,omitempty"`
	Keywords     string `json:"keywords,omitempty"`
	Visibility   string `json:"visibility,omitempty"`
	ImageData    string `json:"image_data,omitempty"`
	Filename     string `json:"filename,omitempty"`
	PostURN      string `json:"post_urn,omitempty"`
	ThreadURN    string `json:"thread_urn,omitempty"`
	ScheduledAt  string `json:"scheduled_at,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Topic        string `json:"topic,omitempty"`
	Period       string `json:"period,omitempty"`
	FromFollowed bool   `json:"from_followed,omitempty"`
	Scrolls      int    `json:"scrolls,omitempty"`
}

// resolveLinkedInShareURN converts LinkedIn fsd_update composite URNs (returned by the
// Voyager feed API) to the share URNs that linkitin's repost() requires.
//
// Feed posts arrive as:
//
//	urn:li:fsd_update:(urn:li:activity:ID,MAIN_FEED,...)
//
// Linkitin expects:
//
//	urn:li:share:ID
//
// Activity IDs and share IDs use the same numeric value for original posts, so
// the conversion is safe for typical feed content.
func resolveLinkedInShareURN(urn string) string {
	if !strings.HasPrefix(urn, "urn:li:fsd_update:") {
		return urn
	}
	// Strip leading "urn:li:fsd_update:(" and take the first comma-separated token.
	inner := strings.TrimPrefix(urn, "urn:li:fsd_update:(")
	if idx := strings.IndexByte(inner, ','); idx >= 0 {
		inner = inner[:idx]
	}
	// Convert activity URN → share URN.
	return strings.Replace(inner, "urn:li:activity:", "urn:li:share:", 1)
}

// linkitinPostJSON represents a post in the linkitin bridge JSON response.
type linkitinPostJSON struct {
	URN         string `json:"urn"`
	ShareURN    string `json:"share_urn"`
	ThreadURN   string `json:"thread_urn"`
	Text        string `json:"text"`
	Likes       int    `json:"likes"`
	Comments    int    `json:"comments"`
	Reposts     int    `json:"reposts"`
	Impressions int    `json:"impressions"`
	CreatedAt   string `json:"created_at"`
	Author      *struct {
		URN       string `json:"urn"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Headline  string `json:"headline"`
	} `json:"author"`
}

// isValidLinkedInPostID reports whether a resolved LinkedIn post ID (URN) is
// usable for actions like commenting or reposting.
//
// Linkitin uses "urn:li:dom:post:N" as an internal DOM-parsed identifier for
// any N when it cannot extract a real LinkedIn URN. These are NOT LinkedIn URNs
// and the LinkedIn API returns 400 for all of them (not just :0). Sponsored
// content URNs (sponsoredContentV2) are also rejected — LinkedIn returns 422
// for comment/repost attempts on ads.
func isValidLinkedInPostID(id string) bool {
	if id == "" {
		return false
	}
	// urn:li:dom:* are linkitin-internal DOM-parsed identifiers, not real
	// LinkedIn URNs. Reject all of them regardless of the numeric suffix.
	if strings.Contains(id, "urn:li:dom:") {
		return false
	}
	// urn:li:content:* are stable content-hash IDs assigned when linkitin's
	// DOM scraper ran as a fallback and no real URN was available. These cannot
	// be used with the LinkedIn comment/repost APIs.
	if strings.Contains(id, "urn:li:content:") {
		return false
	}
	// Sponsored content cannot be commented on or reposted.
	if strings.Contains(id, "sponsoredContent") {
		return false
	}
	return true
}

// resolveLinkedInCommentURN returns the URN format expected by linkitin's
// comment_post (and the underlying LinkedIn Voyager comment API).
//
// The comment API needs urn:li:activity:... as the thread URN.
//   - fsd_update URNs are passed through unchanged — linkitin's internal
//     _build_thread_urn already extracts the activity URN from them.
//   - share URNs (urn:li:share:N) are converted to activity URNs
//     (urn:li:activity:N); the numeric ID is the same for original posts.
//   - Activity URNs and anything else are passed through unchanged.
//
// Note: reposting uses resolveLinkedInShareURN (share URN required there).
func resolveLinkedInCommentURN(urn string) string {
	return strings.Replace(urn, "urn:li:share:", "urn:li:activity:", 1)
}

// parseLinkitinPosts parses the posts array from a linkitin bridge response.
func parseLinkitinPosts(result map[string]interface{}) ([]models.Post, error) {
	postsRaw, ok := result["posts"]
	if !ok {
		return nil, fmt.Errorf("linkitin response missing 'posts' field")
	}

	postsJSON, err := json.Marshal(postsRaw)
	if err != nil {
		return nil, fmt.Errorf("re-marshaling linkitin posts: %w", err)
	}

	var linkitinPosts []linkitinPostJSON
	if err := json.Unmarshal(postsJSON, &linkitinPosts); err != nil {
		return nil, fmt.Errorf("parsing linkitin posts: %w", err)
	}

	now := time.Now()
	posts := make([]models.Post, 0, len(linkitinPosts))
	for _, lp := range linkitinPosts {
		var postedAt time.Time
		if lp.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, lp.CreatedAt); err == nil {
				postedAt = t
			}
		}

		// Prefer share_urn as the post identifier: it's required by linkitin's
		// repost() API. Fall back to the activity/fsd_update URN if absent.
		postID := lp.URN
		if lp.ShareURN != "" {
			postID = lp.ShareURN
		}

		// Skip posts with malformed URNs (e.g. urn:li:dom:post:0 — linkitin
		// uses a :0 ID when it cannot parse the real post identifier).
		if !isValidLinkedInPostID(postID) {
			slog.Debug("skipping linkedin post with invalid URN", "urn", postID)
			continue
		}

		posts = append(posts, models.Post{
			Platform:       "linkedin",
			PlatformPostID: postID,
			Content:        lp.Text,
			Likes:          lp.Likes,
			Reposts:        lp.Reposts,
			Comments:       lp.Comments,
			Impressions:    lp.Impressions,
			PostedAt:       postedAt,
			FetchedAt:      now,
		})
	}

	return posts, nil
}

// mapPeriodToLinkitin converts GoViral period names to linkitin's required format.
func mapPeriodToLinkitin(period string) string {
	switch period {
	case "24h", "day":
		return "past-24h"
	case "7d", "week":
		return "past-week"
	case "30d", "month":
		return "past-month"
	default:
		return "past-week"
	}
}

// parseLinkitinTrendingPosts parses the posts array from a linkitin bridge response into TrendingPost values,
// populating AuthorName, AuthorUsername, and NicheTags.
func parseLinkitinTrendingPosts(result map[string]interface{}, niche string, now time.Time) ([]models.TrendingPost, error) {
	postsRaw, ok := result["posts"]
	if !ok {
		return nil, fmt.Errorf("linkitin response missing 'posts' field")
	}

	postsJSON, err := json.Marshal(postsRaw)
	if err != nil {
		return nil, fmt.Errorf("re-marshaling linkitin posts: %w", err)
	}

	var linkitinPosts []linkitinPostJSON
	if err := json.Unmarshal(postsJSON, &linkitinPosts); err != nil {
		return nil, fmt.Errorf("parsing linkitin posts: %w", err)
	}

	posts := make([]models.TrendingPost, 0, len(linkitinPosts))
	for _, lp := range linkitinPosts {
		var postedAt time.Time
		if lp.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, lp.CreatedAt); err == nil {
				postedAt = t
			}
		}

		var authorName, authorUsername string
		if lp.Author != nil {
			authorName = lp.Author.FirstName + " " + lp.Author.LastName
			authorUsername = lp.Author.URN
		}

		// Prefer share_urn as the post identifier for the same reason as parseLinkitinPosts.
		postID := lp.URN
		if lp.ShareURN != "" {
			postID = lp.ShareURN
		}

		// For trending discovery we keep all posts, including those where linkitin
		// could only derive a DOM-parsed "urn:li:dom:post:N" identifier (which is not
		// a real LinkedIn URN and cannot be used for repost/comment actions).
		// Replace sequential DOM indices with a stable content-hash so that the same
		// post gets the same DB primary key across repeated fetches.
		if strings.Contains(postID, "urn:li:dom:") {
			h := sha256.Sum256([]byte(lp.Text + authorName))
			postID = fmt.Sprintf("urn:li:content:%x", h[:8])
		}

		posts = append(posts, models.TrendingPost{
			Platform:       "linkedin",
			PlatformPostID: postID,
			Content:        lp.Text,
			Likes:          lp.Likes,
			Reposts:        lp.Reposts,
			Comments:       lp.Comments,
			Impressions:    lp.Impressions,
			AuthorName:     authorName,
			AuthorUsername: authorUsername,
			NicheTags:      []string{niche},
			PostedAt:       postedAt,
			FetchedAt:      now,
			ThreadURN:      lp.ThreadURN,
		})
	}

	return posts, nil
}

// ensureLinkitinVenv ensures the shared virtualenv exists and linkitin dependencies are installed.
func ensureLinkitinVenv() (string, error) {
	venvDir := filepath.Join(config.DefaultConfigDir(), "venv")
	venvPython := filepath.Join(venvDir, "bin", "python3")

	// If venv python already exists, ensure linkitin deps are installed.
	if _, err := os.Stat(venvPython); err == nil {
		if err := ensureLinkitinDeps(venvPython); err != nil {
			return "", fmt.Errorf("installing linkitin dependencies: %w", err)
		}
		return venvPython, nil
	}

	// Find system python to create the venv.
	systemPython, err := findLinkitinPython()
	if err != nil {
		return "", err
	}

	// Create the virtualenv.
	cmd := exec.Command(systemPython, "-m", "venv", venvDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("creating venv: %w (output: %s)", err, string(output))
	}

	if err := ensureLinkitinDeps(venvPython); err != nil {
		return "", fmt.Errorf("installing linkitin dependencies: %w", err)
	}

	return venvPython, nil
}

// ensureLinkitinDeps ensures the linkitin PyPI package is installed in the venv.
func ensureLinkitinDeps(pythonPath string) error {
	cmd := exec.Command(pythonPath, "-c", "import linkitin")
	if err := cmd.Run(); err != nil {
		install := exec.Command(pythonPath, "-m", "pip", "install", "linkitin", "-q")
		if output, installErr := install.CombinedOutput(); installErr != nil {
			return fmt.Errorf("installing linkitin: %w (output: %s)", installErr, string(output))
		}
	}
	return nil
}

// findLinkitinPython locates python3 or python on PATH.
func findLinkitinPython() (string, error) {
	for _, name := range []string{"python3", "python"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("neither python3 nor python found on PATH")
}

// ensureLinkitinScript writes the embedded linkitin_bridge.py to ~/.goviral/scripts/
// along with the linkitin package files so the bridge can import them.
func ensureLinkitinScript() (string, error) {
	scriptsDir := filepath.Join(config.DefaultConfigDir(), "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return "", fmt.Errorf("creating scripts directory: %w", err)
	}

	scriptPath := filepath.Join(scriptsDir, "linkitin_bridge.py")

	// Always overwrite to keep the script in sync with the embedded version.
	if err := os.WriteFile(scriptPath, linkitinScript, 0755); err != nil {
		return "", fmt.Errorf("writing script file: %w", err)
	}

	return scriptPath, nil
}
