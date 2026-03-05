#!/usr/bin/env python3
"""Twikit client for fetching X/Twitter user tweets via cookie-based auth.

Usage:
    python3 twikit_guest.py extract_cookies       # auto-extract from Chrome
    python3 twikit_guest.py fetch_user_tweets <username> <limit>

extract_cookies pulls X session cookies from Chrome and saves them
to ~/.goviral/twikit_cookies.json in twikit's format.
Subsequent fetch_user_tweets calls reuse the saved cookies.

Outputs JSON to stdout, logs to stderr.
Auto-installs dependencies via pip if missing.
"""
import json
import os
import sys
import subprocess


COOKIES_PATH = os.environ.get(
    "GOVIRAL_TWIKIT_COOKIES_PATH",
    os.path.join(os.path.expanduser("~"), ".goviral", "twikit_cookies.json"),
)


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


def extract_cookies():
    """Extract X/Twitter cookies from Chrome and save in twikit format."""
    import browser_cookie3

    cookie_jar = browser_cookie3.chrome(domain_name=".x.com")
    cookies = {}
    for cookie in cookie_jar:
        cookies[cookie.name] = cookie.value

    if "auth_token" not in cookies or "ct0" not in cookies:
        return {"error": "could not find auth_token and ct0 cookies for x.com in Chrome — are you logged into X in Chrome?"}

    os.makedirs(os.path.dirname(COOKIES_PATH), exist_ok=True)
    with open(COOKIES_PATH, "w", encoding="utf-8") as f:
        json.dump(cookies, f)

    return {
        "status": "ok",
        "cookies_path": COOKIES_PATH,
        "cookie_count": len(cookies),
    }


def fetch_user_tweets(username, limit):
    import asyncio

    async def _fetch():
        from twikit import Client

        client = Client("en-US")
        client.load_cookies(COOKIES_PATH)

        user = await client.get_user_by_screen_name(username)
        tweets = await client.get_user_tweets(user.id, "Tweets", count=limit)

        results = []
        for tweet in tweets:
            media_list = []
            if hasattr(tweet, "media") and tweet.media:
                for m in tweet.media:
                    media_list.append({
                        "type": getattr(m, "type", "") or "",
                        "url": getattr(m, "media_url_https", "") or getattr(m, "url", "") or "",
                        "preview_url": getattr(m, "preview_image_url", "") or "",
                        "alt_text": getattr(m, "alt_text", "") or "",
                    })
            results.append({
                "id": str(tweet.id),
                "text": tweet.text or "",
                "created_at": tweet.created_at or "",
                "likes": tweet.favorite_count or 0,
                "retweets": tweet.retweet_count or 0,
                "replies": tweet.reply_count or 0,
                "impressions": int(getattr(tweet, "view_count", 0) or 0),
                "media": media_list,
            })
            if len(results) >= limit:
                break

        # Re-save cookies to keep session fresh.
        client.save_cookies(COOKIES_PATH)
        return results

    tweets = asyncio.run(_fetch())
    return {"tweets": tweets}


def search_trending(niches_json, min_likes, limit, period="day"):
    """Search for trending/top tweets matching the given niches."""
    import asyncio
    from datetime import datetime, timedelta

    niches = json.loads(niches_json)

    period_days = {"day": 1, "week": 7, "month": 30}
    days = period_days.get(period, 1)
    now = datetime.utcnow()
    since_date = (now - timedelta(days=days)).strftime("%Y-%m-%d")
    until_date = (now + timedelta(days=1)).strftime("%Y-%m-%d")

    async def _search():
        from twikit import Client

        client = Client("en-US")
        client.load_cookies(COOKIES_PATH)

        seen = set()
        all_tweets = []
        niche_errors = 0

        for niche in niches:
            query = f"{niche} min_faves:{min_likes} lang:en -filter:retweets since:{since_date} until:{until_date}"
            try:
                tweets = await client.search_tweet(query, "Top", count=20)
            except Exception as e:
                niche_errors += 1
                print(f"search for niche '{niche}' failed: {e}", file=sys.stderr)
                continue

            for tweet in tweets:
                if str(tweet.id) in seen:
                    continue
                seen.add(str(tweet.id))

                user = tweet.user
                media_list = []
                if hasattr(tweet, "media") and tweet.media:
                    for m in tweet.media:
                        media_list.append({
                            "type": getattr(m, "type", "") or "",
                            "url": getattr(m, "media_url_https", "") or getattr(m, "url", "") or "",
                            "preview_url": getattr(m, "preview_image_url", "") or "",
                            "alt_text": getattr(m, "alt_text", "") or "",
                        })
                all_tweets.append({
                    "id": str(tweet.id),
                    "text": tweet.text or "",
                    "created_at": tweet.created_at or "",
                    "author_username": user.screen_name if user else "",
                    "author_name": user.name if user else "",
                    "likes": tweet.favorite_count or 0,
                    "retweets": tweet.retweet_count or 0,
                    "replies": tweet.reply_count or 0,
                    "impressions": int(getattr(tweet, "view_count", 0) or 0),
                    "niche": niche,
                    "media": media_list,
                })

                if len(all_tweets) >= limit:
                    break

            if len(all_tweets) >= limit:
                break

        # Sort by engagement (likes + retweets + replies) descending.
        all_tweets.sort(
            key=lambda t: t["likes"] + t["retweets"] + t["replies"],
            reverse=True,
        )

        if len(all_tweets) > limit:
            all_tweets = all_tweets[:limit]

        # Always save cookies — this refreshes the ct0 CSRF token even on auth failure.
        client.save_cookies(COOKIES_PATH)

        # If every niche failed, raise so Go knows to retry (cookies just refreshed above).
        if not all_tweets and niche_errors == len(niches) and niche_errors > 0:
            raise Exception(
                f"all {niche_errors} niche searches failed (session error) — "
                "cookies refreshed, retry should work"
            )

        return all_tweets

    tweets = asyncio.run(_search())
    return {"trending": tweets}


def upload_media(file_path):
    """Upload a media file using cookie-based auth."""
    import asyncio

    async def _upload():
        from twikit import Client

        client = Client("en-US")
        client.load_cookies(COOKIES_PATH)

        media_id = await client.upload_media(file_path)
        client.save_cookies(COOKIES_PATH)
        return {"media_id": str(media_id)}

    return asyncio.run(_upload())


def create_tweet(text, reply_to=None, media_ids_json=None):
    """Create a tweet (or reply) using cookie-based auth."""
    import asyncio

    async def _create():
        from twikit import Client

        client = Client("en-US")
        client.load_cookies(COOKIES_PATH)

        kwargs = {}
        if reply_to:
            kwargs["reply_to"] = reply_to
        if media_ids_json:
            media_ids = json.loads(media_ids_json)
            if media_ids:
                kwargs["media_ids"] = media_ids

        try:
            tweet = await client.create_tweet(text, **kwargs)
        except KeyError as e:
            client.save_cookies(COOKIES_PATH)
            raise Exception(
                f"X API response missing field {e} — this may indicate rate limiting, "
                "a duplicate post, or an API change. Try again in a few seconds."
            )
        if tweet is None:
            client.save_cookies(COOKIES_PATH)  # refresh ct0 for retry
            raise Exception("create_tweet returned None — cookies refreshed, retry should work")
        client.save_cookies(COOKIES_PATH)
        return {"tweet_id": str(tweet.id)}

    return asyncio.run(_create())


def create_quote_tweet(text, quote_tweet_id):
    """Create a quote tweet using cookie-based auth."""
    import asyncio

    async def _create():
        from twikit import Client

        client = Client("en-US")
        client.load_cookies(COOKIES_PATH)

        tweet = await client.create_tweet(
            text,
            attachment_url=f"https://x.com/i/status/{quote_tweet_id}",
        )
        if tweet is None:
            client.save_cookies(COOKIES_PATH)
            raise Exception("create_quote_tweet returned None — cookies refreshed, retry should work")
        client.save_cookies(COOKIES_PATH)
        return {"tweet_id": str(tweet.id)}

    return asyncio.run(_create())


def schedule_quote_tweet(text, quote_tweet_id, scheduled_at_unix):
    """Schedule a quote tweet for future posting via X's native scheduling."""
    import asyncio

    async def _schedule():
        from twikit import Client
        from twikit.client.gql import Endpoint, FEATURES, get_query_id

        client = Client("en-US")
        client.load_cookies(COOKIES_PATH)

        variables = {
            "post_tweet_request": {
                "auto_populate_reply_metadata": False,
                "status": text,
                "exclude_reply_user_ids": [],
                "media_ids": [],
                "attachment_url": f"https://x.com/i/status/{quote_tweet_id}",
            },
            "execute_at": int(scheduled_at_unix),
        }

        data = {
            "variables": variables,
            "queryId": get_query_id(Endpoint.CREATE_SCHEDULED_TWEET),
            "features": FEATURES,
        }

        response, _ = await client.post(
            Endpoint.CREATE_SCHEDULED_TWEET,
            json=data,
            headers=client._base_headers,
        )

        scheduled_tweet_id = response["data"]["tweet"]["rest_id"]
        client.save_cookies(COOKIES_PATH)
        return {"scheduled_tweet_id": str(scheduled_tweet_id)}

    return asyncio.run(_schedule())


def schedule_tweet(text, scheduled_at_unix, media_ids_json=None):
    """Schedule a tweet for future posting via X's native scheduling.

    twikit's create_scheduled_tweet omits the required 'features' dict,
    so we call the GQL endpoint directly with features included.
    """
    import asyncio

    async def _schedule():
        from twikit import Client
        from twikit.client.gql import Endpoint, FEATURES, get_query_id

        client = Client("en-US")
        client.load_cookies(COOKIES_PATH)

        media_ids = []
        if media_ids_json:
            parsed = json.loads(media_ids_json)
            if parsed:
                media_ids = parsed

        variables = {
            "post_tweet_request": {
                "auto_populate_reply_metadata": False,
                "status": text,
                "exclude_reply_user_ids": [],
                "media_ids": media_ids,
            },
            "execute_at": int(scheduled_at_unix),
        }

        data = {
            "variables": variables,
            "queryId": get_query_id(Endpoint.CREATE_SCHEDULED_TWEET),
            "features": FEATURES,
        }

        response, _ = await client.post(
            Endpoint.CREATE_SCHEDULED_TWEET,
            json=data,
            headers=client._base_headers,
        )

        scheduled_tweet_id = response["data"]["tweet"]["rest_id"]
        client.save_cookies(COOKIES_PATH)
        return {"scheduled_tweet_id": str(scheduled_tweet_id)}

    return asyncio.run(_schedule())


def main():
    if len(sys.argv) < 2:
        json.dump({"error": "usage: twikit_guest.py <command> [args...]"}, sys.stdout)
        sys.exit(1)

    command = sys.argv[1]

    if command == "extract_cookies":
        ensure_package("browser_cookie3")

        try:
            result = extract_cookies()
            json.dump(result, sys.stdout)
            if "error" in result:
                sys.exit(1)
        except Exception as e:
            json.dump({"error": str(e)}, sys.stdout)
            sys.exit(1)

    elif command == "fetch_user_tweets":
        if len(sys.argv) < 4:
            json.dump(
                {"error": "usage: twikit_guest.py fetch_user_tweets <username> <limit>"},
                sys.stdout,
            )
            sys.exit(1)

        username = sys.argv[2]
        try:
            limit = int(sys.argv[3])
        except ValueError:
            json.dump({"error": f"invalid limit: {sys.argv[3]}"}, sys.stdout)
            sys.exit(1)

        if not os.path.exists(COOKIES_PATH):
            json.dump(
                {"error": "X cookies not found — sync cookies via the browser extension or provide them in Settings"},
                sys.stdout,
            )
            sys.exit(1)

        ensure_package("twikit")

        try:
            result = fetch_user_tweets(username, limit)
            json.dump(result, sys.stdout)
        except Exception as e:
            json.dump({"error": str(e)}, sys.stdout)
            sys.exit(1)
    elif command == "search_trending":
        if len(sys.argv) < 5:
            json.dump(
                {"error": "usage: twikit_guest.py search_trending <niches_json> <min_likes> <limit>"},
                sys.stdout,
            )
            sys.exit(1)

        niches_json = sys.argv[2]
        try:
            min_likes = int(sys.argv[3])
        except ValueError:
            json.dump({"error": f"invalid min_likes: {sys.argv[3]}"}, sys.stdout)
            sys.exit(1)
        try:
            limit = int(sys.argv[4])
        except ValueError:
            json.dump({"error": f"invalid limit: {sys.argv[4]}"}, sys.stdout)
            sys.exit(1)

        if not os.path.exists(COOKIES_PATH):
            json.dump(
                {"error": "X cookies not found — sync cookies via the browser extension or provide them in Settings"},
                sys.stdout,
            )
            sys.exit(1)

        period = sys.argv[5] if len(sys.argv) > 5 else "day"

        ensure_package("twikit")

        try:
            result = search_trending(niches_json, min_likes, limit, period)
            json.dump(result, sys.stdout)
        except Exception as e:
            json.dump({"error": str(e)}, sys.stdout)
            sys.exit(1)

    elif command == "upload_media":
        if len(sys.argv) < 3:
            json.dump(
                {"error": "usage: twikit_guest.py upload_media <file_path>"},
                sys.stdout,
            )
            sys.exit(1)

        file_path = sys.argv[2]

        if not os.path.exists(COOKIES_PATH):
            json.dump(
                {"error": "X cookies not found — sync cookies via the browser extension or provide them in Settings"},
                sys.stdout,
            )
            sys.exit(1)

        ensure_package("twikit")

        try:
            result = upload_media(file_path)
            json.dump(result, sys.stdout)
        except Exception as e:
            json.dump({"error": str(e)}, sys.stdout)
            sys.exit(1)

    elif command == "create_tweet":
        if len(sys.argv) < 3:
            json.dump(
                {"error": "usage: twikit_guest.py create_tweet <text> [reply_to_id] [media_ids_json]"},
                sys.stdout,
            )
            sys.exit(1)

        text = sys.argv[2]
        reply_to = sys.argv[3] if len(sys.argv) > 3 and sys.argv[3] else None
        media_ids_json = sys.argv[4] if len(sys.argv) > 4 else None

        if not os.path.exists(COOKIES_PATH):
            json.dump(
                {"error": "X cookies not found — sync cookies via the browser extension or provide them in Settings"},
                sys.stdout,
            )
            sys.exit(1)

        ensure_package("twikit")

        try:
            result = create_tweet(text, reply_to, media_ids_json)
            json.dump(result, sys.stdout)
        except Exception as e:
            json.dump({"error": str(e)}, sys.stdout)
            sys.exit(1)

    elif command == "create_quote_tweet":
        if len(sys.argv) < 4:
            json.dump(
                {"error": "usage: twikit_guest.py create_quote_tweet <text> <quote_tweet_id>"},
                sys.stdout,
            )
            sys.exit(1)

        text = sys.argv[2]
        quote_tweet_id = sys.argv[3]

        if not os.path.exists(COOKIES_PATH):
            json.dump(
                {"error": "X cookies not found — sync cookies via the browser extension or provide them in Settings"},
                sys.stdout,
            )
            sys.exit(1)

        ensure_package("twikit")

        try:
            result = create_quote_tweet(text, quote_tweet_id)
            json.dump(result, sys.stdout)
        except Exception as e:
            json.dump({"error": str(e)}, sys.stdout)
            sys.exit(1)

    elif command == "schedule_tweet":
        if len(sys.argv) < 4:
            json.dump(
                {"error": "usage: twikit_guest.py schedule_tweet <text> <scheduled_at_unix> [media_ids_json]"},
                sys.stdout,
            )
            sys.exit(1)

        text = sys.argv[2]
        scheduled_at_unix = sys.argv[3]
        media_ids_json = sys.argv[4] if len(sys.argv) > 4 else None

        if not os.path.exists(COOKIES_PATH):
            json.dump(
                {"error": "X cookies not found — sync cookies via the browser extension or provide them in Settings"},
                sys.stdout,
            )
            sys.exit(1)

        ensure_package("twikit")

        try:
            result = schedule_tweet(text, scheduled_at_unix, media_ids_json)
            json.dump(result, sys.stdout)
        except Exception as e:
            json.dump({"error": str(e)}, sys.stdout)
            sys.exit(1)

    elif command == "schedule_quote_tweet":
        if len(sys.argv) < 5:
            json.dump(
                {"error": "usage: twikit_guest.py schedule_quote_tweet <text> <quote_tweet_id> <scheduled_at_unix>"},
                sys.stdout,
            )
            sys.exit(1)

        text = sys.argv[2]
        quote_tweet_id = sys.argv[3]
        scheduled_at_unix = sys.argv[4]

        if not os.path.exists(COOKIES_PATH):
            json.dump(
                {"error": "X cookies not found — sync cookies via the browser extension or provide them in Settings"},
                sys.stdout,
            )
            sys.exit(1)

        ensure_package("twikit")

        try:
            result = schedule_quote_tweet(text, quote_tweet_id, scheduled_at_unix)
            json.dump(result, sys.stdout)
        except Exception as e:
            json.dump({"error": str(e)}, sys.stdout)
            sys.exit(1)

    else:
        json.dump({"error": f"unknown command: {command}"}, sys.stdout)
        sys.exit(1)


if __name__ == "__main__":
    main()
