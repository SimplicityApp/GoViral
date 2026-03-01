---
name: linkedin-post
description: Interact with LinkedIn — fetch your posts, browse feed, create posts (including scheduled posts), repost content, delete posts, find trending content, search posts, and upload images. Use when the user wants to read or post content on LinkedIn.
allowed-tools: Bash
---

# LinkedIn Operations

Interact with LinkedIn via the linkitin cookie-based bridge. Commands are sent as JSON via stdin and responses come as JSON on stdout.

## Configuration

- **Python**: `~/.goviral/venv/bin/python3`
- **Script**: `internal/platform/linkedin/scripts/linkitin_bridge.py`
- **Invocation pattern**: `echo '<json_command>' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py`

If operations fail with auth errors, tell the user to run `/linkedin-setup`.

## Available Operations

### Get My Posts

Fetch the authenticated user's recent LinkedIn posts.

```bash
echo '{"action": "get_my_posts", "limit": 20}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

**Response:**
```json
{
  "posts": [
    {
      "urn": "urn:li:activity:1234567890",
      "text": "Post content...",
      "likes": 150,
      "comments": 30,
      "reposts": 10,
      "impressions": 5000,
      "created_at": "2024-01-15T10:30:00Z",
      "author": {
        "urn": "urn:li:member:12345",
        "first_name": "John",
        "last_name": "Doe",
        "headline": "Software Engineer"
      }
    }
  ]
}
```

### Get Feed

Fetch the user's LinkedIn feed.

```bash
echo '{"action": "get_feed", "limit": 20}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

Response format is the same as Get My Posts.

### Get Trending Posts

Find trending posts for a specific topic.

```bash
echo '{"action": "get_trending_posts", "topic": "<topic>", "period": "past-24h", "limit": 10, "from_followed": true, "scrolls": 3}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

- `topic`: keyword/topic to search for (e.g., "AI", "startups")
- `period`: `past-24h` (default), `past-week`, `past-month`
- `limit`: max results (default: 10)
- `from_followed`: prioritize followed connections (default: true)
- `scrolls`: how many feed pages to scan (default: 3, increase for more results)

**Example:**
```bash
echo '{"action": "get_trending_posts", "topic": "artificial intelligence", "period": "past-week", "limit": 15}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

### Search Posts

Search LinkedIn posts by keywords.

```bash
echo '{"action": "search_posts", "keywords": "<search_terms>", "limit": 20}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

**Example:**
```bash
echo '{"action": "search_posts", "keywords": "developer productivity tools", "limit": 10}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

### Create Post

Publish a new LinkedIn post.

```bash
echo '{"action": "create_post", "text": "<post_text>", "visibility": "PUBLIC"}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

- `visibility`: `PUBLIC` (default) or `CONNECTIONS`

**Response:**
```json
{"urn": "urn:li:activity:1234567890"}
```

### Upload Image

Upload an image for use in a post. Image data must be base64-encoded.

```bash
echo '{"action": "upload_image", "image_data": "<base64_encoded_data>", "filename": "image.png"}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

**Response:**
```json
{"media_urn": "urn:li:digitalmediaAsset:D1234567890"}
```

### Create Post with Image

Create a post with an attached image in a single operation.

```bash
echo '{"action": "create_post_with_image", "text": "<post_text>", "image_data": "<base64_encoded_data>", "filename": "image.png", "visibility": "PUBLIC"}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

**Response:**
```json
{"urn": "urn:li:activity:1234567890"}
```

### Repost

Repost (share) an existing LinkedIn post from your feed or your own posts.

> **Important**: `share_urn` is only available from `get_feed()` and `get_my_posts()`. Do NOT use posts from search results or trending posts — they lack the `share_urn` field needed for reposts.

```bash
echo '{"action": "repost", "share_urn": "urn:li:share:1234567890", "text": "<optional_comment>", "visibility": "PUBLIC"}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

- `share_urn`: The `urn` field from a post obtained via `get_feed` or `get_my_posts`
- `text`: Optional comment to add when reposting (can be empty string)
- `visibility`: `PUBLIC` (default) or `CONNECTIONS`

**Response:**
```json
{"urn": "urn:li:activity:1234567890"}
```

### Create Scheduled Post

Schedule a post to be published at a future date and time.

```bash
echo '{"action": "create_scheduled_post", "text": "<post_text>", "scheduled_at": "2026-03-01T10:00:00+00:00", "visibility": "PUBLIC"}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

- `text`: The post content
- `scheduled_at`: ISO 8601 formatted timestamp (e.g., `2026-03-01T10:00:00+00:00`)
- `visibility`: `PUBLIC` (default) or `CONNECTIONS`

**Response:**
```json
{"urn": "urn:li:activity:1234567890"}
```

### Create Scheduled Post with Image

Schedule a post with an image to be published at a future date and time.

```bash
echo '{"action": "create_scheduled_post_with_image", "text": "<post_text>", "image_data": "<base64_encoded_data>", "filename": "image.png", "scheduled_at": "2026-03-01T10:00:00+00:00", "visibility": "PUBLIC"}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

- `text`: The post content
- `image_data`: Base64-encoded image data
- `filename`: Name of the image file (e.g., `image.png`)
- `scheduled_at`: ISO 8601 formatted timestamp
- `visibility`: `PUBLIC` (default) or `CONNECTIONS`

**Response:**
```json
{"urn": "urn:li:activity:1234567890"}
```

### Delete Post

Delete an existing LinkedIn post.

```bash
echo '{"action": "delete_post", "post_urn": "urn:li:activity:1234567890"}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

- `post_urn`: The URN of the post to delete (from `get_my_posts()`)

**Response:**
```json
{"status": "ok"}
```

## Safety Rules

**CRITICAL: NEVER perform any write operation without explicit user confirmation.**

Before any write operation (`create_post`, `create_post_with_image`, `create_scheduled_post`, `create_scheduled_post_with_image`, `repost`, `delete_post`):
1. Show the user the exact text that will be posted (or reposted)
2. Show the visibility setting (PUBLIC vs CONNECTIONS)
3. For scheduled posts, confirm the scheduled publication date/time
4. If attaching an image, confirm which file
5. For reposts, show the original post being reposted
6. For delete operations, show which post will be permanently removed
7. Wait for explicit "yes" / "go ahead" / confirmation before executing

## Error Handling

All errors return `{"error": "description"}`. Common patterns:
- **Auth errors / empty responses**: Cookies expired. Tell the user to run `/linkedin-setup`
- **"requires text"**: Missing required field in the command
- **Network errors**: Wait and retry once. If it persists, inform the user
