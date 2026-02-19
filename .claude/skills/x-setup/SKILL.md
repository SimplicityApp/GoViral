---
name: x-setup
description: Set up X/Twitter cookie-based authentication. Extracts session cookies from Chrome and configures the twikit client.
disable-model-invocation: true
allowed-tools: Bash
---

# X/Twitter Authentication Setup

Set up cookie-based authentication for X/Twitter operations. This extracts session cookies from Chrome where the user must already be logged into X.

## Prerequisites

- User must be logged into X/Twitter in Chrome
- Python 3.8+ must be available on PATH

## Setup Steps

### 1. Ensure the GoViral venv exists

```bash
VENV_DIR="$HOME/.goviral/venv"
if [ ! -f "$VENV_DIR/bin/python3" ]; then
  python3 -m venv "$VENV_DIR"
fi
```

### 2. Install twikit dependencies

```bash
"$HOME/.goviral/venv/bin/python3" -m pip install twikit browser-cookie3 -q
```

### 3. Extract cookies from Chrome

```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py extract_cookies
```

**Expected output on success:**
```json
{"status": "ok", "cookies_path": "/home/user/.goviral/twikit_cookies.json", "cookie_count": N}
```

**Expected output on failure:**
```json
{"error": "could not find auth_token and ct0 cookies for x.com in Chrome — are you logged into X in Chrome?"}
```

### 4. Verify cookies exist

```bash
test -f "$HOME/.goviral/twikit_cookies.json" && echo "Cookies ready" || echo "Cookies missing"
```

## Troubleshooting

- If extraction fails, ask the user to: (1) open Chrome, (2) go to x.com, (3) verify they are logged in, (4) try again
- Cookies expire periodically. If X operations start failing with auth errors, run `/x-setup` again
- The cookie file is stored at `~/.goviral/twikit_cookies.json`
