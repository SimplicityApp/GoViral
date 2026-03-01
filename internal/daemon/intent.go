package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/pkg/models"
)

// IntentParser parses natural-language Telegram replies into structured intents.
type IntentParser struct {
	sender claude.MessageSender
}

// NewIntentParser creates a new intent parser.
func NewIntentParser(sender claude.MessageSender) *IntentParser {
	return &IntentParser{sender: sender}
}

var (
	simpleApprove   = regexp.MustCompile(`(?i)^(approve|yes|ok|go|post|lgtm|looks good|ship it)\s*$`)
	simpleReject    = regexp.MustCompile(`(?i)^(reject|no|skip|discard|nah|pass)\s*$`)
	approveSpecific = regexp.MustCompile(`(?i)^approve\s+([\d,\s]+)$`)
	schedulePattern = regexp.MustCompile(`(?i)^schedule\s+(\d+)\s*([hm])(?:\s+for\s+(?:draft\s+)?([\d,\s]+))?$`)
	readPattern     = regexp.MustCompile(`(?i)^(?:read|show|original|full)\s+(?:draft\s+)?(\d+)(?:\s+from\s+batch\s+\d+)?$`)
	rewritePattern  = regexp.MustCompile(`(?i)^rewrite\s+(?:draft\s+)?(\d+)(?:\s+from\s+batch\s+\d+)?\s*(.*)$`)
	repostToggle    = regexp.MustCompile(`(?i)\b(?:as|to be)\s+a\s+(repost|rewrite)(?:\s+instead)?\b`)
	batchRefPattern = regexp.MustCompile(`(?i)\bbatch\s+(\d+)\b`)

	intentSchema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action":     map[string]any{"type": "string"},
			"content_ids": map[string]any{
				"type":  []any{"array", "null"},
				"items": map[string]any{"type": "integer"},
			},
			"edits": map[string]any{
				"type": []any{"array", "null"},
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"draft_number": map[string]any{"type": "integer"},
						"new_text":     map[string]any{"type": "string"},
					},
					"required":             []string{"draft_number", "new_text"},
					"additionalProperties": false,
				},
			},
			"schedule_at_minutes": map[string]any{"type": []any{"integer", "null"}},
			"is_repost":           map[string]any{"type": []any{"boolean", "null"}},
			"message":             map[string]any{"type": "string"},
		},
		"required":             []string{"action", "content_ids", "edits", "schedule_at_minutes", "is_repost", "message"},
		"additionalProperties": false,
	}
)

// Parse interprets a reply text into a DaemonIntent.
// It uses fast-path regex for simple commands and falls back to Claude for complex ones.
func (p *IntentParser) Parse(ctx context.Context, batch *models.DaemonBatch, contents []models.GeneratedContent, replyText string) (*models.DaemonIntent, error) {
	text := strings.TrimSpace(replyText)

	// Fast path: simple approve
	if simpleApprove.MatchString(text) {
		return &models.DaemonIntent{
			Action:  "approve",
			Message: text,
		}, nil
	}

	// Fast path: simple reject
	if simpleReject.MatchString(text) {
		return &models.DaemonIntent{
			Action:  "reject",
			Message: text,
		}, nil
	}

	// Fast path: approve specific drafts
	if m := approveSpecific.FindStringSubmatch(text); m != nil {
		ids, err := parseContentIDs(m[1], contents)
		if err != nil {
			return nil, err
		}
		return &models.DaemonIntent{
			Action:     "approve",
			ContentIDs: ids,
			Message:    text,
		}, nil
	}

	// Fast path: schedule (e.g. "schedule 1h", "schedule 1 h", "schedule 2h for draft 1,2")
	if m := schedulePattern.FindStringSubmatch(text); m != nil {
		dur, err := parseDuration(m[1] + m[2])
		if err != nil {
			return nil, fmt.Errorf("parsing schedule duration: %w", err)
		}
		t := time.Now().Add(dur)
		intent := &models.DaemonIntent{
			Action:     "schedule",
			ScheduleAt: &t,
			Message:    text,
		}
		// Optional draft selection (group 3)
		if m[3] != "" {
			ids, err := parseContentIDs(m[3], contents)
			if err != nil {
				return nil, fmt.Errorf("parsing schedule draft numbers: %w", err)
			}
			intent.ContentIDs = ids
		}
		return intent, nil
	}

	// Fast path: read original post (e.g. "read 1", "show draft 2", "read 1 from batch 5")
	if m := readPattern.FindStringSubmatch(text); m != nil {
		idx, err := strconv.Atoi(m[1])
		if err != nil {
			return nil, fmt.Errorf("invalid draft number %q", m[1])
		}
		if idx < 1 || idx > len(contents) {
			return nil, fmt.Errorf("draft number %d out of range (1-%d)", idx, len(contents))
		}
		return &models.DaemonIntent{
			Action:     "read",
			ContentIDs: []int64{contents[idx-1].ID},
			Message:    text,
		}, nil
	}

	// Fast path: rewrite draft (e.g. "rewrite 1", "rewrite draft 2 more casual")
	if m := rewritePattern.FindStringSubmatch(text); m != nil {
		idx, err := strconv.Atoi(m[1])
		if err != nil {
			return nil, fmt.Errorf("invalid draft number %q", m[1])
		}
		if idx < 1 || idx > len(contents) {
			return nil, fmt.Errorf("draft number %d out of range (1-%d)", idx, len(contents))
		}
		styleDirection := strings.TrimSpace(m[2])
		cleanedStyle, toggle := parseRepostToggle(styleDirection)
		return &models.DaemonIntent{
			Action:     "rewrite",
			ContentIDs: []int64{contents[idx-1].ID},
			IsRepost:   toggle,
			Message:    cleanedStyle,
		}, nil
	}

	// Fallback: Claude-powered intent parsing
	return p.parseWithClaude(ctx, batch, contents, text)
}

func (p *IntentParser) parseWithClaude(ctx context.Context, batch *models.DaemonBatch, contents []models.GeneratedContent, text string) (*models.DaemonIntent, error) {
	systemPrompt := `You are an intent parser for a social media autopilot. Parse the user's reply about a batch of draft posts.

Return a JSON object with these fields:
- "action": one of "approve", "reject", "edit", "schedule", "read", "rewrite"
- "content_ids": array of 1-indexed draft numbers to act on (null to act on all)
- "edits": array of {"draft_number": <1-indexed int>, "new_text": "<new content>"} objects (null if no edits)
- "schedule_at_minutes": number of minutes from now to schedule (null if not scheduling)
- "is_repost": boolean to toggle repost mode on rewrite (true = repost/quote tweet, false = full rewrite, null = keep existing mode)
- "message": brief summary of what the user wants

Examples:
- "post the first two" → {"action":"approve","content_ids":[1,2],"edits":null,"schedule_at_minutes":null,"is_repost":null,"message":"approve drafts 1 and 2"}
- "change draft 2 to talk more about AI" → {"action":"edit","content_ids":null,"edits":[{"draft_number":2,"new_text":"<new content based on original>"}],"schedule_at_minutes":null,"is_repost":null,"message":"edit draft 2"}
- "looks good but schedule for tomorrow morning" → {"action":"schedule","content_ids":null,"edits":null,"schedule_at_minutes":720,"is_repost":null,"message":"schedule for later"}
- "rewrite draft 1 as a repost" → {"action":"rewrite","content_ids":[1],"edits":null,"schedule_at_minutes":null,"is_repost":true,"message":"rewrite as repost"}`

	var draftsDesc strings.Builder
	for i, c := range contents {
		draftsDesc.WriteString(fmt.Sprintf("Draft %d: %s\n\n", i+1, c.GeneratedContent))
	}

	userMessage := fmt.Sprintf("Batch #%d (%s platform) has %d drafts:\n\n%s\nUser reply: %s",
		batch.ID, batch.Platform, len(contents), draftsDesc.String(), text)

	response, err := p.sender.SendMessageJSON(ctx, systemPrompt, userMessage, intentSchema)
	if err != nil {
		return nil, fmt.Errorf("parsing intent with Claude: %w", err)
	}

	var parsed struct {
		Action            string `json:"action"`
		ContentIDs        []int  `json:"content_ids,omitempty"`
		Edits             []struct {
			DraftNumber int    `json:"draft_number"`
			NewText     string `json:"new_text"`
		} `json:"edits,omitempty"`
		ScheduleAtMinutes int    `json:"schedule_at_minutes,omitempty"`
		IsRepost          *bool  `json:"is_repost,omitempty"`
		Message           string `json:"message"`
	}
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, fmt.Errorf("parsing Claude intent response: %w", err)
	}

	intent := &models.DaemonIntent{
		Action:   parsed.Action,
		IsRepost: parsed.IsRepost,
		Message:  parsed.Message,
	}

	// Convert 1-indexed draft numbers to content IDs
	if len(parsed.ContentIDs) > 0 {
		for _, idx := range parsed.ContentIDs {
			if idx >= 1 && idx <= len(contents) {
				intent.ContentIDs = append(intent.ContentIDs, contents[idx-1].ID)
			}
		}
	}

	if len(parsed.Edits) > 0 {
		intent.Edits = make(map[int64]string)
		for _, edit := range parsed.Edits {
			if edit.DraftNumber < 1 || edit.DraftNumber > len(contents) {
				continue
			}
			intent.Edits[contents[edit.DraftNumber-1].ID] = edit.NewText
		}
	}

	if parsed.ScheduleAtMinutes > 0 {
		t := time.Now().Add(time.Duration(parsed.ScheduleAtMinutes) * time.Minute)
		intent.ScheduleAt = &t
	}

	return intent, nil
}

func parseContentIDs(s string, contents []models.GeneratedContent) ([]int64, error) {
	parts := strings.Split(strings.ReplaceAll(s, " ", ""), ",")
	var ids []int64
	for _, p := range parts {
		idx, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, fmt.Errorf("invalid draft number %q", p)
		}
		if idx < 1 || idx > len(contents) {
			return nil, fmt.Errorf("draft number %d out of range (1-%d)", idx, len(contents))
		}
		ids = append(ids, contents[idx-1].ID)
	}
	return ids, nil
}

// extractBatchID looks for a "batch N" reference in the text and returns the batch ID.
// Returns nil if no batch reference is found.
func extractBatchID(text string) *int64 {
	m := batchRefPattern.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	id, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return nil
	}
	return &id
}

// parseRepostToggle extracts a repost/rewrite toggle phrase from a style direction string.
// Returns the cleaned style direction (with toggle phrase removed) and a *bool toggle:
// true = force repost, false = force rewrite, nil = no toggle found.
func parseRepostToggle(style string) (string, *bool) {
	m := repostToggle.FindStringSubmatchIndex(style)
	if m == nil {
		return style, nil
	}

	// m[2]:m[3] is the capture group (repost|rewrite)
	word := strings.ToLower(style[m[2]:m[3]])
	isRepost := word == "repost"

	// Remove the matched toggle phrase and clean up whitespace
	cleaned := strings.TrimSpace(style[:m[0]] + style[m[1]:])
	return cleaned, &isRepost
}

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "h") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "h"))
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * time.Hour, nil
	}
	if strings.HasSuffix(s, "m") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "m"))
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * time.Minute, nil
	}
	// Default to hours if no suffix
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return time.Duration(n) * time.Hour, nil
}

