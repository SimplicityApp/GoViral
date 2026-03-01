package daemon

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shuhao/goviral/pkg/models"
)

// mockSender implements claude.MessageSender for tests.
type mockSender struct {
	response string
	err      error
}

func (m *mockSender) SendMessage(_ context.Context, _, _ string) (string, error) {
	return m.response, m.err
}

func (m *mockSender) SendMessageJSON(_ context.Context, _, _ string, _ map[string]any) (string, error) {
	return m.response, m.err
}

// testContents returns a slice of 3 fake contents with IDs 100, 200, 300.
func testContents() []models.GeneratedContent {
	return []models.GeneratedContent{
		{ID: 100, GeneratedContent: "Draft 1 text"},
		{ID: 200, GeneratedContent: "Draft 2 text"},
		{ID: 300, GeneratedContent: "Draft 3 text"},
	}
}

func testBatch() *models.DaemonBatch {
	return &models.DaemonBatch{ID: 1, Platform: "x", Status: "notified"}
}

// --- schedulePattern regex tests ---

func TestSchedulePattern(t *testing.T) {
	tests := []struct {
		input      string
		wantMatch  bool
		wantNumber string
		wantUnit   string
		wantDrafts string
	}{
		{"schedule 1h", true, "1", "h", ""},
		{"schedule 1 h", true, "1", "h", ""},
		{"schedule 2h for draft 1", true, "2", "h", "1"},
		{"schedule 30m for draft 1,2", true, "30", "m", "1,2"},
		{"schedule 1 h for draft 1", true, "1", "h", "1"},
		{"schedule 1 h for 1", true, "1", "h", "1"},
		{"schedule 1 h for 1, 2", true, "1", "h", "1, 2"},
		{"Schedule 2H for draft 3", true, "2", "H", "3"},
		// Non-matching
		{"schedule", false, "", "", ""},
		{"schedule tomorrow", false, "", "", ""},
		{"schedule 1h extra stuff", false, "", "", ""},
		{"post now", false, "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			m := schedulePattern.FindStringSubmatch(tt.input)
			if tt.wantMatch {
				if m == nil {
					t.Fatalf("expected match for %q, got nil", tt.input)
				}
				if m[1] != tt.wantNumber {
					t.Errorf("number: got %q, want %q", m[1], tt.wantNumber)
				}
				if m[2] != tt.wantUnit {
					t.Errorf("unit: got %q, want %q", m[2], tt.wantUnit)
				}
				if m[3] != tt.wantDrafts {
					t.Errorf("drafts: got %q, want %q", m[3], tt.wantDrafts)
				}
			} else {
				if m != nil {
					t.Fatalf("expected no match for %q, got %v", tt.input, m)
				}
			}
		})
	}
}

// --- Parse() fast-path tests ---

func TestParse_ScheduleFastPath(t *testing.T) {
	parser := NewIntentParser(&mockSender{})
	contents := testContents()
	batch := testBatch()
	ctx := context.Background()

	t.Run("schedule 1h no drafts", func(t *testing.T) {
		before := time.Now()
		intent, err := parser.Parse(ctx, batch, contents, "schedule 1h")
		if err != nil {
			t.Fatal(err)
		}
		if intent.Action != "schedule" {
			t.Errorf("action: got %q, want %q", intent.Action, "schedule")
		}
		if intent.ScheduleAt == nil {
			t.Fatal("schedule_at is nil")
		}
		if intent.ScheduleAt.Before(before.Add(59 * time.Minute)) {
			t.Errorf("schedule_at too early: %v", intent.ScheduleAt)
		}
		if len(intent.ContentIDs) != 0 {
			t.Errorf("expected no content_ids, got %v", intent.ContentIDs)
		}
	})

	t.Run("schedule 1 h for draft 1", func(t *testing.T) {
		intent, err := parser.Parse(ctx, batch, contents, "schedule 1 h for draft 1")
		if err != nil {
			t.Fatal(err)
		}
		if intent.Action != "schedule" {
			t.Errorf("action: got %q, want %q", intent.Action, "schedule")
		}
		if len(intent.ContentIDs) != 1 || intent.ContentIDs[0] != 100 {
			t.Errorf("content_ids: got %v, want [100]", intent.ContentIDs)
		}
	})

	t.Run("schedule 30m for draft 1,2", func(t *testing.T) {
		before := time.Now()
		intent, err := parser.Parse(ctx, batch, contents, "schedule 30m for draft 1,2")
		if err != nil {
			t.Fatal(err)
		}
		if intent.Action != "schedule" {
			t.Errorf("action: got %q, want %q", intent.Action, "schedule")
		}
		if intent.ScheduleAt.Before(before.Add(29 * time.Minute)) {
			t.Errorf("schedule_at too early for 30m: %v", intent.ScheduleAt)
		}
		if len(intent.ContentIDs) != 2 {
			t.Fatalf("content_ids length: got %d, want 2", len(intent.ContentIDs))
		}
		if intent.ContentIDs[0] != 100 || intent.ContentIDs[1] != 200 {
			t.Errorf("content_ids: got %v, want [100 200]", intent.ContentIDs)
		}
	})

	t.Run("schedule with out-of-range draft", func(t *testing.T) {
		_, err := parser.Parse(ctx, batch, contents, "schedule 1h for draft 5")
		if err == nil {
			t.Fatal("expected error for out-of-range draft")
		}
	})
}

func TestParse_SimpleApproveReject(t *testing.T) {
	parser := NewIntentParser(&mockSender{})
	contents := testContents()
	batch := testBatch()
	ctx := context.Background()

	tests := []struct {
		input      string
		wantAction string
	}{
		{"yes", "approve"},
		{"LGTM", "approve"},
		{"ship it", "approve"},
		{"no", "reject"},
		{"skip", "reject"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			intent, err := parser.Parse(ctx, batch, contents, tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if intent.Action != tt.wantAction {
				t.Errorf("action: got %q, want %q", intent.Action, tt.wantAction)
			}
		})
	}
}

// --- parseDuration tests ---

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"1h", 1 * time.Hour},
		{"2h", 2 * time.Hour},
		{"30m", 30 * time.Minute},
		{"5", 5 * time.Hour}, // default to hours
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// --- parseWithClaude via mock ---

func TestParse_ClaudeFallback(t *testing.T) {
	mock := &mockSender{
		response: `{"action":"schedule","schedule_at_minutes":60,"content_ids":[1],"edits":null,"message":"schedule draft 1 in 1 hour"}`,
	}
	parser := NewIntentParser(mock)
	contents := testContents()
	batch := testBatch()

	// Input that doesn't match any regex fast-path
	intent, err := parser.Parse(context.Background(), batch, contents, "post draft 1 in an hour")
	if err != nil {
		t.Fatal(err)
	}
	if intent.Action != "schedule" {
		t.Errorf("action: got %q, want %q", intent.Action, "schedule")
	}
	if intent.ScheduleAt == nil {
		t.Fatal("schedule_at is nil")
	}
	if len(intent.ContentIDs) != 1 || intent.ContentIDs[0] != 100 {
		t.Errorf("content_ids: got %v, want [100]", intent.ContentIDs)
	}
}

// --- readPattern regex tests ---

func TestReadPattern(t *testing.T) {
	tests := []struct {
		input     string
		wantMatch bool
		wantDraft string
	}{
		{"read 1", true, "1"},
		{"Read 2", true, "2"},
		{"show 3", true, "3"},
		{"show draft 1", true, "1"},
		{"original 2", true, "2"},
		{"full draft 3", true, "3"},
		{"read 1 from batch 5", true, "1"},
		{"read draft 2 from batch 12", true, "2"},
		{"FULL DRAFT 1 FROM BATCH 3", true, "1"},
		// Non-matching
		{"read", false, ""},
		{"read draft", false, ""},
		{"approve 1", false, ""},
		{"read 1 extra", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			m := readPattern.FindStringSubmatch(tt.input)
			if tt.wantMatch {
				if m == nil {
					t.Fatalf("expected match for %q, got nil", tt.input)
				}
				if m[1] != tt.wantDraft {
					t.Errorf("draft: got %q, want %q", m[1], tt.wantDraft)
				}
			} else {
				if m != nil {
					t.Fatalf("expected no match for %q, got %v", tt.input, m)
				}
			}
		})
	}
}

// --- Parse() read fast-path tests ---

func TestParse_ReadFastPath(t *testing.T) {
	parser := NewIntentParser(&mockSender{})
	contents := testContents()
	batch := testBatch()
	ctx := context.Background()

	t.Run("read 1", func(t *testing.T) {
		intent, err := parser.Parse(ctx, batch, contents, "read 1")
		if err != nil {
			t.Fatal(err)
		}
		if intent.Action != "read" {
			t.Errorf("action: got %q, want %q", intent.Action, "read")
		}
		if len(intent.ContentIDs) != 1 || intent.ContentIDs[0] != 100 {
			t.Errorf("content_ids: got %v, want [100]", intent.ContentIDs)
		}
	})

	t.Run("show draft 2", func(t *testing.T) {
		intent, err := parser.Parse(ctx, batch, contents, "show draft 2")
		if err != nil {
			t.Fatal(err)
		}
		if intent.Action != "read" {
			t.Errorf("action: got %q, want %q", intent.Action, "read")
		}
		if len(intent.ContentIDs) != 1 || intent.ContentIDs[0] != 200 {
			t.Errorf("content_ids: got %v, want [200]", intent.ContentIDs)
		}
	})

	t.Run("read 1 from batch 5", func(t *testing.T) {
		intent, err := parser.Parse(ctx, batch, contents, "read 1 from batch 5")
		if err != nil {
			t.Fatal(err)
		}
		if intent.Action != "read" {
			t.Errorf("action: got %q, want %q", intent.Action, "read")
		}
		if len(intent.ContentIDs) != 1 || intent.ContentIDs[0] != 100 {
			t.Errorf("content_ids: got %v, want [100]", intent.ContentIDs)
		}
	})

	t.Run("out of range draft", func(t *testing.T) {
		_, err := parser.Parse(ctx, batch, contents, "read 5")
		if err == nil {
			t.Fatal("expected error for out-of-range draft")
		}
	})
}

// --- rewritePattern regex tests ---

func TestRewritePattern(t *testing.T) {
	tests := []struct {
		input     string
		wantMatch bool
		wantDraft string
		wantStyle string
	}{
		{"rewrite 1", true, "1", ""},
		{"Rewrite 2", true, "2", ""},
		{"rewrite draft 1", true, "1", ""},
		{"rewrite draft 2 more casual", true, "2", "more casual"},
		{"rewrite 1 more casual and punchy", true, "1", "more casual and punchy"},
		{"rewrite draft 1 from batch 5", true, "1", ""},
		{"rewrite draft 2 from batch 5 with a professional tone", true, "2", "with a professional tone"},
		{"REWRITE DRAFT 1 FROM BATCH 3", true, "1", ""},
		// Non-matching
		{"rewrite", false, "", ""},
		{"rewrite draft", false, "", ""},
		{"approve 1", false, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			m := rewritePattern.FindStringSubmatch(tt.input)
			if tt.wantMatch {
				if m == nil {
					t.Fatalf("expected match for %q, got nil", tt.input)
				}
				if m[1] != tt.wantDraft {
					t.Errorf("draft: got %q, want %q", m[1], tt.wantDraft)
				}
				gotStyle := strings.TrimSpace(m[2])
				if gotStyle != tt.wantStyle {
					t.Errorf("style: got %q, want %q", gotStyle, tt.wantStyle)
				}
			} else {
				if m != nil {
					t.Fatalf("expected no match for %q, got %v", tt.input, m)
				}
			}
		})
	}
}

// --- Parse() rewrite fast-path tests ---

func TestParse_RewriteFastPath(t *testing.T) {
	parser := NewIntentParser(&mockSender{})
	contents := testContents()
	batch := testBatch()
	ctx := context.Background()

	t.Run("rewrite 1", func(t *testing.T) {
		intent, err := parser.Parse(ctx, batch, contents, "rewrite 1")
		if err != nil {
			t.Fatal(err)
		}
		if intent.Action != "rewrite" {
			t.Errorf("action: got %q, want %q", intent.Action, "rewrite")
		}
		if len(intent.ContentIDs) != 1 || intent.ContentIDs[0] != 100 {
			t.Errorf("content_ids: got %v, want [100]", intent.ContentIDs)
		}
		if intent.Message != "" {
			t.Errorf("message: got %q, want empty", intent.Message)
		}
	})

	t.Run("rewrite draft 2 more casual", func(t *testing.T) {
		intent, err := parser.Parse(ctx, batch, contents, "rewrite draft 2 more casual")
		if err != nil {
			t.Fatal(err)
		}
		if intent.Action != "rewrite" {
			t.Errorf("action: got %q, want %q", intent.Action, "rewrite")
		}
		if len(intent.ContentIDs) != 1 || intent.ContentIDs[0] != 200 {
			t.Errorf("content_ids: got %v, want [200]", intent.ContentIDs)
		}
		if intent.Message != "more casual" {
			t.Errorf("message: got %q, want %q", intent.Message, "more casual")
		}
	})

	t.Run("rewrite draft 1 from batch 5", func(t *testing.T) {
		intent, err := parser.Parse(ctx, batch, contents, "rewrite draft 1 from batch 5")
		if err != nil {
			t.Fatal(err)
		}
		if intent.Action != "rewrite" {
			t.Errorf("action: got %q, want %q", intent.Action, "rewrite")
		}
		if len(intent.ContentIDs) != 1 || intent.ContentIDs[0] != 100 {
			t.Errorf("content_ids: got %v, want [100]", intent.ContentIDs)
		}
	})

	t.Run("rewrite draft 2 from batch 5 with a professional tone", func(t *testing.T) {
		intent, err := parser.Parse(ctx, batch, contents, "rewrite draft 2 from batch 5 with a professional tone")
		if err != nil {
			t.Fatal(err)
		}
		if intent.Action != "rewrite" {
			t.Errorf("action: got %q, want %q", intent.Action, "rewrite")
		}
		if len(intent.ContentIDs) != 1 || intent.ContentIDs[0] != 200 {
			t.Errorf("content_ids: got %v, want [200]", intent.ContentIDs)
		}
		if intent.Message != "with a professional tone" {
			t.Errorf("message: got %q, want %q", intent.Message, "with a professional tone")
		}
	})

	t.Run("rewrite 1 as a repost", func(t *testing.T) {
		intent, err := parser.Parse(ctx, batch, contents, "rewrite 1 as a repost")
		if err != nil {
			t.Fatal(err)
		}
		if intent.Action != "rewrite" {
			t.Errorf("action: got %q, want %q", intent.Action, "rewrite")
		}
		if len(intent.ContentIDs) != 1 || intent.ContentIDs[0] != 100 {
			t.Errorf("content_ids: got %v, want [100]", intent.ContentIDs)
		}
		if intent.IsRepost == nil || !*intent.IsRepost {
			t.Errorf("is_repost: got %v, want true", intent.IsRepost)
		}
		if intent.Message != "" {
			t.Errorf("message: got %q, want empty", intent.Message)
		}
	})

	t.Run("rewrite draft 2 to be a rewrite instead", func(t *testing.T) {
		intent, err := parser.Parse(ctx, batch, contents, "rewrite draft 2 to be a rewrite instead")
		if err != nil {
			t.Fatal(err)
		}
		if intent.Action != "rewrite" {
			t.Errorf("action: got %q, want %q", intent.Action, "rewrite")
		}
		if len(intent.ContentIDs) != 1 || intent.ContentIDs[0] != 200 {
			t.Errorf("content_ids: got %v, want [200]", intent.ContentIDs)
		}
		if intent.IsRepost == nil || *intent.IsRepost {
			t.Errorf("is_repost: got %v, want false", intent.IsRepost)
		}
		if intent.Message != "" {
			t.Errorf("message: got %q, want empty", intent.Message)
		}
	})

	t.Run("rewrite 3 as a repost with casual tone", func(t *testing.T) {
		intent, err := parser.Parse(ctx, batch, contents, "rewrite 3 as a repost with casual tone")
		if err != nil {
			t.Fatal(err)
		}
		if intent.Action != "rewrite" {
			t.Errorf("action: got %q, want %q", intent.Action, "rewrite")
		}
		if len(intent.ContentIDs) != 1 || intent.ContentIDs[0] != 300 {
			t.Errorf("content_ids: got %v, want [300]", intent.ContentIDs)
		}
		if intent.IsRepost == nil || !*intent.IsRepost {
			t.Errorf("is_repost: got %v, want true", intent.IsRepost)
		}
		if intent.Message != "with casual tone" {
			t.Errorf("message: got %q, want %q", intent.Message, "with casual tone")
		}
	})

	t.Run("rewrite 1 more casual (no toggle)", func(t *testing.T) {
		intent, err := parser.Parse(ctx, batch, contents, "rewrite 1 more casual")
		if err != nil {
			t.Fatal(err)
		}
		if intent.IsRepost != nil {
			t.Errorf("is_repost: got %v, want nil", intent.IsRepost)
		}
		if intent.Message != "more casual" {
			t.Errorf("message: got %q, want %q", intent.Message, "more casual")
		}
	})

	t.Run("out of range draft", func(t *testing.T) {
		_, err := parser.Parse(ctx, batch, contents, "rewrite 5")
		if err == nil {
			t.Fatal("expected error for out-of-range draft")
		}
	})
}

// --- parseRepostToggle unit tests ---

func TestParseRepostToggle(t *testing.T) {
	tests := []struct {
		input     string
		wantStyle string
		wantToggle *bool
	}{
		{"as a repost", "", boolPtr(true)},
		{"to be a rewrite instead", "", boolPtr(false)},
		{"as a repost with casual tone", "with casual tone", boolPtr(true)},
		{"to be a repost instead", "", boolPtr(true)},
		{"As A Repost", "", boolPtr(true)},
		{"more casual", "more casual", nil},
		{"with a professional tone", "with a professional tone", nil},
		{"", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotStyle, gotToggle := parseRepostToggle(tt.input)
			if gotStyle != tt.wantStyle {
				t.Errorf("style: got %q, want %q", gotStyle, tt.wantStyle)
			}
			if tt.wantToggle == nil {
				if gotToggle != nil {
					t.Errorf("toggle: got %v, want nil", *gotToggle)
				}
			} else {
				if gotToggle == nil {
					t.Errorf("toggle: got nil, want %v", *tt.wantToggle)
				} else if *gotToggle != *tt.wantToggle {
					t.Errorf("toggle: got %v, want %v", *gotToggle, *tt.wantToggle)
				}
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }
