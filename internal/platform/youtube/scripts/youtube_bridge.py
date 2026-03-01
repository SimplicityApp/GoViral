#!/usr/bin/env python3
"""YouTube bridge script for GoViral.

Communicates via JSON on stdin/stdout.
Uses google-api-python-client for YouTube Data API v3.

Actions:
  upload_video: Upload a video to YouTube
  set_thumbnail: Set a custom thumbnail for a video
"""

import json
import os
import sys

CONFIG_DIR = os.path.expanduser("~/.goviral")
CREDENTIALS_FILE = os.path.join(CONFIG_DIR, "youtube_credentials.json")
TOKEN_FILE = os.path.join(CONFIG_DIR, "youtube_token.json")


def get_youtube_service():
    """Build an authenticated YouTube API service."""
    try:
        from google.oauth2.credentials import Credentials
        from googleapiclient.discovery import build
    except ImportError:
        return None, "google-api-python-client not installed"

    if not os.path.exists(TOKEN_FILE):
        return None, f"YouTube token file not found at {TOKEN_FILE}; run 'goviral youtube-login' first"

    with open(TOKEN_FILE, "r") as f:
        token_data = json.load(f)

    creds = Credentials(
        token=token_data.get("access_token"),
        refresh_token=token_data.get("refresh_token"),
        token_uri="https://oauth2.googleapis.com/token",
        client_id=token_data.get("client_id"),
        client_secret=token_data.get("client_secret"),
    )

    service = build("youtube", "v3", credentials=creds)
    return service, None


def upload_video(cmd):
    """Upload a video to YouTube."""
    try:
        from googleapiclient.http import MediaFileUpload
    except ImportError:
        return {"error": "google-api-python-client not installed"}

    service, err = get_youtube_service()
    if err:
        return {"error": err}

    video_path = cmd.get("video_path", "")
    if not video_path or not os.path.exists(video_path):
        return {"error": f"Video file not found: {video_path}"}

    title = cmd.get("title", "")
    description = cmd.get("description", "")
    tags = cmd.get("tags", [])

    body = {
        "snippet": {
            "title": title,
            "description": description,
            "tags": tags,
            "categoryId": "28",
        },
        "status": {
            "privacyStatus": "public",
            "selfDeclaredMadeForKids": False,
        },
    }

    media = MediaFileUpload(video_path, resumable=True)

    try:
        request = service.videos().insert(
            part="snippet,status",
            body=body,
            media_body=media,
        )
        response = None
        while response is None:
            _, response = request.next_chunk()

        video_id = response["id"]

        # Set thumbnail if provided
        thumbnail_path = cmd.get("thumbnail_path", "")
        if thumbnail_path and os.path.exists(thumbnail_path):
            thumb_media = MediaFileUpload(thumbnail_path)
            service.thumbnails().set(
                videoId=video_id,
                media_body=thumb_media,
            ).execute()

        return {"video_id": video_id, "url": f"https://youtube.com/shorts/{video_id}"}

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
    else:
        result = {"error": f"unknown action: {action}"}

    print(json.dumps(result))


if __name__ == "__main__":
    main()
