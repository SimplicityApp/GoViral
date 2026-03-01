package github

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shuhao/goviral/pkg/models"
)

// DiffLineType classifies a single line within a unified diff hunk.
type DiffLineType int

const (
	// DiffLineContext is an unchanged context line (no leading + or -).
	DiffLineContext DiffLineType = iota
	// DiffLineAddition is a line added in the new version (leading +).
	DiffLineAddition
	// DiffLineDeletion is a line removed from the old version (leading -).
	DiffLineDeletion
)

// DiffLine represents a single line within a diff hunk with its type and raw content.
type DiffLine struct {
	Type    DiffLineType
	Content string // line content excluding the leading +/-/space sigil
}

// DiffHunk represents one @@ … @@ section of a unified diff.
type DiffHunk struct {
	OldStart int
	OldLen   int
	NewStart int
	NewLen   int
	Lines    []DiffLine
}

// ParseUnifiedDiff parses a unified diff patch string (as returned by the GitHub
// API in the file.Patch field) into a slice of structured DiffHunks.
//
// The function is tolerant of missing or malformed hunk headers: lines that
// cannot be parsed are silently skipped so partial patches still yield useful
// results.
func ParseUnifiedDiff(patch string) []DiffHunk {
	if patch == "" {
		return nil
	}

	var hunks []DiffHunk
	var current *DiffHunk

	for _, line := range strings.Split(patch, "\n") {
		// Hunk header: @@ -OldStart,OldLen +NewStart,NewLen @@
		if strings.HasPrefix(line, "@@") {
			if current != nil {
				hunks = append(hunks, *current)
			}
			hunk, ok := parseHunkHeader(line)
			if !ok {
				current = nil
				continue
			}
			current = &hunk
			continue
		}

		if current == nil {
			// Lines before the first hunk header (file header lines like --- / +++)
			// are not part of any hunk.
			continue
		}

		switch {
		case strings.HasPrefix(line, "+"):
			current.Lines = append(current.Lines, DiffLine{
				Type:    DiffLineAddition,
				Content: line[1:],
			})
		case strings.HasPrefix(line, "-"):
			current.Lines = append(current.Lines, DiffLine{
				Type:    DiffLineDeletion,
				Content: line[1:],
			})
		default:
			// Context line: may start with a space or be empty (e.g. trailing newline).
			content := line
			if strings.HasPrefix(line, " ") {
				content = line[1:]
			}
			current.Lines = append(current.Lines, DiffLine{
				Type:    DiffLineContext,
				Content: content,
			})
		}
	}

	if current != nil {
		hunks = append(hunks, *current)
	}

	return hunks
}

// parseHunkHeader extracts OldStart/OldLen/NewStart/NewLen from a hunk header
// line of the form "@@ -1,4 +1,6 @@ optional section heading".
// Returns false if the line does not match the expected format.
func parseHunkHeader(line string) (DiffHunk, bool) {
	// Find the content between the first @@ pair.
	rest := strings.TrimPrefix(line, "@@")
	end := strings.Index(rest, "@@")
	if end < 0 {
		return DiffHunk{}, false
	}
	rangeStr := strings.TrimSpace(rest[:end]) // e.g. "-1,4 +1,6"

	parts := strings.Fields(rangeStr) // ["-1,4", "+1,6"]
	if len(parts) < 2 {
		return DiffHunk{}, false
	}

	oldStart, oldLen, ok1 := parseRange(parts[0], '-')
	newStart, newLen, ok2 := parseRange(parts[1], '+')
	if !ok1 || !ok2 {
		return DiffHunk{}, false
	}

	return DiffHunk{
		OldStart: oldStart,
		OldLen:   oldLen,
		NewStart: newStart,
		NewLen:   newLen,
	}, true
}

// parseRange parses a unified diff range token such as "-12,5" or "+1" (len
// defaults to 1 when the comma part is absent). sigil is the expected leading
// character ('+' or '-').
func parseRange(token string, sigil byte) (start, length int, ok bool) {
	if len(token) == 0 || token[0] != sigil {
		return 0, 0, false
	}
	token = token[1:] // strip leading sigil
	if idx := strings.IndexByte(token, ','); idx >= 0 {
		s, err1 := strconv.Atoi(token[:idx])
		l, err2 := strconv.Atoi(token[idx+1:])
		if err1 != nil || err2 != nil {
			return 0, 0, false
		}
		return s, l, true
	}
	s, err := strconv.Atoi(token)
	if err != nil {
		return 0, 0, false
	}
	return s, 1, true
}

// ---- lockfile / generated-file detection (delegated to pkg/models) ----

// isLockfile returns true when f is a lock file or a known generated file.
func isLockfile(f models.GitHubFileChange) bool {
	return models.IsLockfile(f.Filename)
}

// isConfig returns true when f looks like a configuration / data file.
func isConfig(f models.GitHubFileChange) bool {
	return models.IsConfigFile(f.Filename)
}

// isSource returns true when f is a recognised source-code file.
func isSource(f models.GitHubFileChange) bool {
	return models.IsSourceFile(f.Filename)
}

// fileScore returns a numeric score for how "interesting" a file change is for
// use in a code image.  Higher is better.
//
// Scoring heuristic:
//   - Source files start with a base bonus of 100.
//   - Config files start with a base bonus of 0.
//   - net additions (additions − deletions) add to the score, capped to avoid
//     enormous files dominating purely on line count.
//   - Files with only deletions are slightly penalised (less visual context).
func fileScore(f models.GitHubFileChange) int {
	var base int
	if isSource(f) {
		base = 100
	}

	net := f.Additions - f.Deletions
	// Cap the net contribution so a 1000-line dump doesn't dominate.
	if net > 50 {
		net = 50
	}

	// Small penalty when no additions exist (pure deletions are hard to display).
	if f.Additions == 0 {
		net -= 10
	}

	return base + net
}

// SelectBestFile picks the most visually interesting file change from a commit
// for use in a code image.
//
// Selection logic (in priority order):
//  1. Exclude lock/generated files entirely.
//  2. Among what remains, separate source files from config files.
//  3. If any source files exist, choose the highest-scored one among them.
//  4. Otherwise fall back to the highest-scored config file.
//  5. If all files are lock/generated, return nil.
func SelectBestFile(files []models.GitHubFileChange) *models.GitHubFileChange {
	if len(files) == 0 {
		return nil
	}

	var sources, configs []models.GitHubFileChange
	for _, f := range files {
		if isLockfile(f) {
			continue
		}
		if isConfig(f) {
			configs = append(configs, f)
		} else {
			sources = append(sources, f)
		}
	}

	candidates := sources
	if len(candidates) == 0 {
		candidates = configs
	}
	if len(candidates) == 0 {
		return nil
	}

	best := &candidates[0]
	bestScore := fileScore(candidates[0])
	for i := 1; i < len(candidates); i++ {
		if s := fileScore(candidates[i]); s > bestScore {
			bestScore = s
			best = &candidates[i]
		}
	}

	return best
}

// SummarizeDiff produces a compact, human-readable one-line summary of the
// changes in a commit suitable for embedding in social media posts.
//
// Example output:
//
//	"Modified 3 files: +45 -12 across api/handler.go, models/user.go, config.yaml"
func SummarizeDiff(commit models.GitHubCommit) string {
	if len(commit.Files) == 0 && commit.Additions == 0 && commit.Deletions == 0 {
		return fmt.Sprintf("No file changes recorded for %s", shortSHA(commit.SHA))
	}

	// Determine the action verb based on net change direction.
	action := "Modified"
	if commit.Additions > 0 && commit.Deletions == 0 {
		action = "Added"
	} else if commit.Additions == 0 && commit.Deletions > 0 {
		action = "Removed"
	}

	n := commit.FilesChanged
	if n == 0 {
		n = len(commit.Files)
	}

	noun := "file"
	if n != 1 {
		noun = "files"
	}

	// Collect the most interesting filenames for the trailing list.
	// Show the best-ranked source file first, then fill up to 3 total.
	names := rankedFilenames(commit.Files, 3)

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s %d %s: +%d -%d", action, n, noun, commit.Additions, commit.Deletions)
	if len(names) > 0 {
		sb.WriteString(" across ")
		sb.WriteString(strings.Join(names, ", "))
		if n > len(names) {
			fmt.Fprintf(&sb, " (+%d more)", n-len(names))
		}
	}
	return sb.String()
}

// rankedFilenames returns up to limit filenames from files, ranked so that
// interesting source files appear before lock/config files.
func rankedFilenames(files []models.GitHubFileChange, limit int) []string {
	// Separate into interesting and boring.
	var interesting, boring []string
	for _, f := range files {
		if isLockfile(f) {
			continue
		}
		if isSource(f) {
			interesting = append(interesting, f.Filename)
		} else {
			boring = append(boring, f.Filename)
		}
	}

	ordered := append(interesting, boring...)
	if len(ordered) > limit {
		ordered = ordered[:limit]
	}
	return ordered
}

// ---- helpers ----

// shortSHA returns the first 7 characters of a commit SHA.
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
