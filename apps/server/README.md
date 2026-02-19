# GoViral Server

HTTP API server that powers the web dashboard. Built with [chi](https://github.com/go-chi/chi).

## Run

```sh
go run ./apps/server/
```

Listens on `:8080` by default.

## Configuration

Add a `server` block to `~/.goviral/config.yaml`:

```yaml
server:
  port: 8080
  api_key: "your-secret-key"
  allowed_origins:
    - "http://localhost:5173"
```

## Authentication

All `/api/v1` endpoints require a Bearer token matching `server.api_key`:

```
Authorization: Bearer your-secret-key
```

## API Endpoints

All routes are under `/api/v1`.

### Read Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/posts` | List fetched posts |
| GET | `/trending` | List trending posts |
| GET | `/trending/{id}` | Get trending post by ID |
| GET | `/persona` | Get persona profile |
| GET | `/history` | List generated content |
| GET | `/history/{id}` | Get generated content by ID |
| GET | `/schedule` | List scheduled posts |
| GET | `/config` | Get current config |
| GET | `/operations/{id}` | Poll long-running operation status |

### Write Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/publish` | Publish content to a platform |
| POST | `/schedule` | Schedule a post |
| DELETE | `/schedule/{id}` | Cancel a scheduled post |
| POST | `/schedule/run` | Run due scheduled posts |
| PATCH | `/history/{id}` | Update content status |
| PATCH | `/config` | Update config fields |

### Long-Running Operations (SSE or 202)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/posts/fetch` | Fetch posts from platforms |
| POST | `/trending/discover` | Discover trending posts |
| POST | `/generate` | Generate content |
| POST | `/persona/build` | Build persona profile |

These endpoints support two modes:
- **SSE**: Send `Accept: text/event-stream` header to receive real-time progress events
- **Polling**: Without the header, returns `202 Accepted` with an operation ID; poll `GET /operations/{id}` for status

### OAuth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/{platform}/start` | Start OAuth flow (x or linkedin) |
| GET | `/auth/{platform}/status` | Check OAuth status |

## Production Mode

Embed the web dashboard into the server binary:

```sh
cd apps/web && npm run build
cp -r apps/web/dist/ apps/server/static/
go build -tags embedweb -o goviral-server ./apps/server/
./goviral-server
```

The server serves the SPA at `/` with fallback routing for client-side navigation.

## Repost / Quote Tweet Support

The generate and publish endpoints support creating quote tweets (reposts with commentary).

**Generate** — pass `is_repost: true` in the request body. When set, the AI generates short commentary (defaults to 200 chars max) and the saved content is tagged with `is_repost` and `quote_tweet_id` (the original trending post's platform ID).

```sh
curl -H "Authorization: Bearer your-secret-key" \
     -H "Content-Type: application/json" \
     -X POST http://localhost:8080/api/v1/generate \
     -d '{"trending_post_ids": [42], "target_platform": "x", "is_repost": true}'
```

**Publish** — when publishing repost content, the server automatically calls `PostQuoteTweet` instead of thread posting. No extra flags needed; the server detects repost content by its `is_repost` and `quote_tweet_id` fields.

**History** — generated content responses include `is_repost` (bool) and `quote_tweet_id` (string, omitted when empty).

Quote tweet posting uses twikit's direct GraphQL endpoint (`CreateTweet` with `attachment_url`) since twikit's `create_tweet()` method does not accept a `quote_tweet_id` parameter.

## Examples

```sh
# Health check
curl -H "Authorization: Bearer your-secret-key" http://localhost:8080/api/v1/health

# List posts
curl -H "Authorization: Bearer your-secret-key" http://localhost:8080/api/v1/posts

# Fetch posts (SSE)
curl -H "Authorization: Bearer your-secret-key" \
     -H "Accept: text/event-stream" \
     -X POST http://localhost:8080/api/v1/posts/fetch

# Fetch posts (polling)
curl -H "Authorization: Bearer your-secret-key" \
     -X POST http://localhost:8080/api/v1/posts/fetch
# Returns: {"operation_id": "abc-123"}

curl -H "Authorization: Bearer your-secret-key" \
     http://localhost:8080/api/v1/operations/abc-123

# Generate content
curl -H "Authorization: Bearer your-secret-key" \
     -H "Content-Type: application/json" \
     -X POST http://localhost:8080/api/v1/generate \
     -d '{"platform": "x", "count": 3}'
```
