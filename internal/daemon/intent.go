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
	schedulePattern = regexp.MustCompile(`(?i)^schedule\s+(\d+[hm]?)$`)
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

	// Fast path: schedule
	if m := schedulePattern.FindStringSubmatch(text); m != nil {
		dur, err := parseDuration(m[1])
		if err != nil {
			return nil, fmt.Errorf("parsing schedule duration: %w", err)
		}
		t := time.Now().Add(dur)
		return &models.DaemonIntent{
			Action:     "schedule",
			ScheduleAt: &t,
			Message:    text,
		}, nil
	}

	// Fallback: Claude-powered intent parsing
	return p.parseWithClaude(ctx, batch, contents, text)
}

func (p *IntentParser) parseWithClaude(ctx context.Context, batch *models.DaemonBatch, contents []models.GeneratedContent, text string) (*models.DaemonIntent, error) {
	systemPrompt := `You are an intent parser for a social media autopilot. Parse the user's reply about a batch of draft posts.

Return ONLY a JSON object with these fields:
- "action": one of "approve", "reject", "edit", "schedule"
- "content_ids": optional array of 1-indexed draft numbers to act on (omit to act on all)
- "edits": optional object mapping 1-indexed draft numbers (as strings) to new text
- "schedule_at_minutes": optional number of minutes from now to schedule
- "message": brief summary of what the user wants

Examples:
- "post the first two" → {"action":"approve","content_ids":[1,2],"message":"approve drafts 1 and 2"}
- "change draft 2 to talk more about AI" → {"action":"edit","edits":{"2":"<suggest new content here based on original>"},"message":"edit draft 2"}
- "looks good but schedule for tomorrow morning" → {"action":"schedule","schedule_at_minutes":720,"message":"schedule for later"}`

	var draftsDesc strings.Builder
	for i, c := range contents {
		draftsDesc.WriteString(fmt.Sprintf("Draft %d: %s\n\n", i+1, c.GeneratedContent))
	}

	userMessage := fmt.Sprintf("Batch #%d (%s platform) has %d drafts:\n\n%s\nUser reply: %s",
		batch.ID, batch.Platform, len(contents), draftsDesc.String(), text)

	response, err := p.sender.SendMessage(ctx, systemPrompt, userMessage)
	if err != nil {
		return nil, fmt.Errorf("parsing intent with Claude: %w", err)
	}

	// Extract JSON from response
	response = extractJSON(response)

	var parsed struct {
		Action            string            `json:"action"`
		ContentIDs        []int             `json:"content_ids,omitempty"`
		Edits             map[string]string `json:"edits,omitempty"`
		ScheduleAtMinutes int               `json:"schedule_at_minutes,omitempty"`
		Message           string            `json:"message"`
	}
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, fmt.Errorf("parsing Claude intent response: %w", err)
	}

	intent := &models.DaemonIntent{
		Action:  parsed.Action,
		Message: parsed.Message,
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
		for idxStr, newText := range parsed.Edits {
			idx, err := strconv.Atoi(idxStr)
			if err != nil || idx < 1 || idx > len(contents) {
				continue
			}
			intent.Edits[contents[idx-1].ID] = newText
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

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	// Strip markdown code fences if present
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	// Find the first { and last }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
