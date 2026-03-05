# Internal Packages — `internal/`

## Package Boundaries
- `internal/` packages are only importable by `apps/` and other `internal/` packages
- All return types that cross package boundaries must use interfaces/types from `pkg/models/`
- No circular dependencies between internal packages

## Config Merging (`config/`)
Two-layer config system:
- **Global** (`~/.goviral/config.yaml`): app-level credentials (OAuth ClientID/Secret, shared API keys), server settings, daemon config
- **Per-user** (SQLite `user_config` table, JSON): user OAuth tokens, BYOK API keys, niches, platform usernames

Merging functions:
- `MergedXConfig(global, userCfg)` — app creds from global, user tokens from per-user
- `MergedLinkedInConfig(global, userCfg)` — same pattern
- `ResolvedClaudeConfig(global, userCfg)` — user key if set (BYOK), else global key
- `MergedGitHubToken(userCfg)` — user token ONLY, no global fallback (prevents credential leaking)
- `MergedNiches(global, userCfg)` — user niches if set, else global defaults

Env var overrides: `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`, `CLAUDE_API_KEY`, `GEMINI_API_KEY`

## Database (`db/`)
- **SQLite** via `modernc.org/sqlite` (pure Go, no CGO)
- **WAL mode** for concurrent access, 5s busy timeout
- **Path**: `~/.goviral/goviral.db`
- **Migrations**: idempotent with `CREATE TABLE IF NOT EXISTS`, `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`
- **Multi-tenancy**: all user-scoped tables have `user_id` with composite unique indexes
- **Tables**: `users`, `my_posts`, `persona`, `trending_posts`, `generated_content`, `scheduled_posts`, `daemon_batches`, `github_repos`, `repo_commits`, `user_config`, `ai_usage`

## Platform Client Pattern (`platform/`)
Each platform follows primary + fallback pattern:

```
platform/x/
  client.go      # Primary: X API v2 (Bearer token auth)
  fallback.go    # FallbackClient: tries primary, falls back to twikit
  twikit.go      # Python subprocess bridge (cookie-based)
  twikit_guest.py # Embedded Python script (//go:embed)
```

- Primary client implements `models.PlatformClient`, `models.PlatformPoster`, etc.
- Fallback wraps primary with automatic failover on account-level errors
- Per-user cookie isolation via `cookiePath` / `configDir` parameters

Platforms: `x/`, `linkedin/`, `youtube/`, `tiktok/`, `github/` (GitHub has no fallback)

## AI Pipeline (`ai/`)
- **Claude** (`ai/claude/`): `MessageSender` interface with `SendMessage()` and `SendMessageJSON()` (structured output via JSON schema)
- **Gemini** (`ai/gemini/`): Image generation only — `GenerateImage()` returns `{Data, MIMEType}`
- **Generator** (`ai/generator/`): Implements `ContentGenerator` — `Generate()`, `GenerateRepoPost()`, `GenerateComment()`, `Classify()`, `Compete()`
- **Persona** (`ai/persona/`): `BuildProfile()` analyzes user posts via Claude to create writing style profile
- **Prompts** (`ai/prompts/`): 14 template files with platform-specific prompts, JSON schemas, helper functions

Pipeline: persona analysis -> trending discovery -> classify (rewrite/repost) -> generate variations -> compete (rank) -> approve -> publish

## Python Bridge Embedding
```go
//go:embed scripts/twikit_guest.py
var twikitScript []byte
```
- Scripts extracted to `~/.goviral/` at runtime
- Virtualenv at `~/.goviral/venv/` (handles PEP 668)
- Subprocess execution with JSON stdout/stderr protocol
- Embedded scripts: `twikit_guest.py`, `linkitin_bridge.py`, `youtube_bridge.py`, `tiktok_bridge.py`

## Daemon (`daemon/`)
Autopilot orchestrator running per-platform schedules:
1. Discover trending posts
2. Classify (rewrite vs repost vs comment)
3. Generate content variations
4. Compete and rank
5. Notify via Telegram for approval
6. Publish approved content

- **Dependency injection**: accepts function types (`GenerateFunc`, `PublishFunc`, `DiscoverFunc`)
- **CronScheduler**: per-platform cron expressions via `robfig/cron`
- **Intent parser** (`intent.go`): Claude-based parsing of Telegram replies into structured commands
- **Digest mode**: accumulate content, send daily digest, auto-publish best

## Rate Limiting (`ratelimit/`)
- `CheckAIRateLimit(db, userID, provider, dailyCap)` — checks daily usage against cap
- `RecordAIUsage(db, userID, provider)` — increments counter
- Only applied when using shared (global) API key, not BYOK
- Tracked in `ai_usage` table per user/provider/day

## Code Image Rendering (`codeimg/`)
- Uses `chromedp` (headless Chrome) to render code diffs as PNG
- Templates: VSCode, Terminal, macOS, Minimal, Card
- 15+ syntax themes: Dracula, Nord, Solarized, GitHub, etc.
- `NewRenderer()` starts persistent Chrome allocator; `RenderDiff()` generates images

## Telegram (`telegram/`)
- Bot API client: `SendMessage()`, `EditMessage()`, `GetUpdates()` (long-polling)
- Inline keyboard buttons for approve/reject/skip
- Used by daemon for batch notification and approval workflow

## Auth (`auth/`)
- OAuth 2.0 with PKCE for X (authorization code flow)
- Standard OAuth for LinkedIn and GitHub
- `StartCallbackServer()`: temp HTTP server on localhost for OAuth callback
- `OpenBrowser()`: cross-platform browser opener

## Thread Splitting (`thread/`)
- Max tweet length: 280 characters
- Split strategies (in order): explicit `---` markers -> sentence boundary -> word boundary -> hard split
- Optional `(1/N)` numbering suffix

## Error Wrapping Convention
Always wrap errors with operation context:
```go
fmt.Errorf("fetching repo %s/%s: %w", owner, name, err)
fmt.Errorf("running twikit subprocess: %w (stderr: %s)", err, stderrMsg)
```
