package x

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/pkg/models"
)

//go:embed scripts/twikit_guest.py
var twikitScript []byte

// TwikitClient fetches tweets via a Python twikit subprocess using cookie-based auth.
// Requires a one-time login via Login() which saves cookies to ~/.goviral/twikit_cookies.json.
type TwikitClient struct {
	username   string
	pythonPath string
	scriptPath string
}

// NewTwikitClient creates a TwikitClient. Returns an error if python3/python
// is not on PATH or the embedded script cannot be written to disk.
// It creates a dedicated virtualenv at ~/.goviral/venv/ to avoid
// issues with externally-managed Python environments (PEP 668).
func NewTwikitClient(username string) (*TwikitClient, error) {
	pythonPath, err := ensureVenv()
	if err != nil {
		return nil, fmt.Errorf("setting up python venv: %w", err)
	}

	scriptPath, err := ensureScript()
	if err != nil {
		return nil, fmt.Errorf("writing twikit script: %w", err)
	}

	return &TwikitClient{
		username:   username,
		pythonPath: pythonPath,
		scriptPath: scriptPath,
	}, nil
}

// FetchMyPosts fetches the user's tweets via the twikit Python subprocess.
func (c *TwikitClient) FetchMyPosts(ctx context.Context, limit int) ([]models.Post, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, c.pythonPath, c.scriptPath,
		"fetch_user_tweets", c.username, strconv.Itoa(limit))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Try to extract error from JSON stdout first.
		if stdout.Len() > 0 {
			var errResp twikitErrorResponse
			if jsonErr := json.Unmarshal(stdout.Bytes(), &errResp); jsonErr == nil && errResp.Error != "" {
				return nil, fmt.Errorf("twikit: %s", errResp.Error)
			}
		}
		stderrMsg := stderr.String()
		if stderrMsg != "" {
			return nil, fmt.Errorf("running twikit subprocess: %w (stderr: %s)", err, stderrMsg)
		}
		return nil, fmt.Errorf("running twikit subprocess: %w", err)
	}

	return parseTwikitOutput(stdout.Bytes())
}

// ExtractCookies extracts X session cookies from Chrome and saves them
// to ~/.goviral/twikit_cookies.json for subsequent use. The user must be
// logged into X in Chrome. Only needs to be run once (or when cookies expire).
func (c *TwikitClient) ExtractCookies(ctx context.Context) error {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, c.pythonPath, c.scriptPath, "extract_cookies")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stdout.Len() > 0 {
			var errResp twikitErrorResponse
			if jsonErr := json.Unmarshal(stdout.Bytes(), &errResp); jsonErr == nil && errResp.Error != "" {
				return fmt.Errorf("extracting cookies: %s", errResp.Error)
			}
		}
		stderrMsg := stderr.String()
		if stderrMsg != "" {
			return fmt.Errorf("extract cookies subprocess: %w (stderr: %s)", err, stderrMsg)
		}
		return fmt.Errorf("extract cookies subprocess: %w", err)
	}

	// Check for error in JSON response.
	var errResp twikitErrorResponse
	if err := json.Unmarshal(stdout.Bytes(), &errResp); err == nil && errResp.Error != "" {
		return fmt.Errorf("extracting cookies: %s", errResp.Error)
	}

	return nil
}

// PostTweet creates a new tweet via the twikit Python subprocess.
func (c *TwikitClient) PostTweet(ctx context.Context, text string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, c.pythonPath, c.scriptPath, "create_tweet", text)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", parseTwikitError(err, stdout.Bytes(), stderr.String(), "posting tweet")
	}

	return parseTwikitTweetID(stdout.Bytes())
}

// PostQuoteTweet creates a quote tweet via the twikit Python subprocess.
func (c *TwikitClient) PostQuoteTweet(ctx context.Context, text string, quoteTweetID string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, c.pythonPath, c.scriptPath, "create_quote_tweet", text, quoteTweetID)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", parseTwikitError(err, stdout.Bytes(), stderr.String(), "posting quote tweet")
	}

	return parseTwikitTweetID(stdout.Bytes())
}

// PostReply creates a reply to an existing tweet via the twikit Python subprocess.
func (c *TwikitClient) PostReply(ctx context.Context, text string, inReplyToID string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, c.pythonPath, c.scriptPath, "create_tweet", text, inReplyToID)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", parseTwikitError(err, stdout.Bytes(), stderr.String(), "posting reply")
	}

	return parseTwikitTweetID(stdout.Bytes())
}

func parseTwikitError(err error, stdout []byte, stderr string, action string) error {
	if len(stdout) > 0 {
		var errResp twikitErrorResponse
		if jsonErr := json.Unmarshal(stdout, &errResp); jsonErr == nil && errResp.Error != "" {
			return fmt.Errorf("twikit: %s", errResp.Error)
		}
	}
	if stderr != "" {
		return fmt.Errorf("%s via twikit: %w (stderr: %s)", action, err, stderr)
	}
	return fmt.Errorf("%s via twikit: %w", action, err)
}

func parseTwikitTweetID(data []byte) (string, error) {
	var errResp twikitErrorResponse
	if err := json.Unmarshal(data, &errResp); err == nil && errResp.Error != "" {
		return "", fmt.Errorf("twikit: %s", errResp.Error)
	}

	var resp struct {
		TweetID string `json:"tweet_id"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parsing twikit tweet response: %w", err)
	}
	if resp.TweetID == "" {
		return "", fmt.Errorf("twikit returned empty tweet_id")
	}
	return resp.TweetID, nil
}

// UploadMedia uploads image data via the twikit Python subprocess by writing to a temp file.
func (c *TwikitClient) UploadMedia(ctx context.Context, imageData []byte, mimeType string) (string, error) {
	tmpFile, err := os.CreateTemp("", "goviral-media-*.png")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(imageData); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	tmpFile.Close()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, c.pythonPath, c.scriptPath, "upload_media", tmpFile.Name())
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", parseTwikitError(err, stdout.Bytes(), stderr.String(), "uploading media")
	}

	var errResp twikitErrorResponse
	if err := json.Unmarshal(stdout.Bytes(), &errResp); err == nil && errResp.Error != "" {
		return "", fmt.Errorf("twikit: %s", errResp.Error)
	}

	var resp struct {
		MediaID string `json:"media_id"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return "", fmt.Errorf("parsing twikit upload response: %w", err)
	}
	if resp.MediaID == "" {
		return "", fmt.Errorf("twikit returned empty media_id")
	}
	return resp.MediaID, nil
}

// PostTweetWithMedia posts a tweet with media via the twikit Python subprocess.
func (c *TwikitClient) PostTweetWithMedia(ctx context.Context, text string, mediaIDs []string) (string, error) {
	mediaJSON, err := json.Marshal(mediaIDs)
	if err != nil {
		return "", fmt.Errorf("marshaling media IDs: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, c.pythonPath, c.scriptPath, "create_tweet", text, "", string(mediaJSON))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", parseTwikitError(err, stdout.Bytes(), stderr.String(), "posting tweet with media")
	}

	return parseTwikitTweetID(stdout.Bytes())
}

// PostReplyWithMedia posts a reply with media via the twikit Python subprocess.
func (c *TwikitClient) PostReplyWithMedia(ctx context.Context, text string, inReplyToID string, mediaIDs []string) (string, error) {
	mediaJSON, err := json.Marshal(mediaIDs)
	if err != nil {
		return "", fmt.Errorf("marshaling media IDs: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, c.pythonPath, c.scriptPath, "create_tweet", text, inReplyToID, string(mediaJSON))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", parseTwikitError(err, stdout.Bytes(), stderr.String(), "posting reply with media")
	}

	return parseTwikitTweetID(stdout.Bytes())
}

// ScheduleTweet schedules a tweet for future posting via X's native scheduling.
func (c *TwikitClient) ScheduleTweet(ctx context.Context, text string, scheduledAtUnix int64) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, c.pythonPath, c.scriptPath,
		"schedule_tweet", text, strconv.FormatInt(scheduledAtUnix, 10))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", parseTwikitError(err, stdout.Bytes(), stderr.String(), "scheduling tweet")
	}

	var errResp twikitErrorResponse
	if err := json.Unmarshal(stdout.Bytes(), &errResp); err == nil && errResp.Error != "" {
		return "", fmt.Errorf("twikit: %s", errResp.Error)
	}

	var resp struct {
		ScheduledTweetID string `json:"scheduled_tweet_id"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return "", fmt.Errorf("parsing twikit schedule response: %w", err)
	}
	if resp.ScheduledTweetID == "" {
		return "", fmt.Errorf("twikit returned empty scheduled_tweet_id")
	}
	return resp.ScheduledTweetID, nil
}

// ScheduleQuoteTweet schedules a quote tweet for future posting via X's native scheduling.
func (c *TwikitClient) ScheduleQuoteTweet(ctx context.Context, text string, quoteTweetID string, scheduledAtUnix int64) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, c.pythonPath, c.scriptPath,
		"schedule_quote_tweet", text, quoteTweetID, strconv.FormatInt(scheduledAtUnix, 10))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", parseTwikitError(err, stdout.Bytes(), stderr.String(), "scheduling quote tweet")
	}

	var errResp twikitErrorResponse
	if err := json.Unmarshal(stdout.Bytes(), &errResp); err == nil && errResp.Error != "" {
		return "", fmt.Errorf("twikit: %s", errResp.Error)
	}

	var resp struct {
		ScheduledTweetID string `json:"scheduled_tweet_id"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return "", fmt.Errorf("parsing twikit schedule quote tweet response: %w", err)
	}
	if resp.ScheduledTweetID == "" {
		return "", fmt.Errorf("twikit returned empty scheduled_tweet_id for quote tweet")
	}
	return resp.ScheduledTweetID, nil
}

// FetchTrendingPosts searches for trending/top tweets matching the given niches
// via the twikit Python subprocess using cookie-based auth.
func (c *TwikitClient) FetchTrendingPosts(ctx context.Context, niches []string, period string, minLikes int, limit int) ([]models.TrendingPost, error) {
	nichesJSON, err := json.Marshal(niches)
	if err != nil {
		return nil, fmt.Errorf("marshaling niches: %w", err)
	}

	if period == "" {
		period = "day"
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, c.pythonPath, c.scriptPath,
		"search_trending", string(nichesJSON), strconv.Itoa(minLikes), strconv.Itoa(limit), period)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stdout.Len() > 0 {
			var errResp twikitErrorResponse
			if jsonErr := json.Unmarshal(stdout.Bytes(), &errResp); jsonErr == nil && errResp.Error != "" {
				return nil, fmt.Errorf("twikit: %s", errResp.Error)
			}
		}
		stderrMsg := stderr.String()
		if stderrMsg != "" {
			return nil, fmt.Errorf("running twikit subprocess: %w (stderr: %s)", err, stderrMsg)
		}
		return nil, fmt.Errorf("running twikit subprocess: %w", err)
	}

	return parseTwikitTrendingOutput(stdout.Bytes())
}

// findPython locates python3 or python on PATH.
func findPython() (string, error) {
	for _, name := range []string{"python3", "python"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("neither python3 nor python found on PATH")
}

// ensureVenv creates a virtualenv at ~/.goviral/venv/ if it doesn't exist
// and returns the path to the venv's python binary. This avoids PEP 668
// "externally managed environment" errors with Homebrew Python.
func ensureVenv() (string, error) {
	venvDir := filepath.Join(config.DefaultConfigDir(), "venv")
	venvPython := filepath.Join(venvDir, "bin", "python3")

	// If venv python already exists, reuse it.
	if _, err := os.Stat(venvPython); err == nil {
		return venvPython, nil
	}

	// Find system python to create the venv.
	systemPython, err := findPython()
	if err != nil {
		return "", err
	}

	// Create the virtualenv.
	cmd := exec.Command(systemPython, "-m", "venv", venvDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("creating venv: %w (output: %s)", err, string(output))
	}

	return venvPython, nil
}

// ensureScript writes the embedded twikit_guest.py to ~/.goviral/scripts/.
func ensureScript() (string, error) {
	dir := filepath.Join(config.DefaultConfigDir(), "scripts")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating scripts directory: %w", err)
	}

	scriptPath := filepath.Join(dir, "twikit_guest.py")

	// Always overwrite to keep the script in sync with the embedded version.
	if err := os.WriteFile(scriptPath, twikitScript, 0755); err != nil {
		return "", fmt.Errorf("writing script file: %w", err)
	}

	return scriptPath, nil
}

// parseTwikitOutput parses the JSON output from the twikit Python script into Posts.
func parseTwikitOutput(data []byte) ([]models.Post, error) {
	var errResp twikitErrorResponse
	if err := json.Unmarshal(data, &errResp); err == nil && errResp.Error != "" {
		return nil, fmt.Errorf("twikit: %s", errResp.Error)
	}

	var resp twikitResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing twikit output: %w", err)
	}

	now := time.Now()
	posts := make([]models.Post, 0, len(resp.Tweets))
	for _, t := range resp.Tweets {
		postedAt, _ := time.Parse("Mon Jan 02 15:04:05 +0000 2006", t.CreatedAt)
		posts = append(posts, models.Post{
			Platform:       "x",
			PlatformPostID: t.ID,
			Content:        t.Text,
			Likes:          t.Likes,
			Reposts:        t.Retweets,
			Comments:       t.Replies,
			Impressions:    t.Impressions,
			PostedAt:       postedAt,
			FetchedAt:      now,
		})
	}

	return posts, nil
}

// parseTwikitTrendingOutput parses the JSON output from the search_trending command.
func parseTwikitTrendingOutput(data []byte) ([]models.TrendingPost, error) {
	var errResp twikitErrorResponse
	if err := json.Unmarshal(data, &errResp); err == nil && errResp.Error != "" {
		return nil, fmt.Errorf("twikit: %s", errResp.Error)
	}

	var resp twikitTrendingResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing twikit trending output: %w", err)
	}

	now := time.Now()
	posts := make([]models.TrendingPost, 0, len(resp.Trending))
	for _, t := range resp.Trending {
		postedAt, _ := time.Parse("Mon Jan 02 15:04:05 +0000 2006", t.CreatedAt)
		var media []models.MediaAttachment
		for _, m := range t.Media {
			media = append(media, models.MediaAttachment{
				Type:       m.Type,
				URL:        m.URL,
				PreviewURL: m.PreviewURL,
				AltText:    m.AltText,
			})
		}
		posts = append(posts, models.TrendingPost{
			Platform:       "x",
			PlatformPostID: t.ID,
			AuthorUsername: t.AuthorUsername,
			AuthorName:     t.AuthorName,
			Content:        t.Text,
			Likes:          t.Likes,
			Reposts:        t.Retweets,
			Comments:       t.Replies,
			Impressions:    t.Impressions,
			NicheTags:      []string{t.Niche},
			Media:          media,
			PostedAt:       postedAt,
			FetchedAt:      now,
		})
	}

	return posts, nil
}

// JSON response types for the twikit Python script output.

type twikitResponse struct {
	Tweets []twikitTweet `json:"tweets"`
}

type twikitMediaItem struct {
	Type       string `json:"type"`
	URL        string `json:"url"`
	PreviewURL string `json:"preview_url"`
	AltText    string `json:"alt_text"`
}

type twikitTweet struct {
	ID          string           `json:"id"`
	Text        string           `json:"text"`
	CreatedAt   string           `json:"created_at"`
	Likes       int              `json:"likes"`
	Retweets    int              `json:"retweets"`
	Replies     int              `json:"replies"`
	Impressions int              `json:"impressions"`
	Media       []twikitMediaItem `json:"media"`
}

type twikitTrendingResponse struct {
	Trending []twikitTrendingTweet `json:"trending"`
}

type twikitTrendingTweet struct {
	ID             string           `json:"id"`
	Text           string           `json:"text"`
	CreatedAt      string           `json:"created_at"`
	AuthorUsername string           `json:"author_username"`
	AuthorName     string           `json:"author_name"`
	Likes          int              `json:"likes"`
	Retweets       int              `json:"retweets"`
	Replies        int              `json:"replies"`
	Impressions    int              `json:"impressions"`
	Niche          string           `json:"niche"`
	Media          []twikitMediaItem `json:"media"`
}

type twikitErrorResponse struct {
	Error string `json:"error"`
}
