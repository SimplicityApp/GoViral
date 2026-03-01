package tiktok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/shuhao/goviral/internal/config"
)

// Client interacts with the TikTok Content Posting API.
type Client struct {
	cfg        config.TikTokConfig
	httpClient *http.Client
}

// NewClient creates a TikTok API client.
func NewClient(cfg config.TikTokConfig) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{},
	}
}

// UploadVideo uploads a video to TikTok via the Content Posting API.
// Flow: init upload → upload video file → publish.
// Returns the publish ID on success.
func (c *Client) UploadVideo(ctx context.Context, videoPath string, description string, tags []string) (string, error) {
	if c.cfg.AccessToken == "" {
		return "", fmt.Errorf("no TikTok access token configured; run 'goviral tiktok-login' first")
	}

	videoFile, err := os.Open(videoPath)
	if err != nil {
		return "", fmt.Errorf("opening video file %s: %w", videoPath, err)
	}
	defer videoFile.Close()

	stat, err := videoFile.Stat()
	if err != nil {
		return "", fmt.Errorf("stat video file: %w", err)
	}

	// Step 1: Initialize upload
	initResp, err := c.initUpload(ctx, stat.Size())
	if err != nil {
		return "", fmt.Errorf("initializing TikTok upload: %w", err)
	}

	// Step 2: Upload video file
	videoData, err := io.ReadAll(videoFile)
	if err != nil {
		return "", fmt.Errorf("reading video file: %w", err)
	}

	if err := c.uploadChunk(ctx, initResp.UploadURL, videoData); err != nil {
		return "", fmt.Errorf("uploading video chunk: %w", err)
	}

	// Step 3: Publish
	// Build caption with hashtags
	caption := description
	for _, tag := range tags {
		if tag != "" {
			caption += " #" + tag
		}
	}

	publishID, err := c.publish(ctx, initResp.PublishID, caption)
	if err != nil {
		return "", fmt.Errorf("publishing TikTok video: %w", err)
	}

	return publishID, nil
}

// ScheduleVideo schedules a video for future posting on TikTok.
func (c *Client) ScheduleVideo(ctx context.Context, videoPath string, description string, tags []string, scheduledAt time.Time) (string, error) {
	// TikTok Content Posting API supports scheduled publishing via the schedule_publish_time field.
	if c.cfg.AccessToken == "" {
		return "", fmt.Errorf("no TikTok access token configured; run 'goviral tiktok-login' first")
	}

	videoFile, err := os.Open(videoPath)
	if err != nil {
		return "", fmt.Errorf("opening video file %s: %w", videoPath, err)
	}
	defer videoFile.Close()

	stat, err := videoFile.Stat()
	if err != nil {
		return "", fmt.Errorf("stat video file: %w", err)
	}

	initResp, err := c.initUpload(ctx, stat.Size())
	if err != nil {
		return "", fmt.Errorf("initializing TikTok upload: %w", err)
	}

	videoData, err := io.ReadAll(videoFile)
	if err != nil {
		return "", fmt.Errorf("reading video file: %w", err)
	}

	if err := c.uploadChunk(ctx, initResp.UploadURL, videoData); err != nil {
		return "", fmt.Errorf("uploading video chunk: %w", err)
	}

	caption := description
	for _, tag := range tags {
		if tag != "" {
			caption += " #" + tag
		}
	}

	publishID, err := c.publishScheduled(ctx, initResp.PublishID, caption, scheduledAt)
	if err != nil {
		return "", fmt.Errorf("scheduling TikTok video: %w", err)
	}

	return publishID, nil
}

type initUploadResponse struct {
	PublishID string
	UploadURL string
}

func (c *Client) initUpload(ctx context.Context, fileSize int64) (*initUploadResponse, error) {
	body := map[string]interface{}{
		"post_info": map[string]interface{}{
			"title":           "",
			"privacy_level":   "PUBLIC_TO_EVERYONE",
			"disable_duet":    false,
			"disable_stitch":  false,
			"disable_comment": false,
		},
		"source_info": map[string]interface{}{
			"source":            "FILE_UPLOAD",
			"video_size":        fileSize,
			"chunk_size":        fileSize,
			"total_chunk_count": 1,
		},
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling init request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://open.tiktokapis.com/v2/post/publish/inbox/video/init/", bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("creating init request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("init upload request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("init upload failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			PublishID string `json:"publish_id"`
			UploadURL string `json:"upload_url"`
		} `json:"data"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing init response: %w", err)
	}
	if result.Error.Code != "" && result.Error.Code != "ok" {
		return nil, fmt.Errorf("TikTok init error: %s - %s", result.Error.Code, result.Error.Message)
	}

	return &initUploadResponse{
		PublishID: result.Data.PublishID,
		UploadURL: result.Data.UploadURL,
	}, nil
}

func (c *Client) uploadChunk(ctx context.Context, uploadURL string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, "PUT", uploadURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating chunk upload request: %w", err)
	}
	req.Header.Set("Content-Type", "video/mp4")
	req.Header.Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", len(data)-1, len(data)))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("chunk upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chunk upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) publish(ctx context.Context, publishID string, caption string) (string, error) {
	body := map[string]interface{}{
		"publish_id": publishID,
		"post_info": map[string]interface{}{
			"title":         caption,
			"privacy_level": "PUBLIC_TO_EVERYONE",
		},
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshaling publish request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://open.tiktokapis.com/v2/post/publish/video/init/", bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("creating publish request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("publish request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("publish failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			PublishID string `json:"publish_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing publish response: %w", err)
	}

	return result.Data.PublishID, nil
}

func (c *Client) publishScheduled(ctx context.Context, publishID string, caption string, scheduledAt time.Time) (string, error) {
	body := map[string]interface{}{
		"publish_id": publishID,
		"post_info": map[string]interface{}{
			"title":                 caption,
			"privacy_level":         "PUBLIC_TO_EVERYONE",
			"schedule_publish_time": scheduledAt.Unix(),
		},
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshaling scheduled publish request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://open.tiktokapis.com/v2/post/publish/video/init/", bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("creating scheduled publish request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("scheduled publish request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("scheduled publish failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			PublishID string `json:"publish_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing scheduled publish response: %w", err)
	}

	return result.Data.PublishID, nil
}
