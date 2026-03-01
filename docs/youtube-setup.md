# YouTube Shorts Setup Guide

## Prerequisites

- GoViral installed with YouTube platform support
- A Google account with a YouTube channel

## Step 1: Create Google OAuth Credentials

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a new project (or select an existing one)
3. Enable the **YouTube Data API v3**:
   - Go to **APIs & Services** > **Library**
   - Search for "YouTube Data API v3"
   - Click **Enable**
4. Create OAuth credentials:
   - Go to **APIs & Services** > **Credentials**
   - Click **Create Credentials** > **OAuth 2.0 Client ID**
   - If prompted, configure the OAuth consent screen first:
     - User type: **External**
     - App name: `goviral`
     - Scopes: add `youtube.upload` and `youtube`
   - Application type: **Web application**
   - Name: `goviral`
   - Authorized redirect URIs: add `http://localhost:8989/callback`
   - Click **Create**
5. Copy the **Client ID** and **Client Secret**

## Step 2: Add Credentials to Config

Edit `~/.goviral/config.yaml`:

```yaml
youtube:
  client_id: "YOUR_CLIENT_ID.apps.googleusercontent.com"
  client_secret: "YOUR_CLIENT_SECRET"
```

## Step 3: Authenticate

Run the OAuth login command:

```bash
goviral youtube-login
```

This will:
1. Open your browser to Google's OAuth consent screen
2. Start a local server on `localhost:8989` to receive the callback
3. After you approve, exchange the code for access + refresh tokens
4. Save tokens to `~/.goviral/config.yaml` and `~/.goviral/youtube_token.json`

## Step 4: Verify

Check that `~/.goviral/config.yaml` now has populated `access_token` and `refresh_token` fields under `youtube:`.

Via the API:
```bash
curl -H "Authorization: Bearer YOUR_API_KEY" http://localhost:8080/api/v1/config
```
The response should show `"youtube": { "has_auth": true }`.

## Usage

### CLI

```bash
# Post video content by ID
goviral post --id 5 --video /path/to/video.mp4

# With custom thumbnail (YouTube only)
goviral post --id 5 --video /path/to/video.mp4 --thumbnail /path/to/thumb.jpg
```

### API

```bash
# Publish video content to YouTube
curl -X POST http://localhost:8080/api/v1/youtube/upload \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"content_id": 5}'

# Or use the generic publish endpoint (routes by target_platform)
curl -X POST http://localhost:8080/api/v1/publish \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"content_id": 5}'
```

### Web Dashboard

Navigate to the **Video** page in the sidebar to upload and publish videos.

## YouTube Shorts Requirements

For a video to be auto-detected as a YouTube Short:
- **Vertical aspect ratio** (9:16, e.g., 1080x1920)
- **Duration under 60 seconds**
- Adding `#Shorts` to the title or description helps but is not required

## Token Refresh

YouTube access tokens expire after 1 hour. The refresh token is stored and used automatically by both the Go client and the Python bridge. If auth fails, re-run:

```bash
goviral youtube-login
```

## Troubleshooting

- **"No YouTube access token configured"** — Run `goviral youtube-login`
- **403 Forbidden** — Check that YouTube Data API v3 is enabled in Google Cloud Console
- **"redirect_uri_mismatch"** — Ensure `http://localhost:8989/callback` is in your authorized redirect URIs
- **Upload fails silently** — Check that the video file is a supported format (mp4, webm, mov, avi)
- **Python bridge errors** — The fallback uses `google-api-python-client`; ensure Python 3 is installed
