# Autopilot Daemon & Telegram Bot Setup

GoViral includes an autopilot daemon that runs the full content pipeline on a schedule: discover trending posts, generate drafts, send previews via Telegram, and post after approval. This guide covers setup and usage.

## How It Works

```
Cron tick
  -> Fetch trending posts for the platform
  -> Generate draft content using your persona
  -> Save as a batch in the database
  -> Send preview to Telegram with reply instructions
  -> Wait for your reply (approve / reject / edit / schedule)
  -> Claude parses your intent from natural language
  -> Execute: post immediately, schedule for later, or discard
```

Batches that go unanswered are automatically archived after a configurable timeout (default: 2 hours).

## 1. Create a Telegram Bot

1. Open Telegram and search for **@BotFather**.
2. Send `/newbot` and follow the prompts to name your bot.
3. BotFather gives you a **bot token** like `123456789:ABCdefGHI...`. Copy it.
4. Start a conversation with your new bot (search for it by username and press **Start**).
5. To find your **chat ID**, send any message to your bot, then open:
   ```
   https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates
   ```
   Look for `"chat":{"id":123456789}` in the response. That number is your chat ID.

## 2. Add Config

Edit `~/.goviral/config.yaml`:

```yaml
telegram:
  bot_token: "123456789:ABCdefGHIjklMNOpqrSTUvwxYZ"
  chat_id: 123456789
  # webhook_url: "https://your-server.com/api/v1/telegram/webhook/SECRET"  # optional

daemon:
  enabled: true
  schedules:
    x: "0 9 * * 1-5"         # weekdays at 9 AM
    linkedin: "0 10 * * 2,4"  # Tue/Thu at 10 AM
  max_per_batch: 3            # drafts per batch
  auto_skip_after: "2h"       # archive unanswered batches
  trending_limit: 10          # trending posts to consider
  min_likes: 10               # minimum engagement filter
  period: "week"              # trending period: day, week, month
```

### Schedule Syntax

Schedules use standard cron expressions (5 fields):

| Field | Values |
|-------|--------|
| Minute | 0-59 |
| Hour | 0-23 |
| Day of month | 1-31 |
| Month | 1-12 |
| Day of week | 0-6 (Sun=0) |

Examples:
- `0 9 * * *` — every day at 9:00 AM
- `0 9 * * 1-5` — weekdays at 9:00 AM
- `0 */6 * * *` — every 6 hours
- `30 8,18 * * *` — 8:30 AM and 6:30 PM daily

## 3. Telegram Receiving Mode

The daemon supports two ways to receive your replies:

### Long Polling (default)

When `webhook_url` is empty, the daemon polls Telegram every 30 seconds for new messages. This works anywhere, including localhost, with no extra setup.

### Webhook Mode

When `webhook_url` is set, the daemon registers a webhook with Telegram so replies arrive instantly. The URL must be HTTPS and publicly reachable. A secret path segment is appended automatically (derived from your bot token hash) to prevent unauthorized access.

Set `webhook_url` to your server's public URL:
```yaml
telegram:
  webhook_url: "https://your-server.com"
```

The actual webhook endpoint registered with Telegram will be:
```
https://your-server.com/api/v1/telegram/webhook/{secret}
```

## 4. Start the Server

The daemon runs inside the server process. Start the server normally:

```bash
go run apps/server/*.go
```

If `daemon.enabled` is `true` and a Telegram bot token is configured, the daemon starts automatically. You'll see:

```
INFO daemon started platforms=2
INFO starting telegram long-poll receiver
```

## 5. Replying to Telegram Notifications

When the daemon generates a batch, you receive a Telegram message with draft previews and instructions. Reply to that message with:

| Reply | Action |
|-------|--------|
| `approve` | Post all drafts immediately |
| `reject` | Discard the batch |
| `approve 1,3` | Post only drafts 1 and 3 |
| `schedule 2h` | Schedule all drafts for 2 hours from now |
| `schedule 30m` | Schedule for 30 minutes from now |
| Natural language | Claude interprets your intent |

Natural language examples that work:
- "looks good, post it"
- "post the first two but skip the third"
- "make draft 2 shorter and more punchy, then post all"
- "schedule for tomorrow morning"
- "nah, not feeling these"

The daemon uses fast regex matching for simple commands (approve, reject, schedule) and falls back to Claude for complex/ambiguous replies.

## 6. Web Dashboard

The Autopilot page in the web dashboard (`/{platform}/autopilot`) provides:

- **Status panel** — running/stopped indicator, per-platform next-run times, Start/Stop/Run Now buttons
- **Batches list** — filterable by status, with content previews and approve/reject buttons
- **Config panel** — edit schedules, batch size, and Telegram settings from the browser

You can also manage Telegram bot settings from **Settings** in the web dashboard.

### Running from the Web UI

Even without Telegram, you can:
1. Navigate to `/{platform}/autopilot`
2. Click **Run Now** to trigger a batch immediately
3. Review drafts in the batch list
4. Click **Approve** or **Reject** directly from the web

## 7. API Endpoints

All daemon endpoints require authentication (same API key as other endpoints).

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/daemon/status` | Daemon running state, per-platform info |
| `GET` | `/api/v1/daemon/batches` | List batches (query: `platform`, `status`, `limit`) |
| `GET` | `/api/v1/daemon/batches/{id}` | Single batch with content details |
| `POST` | `/api/v1/daemon/batches/{id}/action` | Approve/reject/edit a batch |
| `POST` | `/api/v1/daemon/run` | Trigger immediate pipeline run |
| `GET` | `/api/v1/daemon/config` | Current daemon + Telegram config |
| `PATCH` | `/api/v1/daemon/config` | Update daemon settings |
| `POST` | `/api/v1/daemon/start` | Start the daemon |
| `POST` | `/api/v1/daemon/stop` | Stop the daemon |

### Batch Action Request

```json
POST /api/v1/daemon/batches/42/action

{
  "action": "approve",
  "content_ids": [10, 12],
  "schedule_at": "2026-02-20T09:00:00Z"
}
```

Action values: `approve`, `reject`, `edit`, `schedule`.

### Run Now Request

```json
POST /api/v1/daemon/run

{
  "platform": "x"
}
```

## 8. Batch Lifecycle

```
pending -> notified -> awaiting_reply -> approved -> posted
                                      -> rejected
                                      -> scheduled
                    -> archived (auto-skip after timeout)
                    -> failed (on error)
```

## Prerequisites

The daemon reuses your existing GoViral setup. Before enabling it, make sure:

- Platform credentials are configured (X and/or LinkedIn)
- A persona has been built for each platform you want to automate (`POST /api/v1/persona/build`)
- Claude API key is set (used for content generation and intent parsing)
- Niche tags are configured for the platforms you want to automate
