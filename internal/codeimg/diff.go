// Package codeimg renders GitHub-style diff images using headless Chrome.
package codeimg

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// DiffLineType represents the kind of a single diff line.
type DiffLineType string

const (
	// DiffLineContext is an unchanged context line.
	DiffLineContext DiffLineType = "context"
	// DiffLineAddition is a line added in the new version.
	DiffLineAddition DiffLineType = "addition"
	// DiffLineDeletion is a line removed from the old version.
	DiffLineDeletion DiffLineType = "deletion"
	// DiffLineHunkHeader is the @@ ... @@ separator between hunks.
	DiffLineHunkHeader DiffLineType = "hunk_header"
)

// TemplateDiffLine is a single parsed diff line ready for the HTML template.
type TemplateDiffLine struct {
	// OldLineNum is the line number in the original file (0 = not applicable).
	OldLineNum int
	// NewLineNum is the line number in the patched file (0 = not applicable).
	NewLineNum int
	// Type classifies the line for styling purposes.
	Type DiffLineType
	// Content is the raw text of the line, without the leading +/-/space prefix.
	Content string
}

// TemplateDiffData holds everything the HTML template needs to render a diff.
type TemplateDiffData struct {
	// Filename is the name of the changed file (may be empty until set by the caller).
	Filename string
	// Language is the detected programming language derived from the file extension.
	Language string
	// Additions is the total number of added lines visible in this diff.
	Additions int
	// Deletions is the total number of deleted lines visible in this diff.
	Deletions int
	// Lines are the parsed diff lines in order.
	Lines []TemplateDiffLine

	// Description is an optional human-readable explanation of the changes
	// (used by templates that support a description area, e.g. "card").
	Description string
	// RepoName is the "owner/repo" string for display in template headers.
	RepoName string
	// Theme provides the colour palette for CSS variable injection.
	Theme ThemeColors
}

// defaultMaxLines is used when the caller passes maxLines <= 0.
const defaultMaxLines = 25

// FormatDiffForTemplate parses a unified diff patch string into a
// TemplateDiffData value ready for rendering. maxLines limits the total
// number of lines included; pass 0 or a negative value to use the default
// of 25.
//
// The function understands standard unified-diff syntax:
//
//	@@ -oldStart,oldLen +newStart,newLen @@ optional section heading
//	 context line
//	-deleted line
//	+added line
func FormatDiffForTemplate(patch string, maxLines int) TemplateDiffData {
	if maxLines <= 0 {
		maxLines = defaultMaxLines
	}

	data := TemplateDiffData{}

	// Track old/new line counters across the whole patch.
	oldLine := 0
	newLine := 0

	lines := strings.Split(patch, "\n")
	for _, raw := range lines {
		if len(data.Lines) >= maxLines {
			break
		}

		// Hunk header: @@ -old,len +new,len @@ optional text
		if strings.HasPrefix(raw, "@@") {
			oldStart, newStart := parseHunkHeader(raw)
			oldLine = oldStart
			newLine = newStart

			data.Lines = append(data.Lines, TemplateDiffLine{
				Type:    DiffLineHunkHeader,
				Content: raw,
			})
			continue
		}

		// Skip file header lines produced by git (--- a/... / +++ b/...).
		if strings.HasPrefix(raw, "--- ") || strings.HasPrefix(raw, "+++ ") {
			continue
		}

		// Skip "diff --git a/... b/..." lines.
		if strings.HasPrefix(raw, "diff ") || strings.HasPrefix(raw, "index ") ||
			strings.HasPrefix(raw, "new file") || strings.HasPrefix(raw, "deleted file") ||
			strings.HasPrefix(raw, "rename ") || strings.HasPrefix(raw, "similarity ") {
			continue
		}

		switch {
		case strings.HasPrefix(raw, "+"):
			data.Additions++
			data.Lines = append(data.Lines, TemplateDiffLine{
				NewLineNum: newLine,
				Type:       DiffLineAddition,
				Content:    raw[1:],
			})
			newLine++

		case strings.HasPrefix(raw, "-"):
			data.Deletions++
			data.Lines = append(data.Lines, TemplateDiffLine{
				OldLineNum: oldLine,
				Type:       DiffLineDeletion,
				Content:    raw[1:],
			})
			oldLine++

		case strings.HasPrefix(raw, " ") || raw == "":
			// Context line — a leading space is present in well-formed diffs;
			// a completely empty string can occur at the very end of a hunk.
			content := raw
			if strings.HasPrefix(raw, " ") {
				content = raw[1:]
			}
			data.Lines = append(data.Lines, TemplateDiffLine{
				OldLineNum: oldLine,
				NewLineNum: newLine,
				Type:       DiffLineContext,
				Content:    content,
			})
			oldLine++
			newLine++

		default:
			// Unrecognised line — render as context so we don't drop it silently.
			data.Lines = append(data.Lines, TemplateDiffLine{
				OldLineNum: oldLine,
				NewLineNum: newLine,
				Type:       DiffLineContext,
				Content:    raw,
			})
			oldLine++
			newLine++
		}
	}

	return data
}

// parseHunkHeader extracts the starting old and new line numbers from an
// @@ -old,len +new,len @@ header.  On parse failure it returns (1, 1).
func parseHunkHeader(line string) (oldStart, newStart int) {
	// Example: "@@ -10,6 +10,8 @@ func Foo() {"
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return 1, 1
	}

	oldStart = parseHunkRange(parts[1]) // "-10,6"
	newStart = parseHunkRange(parts[2]) // "+10,8"
	return oldStart, newStart
}

// parseHunkRange extracts the starting line number from a "-N,M" or "+N,M"
// token as found in hunk headers.
func parseHunkRange(token string) int {
	// Strip leading - or +.
	if len(token) == 0 {
		return 1
	}
	s := token[1:]

	// Take only the part before the comma.
	if idx := strings.IndexByte(s, ','); idx >= 0 {
		s = s[:idx]
	}

	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// FormatDiffForTemplateRange parses a single file's unified diff patch and
// returns only the lines whose global line number falls in [startLine, endLine].
// globalOffset is the number of numbered lines already consumed by prior files
// (using the same counting logic as NumberedDiffBlock in the prompts package).
// A synthetic hunk header is inserted if the range doesn't start on one.
func FormatDiffForTemplateRange(patch string, globalOffset, startLine, endLine int) TemplateDiffData {
	data := TemplateDiffData{}
	if patch == "" {
		return data
	}

	oldLine := 0
	newLine := 0
	lineNum := globalOffset // global counter matching NumberedDiffBlock logic

	rawLines := strings.Split(patch, "\n")
	needsHeader := true // whether we need to synthesize a hunk header

	for _, raw := range rawLines {
		// Skip git header lines (same as NumberedDiffBlock)
		if strings.HasPrefix(raw, "diff ") || strings.HasPrefix(raw, "index ") ||
			strings.HasPrefix(raw, "--- ") || strings.HasPrefix(raw, "+++ ") ||
			strings.HasPrefix(raw, "new file") || strings.HasPrefix(raw, "deleted file") ||
			strings.HasPrefix(raw, "rename ") || strings.HasPrefix(raw, "similarity ") {
			continue
		}

		// Hunk header: update counters but also counts as a numbered line
		if strings.HasPrefix(raw, "@@") {
			oldStart, newStart := parseHunkHeader(raw)
			oldLine = oldStart
			newLine = newStart

			lineNum++
			if lineNum >= startLine && lineNum <= endLine {
				data.Lines = append(data.Lines, TemplateDiffLine{
					Type:    DiffLineHunkHeader,
					Content: raw,
				})
				needsHeader = false
			}
			continue
		}

		lineNum++
		if lineNum < startLine {
			// Advance counters without emitting
			switch {
			case strings.HasPrefix(raw, "+"):
				newLine++
			case strings.HasPrefix(raw, "-"):
				oldLine++
			default:
				oldLine++
				newLine++
			}
			continue
		}
		if lineNum > endLine {
			break
		}

		// Synthesize a hunk header if needed
		if needsHeader {
			header := fmt.Sprintf("@@ -%d +%d @@", oldLine, newLine)
			data.Lines = append(data.Lines, TemplateDiffLine{
				Type:    DiffLineHunkHeader,
				Content: header,
			})
			needsHeader = false
		}

		switch {
		case strings.HasPrefix(raw, "+"):
			data.Additions++
			data.Lines = append(data.Lines, TemplateDiffLine{
				NewLineNum: newLine,
				Type:       DiffLineAddition,
				Content:    raw[1:],
			})
			newLine++
		case strings.HasPrefix(raw, "-"):
			data.Deletions++
			data.Lines = append(data.Lines, TemplateDiffLine{
				OldLineNum: oldLine,
				Type:       DiffLineDeletion,
				Content:    raw[1:],
			})
			oldLine++
		default:
			content := raw
			if strings.HasPrefix(raw, " ") {
				content = raw[1:]
			}
			data.Lines = append(data.Lines, TemplateDiffLine{
				OldLineNum: oldLine,
				NewLineNum: newLine,
				Type:       DiffLineContext,
				Content:    content,
			})
			oldLine++
			newLine++
		}
	}

	return data
}

// DetectLanguage maps a filename's extension to a human-readable language
// name used in the template for display and (future) syntax highlighting.
func DetectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go":
		return "Go"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "JavaScript"
	case ".py":
		return "Python"
	case ".rs":
		return "Rust"
	case ".java":
		return "Java"
	case ".kt", ".kts":
		return "Kotlin"
	case ".swift":
		return "Swift"
	case ".c", ".h":
		return "C"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "C++"
	case ".cs":
		return "C#"
	case ".rb":
		return "Ruby"
	case ".php":
		return "PHP"
	case ".sh", ".bash", ".zsh":
		return "Shell"
	case ".yaml", ".yml":
		return "YAML"
	case ".json":
		return "JSON"
	case ".toml":
		return "TOML"
	case ".md", ".mdx":
		return "Markdown"
	case ".sql":
		return "SQL"
	case ".html", ".htm":
		return "HTML"
	case ".css", ".scss", ".sass":
		return "CSS"
	case ".proto":
		return "Protobuf"
	case ".tf", ".tfvars":
		return "Terraform"
	case ".dockerfile", "":
		if strings.EqualFold(filepath.Base(filename), "dockerfile") {
			return "Dockerfile"
		}
		return ""
	default:
		return fmt.Sprintf("%s file", strings.TrimPrefix(ext, "."))
	}
}
