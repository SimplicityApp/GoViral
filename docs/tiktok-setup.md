# TikTok Setup Guide

## Two Authentication Options

| Method | Requires API approval? | Setup time | Best for |
|--------|----------------------|------------|----------|
| **Option A: OAuth 2.0** | Yes (1-5 days review) | Medium | Production use |
| **Option B: Cookie fallback** | No | Immediate | Quick start, personal use |

---

## Option A: OAuth 2.0 (Official API)

### Step 1: Create TikTok Developer App

1. Go to [TikTok Developer Portal](https://developers.tiktok.com)
2. Create a new app with these settings:

| Field | Value |
|-------|-------|
| **App name** | `goviral` |
| **Category** | Social Networking |
| **Description** | `A developer tool that generates and publishes viral short-form video content to TikTok. Supports video uploads, scheduling, and content management.` |
| **Terms of Service URL** | Your TOS URL (e.g., a GitHub Pages link) |
| **Privacy Policy URL** | Your privacy policy URL |
| **Platforms** | Web |

3. Add **Products**:
   - **Login Kit** (for OAuth)
   - **Content Posting API** (for video uploads)

4. Add **Scopes**:
   - `user.info.basic`
   - `video.upload`
   - `video.publish`

5. For the **"Explain how each product and scope works"** field:
   ```
   GoViral is a content management tool that publishes short-form videos to
   TikTok via the Content Posting API. The integration flow: 1) User
   authenticates via TikTok Login Kit (OAuth 2.0) to connect their account.
   2) User selects a video file and writes a caption with hashtags. 3) The
   app uploads the video using the Content Posting API video upload endpoint.
   4) User can optionally schedule videos for future publishing. Scopes used:
   user.info.basic to identify the connected account, video.upload to upload
   video files, video.publish to publish uploaded videos to the user's TikTok
   profile.
   ```

6. **Demo video**: Record a screen capture showing the end-to-end flow in sandbox mode (OAuth login -> video upload -> published video). You must demonstrate this working before submitting for review.

7. Add `http://localhost:8990/callback` as a redirect URI.

8. Copy the **Client Key** and **Client Secret**.

### Step 2: Add Credentials to Config

Edit `~/.goviral/config.yaml`:

```yaml
tiktok:
  client_key: "YOUR_CLIENT_KEY"
  client_secret: "YOUR_CLIENT_SECRET"
```

### Step 3: Authenticate

```bash
goviral tiktok-login
```

This will:
1. Open your browser to TikTok's OAuth consent screen
2. Start a local server on `localhost:8990` to receive the callback
3. Exchange the code for access + refresh tokens
4. Save tokens to `~/.goviral/config.yaml`

### App Review Notes

- **Sandbox mode** is available immediately after creating the app (can only post to your own account with limited quota)
- **Full review** takes 1-5 business days
- You must record a demo video showing the integration working in sandbox before submitting

---

## Option B: Cookie-Based Fallback (No API Approval Needed)

This uses the `tiktok-uploader` Python package which automates TikTok via Playwright (headless browser). No API credentials required.

### Step 1: Log into TikTok in Chrome

Open Chrome and log into your TikTok account at [tiktok.com](https://www.tiktok.com).

### Step 2: Export Cookies

Use a browser extension like **Get cookies.txt** (or similar) to export your TikTok cookies.

Save the exported cookies to:
```
~/.goviral/tiktok_cookies.json
```

Alternatively, run the helper command for instructions:
```bash
goviral tiktok-login --cookies
```

### Step 3: Dependencies

The cookie fallback requires Python 3 and Playwright. These are installed automatically on first use, but you can install manually:

```bash
pip install tiktok-uploader playwright
playwright install chromium
```

### Cookie Expiry

TikTok session cookies expire periodically. If uploads start failing with auth errors, re-export your cookies from Chrome.

---

## Usage

### CLI

```bash
# Post video content by ID
goviral post --id 5 --video /path/to/video.mp4
```

### API

```bash
# Publish video content to TikTok
curl -X POST http://localhost:8080/api/v1/tiktok/upload \
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

Navigate to the **Video** page in the sidebar, select **TikTok**, and publish video content.

## TikTok Video Requirements

- **Duration**: 1 second to 10 minutes
- **Format**: mp4 recommended
- **Aspect ratio**: 9:16 (vertical) recommended for best performance
- **File size**: Up to 4GB via official API, varies for cookie fallback

## Rate Limits (Official API)

- 6 requests per minute
- 15 videos per day per user

## Troubleshooting

- **"No TikTok access token configured"** — Run `goviral tiktok-login` or use `--cookies` mode
- **"Cookie file not found"** — Export cookies to `~/.goviral/tiktok_cookies.json`
- **Cookie auth fails** — Re-export cookies from Chrome (they expire)
- **"tiktok-uploader not installed"** — Run `pip install tiktok-uploader playwright && playwright install chromium`
- **Playwright browser errors** — Run `playwright install chromium` to ensure the browser is installed
- **Upload succeeds but no video ID returned** — Normal for cookie fallback; the video was uploaded successfully
