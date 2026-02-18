#!/usr/bin/env python3
"""Bridge script for GoViral subprocess integration.

Reads JSON commands from stdin (one per line), executes them via LikitClient,
and writes JSON responses to stdout (one per line).

Commands:
    {"action": "login", "li_at": "...", "jsessionid": "..."}
    {"action": "login_browser"}
    {"action": "get_my_posts", "limit": 20}
    {"action": "get_feed", "limit": 20}
    {"action": "create_post", "text": "...", "visibility": "PUBLIC"}
    {"action": "search_posts", "keywords": "...", "limit": 20}
    {"action": "upload_image", "image_data": "<base64>", "filename": "image.png"}
    {"action": "create_post_with_image", "text": "...", "image_data": "<base64>", "filename": "image.png"}
"""
import json
import os
import subprocess
import sys


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


# Ensure dependencies are available.
ensure_package("httpx")
ensure_package("pydantic")


import asyncio
import base64

# Add parent directory to path so likit package can be found when running
# from the scripts directory or from Go's embedded copy.
script_dir = os.path.dirname(os.path.abspath(__file__))
parent_dir = os.path.dirname(script_dir)
if parent_dir not in sys.path:
    sys.path.insert(0, parent_dir)

# Also add the likit package directory for when the bridge is embedded
# alongside the likit package in Go.
likit_pkg_dir = os.path.join(script_dir, "..")
if likit_pkg_dir not in sys.path:
    sys.path.insert(0, os.path.abspath(likit_pkg_dir))

from likit import LikitClient


async def handle_command(client, cmd):
    action = cmd.get("action", "")

    if action == "login":
        li_at = cmd.get("li_at", "")
        jsessionid = cmd.get("jsessionid", "")
        if not li_at or not jsessionid:
            return {"error": "login requires li_at and jsessionid"}
        await client.login_with_cookies(li_at, jsessionid)
        return {"status": "ok"}

    elif action == "login_browser":
        await client.login_from_browser()
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

    else:
        return {"error": f"unknown action: {action}"}


async def main():
    client = LikitClient()

    # Try to load saved cookies on startup.
    try:
        await client.login_from_saved()
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


if __name__ == "__main__":
    asyncio.run(main())
