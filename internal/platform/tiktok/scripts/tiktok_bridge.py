#!/usr/bin/env python3
"""TikTok bridge script for GoViral.

Communicates via JSON on stdin/stdout.
Uses tiktok-uploader library for cookie-based uploads.

Actions:
  upload_video: Upload a video to TikTok
  schedule_video: Schedule a video for future posting
"""

import json
import os
import sys
from datetime import datetime


def upload_video(cmd):
    """Upload a video to TikTok using tiktok-uploader."""
    try:
        from tiktok_uploader.upload import upload_video as tiktok_upload
    except ImportError:
        return {"error": "tiktok-uploader not installed; run: pip install tiktok-uploader"}

    video_path = cmd.get("video_path", "")
    if not video_path or not os.path.exists(video_path):
        return {"error": f"Video file not found: {video_path}"}

    description = cmd.get("description", "")
    tags = cmd.get("tags", [])
    cookie_path = cmd.get("cookie_path", "")

    # Build caption with hashtags
    caption = description
    for tag in tags:
        if tag:
            caption += f" #{tag}"

    if not cookie_path or not os.path.exists(cookie_path):
        return {"error": f"Cookie file not found: {cookie_path}; run 'goviral tiktok-login' first"}

    try:
        result = tiktok_upload(
            filename=video_path,
            description=caption,
            cookies=cookie_path,
            headless=True,
        )
        return {"video_id": str(result) if result else "uploaded", "status": "success"}
    except Exception as e:
        return {"error": str(e)}


def schedule_video(cmd):
    """Schedule a video for future posting on TikTok."""
    try:
        from tiktok_uploader.upload import upload_video as tiktok_upload
    except ImportError:
        return {"error": "tiktok-uploader not installed; run: pip install tiktok-uploader"}

    video_path = cmd.get("video_path", "")
    if not video_path or not os.path.exists(video_path):
        return {"error": f"Video file not found: {video_path}"}

    description = cmd.get("description", "")
    tags = cmd.get("tags", [])
    schedule_at = cmd.get("schedule_at", 0)
    cookie_path = cmd.get("cookie_path", "")

    caption = description
    for tag in tags:
        if tag:
            caption += f" #{tag}"

    if not cookie_path or not os.path.exists(cookie_path):
        return {"error": f"Cookie file not found: {cookie_path}; run 'goviral tiktok-login' first"}

    schedule_time = datetime.fromtimestamp(schedule_at) if schedule_at else None

    try:
        result = tiktok_upload(
            filename=video_path,
            description=caption,
            cookies=cookie_path,
            schedule=schedule_time,
            headless=True,
        )
        return {"video_id": str(result) if result else "scheduled", "status": "scheduled"}
    except Exception as e:
        return {"error": str(e)}


def main():
    raw = sys.stdin.read()
    try:
        cmd = json.loads(raw)
    except json.JSONDecodeError as e:
        print(json.dumps({"error": f"invalid JSON input: {e}"}))
        sys.exit(1)

    action = cmd.get("action", "")

    if action == "upload_video":
        result = upload_video(cmd)
    elif action == "schedule_video":
        result = schedule_video(cmd)
    else:
        result = {"error": f"unknown action: {action}"}

    print(json.dumps(result))


if __name__ == "__main__":
    main()
