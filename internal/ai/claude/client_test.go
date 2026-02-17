package claude

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
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

func newTestClient(serverURL string) *Client {
	return &Client{
		apiKey: "test-api-key",
		model:  "claude-test",
		httpClient: &http.Client{
			Transport: &rewriteTransport{serverURL: serverURL},
		},
	}
}

func TestSendMessage_Success(t *testing.T) {
	resp := mustReadFile(t, "../../../testdata/claude_persona_response.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("x-api-key"); got != "test-api-key" {
			t.Errorf("x-api-key = %q, want 'test-api-key'", got)
		}
		if got := r.Header.Get("anthropic-version"); got != anthropicVersion {
			t.Errorf("anthropic-version = %q, want %q", got, anthropicVersion)
		}
		if got := r.Header.Get("content-type"); got != "application/json" {
			t.Errorf("content-type = %q, want 'application/json'", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.SendMessage(context.Background(), "system prompt", "user message")
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if result == "" {
		t.Fatal("SendMessage() returned empty result")
	}
	if !strings.Contains(result, "writing_tone") {
		t.Errorf("SendMessage() result doesn't contain expected persona content: %q", result[:min(len(result), 100)])
	}
}

func TestSendMessage_ResponseParsing(t *testing.T) {
	resp := mustReadFile(t, "../../../testdata/claude_generate_response.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.SendMessage(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	// Should contain the generated content JSON array
	if !strings.Contains(result, "confidence_score") {
		t.Errorf("SendMessage() result doesn't contain expected content: %q", result[:min(len(result), 100)])
	}
}

func TestSendMessage_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"type":"authentication_error","message":"invalid api key"}}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.SendMessage(context.Background(), "system", "user")
	if err == nil {
		t.Fatal("SendMessage() expected error for 401, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to mention 401, got: %v", err)
	}
}

func TestSendMessage_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	client := newTestClient(server.URL)
	_, err := client.SendMessage(ctx, "system", "user")
	if err == nil {
		t.Fatal("SendMessage() expected error for 429 response, got nil")
	}
}

func TestSendMessage_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"type":"server_error","message":"internal server error"}}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.SendMessage(context.Background(), "system", "user")
	if err == nil {
		t.Fatal("SendMessage() expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "internal server error") {
		t.Errorf("expected error to contain 'internal server error', got: %v", err)
	}
}

func TestSendMessage_EmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content":[]}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.SendMessage(context.Background(), "system", "user")
	if err == nil {
		t.Fatal("SendMessage() expected error for empty content blocks, got nil")
	}
	if !strings.Contains(err.Error(), "no content blocks") {
		t.Errorf("expected error about no content blocks, got: %v", err)
	}
}

func TestSendMessage_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := newTestClient(server.URL)
	_, err := client.SendMessage(ctx, "system", "user")
	if err == nil {
		t.Fatal("SendMessage() expected error for cancelled context, got nil")
	}
}
