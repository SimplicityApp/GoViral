package telegram

import (
	"fmt"
	"strings"

	"github.com/shuhao/goviral/pkg/models"
)

// clampRunes truncates s to maxRunes runes, appending "..." if truncated.
func clampRunes(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes-3]) + "..."
}

// FormatBatchNotification formats a batch notification for Telegram.
// trendingPosts is an optional map of trending post ID -> TrendingPost for showing source context.
func FormatBatchNotification(batch *models.DaemonBatch, contents []models.GeneratedContent, trendingPosts map[int64]models.TrendingPost) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("*GoViral Autopilot* - %s\n\n", strings.ToUpper(batch.Platform)))
	sb.WriteString(fmt.Sprintf("Batch #%d | %d drafts ready for review\n\n", batch.ID, len(contents)))

	for i, c := range contents {
		label := fmt.Sprintf("*Draft %d:*", i+1)
		if c.IsRepost {
			label = fmt.Sprintf("*Draft %d \\[repost\\]:*", i+1)
		}
		sb.WriteString(label + "\n")

		// Show source trending post context if available
		if tp, ok := trendingPosts[c.SourceTrendingID]; ok {
			preview := clampRunes(tp.Content, 50)
			// Replace newlines so the italic block stays on one line
			preview = strings.ReplaceAll(preview, "\n", " ")
			sb.WriteString(fmt.Sprintf("@%s: \"%s\"\n", EscapeMarkdown(tp.AuthorUsername), EscapeMarkdown(preview)))
			sb.WriteString(fmt.Sprintf("Likes: %s | Reposts: %s | Comments: %s | Views: %s\n",
				formatCount(tp.Likes), formatCount(tp.Reposts), formatCount(tp.Comments), formatCount(tp.Impressions)))
		}

		contentPreview := clampRunes(c.GeneratedContent, 150)
		contentPreview = EscapeMarkdown(contentPreview)
		sb.WriteString(contentPreview + "\n\n")
	}

	sb.WriteString("---\n")
	sb.WriteString("Reply to this message:\n")
	sb.WriteString("  `approve` - post all drafts now\n")
	sb.WriteString("  `reject` - discard this batch\n")
	sb.WriteString("  `approve 1,3` - post specific drafts\n")
	sb.WriteString("  `read 1` - show original + generated draft\n")
	sb.WriteString("  `rewrite 1` - AI rewrite of draft 1\n")
	sb.WriteString("  `schedule 2h` - schedule for later\n")
	sb.WriteString("  Or describe edits in natural language")

	return sb.String()
}

// formatCount formats a number compactly: 1234 -> "1.2K", 1234567 -> "1.2M".
func formatCount(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
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

// FormatDraftDetail formats the full original trending post AND generated content
// for the "read" command. Shows a type label (Post/Repost/Comment) and both pieces.
func FormatDraftDetail(draftNum int, tp *models.TrendingPost, gc *models.GeneratedContent) string {
	var sb strings.Builder

	// Determine type label
	typeLabel := "Post"
	if gc.IsComment {
		typeLabel = "Comment"
	} else if gc.IsRepost {
		typeLabel = "Repost"
	}

	sb.WriteString(fmt.Sprintf("*Draft %d* \\[%s\\]\n\n", draftNum, typeLabel))

	// Original post section (if available)
	if tp != nil {
		sb.WriteString("*Source post:*\n")
		sb.WriteString(fmt.Sprintf("@%s\n", EscapeMarkdown(tp.AuthorUsername)))
		sb.WriteString(fmt.Sprintf("Likes: %s | Reposts: %s | Comments: %s | Views: %s\n",
			formatCount(tp.Likes), formatCount(tp.Reposts), formatCount(tp.Comments), formatCount(tp.Impressions)))
		sb.WriteString(fmt.Sprintf("Posted: %s\n\n", tp.PostedAt.Format("2006-01-02")))
		sb.WriteString(EscapeMarkdown(tp.Content))
		sb.WriteString("\n\n")
	}

	// Generated content section
	sb.WriteString(fmt.Sprintf("*Your %s:*\n", strings.ToLower(typeLabel)))
	sb.WriteString(EscapeMarkdown(gc.GeneratedContent))

	return sb.String()
}

// FormatCommentBatchNotification formats a comment batch notification for Telegram.
func FormatCommentBatchNotification(batch *models.DaemonBatch, contents []models.GeneratedContent, trendingPosts map[int64]models.TrendingPost) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("*GoViral Autopilot* - %s COMMENTS\n\n", strings.ToUpper(batch.Platform)))
	sb.WriteString(fmt.Sprintf("Batch #%d | %d comments ready for review\n\n", batch.ID, len(contents)))

	for i, c := range contents {
		sb.WriteString(fmt.Sprintf("*Comment %d:*\n", i+1))

		if tp, ok := trendingPosts[c.SourceTrendingID]; ok {
			preview := clampRunes(tp.Content, 50)
			preview = strings.ReplaceAll(preview, "\n", " ")
			sb.WriteString(fmt.Sprintf("Replying to @%s: \"%s\"\n", EscapeMarkdown(tp.AuthorUsername), EscapeMarkdown(preview)))
		}

		contentPreview := clampRunes(c.GeneratedContent, 150)
		sb.WriteString(fmt.Sprintf("Your comment: %s\n\n", EscapeMarkdown(contentPreview)))
	}

	sb.WriteString("---\n")
	sb.WriteString("Reply: `approve`, `reject`, `approve 1,3`, `read 1`, `rewrite 1`, `edit`, `schedule 2h`")

	return sb.String()
}

// FormatDigestNotification formats a nightly digest notification for Telegram.
// It shows the top-ranked contents from a competition pass, with source context
// and reasoning, plus an archive count for the discarded candidates.
func FormatDigestNotification(batch *models.DaemonBatch, contents []models.GeneratedContent, trendingPosts map[int64]models.TrendingPost, rankings []models.CompeteResult, totalCandidates int) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("*GoViral Nightly Digest* - %s\n\n", strings.ToUpper(batch.Platform)))

	candidateCount := totalCandidates
	if candidateCount == 0 {
		candidateCount = len(contents)
	}
	winnerCount := len(rankings)
	if winnerCount == 0 {
		winnerCount = len(contents)
	}
	sb.WriteString(fmt.Sprintf("Batch #%d | %d best posts from %d candidates\n\n", batch.ID, winnerCount, candidateCount))

	// Build a lookup map from content ID to GeneratedContent for O(1) access.
	contentByID := make(map[int64]models.GeneratedContent, len(contents))
	for _, c := range contents {
		contentByID[c.ID] = c
	}

	for _, r := range rankings {
		c, ok := contentByID[r.ContentID]
		if !ok {
			continue
		}

		// Type label
		typeLabel := "post"
		if c.IsComment {
			typeLabel = "comment"
		} else if c.IsRepost {
			typeLabel = "repost"
		}

		sb.WriteString(fmt.Sprintf("*#%d \\[%s\\] (Score: %.1f):*\n", r.Rank, typeLabel, r.Score))

		// Source trending post context.
		if tp, ok := trendingPosts[c.SourceTrendingID]; ok {
			tpPreview := clampRunes(tp.Content, 50)
			tpPreview = strings.ReplaceAll(tpPreview, "\n", " ")
			sb.WriteString(fmt.Sprintf("@%s: \"%s\"\n", EscapeMarkdown(tp.AuthorUsername), EscapeMarkdown(tpPreview)))
		}

		// Generated content preview (clamped to 150).
		contentPreview := clampRunes(c.GeneratedContent, 150)
		contentPreview = strings.ReplaceAll(contentPreview, "\n", " ")
		sb.WriteString(fmt.Sprintf("Your %s: \"%s\"\n", typeLabel, EscapeMarkdown(contentPreview)))

		// Competition reasoning (clamped to 80).
		if r.Reasoning != "" {
			reason := clampRunes(r.Reasoning, 80)
			sb.WriteString(fmt.Sprintf("Reason: %s\n", EscapeMarkdown(reason)))
		}

		sb.WriteString("\n")
	}

	// No rankings: competition didn't run or failed — display all batch contents without scores
	if len(rankings) == 0 {
		for i, c := range contents {
			typeLabel := "post"
			if c.IsComment {
				typeLabel = "comment"
			} else if c.IsRepost {
				typeLabel = "repost"
			}
			sb.WriteString(fmt.Sprintf("*#%d \\[%s\\]:*\n", i+1, typeLabel))
			if tp, ok := trendingPosts[c.SourceTrendingID]; ok {
				tpPreview := clampRunes(tp.Content, 50)
				tpPreview = strings.ReplaceAll(tpPreview, "\n", " ")
				sb.WriteString(fmt.Sprintf("@%s: \"%s\"\n", EscapeMarkdown(tp.AuthorUsername), EscapeMarkdown(tpPreview)))
			}
			contentPreview := clampRunes(c.GeneratedContent, 150)
			contentPreview = strings.ReplaceAll(contentPreview, "\n", " ")
			sb.WriteString(fmt.Sprintf("Your %s: \"%s\"\n\n", typeLabel, EscapeMarkdown(contentPreview)))
		}
	}

	archivedCount := candidateCount - winnerCount
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("%d drafts auto-archived\n", archivedCount))
	sb.WriteString("Reply: `approve`, `reject`, `approve 1,3`, `read 1`, `rewrite 1`, `schedule 2h`")

	return sb.String()
}

// FormatAutoPublishReport formats an informational report after auto-publishing content.
// No approval buttons — this is informational only.
func FormatAutoPublishReport(batch *models.DaemonBatch, results []models.AutoPublishResult, contents []models.GeneratedContent, trendingPosts map[int64]models.TrendingPost, rankings []models.CompeteResult, totalCandidates int) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("*GoViral Auto\\-Pilot* — %s\n\n", strings.ToUpper(batch.Platform)))

	publishedCount := len(results)
	sb.WriteString(fmt.Sprintf("Batch #%d | %d item auto\\-published from %d candidates\n\n", batch.ID, publishedCount, totalCandidates))

	// Build lookup maps
	contentByID := make(map[int64]models.GeneratedContent, len(contents))
	for _, c := range contents {
		contentByID[c.ID] = c
	}
	rankByContentID := make(map[int64]models.CompeteResult, len(rankings))
	for _, r := range rankings {
		rankByContentID[r.ContentID] = r
	}

	for i, res := range results {
		actionLabel := res.Action
		if len(actionLabel) > 0 {
			actionLabel = strings.ToUpper(actionLabel[:1]) + actionLabel[1:]
		}
		sb.WriteString(fmt.Sprintf("*%s #%d:*\n", actionLabel, i+1))

		c, ok := contentByID[res.ContentID]
		if ok {
			// Source trending post context
			if tp, tpOk := trendingPosts[c.SourceTrendingID]; tpOk {
				tpPreview := clampRunes(tp.Content, 50)
				tpPreview = strings.ReplaceAll(tpPreview, "\n", " ")
				sb.WriteString(fmt.Sprintf("Source: @%s — \"%s\"\n", EscapeMarkdown(tp.AuthorUsername), EscapeMarkdown(tpPreview)))
			}

			// Generated content preview
			contentPreview := clampRunes(c.GeneratedContent, 150)
			contentPreview = strings.ReplaceAll(contentPreview, "\n", " ")
			sb.WriteString(fmt.Sprintf("Content: \"%s\"\n", EscapeMarkdown(contentPreview)))
		}

		// Competition score and reasoning
		if rank, rOk := rankByContentID[res.ContentID]; rOk {
			reason := clampRunes(rank.Reasoning, 80)
			sb.WriteString(fmt.Sprintf("Score: %.1f | %s\n", rank.Score, EscapeMarkdown(reason)))
		}

		// Post IDs
		if len(res.PostIDs) > 0 {
			sb.WriteString(fmt.Sprintf("Posted: `%s`\n", strings.Join(res.PostIDs, ", ")))
		}

		sb.WriteString("\n")
	}

	sb.WriteString("---\n")
	sb.WriteString("Auto\\-published by GoViral Autopilot")

	return sb.String()
}

// EscapeMarkdown escapes Telegram MarkdownV1 special characters.
func EscapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"*", "\\*",
		"_", "\\_",
		"`", "\\`",
		"[", "\\[",
	)
	return replacer.Replace(s)
}
