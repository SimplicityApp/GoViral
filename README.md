# GoViral

CLI + web dashboard for creating viral content on X and LinkedIn. Analyzes your writing style, discovers trending posts in your niches, and uses AI to generate content that matches your voice.

## Monorepo Structure

```
apps/cli/       Go CLI tool (Phase 1)
apps/server/    Go HTTP API server (Phase 2)
apps/web/       React web dashboard (Phase 2)
```

## Prerequisites

- Go 1.24+
- Node 22+ (for web dashboard)
- API keys: Claude (required), X and/or LinkedIn credentials

## Quick Start

### 1. Configure

Run the interactive setup wizard:

```sh
go run ./apps/cli/ init
```

This creates `~/.goviral/config.yaml` with your API keys and `~/.goviral/goviral.db`.

Alternatively, create `~/.goviral/config.yaml` manually:

```yaml
claude:
  api_key: "sk-ant-..."
  model: "claude-sonnet-4-20250514"    # optional, this is the default
x:
  bearer_token: "..."
  username: "yourhandle"
  client_id: "..."          # for OAuth posting
  client_secret: "..."      # for OAuth posting
linkedin:
  client_id: "..."
  client_secret: "..."
  access_token: "..."
gemini:
  api_key: "..."            # optional, for image generation
niches:
  - "AI"
  - "startups"
server:
  port: 8080
  api_key: "your-secret"
  allowed_origins:
    - "http://localhost:5173"
```

### 2. Run the CLI

```sh
go build -o goviral ./apps/cli/
./goviral fetch --platform x
./goviral profile build
./goviral trending --platform x
./goviral generate --platform x
```

### 3. Run the Web Dashboard

Development mode (two terminals):

```sh
# Terminal 1: API server
go run ./apps/server/

# Terminal 2: Web dev server (proxies /api to :8080)
cd apps/web && npm install && npm run dev
```

Production mode (single binary with embedded web UI):

```sh
cd apps/web && npm run build
cp -r apps/web/dist/ apps/server/static/
go build -tags embedweb -o goviral-server ./apps/server/
./goviral-server
```

## App READMEs

- [CLI](apps/cli/README.md)
- [Server](apps/server/README.md)
- [Web Dashboard](apps/web/README.md)
