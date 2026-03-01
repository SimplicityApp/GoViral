# GoViral CLI

Command-line tool for fetching posts, building persona profiles, discovering trends, and generating viral content.

## Build

```sh
go build -o goviral ./apps/cli/
```

## Setup

```sh
goviral init
```

Walks you through configuring API keys and creates `~/.goviral/config.yaml` + the SQLite database.

## Authentication

```sh
goviral auth x            # OAuth 2.0 with PKCE for X
goviral auth linkedin     # OAuth 2.0 for LinkedIn
goviral auth              # Interactive platform picker
goviral twikit-login      # Extract X cookies from Chrome (fallback)
goviral linkitin-login       # Extract LinkedIn cookies from Chrome (fallback)
```

The `auth` commands accept `--port` (default `8080`) for the local OAuth callback server.

## Command Reference

| Command | Description | Key Flags |
|---------|-------------|-----------|
| `init` | Interactive setup wizard | |
| `auth [x\|linkedin]` | OAuth authentication | `--port` |
| `twikit-login` | Extract X cookies from Chrome | |
| `linkitin-login` | Extract LinkedIn cookies from Chrome | |
| `fetch` | Fetch your posts from X/LinkedIn | `-p, --platform` (x, linkedin, all), `-l, --limit` (default 50) |
| `posts` | View fetched posts | `-p, --platform` (x, linkedin, all), `-l, --limit` (default 20) |
| `trending` | Discover trending posts in your niches | `-p, --platform`, `--period` (day, week, month), `--min-likes` (default 100), `-l, --limit`, `--per-niche` (default 5) |
| `generate` | Generate viral content from trending posts | `-p, --platform` (default x), `--auto`, `-c, --count` (default 3), `--max-chars`, `--images` |
| `post` | Post or schedule generated content to X | `--id`, `--numbered` (default true), `--at` (schedule time), `--scheduled`, `--run-scheduled`, `--dry-run` |
| `profile build` | Build persona profile from your posts | |
| `profile show` | Display current persona profile | |
| `profile refresh` | Rebuild persona with latest posts | |
| `history` | View past generated content | `--status` (draft, approved, posted), `--id` |

## Typical Workflow

```sh
goviral fetch --platform x          # 1. Pull your posts
goviral profile build               # 2. Analyze your writing style
goviral trending --platform x       # 3. Find trending content
goviral generate --platform x       # 4. AI rewrites in your voice
goviral post --id 5 --dry-run       # 5. Preview thread splitting
goviral post --id 5                 # 6. Publish to X
```
