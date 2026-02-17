package thread

import (
	"fmt"
	"strings"
)

const maxTweetLength = 280

// SplitResult contains the parts of a split thread.
type SplitResult struct {
	Parts []string
}

// Split splits content into tweet-sized parts for threading.
// It checks for explicit "---" markers first, then auto-splits at sentence
// and word boundaries if the content exceeds 280 characters.
// If numbered is true, each part gets a " (1/N)" suffix.
func Split(content string, numbered bool) SplitResult {
	content = strings.TrimSpace(content)

	// Check for explicit --- markers
	if strings.Contains(content, "---") {
		parts := splitOnMarkers(content)
		if numbered {
			parts = addNumbering(parts)
		}
		return SplitResult{Parts: parts}
	}

	// If content fits in a single tweet, no splitting needed
	if len(content) <= maxTweetLength {
		return SplitResult{Parts: []string{content}}
	}

	// Auto-split
	parts := autoSplit(content, numbered)
	if numbered {
		parts = addNumbering(parts)
	}
	return SplitResult{Parts: parts}
}

func splitOnMarkers(content string) []string {
	raw := strings.Split(content, "---")
	var parts []string
	for _, part := range raw {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func autoSplit(content string, numbered bool) []string {
	// Reserve space for numbering suffix like " (1/N)"
	// Max suffix length: " (XX/XX)" = 8 chars, but typically " (1/9)" = 6
	suffixReserve := 0
	if numbered {
		suffixReserve = 8
	}
	maxLen := maxTweetLength - suffixReserve

	var parts []string
	remaining := content

	for len(remaining) > 0 {
		remaining = strings.TrimSpace(remaining)
		if len(remaining) <= maxLen {
			parts = append(parts, remaining)
			break
		}

		// Try to split at sentence boundary
		splitIdx := findSentenceBoundary(remaining, maxLen)
		if splitIdx <= 0 {
			// Fall back to word boundary
			splitIdx = findWordBoundary(remaining, maxLen)
		}
		if splitIdx <= 0 {
			// Hard split as last resort
			splitIdx = maxLen
		}

		parts = append(parts, strings.TrimSpace(remaining[:splitIdx]))
		remaining = remaining[splitIdx:]
	}

	return parts
}

func findSentenceBoundary(text string, maxLen int) int {
	best := -1
	for i, ch := range text {
		if i >= maxLen {
			break
		}
		if ch == '.' || ch == '!' || ch == '?' {
			// Check if next char is a space or end of string (actual sentence end)
			nextIdx := i + 1
			if nextIdx >= len(text) || text[nextIdx] == ' ' || text[nextIdx] == '\n' {
				best = nextIdx
			}
		}
	}
	return best
}

func findWordBoundary(text string, maxLen int) int {
	best := -1
	for i := range text {
		if i >= maxLen {
			break
		}
		if text[i] == ' ' || text[i] == '\n' {
			best = i
		}
	}
	return best
}

func addNumbering(parts []string) []string {
	total := len(parts)
	if total <= 1 {
		return parts
	}
	numbered := make([]string, total)
	for i, part := range parts {
		suffix := fmt.Sprintf(" (%d/%d)", i+1, total)
		// Trim part if adding suffix would exceed limit
		if len(part)+len(suffix) > maxTweetLength {
			part = part[:maxTweetLength-len(suffix)]
			// Try to end at a word boundary
			if lastSpace := strings.LastIndex(part, " "); lastSpace > len(part)-20 {
				part = part[:lastSpace]
			}
		}
		numbered[i] = part + suffix
	}
	return numbered
}
