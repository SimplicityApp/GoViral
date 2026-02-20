package telegram

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
	apiBase         = "https://api.telegram.org/bot"
	apiTimeout      = 30 * time.Second // regular API calls
	pollHTTPTimeout = 60 * time.Second // must exceed the Telegram poll timeout
	pollTimeout     = 25               // seconds Telegram waits for an update
)

// Client is a Telegram Bot API client.
type Client struct {
	token          string
	httpClient     *http.Client // for regular API calls
	pollHTTPClient *http.Client // for long-polling getUpdates
}

// NewClient creates a new Telegram Bot API client.
func NewClient(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: apiTimeout,
		},
		pollHTTPClient: &http.Client{
			Timeout: pollHTTPTimeout,
		},
	}
}

// Update represents a Telegram update (message, callback, etc.).
type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

// Message represents a Telegram message.
type Message struct {
	MessageID      int64  `json:"message_id"`
	Chat           Chat   `json:"chat"`
	Text           string `json:"text"`
	ReplyToMessage *Message `json:"reply_to_message,omitempty"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID int64 `json:"id"`
}

type apiResponse struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Desc   string          `json:"description,omitempty"`
}

type sendMessageRequest struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type editMessageRequest struct {
	ChatID    int64  `json:"chat_id"`
	MessageID int64  `json:"message_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type sentMessage struct {
	MessageID int64 `json:"message_id"`
}

// SendMessage sends a text message and returns the message ID.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text, parseMode string) (int64, error) {
	body := sendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: parseMode,
	}
	data, err := c.call(ctx, "sendMessage", body)
	if err != nil {
		return 0, fmt.Errorf("sending message: %w", err)
	}
	var msg sentMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return 0, fmt.Errorf("parsing send response: %w", err)
	}
	return msg.MessageID, nil
}

// SendMessageWithMarkdown sends a Markdown-formatted message.
func (c *Client) SendMessageWithMarkdown(ctx context.Context, chatID int64, text string) (int64, error) {
	return c.SendMessage(ctx, chatID, text, "Markdown")
}

// EditMessage edits an existing message.
func (c *Client) EditMessage(ctx context.Context, chatID, messageID int64, text string) error {
	body := editMessageRequest{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      text,
		ParseMode: "Markdown",
	}
	_, err := c.call(ctx, "editMessageText", body)
	if err != nil {
		return fmt.Errorf("editing message: %w", err)
	}
	return nil
}

// GetUpdates retrieves updates using long polling.
// It uses a dedicated HTTP client with a longer timeout than the poll window.
func (c *Client) GetUpdates(ctx context.Context, offset int64) ([]Update, error) {
	body := map[string]interface{}{
		"offset":  offset,
		"timeout": pollTimeout,
	}
	data, err := c.callWithClient(ctx, c.pollHTTPClient, "getUpdates", body)
	if err != nil {
		return nil, fmt.Errorf("getting updates: %w", err)
	}
	var updates []Update
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, fmt.Errorf("parsing updates: %w", err)
	}
	return updates, nil
}

// SetWebhook registers a webhook URL with Telegram.
func (c *Client) SetWebhook(ctx context.Context, url string) error {
	body := map[string]interface{}{
		"url": url,
	}
	_, err := c.call(ctx, "setWebhook", body)
	if err != nil {
		return fmt.Errorf("setting webhook: %w", err)
	}
	return nil
}

// DeleteWebhook removes the webhook.
func (c *Client) DeleteWebhook(ctx context.Context) error {
	_, err := c.call(ctx, "deleteWebhook", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("deleting webhook: %w", err)
	}
	return nil
}

// ParseWebhookUpdate parses an incoming webhook request body into an Update.
func ParseWebhookUpdate(r *http.Request) (*Update, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("reading webhook body: %w", err)
	}
	var update Update
	if err := json.Unmarshal(body, &update); err != nil {
		return nil, fmt.Errorf("parsing webhook update: %w", err)
	}
	return &update, nil
}

func (c *Client) call(ctx context.Context, method string, body interface{}) (json.RawMessage, error) {
	return c.callWithClient(ctx, c.httpClient, method, body)
}

func (c *Client) callWithClient(ctx context.Context, client *http.Client, method string, body interface{}) (json.RawMessage, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	url := apiBase + c.token + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("telegram API error: %s", apiResp.Desc)
	}

	return apiResp.Result, nil
}
