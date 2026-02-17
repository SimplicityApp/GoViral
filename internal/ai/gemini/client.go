package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/shuhao/goviral/internal/config"
)

// GeneratedImage holds the raw image data and MIME type.
type GeneratedImage struct {
	Data     []byte
	MIMEType string
}

// Client interacts with the Google Gemini API for image generation.
type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient creates a new Gemini client.
func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// GenerateImage sends a prompt to Gemini and returns a generated image.
func (c *Client) GenerateImage(ctx context.Context, prompt string) (*GeneratedImage, error) {
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"responseModalities": []string{"TEXT", "IMAGE"},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("executing request: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limited (HTTP 429)")
			if attempt == maxRetries {
				return nil, lastErr
			}
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, string(respBody))
		}

		return parseImageResponse(respBody)
	}

	return nil, lastErr
}

func parseImageResponse(body []byte) (*GeneratedImage, error) {
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text       string `json:"text,omitempty"`
					InlineData *struct {
						MIMEType string `json:"mimeType"`
						Data     string `json:"data"`
					} `json:"inlineData,omitempty"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing Gemini response: %w", err)
	}

	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil && isImageMIME(part.InlineData.MIMEType) {
				data, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
				if err != nil {
					return nil, fmt.Errorf("decoding image data: %w", err)
				}
				return &GeneratedImage{
					Data:     data,
					MIMEType: part.InlineData.MIMEType,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("no image found in Gemini response")
}

func isImageMIME(mime string) bool {
	switch mime {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return true
	}
	return false
}

// SaveImage writes the generated image to ~/.goviral/images/{name}.png and returns the file path.
func SaveImage(img *GeneratedImage, name string) (string, error) {
	dir := filepath.Join(config.DefaultConfigDir(), "images")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating images directory: %w", err)
	}

	ext := ".png"
	switch img.MIMEType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/webp":
		ext = ".webp"
	case "image/gif":
		ext = ".gif"
	}

	path := filepath.Join(dir, name+ext)
	if err := os.WriteFile(path, img.Data, 0644); err != nil {
		return "", fmt.Errorf("writing image file: %w", err)
	}

	return path, nil
}
