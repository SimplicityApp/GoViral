package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	apiURL         = "https://api.anthropic.com/v1/messages"
	anthropicVersion = "2023-06-01"
	defaultMaxTokens = 4096
	maxRetries       = 3
	retryBaseDelay   = time.Second
)

// MessageSender defines the interface for sending messages to an LLM.
type MessageSender interface {
	SendMessage(ctx context.Context, systemPrompt string, userMessage string) (string, error)
}

// Client is an Anthropic Claude API client.
type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient creates a new Claude API client.
func NewClient(apiKey string, model string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system"`
	Messages  []apiMessage `json:"messages"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiResponse struct {
	Content []apiContentBlock `json:"content"`
	Error   *apiError         `json:"error,omitempty"`
}

type apiContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type apiErrorResponse struct {
	Error apiError `json:"error"`
}

// SendMessage sends a message to the Claude API and returns the text response.
func (c *Client) SendMessage(ctx context.Context, systemPrompt string, userMessage string) (string, error) {
	reqBody := apiRequest{
		Model:     c.model,
		MaxTokens: defaultMaxTokens,
		System:    systemPrompt,
		Messages: []apiMessage{
			{Role: "user", Content: userMessage},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request body: %w", err)
	}

	var lastErr error
	for attempt := range maxRetries {
		result, err := c.doRequest(ctx, bodyBytes)
		if err == nil {
			return result, nil
		}

		if !isRetryable(err) {
			return "", err
		}

		lastErr = err
		delay := retryBaseDelay * time.Duration(1<<uint(attempt))
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("waiting for retry: %w", ctx.Err())
		case <-time.After(delay):
		}
	}

	return "", fmt.Errorf("sending message after %d retries: %w", maxRetries, lastErr)
}

func (c *Client) doRequest(ctx context.Context, body []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating HTTP request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("executing HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", &rateLimitError{statusCode: resp.StatusCode}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("authenticating with Claude API: invalid API key (status 401)")
	}

	if resp.StatusCode != http.StatusOK {
		var errResp apiErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return "", fmt.Errorf("calling Claude API (status %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return "", fmt.Errorf("calling Claude API: unexpected status %d", resp.StatusCode)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("unmarshaling response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("parsing Claude response: no content blocks returned")
	}

	return apiResp.Content[0].Text, nil
}

type rateLimitError struct {
	statusCode int
}

func (e *rateLimitError) Error() string {
	return fmt.Sprintf("rate limited by Claude API (status %d)", e.statusCode)
}

func isRetryable(err error) bool {
	_, ok := err.(*rateLimitError)
	return ok
}
