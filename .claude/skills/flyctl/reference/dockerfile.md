# Dockerfile Deployment on Fly.io

## Quick Start

```bash
cd /path/to/project-with-dockerfile
fly launch
```

Fly detects Dockerfile and:
- Reads `EXPOSE` for internal port (default: 8080)
- Builds image remotely
- Deploys to nearest region

## Launch Options

```bash
fly launch                    # Interactive
fly launch --no-deploy        # Configure only
fly launch --now -y           # Accept defaults, deploy
fly launch --internal-port 3000  # Override port
```

## Configuration

### fly.toml Basics

```toml
app = "my-app"
primary_region = "ord"

[build]
dockerfile = "Dockerfile"      # Custom path (optional)

[env]
NODE_ENV = "production"
PORT = "8080"

[http_service]
internal_port = 8080          # Must match EXPOSE/app port
force_https = true
auto_stop_machines = "stop"
auto_start_machines = true
min_machines_running = 0

[[http_service.checks]]
interval = "10s"
timeout = "2s"
grace_period = "5s"
path = "/health"
```

### Environment Variables

In fly.toml (non-sensitive):
```toml
[env]
DATABASE_HOST = "my-db.internal"
```

Via CLI (sensitive):
```bash
fly secrets set API_KEY="secret123"
fly secrets set DB_PASS="password" REDIS_URL="redis://..."
```

### Build Arguments

```bash
fly deploy --build-arg VERSION=1.2.3
```

Or in fly.toml:
```toml
[build.args]
VERSION = "1.2.3"
```

### Build Secrets

For secrets needed at build time:
```bash
fly deploy --build-secret NPM_TOKEN=xxx
```

In Dockerfile:
```dockerfile
RUN --mount=type=secret,id=NPM_TOKEN \
    NPM_TOKEN=$(cat /run/secrets/NPM_TOKEN) npm install
```

## Deploy Options

```bash
fly deploy                       # Standard (remote build)
fly deploy --local-only          # Build locally
fly deploy --no-cache            # Fresh build
fly deploy --strategy bluegreen  # Zero-downtime
fly deploy --detach              # Don't wait for completion
```

### Strategies

| Strategy | Description |
|----------|-------------|
| `rolling` | Replace instances gradually (default) |
| `bluegreen` | Start new, switch traffic, stop old |
| `canary` | Deploy to subset first |
| `immediate` | Replace all at once |

## Multi-Stage Builds

```dockerfile
FROM node:20 AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:20-slim
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
EXPOSE 8080
CMD ["node", "dist/index.js"]
```

## Health Checks

```toml
[[http_service.checks]]
interval = "15s"
timeout = "5s"
grace_period = "10s"
path = "/health"
method = "GET"
```

Or TCP check:
```toml
[[services.tcp_checks]]
interval = "10s"
timeout = "2s"
```

## Persistent Storage

```bash
fly volumes create appdata --size 5 --region ord
```

```toml
[mounts]
source = "appdata"
destination = "/app/data"
```

## Process Groups

Run multiple processes from one image:

```toml
[processes]
app = "node dist/server.js"
worker = "node dist/worker.js"

[[http_service]]
processes = ["app"]
internal_port = 8080
```

Scale separately:
```bash
fly scale count app=3 worker=2
```

## Private Networking

Apps communicate via `.internal` hostnames:
```
my-app.internal:8080
my-db.internal:5432
```

Allocate private IP:
```bash
fly ips allocate-v6 --private
```

## Common Dockerfile Patterns

### Node.js
```dockerfile
FROM node:20-slim
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
EXPOSE 8080
CMD ["node", "index.js"]
```

### Python
```dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8080
CMD ["gunicorn", "-b", "0.0.0.0:8080", "app:app"]
```

### Go
```dockerfile
FROM golang:1.22 AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server

FROM alpine:3.19
COPY --from=builder /app/server /server
EXPOSE 8080
CMD ["/server"]
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Port mismatch | Ensure `internal_port` matches `EXPOSE` |
| Build fails | Try `--local-only`, check Dockerfile |
| Health check fails | Verify endpoint, increase grace_period |
| OOM killed | Increase memory: `fly scale memory 512` |
