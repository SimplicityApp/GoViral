---
name: x-post
description: Interact with X/Twitter — fetch user tweets, search trending posts, create tweets, quote tweets, schedule tweets, and upload media. Use when the user wants to read or post content on X/Twitter.
allowed-tools: Bash
---

# X/Twitter Operations

Interact with X/Twitter via the twikit cookie-based client. All commands output JSON to stdout.

## Configuration

- **Python**: `~/.goviral/venv/bin/python3`
- **Script**: `internal/platform/x/scripts/twikit_guest.py`
- **Cookies**: `~/.goviral/twikit_cookies.json`

If cookies don't exist, tell the user to run `/x-setup` first.

## Available Operations

### Fetch User Tweets

Retrieve recent tweets from a specific user.

```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py fetch_user_tweets <username> <limit>
```

**Example:**
```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py fetch_user_tweets elonmusk 20
```

**Response:**
```json
{
  "tweets": [
    {
      "id": "1234567890",
      "text": "Tweet content...",
      "created_at": "2024-01-15T10:30:00Z",
      "likes": 1500,
      "retweets": 300,
      "replies": 50,
      "impressions": 50000,
      "media": [{"type": "photo", "url": "...", "preview_url": "...", "alt_text": ""}]
    }
  ]
}
```

### Search Trending Posts

Find trending/top tweets matching specific niches.

```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py search_trending '<niches_json>' <min_likes> <limit> [period]
```

- `niches_json`: JSON array of topic strings, e.g. `'["AI", "startups"]'`
- `min_likes`: minimum like count filter
- `limit`: max number of results
- `period`: `day` (default), `week`, or `month`

**Example:**
```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py search_trending '["AI", "machine learning"]' 100 10 week
```

**Response:**
```json
{
  "trending": [
    {
      "id": "1234567890",
      "text": "Trending tweet content...",
      "created_at": "...",
      "author_username": "user123",
      "author_name": "User Name",
      "likes": 5000,
      "retweets": 800,
      "replies": 200,
      "impressions": 100000,
      "niche": "AI",
      "media": []
    }
  ]
}
```

### Upload Media

Upload an image or video file. Returns a media_id for use with create_tweet.

```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py upload_media <file_path>
```

**Response:**
```json
{"media_id": "1234567890123456789"}
```

### Create Tweet

Post a new tweet. Optionally reply to another tweet or attach media.

```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py create_tweet '<text>' [reply_to_id] ['<media_ids_json>']
```

- `text`: the tweet text (required)
- `reply_to_id`: tweet ID to reply to (optional, pass empty string `""` to skip)
- `media_ids_json`: JSON array of media IDs from upload_media (optional)

**Example — simple tweet:**
```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py create_tweet 'Hello world!'
```

**Example — tweet with media:**
```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py create_tweet 'Check this out!' "" '["1234567890123456789"]'
```

**Response:**
```json
{"tweet_id": "1234567890"}
```

### Create Quote Tweet

Quote retweet an existing tweet with your commentary.

```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py create_quote_tweet '<text>' <quote_tweet_id>
```

**Response:**
```json
{"tweet_id": "1234567890"}
```

### Schedule Tweet

Schedule a tweet for future posting. Uses X's native scheduling.

```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py schedule_tweet '<text>' <scheduled_at_unix> ['<media_ids_json>']
```

- `scheduled_at_unix`: Unix timestamp for when to post
- `media_ids_json`: optional JSON array of media IDs

**Example:**
```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py schedule_tweet 'Scheduled post!' 1708300800
```

**Response:**
```json
{"scheduled_tweet_id": "1234567890"}
```

## Safety Rules

**CRITICAL: NEVER create, quote, or schedule a tweet without explicit user confirmation.**

Before any write operation (create_tweet, create_quote_tweet, schedule_tweet):
1. Show the user the exact content that will be posted
2. If scheduling, show the human-readable date/time
3. If attaching media, confirm which files
4. Wait for explicit "yes" / "go ahead" / confirmation before executing

## Error Handling

All errors return `{"error": "description"}`. Common errors:
- **"not logged in"**: Cookies missing or expired. Tell the user to run `/x-setup`
- **Network/rate limit errors**: Wait and retry once. If it persists, inform the user
- **"unknown command"**: Check the command name spelling
