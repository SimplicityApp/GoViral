# Laravel on Fly.io

## Quick Start

```bash
cd /path/to/laravel-app
fly launch
```

Fly auto-detects Laravel and:
- Generates optimized Dockerfile with PHP/Nginx
- Sets default secrets (APP_KEY, etc.)
- Creates `.fly/` directory with server configs
- Offers Postgres, Redis, Tigris extensions

## Generated Files

| File | Purpose |
|------|---------|
| `fly.toml` | App configuration |
| `Dockerfile` | Container build |
| `.fly/` | Nginx/PHP configs |
| `.dockerignore` | Build exclusions |
| `.github/workflows/fly-deploy.yml` | CI/CD |

## Environment Setup

### Required Secrets

```bash
fly secrets set APP_KEY="base64:..."
fly secrets set DB_PASSWORD="..."
```

### fly.toml Environment

```toml
[env]
APP_ENV = "production"
APP_URL = "https://my-app.fly.dev"
LOG_CHANNEL = "stderr"
LOG_LEVEL = "error"
SESSION_DRIVER = "cookie"
```

## Database Options

### MySQL (Managed)

```bash
fly mysql create
# Sets DATABASE_URL automatically
```

### PostgreSQL

```bash
fly mpg create                    # Managed (recommended)
fly postgres create               # Unmanaged
fly postgres attach my-postgres   # Attach to app
```

Update Laravel config:
```toml
[env]
DB_CONNECTION = "pgsql"
```

### SQLite with Volume

```bash
fly volumes create sqlite_data --size 1
```

```toml
[mounts]
source = "sqlite_data"
destination = "/var/www/html/storage/database"

[env]
DB_CONNECTION = "sqlite"
DB_DATABASE = "/var/www/html/storage/database/database.sqlite"
```

## Redis (Cache/Queue)

```bash
fly redis create
# Sets REDIS_URL automatically
```

```toml
[env]
CACHE_DRIVER = "redis"
QUEUE_CONNECTION = "redis"
SESSION_DRIVER = "redis"
```

## Storage (Tigris CDN)

```bash
fly storage create
# Sets AWS_* credentials automatically
```

```toml
[env]
FILESYSTEM_DISK = "s3"
```

## Cron & Queues

### Scheduler (cron)

Add to fly.toml:
```toml
[processes]
app = ""
scheduler = "php artisan schedule:work"
```

### Queue Worker

```toml
[processes]
app = ""
worker = "php artisan queue:work --tries=3"
```

Scale separately:
```bash
fly scale count app=2 worker=1 scheduler=1
```

## Persistent Storage

Mount volume for storage folder:

```bash
fly volumes create storage_data --size 1
```

```toml
[mounts]
source = "storage_data"
destination = "/var/www/html/storage/app"
```

## Deploy Commands

Run migrations on deploy:
```toml
[deploy]
release_command = "php artisan migrate --force"
```

## PHP/Node Versions

Customize in Dockerfile or via scanner args:
```bash
fly launch -- --php 8.3 --node 20
```

## Common Issues

| Issue | Solution |
|-------|----------|
| 500 errors | Check `fly logs`, ensure APP_KEY set |
| Session lost | Use cookie/redis session driver |
| Storage permission | Mount volume, check permissions |
| Slow cold start | Set `min_machines_running = 1` |

## Useful Commands

```bash
fly ssh console -C "php artisan tinker"
fly ssh console -C "php artisan migrate:status"
fly ssh console -C "php artisan config:clear"
```
