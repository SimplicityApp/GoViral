# Server — `apps/server/`

## Architecture
- **main.go**: Bootstrap (config, db, daemon, routes), scheduled post execution (every 1 min), graceful shutdown
- **server.go**: `Server` struct with `Cfg`, `DB`, `Router`, lifecycle methods (`Start`, `Shutdown`)
- **Layering**: handler -> service -> internal packages. Handlers decode requests and call services; services contain business logic.

## Handler Patterns

**Dependency injection** — all handlers receive dependencies via constructor:
```go
h := NewGenerateHandler(genService, opStore, db, cfg)
```

**Request decoding**:
```go
var req dto.GenerateRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil { ... }
```

**User ID from context** (set by middleware):
```go
userID := middleware.UserIDFromContext(r.Context())
reqID := middleware.RequestIDFromContext(r.Context())
```

**Response writing**:
```go
middleware.WriteJSON(w, http.StatusOK, data)
middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid request", reqID)
```

## SSE Pattern
Long-running operations support two modes based on `Accept` header:

**SSE mode** (`Accept: text/event-stream`):
```go
if handler.WantsSSE(r) {
    progress := make(chan dto.ProgressEvent)
    go h.doWork(ctx, progress)
    handler.StreamProgress(w, r, progress)  // blocks until channel closes
    return
}
```

**Polling mode** (202 Accepted + operation ID):
```go
opID := h.store.Create()
go func() {
    // do work, drain progress, then:
    h.store.Complete(opID, result)  // or h.store.Fail(opID, errMsg)
}()
middleware.WriteJSON(w, http.StatusAccepted, dto.OperationResponse{ID: opID, Status: "running"})
```

**ProgressEvent types**: `"progress"`, `"complete"`, `"error"`, `"warning"`

**SSE utilities** (`handler/sse.go`): `WantsSSE()`, `SSEWriter`, `StreamProgress()` with 15s heartbeat

**Operation store** (`service/operation_store.go`): In-memory map with TTL cleanup (30 min default), IDs prefixed `op_`

## Middleware Stack (order matters)
1. **Recovery** — panic handler, returns 500 JSON
2. **Logging** — request logging with generated `request_id` (UUID)
3. **CORS** — wildcard subdomain support (*.vercel.app)
4. **UserID** — validates `X-User-ID` header (UUID v4), upserts user in DB. Skips `/health` and `/oauth/*`

## DTO Conventions
- `dto/requests.go` — all request structs
- `dto/responses.go` — all response structs
- `dto/errors.go` — error code constants: `VALIDATION_ERROR`, `NOT_FOUND`, `UNAUTHORIZED`, `INTERNAL_ERROR`, `PLATFORM_ERROR`, `CONFLICT`

## Embed Pattern
- Build tag `-tags embedweb` enables `embed.go`: embeds `static/` directory, serves SPA with index.html fallback (skips `/api/*`)
- Dev mode (`!embedweb`): `embed_dev.go` returns 404 with hint to build with embedweb tag
- Extension embed: `extension_embed.go` serves Chrome extension files from `extension/` directory

## Platform Factory
`service/platform_factory.go` — package-level factory variables for testability:
```go
var newXPoster = func(cfg config.XConfig, cookiePath string) models.PlatformPoster { ... }
var newLinkedInPoster = func(cfg config.LinkedInConfig, configDir string) models.LinkedInPoster { ... }
var newYouTubePoster = func(cfg config.YouTubeConfig) models.YouTubePoster { ... }
var newTikTokPoster = func(cfg config.TikTokConfig) models.TikTokPoster { ... }
```
Per-user isolation via `cookiePath`/`configDir` parameters written by `handler/cookie_temp.go`.

## Key Files
| File | Purpose |
|------|---------|
| `main.go` | Bootstrap, route registration, scheduled post runner |
| `server.go` | Server struct and lifecycle |
| `handler/sse.go` | SSE streaming utilities |
| `handler/fetch_posts.go` | Fetch user posts (SSE/202) |
| `handler/generate.go` | Generate content variations (SSE/202) |
| `handler/discover_trending.go` | Discover trending posts (SSE/202) |
| `handler/build_persona.go` | Build persona (SSE/202) |
| `handler/publish.go` | Publish to X/LinkedIn/YouTube/TikTok |
| `handler/config_handler.go` | Read merged config (secrets masked) |
| `handler/config_write.go` | Update per-user config |
| `handler/daemon.go` | Daemon control and batch management |
| `handler/repo.go` | GitHub repo operations |
| `handler/comment.go` | Comment generation and posting |
| `service/operation_store.go` | In-memory operation tracking with TTL |
| `service/generate_service.go` | Content generation with rate limiting |
| `service/platform_factory.go` | Testable factory variables |
| `service/publish_service.go` | Cross-platform publishing |
| `middleware/user.go` | X-User-ID validation and user upsert |
| `middleware/errors.go` | WriteError, WriteJSON helpers |
