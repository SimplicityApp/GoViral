---
name: linkedin-setup
description: Set up LinkedIn cookie-based authentication. Extracts session cookies from Chrome or accepts manual cookie input for the linkitin client.
disable-model-invocation: true
allowed-tools: Bash
---

# LinkedIn Authentication Setup

Set up cookie-based authentication for LinkedIn operations using the linkitin bridge.

## Prerequisites

- Python 3.8+ available on PATH
- For browser extraction: user must be logged into LinkedIn in Chrome
- For manual cookies: user needs `li_at` and `JSESSIONID` values from browser DevTools

## Setup Steps

### 1. Ensure the GoViral venv exists

```bash
VENV_DIR="$HOME/.goviral/venv"
if [ ! -f "$VENV_DIR/bin/python3" ]; then
  python3 -m venv "$VENV_DIR"
fi
```

### 2. Install linkitin dependencies

```bash
"$HOME/.goviral/venv/bin/python3" -m pip install httpx pydantic browser-cookie3 -q
```

### 3. Ensure the linkitin package is installed

The linkitin Python package must be importable. Check and install:

```bash
"$HOME/.goviral/venv/bin/python3" -c "import linkitin" 2>/dev/null || \
  "$HOME/.goviral/venv/bin/python3" -m pip install -e "$HOME/Project/linkitin/" -q
```

If the above fails because `~/Project/linkitin/` doesn't exist, tell the user they need to clone or install the linkitin package.

### 4. Extract cookies

**Option A — Browser extraction (preferred):**

```bash
echo '{"action": "login_browser"}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

**Option B — Manual cookies:**

Ask the user for their `li_at` and `JSESSIONID` cookie values (from Chrome DevTools > Application > Cookies > linkedin.com), then:

```bash
echo '{"action": "login", "li_at": "<LI_AT_VALUE>", "jsessionid": "<JSESSIONID_VALUE>"}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

**Expected success response:**
```json
{"status": "ok"}
```

### 5. Verify by fetching posts

```bash
echo '{"action": "get_my_posts", "limit": 1}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

If this returns posts, authentication is working.

## Troubleshooting

- **Browser extraction fails**: Ask user to open Chrome, navigate to linkedin.com, verify login, try again
- **Manual cookies don't work**: Cookies may be expired. Have user get fresh values from DevTools
- **"linkitin package not found"**: The linkitin Python package needs to be installed. Check `~/Project/linkitin/` or ask user where it's located
- **Cookies expire**: If LinkedIn operations start failing, run `/linkedin-setup` again
