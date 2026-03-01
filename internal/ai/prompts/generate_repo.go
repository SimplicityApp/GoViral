package prompts

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shuhao/goviral/pkg/models"
)

// SystemPromptRepoX is the system prompt for generating X (Twitter) posts from GitHub commits.
var SystemPromptRepoX = `You are a "build in public" ghostwriter for X (Twitter). Your job is to transform raw GitHub commits into viral, persona-matched posts that resonate with the developer community.

` + ToneDirective(PlatformX, false) + `

## X Platform Guidelines
- 280 characters max per tweet (thread-ready if longer)
- Punchy hooks that stop the scroll — first line must grab attention
- Curiosity gaps that make people click/expand
- 0-2 hashtags maximum (X algorithm penalizes hashtag spam)
- Short paragraphs, line breaks for readability
- Avoid looking like a changelog or release note

## Commit-Specific Angles to Explore
- "Just shipped X" announcements with a concrete benefit or number
- Before/after comparisons that show the delta clearly
- Performance improvements with real metrics (latency, throughput, memory)
- "TIL" technical insights disguised as lessons rather than patches
- Build-in-public progress: the struggle, the fix, the win
- Focus on the "why" behind the change, not just what changed
- Turn technical commits into relatable developer stories that non-specialists can appreciate

For each variation, include:
- "content": the post (ready to copy-paste, 280 chars or fewer per tweet)
- "viral_mechanic": brief note on what viral mechanic you used
- "confidence_score": number 1-10 on viral potential`

// SystemPromptRepoLinkedIn is the system prompt for generating LinkedIn posts from GitHub commits.
var SystemPromptRepoLinkedIn = `You are a "build in public" storyteller for LinkedIn. Your job is to transform raw GitHub commits into compelling, persona-matched posts that educate and inspire the professional developer community.

` + ToneDirective(PlatformLinkedIn, false) + `

## LinkedIn Guidelines
- 1000-2000 characters sweet spot for engagement
- Strong hook in the first 2 lines (before "...see more" truncation) — lead with the insight, not the commit
- Storytelling structure: hook → context → insight → takeaway → CTA
- Strategic line breaks — one thought per line, white space is your friend
- Professional but conversational — not corporate, not casual
- End with an engagement hook: "Agree?", "Has this bitten you before?", "What would you do differently?"
- 1-3 relevant hashtags at the end (not inline)
- Lists and numbered points perform well for technical breakdowns

## Commit-Specific Angles to Explore
- Problem → solution narratives: what broke, what you tried, what finally worked
- Lessons learned from the implementation — what you'd do differently
- Scaling challenges: what assumption failed at scale and how the commit addresses it
- Engineering decisions: the tradeoffs considered, the option you chose and why
- More narrative and educational than X — teach something concrete from the commit
- Turn the commit diff into a teachable moment for other developers

For each variation, include:
- "content": the post (ready to copy-paste, 1000-2000 chars)
- "viral_mechanic": brief note on what viral mechanic you used
- "confidence_score": number 1-10 on viral potential`

// RepoPostPrompt returns the system prompt for generating posts from GitHub commits.
func RepoPostPrompt(platform Platform) string {
	switch platform {
	case PlatformLinkedIn:
		return SystemPromptRepoLinkedIn
	default: // x
		return SystemPromptRepoX
	}
}

// NumberedDiffBlock builds a line-numbered diff block from per-file changes,
// skipping lockfiles and generated files. Each line gets a global L-number
// (e.g. "L1:", "L2:") so the AI can reference them precisely. File headers
// ("=== filename ===") are not numbered. Returns the formatted block and the
// total number of numbered lines emitted.
func NumberedDiffBlock(files []models.GitHubFileChange, maxLines int) (string, int) {
	if maxLines <= 0 {
		maxLines = 120
	}

	var b strings.Builder
	lineNum := 0

	for _, f := range files {
		if models.IsLockfile(f.Filename) {
			continue
		}
		if f.Patch == "" {
			continue
		}
		if lineNum >= maxLines {
			break
		}

		fmt.Fprintf(&b, "=== %s ===\n", f.Filename)

		for _, raw := range strings.Split(f.Patch, "\n") {
			if lineNum >= maxLines {
				break
			}
			// Skip git header lines within the patch
			if strings.HasPrefix(raw, "diff ") || strings.HasPrefix(raw, "index ") ||
				strings.HasPrefix(raw, "--- ") || strings.HasPrefix(raw, "+++ ") ||
				strings.HasPrefix(raw, "new file") || strings.HasPrefix(raw, "deleted file") ||
				strings.HasPrefix(raw, "rename ") || strings.HasPrefix(raw, "similarity ") {
				continue
			}
			lineNum++
			fmt.Fprintf(&b, "L%d: %s\n", lineNum, raw)
		}
	}

	return b.String(), lineNum
}

// BuildRepoUserMessage constructs the user message for repo-to-post generation.
func BuildRepoUserMessage(req models.RepoPostRequest) string {
	personaJSON, err := json.Marshal(req.Persona.Profile)
	if err != nil {
		personaJSON = []byte("{}")
	}

	commit := req.Commit
	repo := req.Repo

	// Short SHA (first 7 chars) for readability.
	shortSHA := commit.SHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## My Persona Profile\n%s\n\n", string(personaJSON))
	fmt.Fprintf(&b, "## Target Platform\n%s\n\n", req.TargetPlatform)

	fmt.Fprintf(&b, "## Commit Details\n")
	fmt.Fprintf(&b, "SHA: %s\n", shortSHA)
	fmt.Fprintf(&b, "Message: %s\n", commit.Message)
	fmt.Fprintf(&b, "Files changed: %d\n", commit.FilesChanged)
	fmt.Fprintf(&b, "Additions: %d  Deletions: %d\n\n", commit.Additions, commit.Deletions)

	if req.IncludeCodeImage && len(commit.Files) > 0 {
		// Use numbered diff block so the AI can select code snippets
		numbered, totalLines := NumberedDiffBlock(commit.Files, 120)
		if totalLines > 0 {
			fmt.Fprintf(&b, "### Numbered Diff (L1–L%d)\n```\n%s```\n\n", totalLines, numbered)
		}
	} else {
		// Fallback: raw truncated diff
		const maxDiffChars = 2000
		diff := commit.DiffPatch
		diffTruncated := false
		if len(diff) > maxDiffChars {
			diff = diff[:maxDiffChars]
			diffTruncated = true
		}
		if diff != "" {
			fmt.Fprintf(&b, "### Diff Excerpt\n```diff\n%s\n```", diff)
			if diffTruncated {
				fmt.Fprintf(&b, "\n... (diff truncated for brevity)")
			}
			fmt.Fprintf(&b, "\n\n")
		}
	}

	fmt.Fprintf(&b, "## Repository Context\n")
	fmt.Fprintf(&b, "Name: %s\n", repo.Name)
	if repo.Language != "" {
		fmt.Fprintf(&b, "Primary language: %s\n", repo.Language)
	}
	if repo.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", repo.Description)
	}
	fmt.Fprintf(&b, "\n")

	if req.TargetAudience != "" {
		fmt.Fprintf(&b, "## Target Audience\n")
		fmt.Fprintf(&b, "Write this post specifically for: %s\n", req.TargetAudience)
		fmt.Fprintf(&b, "Use language, references, and examples that resonate with this audience.\n\n")
	}

	fmt.Fprintf(&b, "## Instructions\n")
	fmt.Fprintf(&b, "1. Study the commit and understand what changed and why\n")
	if req.TargetAudience != "" {
		fmt.Fprintf(&b, "2. Identify the most compelling angle for a build-in-public post that resonates with %s\n", req.TargetAudience)
	} else {
		fmt.Fprintf(&b, "2. Identify the most compelling angle for a build-in-public post\n")
	}
	fmt.Fprintf(&b, "3. Write in the persona voice above — authentic and non-generic\n")
	fmt.Fprintf(&b, "4. Focus on the human story behind the code change\n")
	fmt.Fprintf(&b, "5. Optimize for %s platform\n", req.TargetPlatform)
	if req.MaxChars > 0 {
		fmt.Fprintf(&b, "6. Each post MUST be %d characters or fewer\n", req.MaxChars)
	}

	if req.IncludeCodeImage {
		fmt.Fprintf(&b, "\n## Code Snippet Selection\n")
		fmt.Fprintf(&b, "For each variation, also select a code snippet for the accompanying image:\n")
		fmt.Fprintf(&b, "- Pick 8-25 consecutive diff lines (by L-number) that best illustrate the change your post discusses\n")
		fmt.Fprintf(&b, "- Prefer additions (+lines) over deletions for visual appeal\n")
		fmt.Fprintf(&b, "- Include context lines around the additions for readability\n")
		fmt.Fprintf(&b, "- Return the filename, start_line, and end_line in the code_snippet field\n")
		fmt.Fprintf(&b, "- Write a short image_description (1 sentence, max 80 chars) that explains what the code snippet shows — e.g. \"Added retry logic with exponential backoff\". This appears as an overlay on the code image.\n")
	}

	if len(req.Links) > 0 {
		fmt.Fprintf(&b, "\n## Required Links\n")
		fmt.Fprintf(&b, "Every post variation MUST end with the following links on separate lines.\n")
		fmt.Fprintf(&b, "Use the exact format \"Label: URL\" so readers know what each link is before clicking (platforms like LinkedIn shorten URLs, making bare links unrecognizable).\n")
		fmt.Fprintf(&b, "Example format:\n")
		fmt.Fprintf(&b, "  GitHub: https://github.com/owner/repo\n")
		fmt.Fprintf(&b, "  PyPI: https://pypi.org/project/name\n\n")
		fmt.Fprintf(&b, "Links to include:\n")
		for _, link := range req.Links {
			fmt.Fprintf(&b, "- %s: %s\n", link.Label, link.URL)
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "\nGenerate %d variations.\n", req.Count)

	if req.StyleDirection != "" {
		fmt.Fprintf(&b, "\nStyle direction from the user: %s\nIncorporate this tone/style preference into your post.\n", req.StyleDirection)
	}

	return b.String()
}
