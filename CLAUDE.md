# GoViral — Project Instructions

## Overview
GoViral is a monorepo for viral content creation on X, LinkedIn, YouTube, and TikTok. It includes:
- **CLI** (`apps/cli/`) — Interactive terminal tool with Cobra + bubbletea/huh
- **Server** (`apps/server/`) — HTTP API with chi router, SSE for long-running ops
- **Web Dashboard** (`apps/web/`) — React 19 + TypeScript + Vite + Tailwind SPA

## Project Structure
```
apps/
  cli/           # Cobra CLI (18 command files)
  server/        # HTTP API server
    handler/     # 29 route handlers
    service/     # 10 business-logic services
    middleware/   # recovery, logging, cors, user-id
    dto/         # request/response types + error codes
    router/      # chi route group setup
    embed.go     # -tags embedweb: serve web assets (SPA fallback)
  web/           # React dashboard (SPA)
internal/
  ai/claude/     # Claude API client (text generation)
  ai/gemini/     # Gemini API client (image generation)
  ai/persona/    # Persona analysis logic
  ai/generator/  # Content generation, classification, competition
  ai/prompts/    # Prompt templates and JSON schemas (14 files)
  auth/          # OAuth flows (X, LinkedIn, GitHub)
  config/        # Config management (global + per-user merging)
  db/            # SQLite via modernc.org/sqlite (WAL mode)
  daemon/        # Autopilot: trending -> generate -> approve -> publish
  platform/x/         # X API v2 + twikit fallback
  platform/linkedin/  # LinkedIn API + linkitin fallback
  platform/youtube/   # YouTube Data API v3 + bridge fallback
  platform/tiktok/    # TikTok Content Posting API + bridge fallback
  platform/github/    # GitHub REST API v3 (repos, commits, diffs)
  ratelimit/     # Daily AI usage caps (shared key protection)
  telegram/      # Telegram Bot API (daemon notifications + approval)
  codeimg/       # Code diff -> PNG rendering (chromedp + syntax themes)
  thread/        # Tweet thread splitting logic
pkg/
  models/        # Shared interfaces and data models (contract layer)
```

## Build & Run Commands
```bash
# CLI
go run ./apps/cli ...           # Run CLI commands
go build -o goviral ./apps/cli  # Build CLI binary

# Server (dev)
go run ./apps/server            # Start API server on :8080

# Server (production with embedded web)
cd apps/web && npm run build
cp -r apps/web/dist/* apps/server/static/
go build -tags embedweb -o goviral-server ./apps/server

# Web (dev)
cd apps/web && npm run dev      # Vite dev server, proxies /api -> :8080

# Type check
cd apps/web && npx tsc --noEmit

# Build all Go
go build ./...
```

## Conventions
- Go module path: `github.com/shuhao/goviral`
- Use `modernc.org/sqlite` (pure Go, no CGO)
- All shared data models in `pkg/models/` — no business logic, no external deps beyond stdlib + uuid
- All business logic in `internal/`
- CLI-specific code only in `apps/cli/`
- Error handling: wrap errors with context using `fmt.Errorf("doing X: %w", err)`
- Use `lipgloss` for terminal output styling
- Config stored at `~/.goviral/config.yaml` (global) + SQLite `user_config` table (per-user)
- Server uses `go-chi/chi/v5` with middleware stack: Recovery -> Logging -> CORS -> UserID
- **Fallback client pattern**: official API client -> cookie-based fallback (twikit for X, linkitin for LinkedIn, Python bridges for YouTube/TikTok)
- Python bridges are embedded via `//go:embed` into Go binaries, extracted to `~/.goviral/` at runtime
- Python venv at `~/.goviral/venv/`
- Server embeds web assets with `-tags embedweb` build tag
- Web dev server proxies `/api` to Go server at `:8080`
- Long-running operations use SSE (Accept: text/event-stream) with 202 Accepted + operation polling as fallback
- **Config merging**: app-level creds from global config, user-level creds from per-user config in DB; BYOK users get their own API key, shared-key users get rate-limited
- **Multi-tenancy**: all user-scoped DB tables have `user_id` column with composite unique indexes

## Config Overview
```yaml
# ~/.goviral/config.yaml — app-level (shared) credentials
x:
  client_id: ...       # OAuth app credentials
  client_secret: ...
linkedin:
  client_id: ...
  client_secret: ...
youtube:
  client_id: ...
  client_secret: ...
tiktok:
  client_key: ...
  client_secret: ...
claude:
  api_key: ...         # Shared API key (rate-limited per user)
  model: claude-sonnet-4-20250514
gemini:
  api_key: ...
  model: gemini-2.0-flash-exp
github:
  client_id: ...
  client_secret: ...
server:
  port: 8080
  allowed_origins: [...]
daemon:
  enabled: false
  schedules: { x: "0 9 * * *", linkedin: "0 10 * * *" }
  max_per_batch: 3
telegram:
  bot_token: ...
  chat_id: ...
niches: [...]            # Default niches for trending discovery
```

Per-user config is stored as JSON in SQLite `user_config` table, containing user OAuth tokens, BYOK API keys, personal niches, and platform usernames.

## Agent Team Roles

When using Agent Teams, the following ownership boundaries apply:

### Lead Agent
- Owns: `apps/cli/cmd/`, `apps/cli/main.go`, `internal/config/`, `internal/db/`, `internal/auth/`, `internal/thread/`, `pkg/models/`, `go.mod`, `go.sum`
- Responsibility: Project scaffolding, Cobra CLI wiring, config management, database layer, auth flows, shared models, final integration

### Teammate: platform-apis
- Owns: `internal/platform/x/`, `internal/platform/linkedin/`, `internal/platform/youtube/`, `internal/platform/tiktok/`, `internal/platform/github/`
- Responsibility: Platform API clients, rate limiting, fallback clients, Python bridge scripts
- Must use interfaces from `pkg/models/` for return types

### Teammate: ai-layer
- Owns: `internal/ai/claude/`, `internal/ai/gemini/`, `internal/ai/persona/`, `internal/ai/generator/`, `internal/ai/prompts/`
- Responsibility: Claude API client, Gemini API client, persona analysis, content generation/classification/competition, prompt templates
- Must use interfaces from `pkg/models/` for return types

### Teammate: automation
- Owns: `internal/daemon/`, `internal/telegram/`, `internal/ratelimit/`, `internal/codeimg/`
- Responsibility: Autopilot daemon, Telegram bot integration, rate limiting, code image rendering

### Teammate: server
- Owns: `apps/server/` (handler, service, middleware, dto, router)
- Responsibility: HTTP API, SSE endpoints, operation store, platform factory, request/response DTOs, embedded web assets

### Teammate: web-frontend
- Owns: `apps/web/`
- Responsibility: React dashboard, UI components, API client hooks, routing, state management

### Teammate: testing
- Owns: all `*_test.go` files, `testdata/` directory
- Responsibility: Unit tests, integration tests, test fixtures, mocks
- Waits for other teammates to finish before writing tests

## Dependencies

### Go
- github.com/spf13/cobra (CLI framework)
- github.com/charmbracelet/bubbletea (interactive TUI)
- github.com/charmbracelet/huh (form prompts)
- github.com/charmbracelet/lipgloss (terminal styling)
- github.com/go-chi/chi/v5 (HTTP router)
- github.com/google/uuid (UUIDs)
- modernc.org/sqlite (database, pure Go)
- gopkg.in/yaml.v3 (YAML parsing)
- golang.org/x/sync (concurrency primitives)
- github.com/robfig/cron/v3 (daemon scheduling)
- github.com/chromedp/chromedp (headless Chrome for code images)

### Web (apps/web/)
- React 19, React Router v7, React DOM
- Vite 7, TypeScript ~5.9
- Tailwind CSS v4
- TanStack Query v5 (data fetching)
- Zustand (state management)
- Lucide React (icons), Sonner (toasts)
