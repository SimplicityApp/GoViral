package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shuhao/goviral/apps/server/dto"
)

const sseHeartbeatInterval = 15 * time.Second

// WantsSSE returns true if the client accepts text/event-stream.
func WantsSSE(r *http.Request) bool {
	return r.Header.Get("Accept") == "text/event-stream"
}

// SSEWriter sends SSE events to the client.
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewSSEWriter prepares the response for SSE streaming.
func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, _ := w.(http.Flusher)
	if flusher != nil {
		flusher.Flush()
	}

	return &SSEWriter{w: w, flusher: flusher}
}

// SendEvent writes a single SSE event.
func (s *SSEWriter) SendEvent(event dto.ProgressEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling SSE event: %w", err)
	}

	_, err = fmt.Fprintf(s.w, "data: %s\n\n", data)
	if err != nil {
		return fmt.Errorf("writing SSE event: %w", err)
	}

	if s.flusher != nil {
		s.flusher.Flush()
	}
	return nil
}

// SendHeartbeat writes a comment line to keep the connection alive.
func (s *SSEWriter) SendHeartbeat() {
	fmt.Fprint(s.w, ": heartbeat\n\n")
	if s.flusher != nil {
		s.flusher.Flush()
	}
}

// StreamProgress reads from a progress channel, sending each event via SSE.
// It sends heartbeats every 15 seconds to keep the connection alive.
// Returns when the channel is closed or the request context is cancelled.
func StreamProgress(w http.ResponseWriter, r *http.Request, ch <-chan dto.ProgressEvent) {
	sse := NewSSEWriter(w)
	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return
			}
			if err := sse.SendEvent(event); err != nil {
				return
			}
		case <-heartbeat.C:
			sse.SendHeartbeat()
		case <-r.Context().Done():
			return
		}
	}
}
