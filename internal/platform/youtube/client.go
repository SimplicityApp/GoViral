package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuhao/goviral/internal/config"
)

// Client interacts with the YouTube Data API v3.
type Client struct {
	cfg        config.YouTubeConfig
	httpClient *http.Client
}

// NewClient creates a YouTube API client.
func NewClient(cfg config.YouTubeConfig) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{},
	}
}

// UploadVideo uploads a video to YouTube using the resumable upload protocol.
// For Shorts, the video should be vertical and <60s. Add #Shorts to the title/description.
// Returns the video ID on success.
func (c *Client) UploadVideo(ctx context.Context, videoPath string, title string, description string, tags []string) (string, error) {
	if c.cfg.AccessToken == "" {
		return "", fmt.Errorf("no YouTube access token configured; run 'goviral youtube-login' first")
	}

	// Build video resource metadata
	snippet := map[string]interface{}{
		"title":       title,
		"description": description,
		"tags":        tags,
		"categoryId":  "28", // Science & Technology
	}
	status := map[string]interface{}{
		"privacyStatus":           "public",
		"selfDeclaredMadeForKids": false,
	}
	videoResource := map[string]interface{}{
		"snippet": snippet,
		"status":  status,
	}

	metadataJSON, err := json.Marshal(videoResource)
	if err != nil {
		return "", fmt.Errorf("marshaling video metadata: %w", err)
	}

	// Open the video file
	videoFile, err := os.Open(videoPath)
	if err != nil {
		return "", fmt.Errorf("opening video file %s: %w", videoPath, err)
	}
	defer videoFile.Close()

	stat, err := videoFile.Stat()
	if err != nil {
		return "", fmt.Errorf("stat video file: %w", err)
	}

	// Build multipart request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Part 1: metadata
	metaPart, err := writer.CreatePart(map[string][]string{
		"Content-Type": {"application/json; charset=UTF-8"},
	})
	if err != nil {
		return "", fmt.Errorf("creating metadata part: %w", err)
	}
	if _, err := metaPart.Write(metadataJSON); err != nil {
		return "", fmt.Errorf("writing metadata part: %w", err)
	}

	// Part 2: video file
	ext := strings.ToLower(filepath.Ext(videoPath))
	mimeType := "video/mp4"
	switch ext {
	case ".webm":
		mimeType = "video/webm"
	case ".mov":
		mimeType = "video/quicktime"
	case ".avi":
		mimeType = "video/x-msvideo"
	}

	videoPart, err := writer.CreatePart(map[string][]string{
		"Content-Type": {mimeType},
	})
	if err != nil {
		return "", fmt.Errorf("creating video part: %w", err)
	}
	if _, err := io.Copy(videoPart, videoFile); err != nil {
		return "", fmt.Errorf("copying video data: %w", err)
	}
	writer.Close()

	_ = stat // used for Content-Length if needed

	url := "https://www.googleapis.com/upload/youtube/v3/videos?uploadType=multipart&part=snippet,status"
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return "", fmt.Errorf("creating upload request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("uploading video: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("YouTube upload failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing upload response: %w", err)
	}

	return result.ID, nil
}

// UploadVideoWithThumbnail uploads a video then sets a custom thumbnail.
func (c *Client) UploadVideoWithThumbnail(ctx context.Context, videoPath string, thumbnailPath string, title string, description string, tags []string) (string, error) {
	videoID, err := c.UploadVideo(ctx, videoPath, title, description, tags)
	if err != nil {
		return "", err
	}

	if err := c.setThumbnail(ctx, videoID, thumbnailPath); err != nil {
		// Video was uploaded successfully, log thumbnail error but return the video ID
		return videoID, fmt.Errorf("video uploaded (id=%s) but thumbnail failed: %w", videoID, err)
	}

	return videoID, nil
}

func (c *Client) setThumbnail(ctx context.Context, videoID string, thumbnailPath string) error {
	thumbData, err := os.ReadFile(thumbnailPath)
	if err != nil {
		return fmt.Errorf("reading thumbnail %s: %w", thumbnailPath, err)
	}

	url := fmt.Sprintf("https://www.googleapis.com/upload/youtube/v3/thumbnails/set?videoId=%s&uploadType=media", videoID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(thumbData))
	if err != nil {
		return fmt.Errorf("creating thumbnail request: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(thumbnailPath))
	mimeType := "image/png"
	if ext == ".jpg" || ext == ".jpeg" {
		mimeType = "image/jpeg"
	}

	req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	req.Header.Set("Content-Type", mimeType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("uploading thumbnail: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("thumbnail upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
