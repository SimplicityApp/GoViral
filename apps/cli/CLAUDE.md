# CLI — `apps/cli/`

## Structure
- `main.go` — Entry point, calls `cmd.Execute()`
- `cmd/root.go` — Root Cobra command, config loading
- `cmd/*.go` — 18 command files registered via `rootCmd.AddCommand()` in `init()`

## Commands
| File | Command | Purpose |
|------|---------|---------|
| `init.go` | `init` | Interactive setup wizard for config and database |
| `auth.go` | `auth` | OAuth flows for X and LinkedIn |
| `fetch.go` | `fetch` | Fetch user's posts from platforms |
| `trending.go` | `trending` | Discover trending posts in niches |
| `generate.go` | `generate` | Generate viral content from trending posts |
| `post.go` | `post` | Post/schedule generated content (threads, media, quotes) |
| `profile.go` | `profile` | Build/show/refresh persona profiles |
| `comment.go` | `comment` | Generate and post AI comments |
| `repo.go` | `repo` | GitHub repo management and commit-to-post |
| `history.go` | `history` | Display content history |
| `posts.go` | `posts` | List stored posts |
| `sync_cookies.go` | `sync-cookies` | Sync cookies for fallback clients |
| `twikit_login.go` | `twikit-login` | Cookie-based X authentication |
| `linkitin_login.go` | `linkitin-login` | Cookie-based LinkedIn authentication |
| `youtube_login.go` | `youtube-login` | YouTube authentication |
| `tiktok_login.go` | `tiktok-login` | TikTok authentication |

## Patterns

**Interactive TUI**: `huh` library for form prompts (selects, inputs), `lipgloss` for styled terminal output with color codes.

**Config access**: Direct file access to `~/.goviral/config.yaml` via `config.Load("")`. No server dependency — CLI reads config and database directly.

**Fallback client usage**: Commands create `FallbackClient` instances that try official API first, then cookie-based fallback:
```go
client := xplatform.NewFallbackClient(cfg.X)
```

**Generate pipeline**: Load persona -> select trending posts -> call Claude for content -> optionally generate images via Gemini -> store as "draft" in database.

**Post pipeline**: Preview with thread splitting -> upload media -> post (with 1s delay between thread parts) -> update status to "posted".

## Gemini Image Generation (CLI-specific)
The full image generation pipeline lives in the CLI's `generate.go`:
1. `--images` flag or user prompt triggers image generation
2. `gen.GenerateImagePrompt(ctx, content, platform)` creates prompt
3. `gemini.NewClient().GenerateImage(ctx, prompt)` generates image
4. `gemini.SaveImage(img, name)` saves with naming: `gen_{trendingID}_{variation}_{timestamp}`
5. Path stored in `GeneratedContent.ImagePath`

## Terminal Styling
Consistent `lipgloss` styles: `headerStyle`, `successStyle`, `errorStyle`, `metricsStyle` used across commands for uniform appearance.
