---
name: linkedin-post
description: Interact with LinkedIn — fetch your posts, browse feed, create posts, find trending content, search posts, and upload images. Use when the user wants to read or post content on LinkedIn.
allowed-tools: Bash
---

# LinkedIn Operations

Interact with LinkedIn via the likit cookie-based bridge. Commands are sent as JSON via stdin and responses come as JSON on stdout.

## Configuration

- **Python**: `~/.goviral/venv/bin/python3`
- **Script**: `internal/platform/linkedin/scripts/likit_bridge.py`
- **Invocation pattern**: `echo '<json_command>' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/likit_bridge.py`

If operations fail with auth errors, tell the user to run `/linkedin-setup`.

## Available Operations

### Get My Posts

Fetch the authenticated user's recent LinkedIn posts.

```bash
echo '{"action": "get_my_posts", "limit": 20}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/likit_bridge.py
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
echo '{"action": "get_feed", "limit": 20}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/likit_bridge.py
```

Response format is the same as Get My Posts.

### Get Trending Posts

Find trending posts for a specific topic.

```bash
echo '{"action": "get_trending_posts", "topic": "<topic>", "period": "past-24h", "limit": 10, "from_followed": true, "scrolls": 3}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/likit_bridge.py
```

- `topic`: keyword/topic to search for (e.g., "AI", "startups")
- `period`: `past-24h` (default), `past-week`, `past-month`
- `limit`: max results (default: 10)
- `from_followed`: prioritize followed connections (default: true)
- `scrolls`: how many feed pages to scan (default: 3, increase for more results)

**Example:**
```bash
echo '{"action": "get_trending_posts", "topic": "artificial intelligence", "period": "past-week", "limit": 15}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/likit_bridge.py
```

### Search Posts

Search LinkedIn posts by keywords.

```bash
echo '{"action": "search_posts", "keywords": "<search_terms>", "limit": 20}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/likit_bridge.py
```

**Example:**
```bash
echo '{"action": "search_posts", "keywords": "developer productivity tools", "limit": 10}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/likit_bridge.py
```

### Create Post

Publish a new LinkedIn post.

```bash
echo '{"action": "create_post", "text": "<post_text>", "visibility": "PUBLIC"}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/likit_bridge.py
```

- `visibility`: `PUBLIC` (default) or `CONNECTIONS`

**Response:**
```json
{"urn": "urn:li:activity:1234567890"}
```

### Upload Image

Upload an image for use in a post. Image data must be base64-encoded.

```bash
echo '{"action": "upload_image", "image_data": "<base64_encoded_data>", "filename": "image.png"}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/likit_bridge.py
```

**Response:**
```json
{"media_urn": "urn:li:digitalmediaAsset:D1234567890"}
```

### Create Post with Image

Create a post with an attached image in a single operation.

```bash
echo '{"action": "create_post_with_image", "text": "<post_text>", "image_data": "<base64_encoded_data>", "filename": "image.png", "visibility": "PUBLIC"}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/likit_bridge.py
```

**Response:**
```json
{"urn": "urn:li:activity:1234567890"}
```

## Safety Rules

**CRITICAL: NEVER create a LinkedIn post without explicit user confirmation.**

Before any write operation (create_post, create_post_with_image):
1. Show the user the exact text that will be posted
2. Show the visibility setting (PUBLIC vs CONNECTIONS)
3. If attaching an image, confirm which file
4. Wait for explicit "yes" / "go ahead" / confirmation before executing

## Error Handling

All errors return `{"error": "description"}`. Common patterns:
- **Auth errors / empty responses**: Cookies expired. Tell the user to run `/linkedin-setup`
- **"requires text"**: Missing required field in the command
- **Network errors**: Wait and retry once. If it persists, inform the user
