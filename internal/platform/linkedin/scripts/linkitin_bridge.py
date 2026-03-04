#!/usr/bin/env python3
"""Bridge script for GoViral subprocess integration.

Reads JSON commands from stdin (one per line), executes them via LinkitinClient,
and writes JSON responses to stdout (one per line).

Commands:
    {"action": "login", "li_at": "...", "jsessionid": "..."}
    {"action": "login_browser"}
    {"action": "get_my_posts", "limit": 20}
    {"action": "get_feed", "limit": 20}
    {"action": "create_post", "text": "...", "visibility": "PUBLIC"}
    {"action": "get_trending_posts", "topic": "...", "period": "past-24h", "limit": 10, "from_followed": true, "scrolls": 3}
    {"action": "search_posts", "keywords": "...", "limit": 20}
    {"action": "upload_image", "image_data": "<base64>", "filename": "image.png"}
    {"action": "create_post_with_image", "text": "...", "image_data": "<base64>", "filename": "image.png"}
    {"action": "repost", "share_urn": "urn:li:share:...", "text": "...", "visibility": "PUBLIC"}
    {"action": "create_scheduled_post", "text": "...", "scheduled_at": "2026-03-01T10:00:00+00:00", "visibility": "PUBLIC"}
    {"action": "create_scheduled_post_with_image", "text": "...", "image_data": "<base64>", "filename": "image.png", "scheduled_at": "2026-03-01T10:00:00+00:00", "visibility": "PUBLIC"}
    {"action": "delete_post", "post_urn": "urn:li:activity:..."}
    {"action": "comment_post", "post_urn": "urn:li:activity:...", "text": "Great insight!"}
"""
import json
import os
import subprocess
import sys

# Marker file written when Chrome proxy mode is activated.
# Subsequent subprocesses check this file to restore proxy mode without
# re-running the full browser login flow.
_GOVIRAL_DIR = os.path.join(os.path.expanduser("~"), ".goviral")
_CHROME_PROXY_MARKER = os.path.join(_GOVIRAL_DIR, "linkitin_chrome_proxy")


def ensure_package(package_name, pip_name=None):
    try:
        __import__(package_name)
    except ImportError:
        pip_name = pip_name or package_name
        print(f"{pip_name} not found, installing...", file=sys.stderr)
        subprocess.check_call(
            [sys.executable, "-m", "pip", "install", pip_name, "-q"],
            stdout=sys.stderr,
            stderr=sys.stderr,
        )


# Ensure linkitin is available (installed from PyPI).
ensure_package("linkitin")


import asyncio
import base64

# ---------------------------------------------------------------------------
# Headless Chromium support for non-macOS (Linux servers).
#
# linkitin's chrome_data module uses osascript (macOS AppleScript) to execute
# JavaScript inside a Chrome tab on linkedin.com. On Linux we replace this
# with headless Chromium via Playwright, providing the same interface.
# ---------------------------------------------------------------------------

_headless_page = None  # Playwright Page, lazily initialized
_headless_pw = None    # Playwright instance
_headless_browser = None


def _setup_headless_chrome(li_at, jsessionid):
    """Launch headless Chromium and navigate to LinkedIn with cookies."""
    global _headless_page, _headless_pw, _headless_browser

    if _headless_page is not None:
        return

    ensure_package("playwright")
    from playwright.sync_api import sync_playwright

    chromium_path = os.environ.get("CHROMIUM_PATH")
    if not chromium_path:
        # Common paths on Linux
        for p in ["/usr/bin/chromium", "/usr/bin/chromium-browser", "/usr/bin/google-chrome"]:
            if os.path.exists(p):
                chromium_path = p
                break

    if not chromium_path:
        print("[goviral] no Chromium found, headless mode unavailable", file=sys.stderr)
        return

    print(f"[goviral] launching headless Chromium ({chromium_path})...", file=sys.stderr)
    _headless_pw = sync_playwright().start()
    _headless_browser = _headless_pw.chromium.launch(
        executable_path=chromium_path,
        headless=True,
        args=[
            "--no-sandbox",
            "--disable-gpu",
            "--disable-dev-shm-usage",
            "--disable-software-rasterizer",
        ],
    )
    context = _headless_browser.new_context(
        user_agent=(
            "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
            "AppleWebKit/537.36 (KHTML, like Gecko) "
            "Chrome/131.0.0.0 Safari/537.36"
        ),
        viewport={"width": 1920, "height": 1080},
    )

    # Strip quotes from JSESSIONID if present (LinkedIn stores it as "ajax:...")
    clean_jsessionid = jsessionid.strip('"')

    context.add_cookies([
        {
            "name": "li_at",
            "value": li_at,
            "domain": ".linkedin.com",
            "path": "/",
            "httpOnly": True,
            "secure": True,
        },
        {
            "name": "JSESSIONID",
            "value": f'"{clean_jsessionid}"',
            "domain": ".linkedin.com",
            "path": "/",
            "secure": True,
        },
    ])

    page = context.new_page()
    page.goto("https://www.linkedin.com/feed/", wait_until="domcontentloaded", timeout=30000)
    _headless_page = page
    print("[goviral] headless Chromium ready", file=sys.stderr)

    import atexit
    atexit.register(_cleanup_headless)


def _cleanup_headless():
    global _headless_page, _headless_browser, _headless_pw
    try:
        if _headless_browser:
            _headless_browser.close()
        if _headless_pw:
            _headless_pw.stop()
    except Exception:
        pass
    _headless_page = None
    _headless_browser = None
    _headless_pw = None


def _headless_find_and_exec(js_code):
    """Execute JS in headless LinkedIn page (drop-in for osascript version)."""
    if _headless_page is None:
        from linkitin.exceptions import AuthError
        raise AuthError("headless browser not initialized")
    try:
        result = _headless_page.evaluate(js_code)
        return str(result) if result is not None else ""
    except Exception as e:
        err = str(e)
        # Navigation-triggering JS (window.location.href = ...) may cause
        # Playwright to report an error. Wait for the new page and return.
        if "navigation" in err.lower() or "context was destroyed" in err.lower():
            try:
                _headless_page.wait_for_load_state("domcontentloaded", timeout=15000)
            except Exception:
                pass
            return ""
        raise


if sys.platform != "darwin":
    # Monkey-patch linkitin's osascript-based functions to use headless Chromium.
    import linkitin.chrome_data as _cd
    _cd._find_linkedin_tab_and_exec = _headless_find_and_exec

    import linkitin.chrome_proxy as _cp
    _cp._find_linkedin_tab_and_exec = _headless_find_and_exec


from linkitin import LinkitinClient


async def handle_command(client, cmd):
    action = cmd.get("action", "")

    if action == "login":
        li_at = cmd.get("li_at", "")
        jsessionid = cmd.get("jsessionid", "")
        if not li_at or not jsessionid:
            return {"error": "login requires li_at and jsessionid"}
        await client.login_with_cookies(li_at, jsessionid)
        # Initialize headless browser with new cookies.
        if sys.platform != "darwin":
            _setup_headless_chrome(li_at, jsessionid)
        return {"status": "ok"}

    elif action == "login_browser":
        await client.login_from_browser()
        if client.session.use_chrome_proxy:
            # Persist Chrome proxy mode so future subprocesses can restore it.
            os.makedirs(_GOVIRAL_DIR, exist_ok=True)
            open(_CHROME_PROXY_MARKER, "w").close()
        return {"status": "ok"}

    elif action == "login_saved":
        ok = await client.login_from_saved()
        if ok:
            return {"status": "ok"}
        return {"error": "no saved cookies or cookies are expired"}

    elif action == "get_my_posts":
        limit = cmd.get("limit", 20)
        posts = await client.get_my_posts(limit=limit)
        return {"posts": [p.model_dump(mode="json") for p in posts]}

    elif action == "get_feed":
        limit = cmd.get("limit", 20)
        posts = await client.get_feed(limit=limit)
        return {"posts": [p.model_dump(mode="json") for p in posts]}

    elif action == "create_post":
        text = cmd.get("text", "")
        if not text:
            return {"error": "create_post requires text"}
        visibility = cmd.get("visibility", "PUBLIC")
        urn = await client.create_post(text=text, visibility=visibility)
        return {"urn": urn}

    elif action == "get_trending_posts":
        topic = cmd.get("topic", "")
        period = cmd.get("period", "past-24h")
        limit = cmd.get("limit", 10)
        from_followed = cmd.get("from_followed", True)
        scrolls = cmd.get("scrolls", 3)
        posts = await client.get_trending_posts(
            topic=topic, period=period, limit=limit,
            from_followed=from_followed, scrolls=scrolls,
        )
        return {"posts": [p.model_dump(mode="json") for p in posts]}

    elif action == "search_posts":
        keywords = cmd.get("keywords", "")
        if not keywords:
            return {"error": "search_posts requires keywords"}
        limit = cmd.get("limit", 20)
        posts = await client.search_posts(keywords=keywords, limit=limit)
        return {"posts": [p.model_dump(mode="json") for p in posts]}

    elif action == "upload_image":
        image_b64 = cmd.get("image_data", "")
        if not image_b64:
            return {"error": "upload_image requires image_data (base64)"}
        image_data = base64.b64decode(image_b64)
        filename = cmd.get("filename", "image.png")
        media_urn = await client.upload_image(image_data, filename)
        return {"media_urn": media_urn}

    elif action == "create_post_with_image":
        text = cmd.get("text", "")
        image_b64 = cmd.get("image_data", "")
        if not text or not image_b64:
            return {"error": "create_post_with_image requires text and image_data"}
        image_data = base64.b64decode(image_b64)
        filename = cmd.get("filename", "image.png")
        visibility = cmd.get("visibility", "PUBLIC")
        urn = await client.create_post_with_image(
            text=text, image_data=image_data, filename=filename, visibility=visibility
        )
        return {"urn": urn}

    elif action == "repost":
        post_urn = cmd.get("post_urn", "")
        if not post_urn:
            return {"error": "repost requires post_urn"}
        text = cmd.get("text", "")
        visibility = cmd.get("visibility", "PUBLIC")
        urn = await client.repost(share_urn=post_urn, text=text)
        return {"urn": urn}

    elif action == "create_scheduled_post":
        text = cmd.get("text", "")
        scheduled_at_str = cmd.get("scheduled_at", "")
        if not text or not scheduled_at_str:
            return {"error": "create_scheduled_post requires text and scheduled_at"}
        from datetime import datetime
        scheduled_at = datetime.fromisoformat(scheduled_at_str)
        visibility = cmd.get("visibility", "PUBLIC")
        urn = await client.create_scheduled_post(text=text, scheduled_at=scheduled_at, visibility=visibility)
        return {"urn": urn}

    elif action == "create_scheduled_post_with_image":
        text = cmd.get("text", "")
        image_b64 = cmd.get("image_data", "")
        scheduled_at_str = cmd.get("scheduled_at", "")
        if not text or not image_b64 or not scheduled_at_str:
            return {"error": "create_scheduled_post_with_image requires text, image_data, and scheduled_at"}
        from datetime import datetime
        image_data = base64.b64decode(image_b64)
        filename = cmd.get("filename", "image.png")
        scheduled_at = datetime.fromisoformat(scheduled_at_str)
        visibility = cmd.get("visibility", "PUBLIC")
        urn = await client.create_scheduled_post_with_image(
            text=text, image_data=image_data, filename=filename,
            scheduled_at=scheduled_at, visibility=visibility
        )
        return {"urn": urn}

    elif action == "comment_post":
        post_urn = cmd.get("post_urn", "")
        text = cmd.get("text", "")
        if not post_urn or not text:
            return {"error": "comment_post requires post_urn and text"}
        thread_urn = cmd.get("thread_urn", "") or ""
        urn = await client.comment_post(post_urn=post_urn, text=text, thread_urn=thread_urn)
        return {"urn": urn}

    elif action == "delete_post":
        post_urn = cmd.get("post_urn", "")
        if not post_urn:
            return {"error": "delete_post requires post_urn"}
        await client.delete_post(post_urn)
        return {"status": "ok"}

    else:
        return {"error": f"unknown action: {action}"}


async def main():
    cookies_path = os.path.join(_GOVIRAL_DIR, "linkitin_cookies.json")
    client = LinkitinClient(cookies_path=cookies_path)

    # Try to load saved cookies on startup.
    loaded = False
    try:
        loaded = await client.login_from_saved()
    except Exception:
        pass

    if loaded and sys.platform != "darwin":
        # Initialize headless Chromium with the saved cookies.
        li_at = getattr(client.session, "_li_at", None)
        jsessionid = getattr(client.session, "_jsessionid", None)
        if not li_at or not jsessionid:
            # Read cookies from the file directly as fallback.
            try:
                with open(cookies_path) as f:
                    cdata = json.load(f)
                li_at = cdata.get("li_at", "")
                jsessionid = cdata.get("jsessionid", cdata.get("JSESSIONID", ""))
            except Exception:
                pass
        if li_at and jsessionid:
            try:
                _setup_headless_chrome(li_at, jsessionid)
            except Exception as e:
                print(f"[goviral] headless Chrome setup failed: {e}", file=sys.stderr)

    if not loaded and os.path.exists(_CHROME_PROXY_MARKER):
        if sys.platform != "darwin":
            # Chrome proxy requires osascript (macOS only) — remove stale marker.
            try:
                os.unlink(_CHROME_PROXY_MARKER)
            except OSError:
                pass
        else:
            # Chrome proxy mode was previously activated (extract-cookies used).
            # Restore it: route all requests through Chrome's live session.
            try:
                from linkitin.chrome_proxy import chrome_validate_session
                if chrome_validate_session():
                    client.session.use_chrome_proxy = True
                else:
                    # Chrome no longer has a valid LinkedIn session — clear the marker.
                    os.unlink(_CHROME_PROXY_MARKER)
            except Exception:
                pass

    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            cmd = json.loads(line)
            result = await handle_command(client, cmd)
        except Exception as e:
            result = {"error": str(e)}
        print(json.dumps(result), flush=True)

    await client.close()
    _cleanup_headless()


if __name__ == "__main__":
    asyncio.run(main())
