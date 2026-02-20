package telegram

import (
	"fmt"
	"strings"

	"github.com/shuhao/goviral/pkg/models"
)

// FormatBatchNotification formats a batch notification for Telegram.
func FormatBatchNotification(batch *models.DaemonBatch, contents []models.GeneratedContent) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("*GoViral Autopilot* - %s\n\n", strings.ToUpper(batch.Platform)))
	sb.WriteString(fmt.Sprintf("Batch #%d | %d drafts ready for review\n\n", batch.ID, len(contents)))

	for i, c := range contents {
		preview := c.GeneratedContent
		if len(preview) > 280 {
			preview = preview[:277] + "..."
		}
		// Escape markdown special characters in content
		preview = escapeMarkdown(preview)
		label := fmt.Sprintf("*Draft %d:*", i+1)
		if c.IsRepost {
			label = fmt.Sprintf("*Draft %d \\[repost\\]:*", i+1)
		}
		sb.WriteString(fmt.Sprintf("%s\n%s\n\n", label, preview))
	}

	sb.WriteString("---\n")
	sb.WriteString("Reply to this message:\n")
	sb.WriteString("  `approve` - post all drafts now\n")
	sb.WriteString("  `reject` - discard this batch\n")
	sb.WriteString("  `approve 1,3` - post specific drafts\n")
	sb.WriteString("  `schedule 2h` - schedule for later\n")
	sb.WriteString("  Or describe edits in natural language")

	return sb.String()
}

// FormatBatchApproved formats a batch approval confirmation.
func FormatBatchApproved(batch *models.DaemonBatch, action string) string {
	return fmt.Sprintf("Batch #%d %s (%s)", batch.ID, action, batch.Platform)
}

// FormatPostResult formats the posting result.
func FormatPostResult(batch *models.DaemonBatch, postIDs []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Batch #%d posted on %s\n\n", batch.ID, batch.Platform))
	for i, id := range postIDs {
		sb.WriteString(fmt.Sprintf("Post %d: `%s`\n", i+1, id))
	}
	return sb.String()
}

func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"*", "\\*",
		"_", "\\_",
		"`", "\\`",
		"[", "\\[",
	)
	return replacer.Replace(s)
}
