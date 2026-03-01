# GoViral — Project Instructions

## Overview
GoViral is a monorepo for viral content creation on X and LinkedIn. It includes:
- **CLI** (`apps/cli/`) — Interactive terminal tool with Cobra + bubbletea/huh
- **Server** (`apps/server/`) — HTTP API with chi router, SSE for long-running ops
- **Web Dashboard** (`apps/web/`) — React 19 + TypeScript + Vite + Tailwind SPA

## Project Structure
```
apps/
  cli/           # Cobra CLI (Phase 1)
  server/        # HTTP API server (Phase 2)
    handler/     # 18 route handlers
    service/     # 8 business-logic services
    middleware/   # auth, cors, errors, logging
    dto/         # request/response types
    router/      # chi route registration
    embed.go     # -tags embedweb: serve web assets
  web/           # React dashboard (Phase 2)
internal/
  ai/claude/     # Claude API client
  ai/gemini/     # Gemini API client
  ai/persona/    # Persona analysis logic
  ai/generator/  # Content generation prompts
  auth/          # OAuth flows (X and LinkedIn)
  config/        # Config management (~/.goviral/config.yaml)
  db/            # SQLite via modernc.org/sqlite
  platform/x/    # X API + twikit fallback
  platform/linkedin/  # LinkedIn API + linkitin fallback
  thread/        # Thread splitting logic
  trending/      # Trending topic discovery
pkg/
  models/        # Shared data models and interfaces
```

## Conventions
- Go module path: `github.com/shuhao/goviral`
- Use `modernc.org/sqlite` (pure Go, no CGO)
- All shared data models in `pkg/models/`
- All business logic in `internal/`
- CLI-specific code only in `apps/cli/`
- Error handling: wrap errors with context using `fmt.Errorf("doing X: %w", err)`
- Use `lipgloss` for terminal output styling
- Config stored at `~/.goviral/config.yaml`
- Server uses `go-chi/chi/v5` with middleware stack (auth, CORS, error recovery, logging)
- **Fallback client pattern**: official API client → cookie-based fallback (twikit for X, linkitin for LinkedIn)
- Python bridges (`twikit_guest.py`, `linkitin_bridge.py`) are embedded via `//go:embed` into Go binaries
- Python venv at `~/.goviral/venv/`
- Server embeds web assets with `-tags embedweb` build tag (copy `apps/web/dist/` to `apps/server/static/` first)
- Web dev server proxies `/api` to Go server at `:8080`
- Long-running operations (fetch posts, build persona, discover trending) use SSE for real-time updates

## Agent Team Roles

When using Agent Teams, the following ownership boundaries apply:

### Lead Agent
- Owns: `apps/cli/cmd/`, `apps/cli/main.go`, `internal/config/`, `internal/db/`, `internal/auth/`, `internal/thread/`, `pkg/models/`, `go.mod`, `go.sum`
- Responsibility: Project scaffolding, Cobra CLI wiring, config management, database layer, auth flows, shared models, final integration

### Teammate: platform-apis
- Owns: `internal/platform/x/`, `internal/platform/linkedin/`
- Responsibility: X API v2 client, LinkedIn API client, all HTTP calls, rate limiting, fallback clients (twikit/linkitin), Python bridge scripts
- Must use interfaces from `pkg/models/` for return types

### Teammate: ai-layer
- Owns: `internal/ai/claude/`, `internal/ai/gemini/`, `internal/ai/persona/`, `internal/ai/generator/`
- Responsibility: Claude API client, Gemini API client, persona analysis logic, content generation prompts and parsing
- Must use interfaces from `pkg/models/` for return types

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
- modernc.org/sqlite (database)
- gopkg.in/yaml.v3 (YAML parsing)
- golang.org/x/sync (concurrency primitives)

### Web (apps/web/)
- React 19, React Router v7, React DOM
- Vite 7, TypeScript ~5.9
- Tailwind CSS v4
- TanStack Query v5 (data fetching)
- Zustand (state management)
- Lucide React (icons), Sonner (toasts)
