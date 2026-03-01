---
name: content-workflow
description: End-to-end viral content workflow — fetch trending posts, build a persona profile, generate viral content variations, preview, and optionally post to X or LinkedIn. Use when the user wants to create viral content or asks for a content generation workflow.
allowed-tools: Bash, Read, Grep, Glob, Write
---

# Viral Content Workflow

This skill orchestrates the full GoViral content pipeline: discover trending content, analyze the user's writing style, generate viral variations, and post.

## Workflow Steps

### Step 1: Identify the target platform

Ask the user which platform they want to create content for:
- **X/Twitter** — uses the x-post skill's commands
- **LinkedIn** — uses the linkedin-post skill's commands
- **Both** — run the workflow for each platform

### Step 2: Fetch the user's recent posts (for persona)

Build a writing style profile from the user's existing posts.

**For X:**
```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py fetch_user_tweets <username> 30
```

**For LinkedIn:**
```bash
echo '{"action": "get_my_posts", "limit": 30}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

Collect at least 15-30 posts for a good persona analysis. If the user has fewer posts, use whatever is available.

### Step 3: Analyze persona

Using the fetched posts, analyze the user's writing style. Look for:
- **Tone**: casual, professional, witty, provocative, etc.
- **Typical length**: short punchy tweets vs long-form threads
- **Common themes**: recurring topics they post about
- **Structural patterns**: threads, single posts, questions, lists, stories
- **Engagement patterns**: what types of their posts get the most likes/retweets
- **Unique quirks**: distinctive habits (emoji usage, hashtag style, catchphrases)

Summarize the persona in 2-3 paragraphs. This will guide content generation.

### Step 4: Discover trending content

Find high-performing posts in the user's niche(s).

Ask the user for their niche topics (e.g., "AI", "startups", "developer tools"). Then:

**For X:**
```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py search_trending '["niche1", "niche2"]' 100 10 week
```

**For LinkedIn:**
```bash
echo '{"action": "get_trending_posts", "topic": "niche1", "period": "past-week", "limit": 10}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

### Step 5: Generate content variations

For each trending post worth reworking, generate 2-3 variations that:
1. **Match the user's voice** (based on Step 3 persona analysis)
2. **Preserve the viral mechanic** (what made the original go viral — controversy, relatability, surprising data, etc.)
3. **Add original value** — don't just paraphrase; add a unique angle, personal experience, or contrarian take

For each variation, provide:
- The ready-to-post content
- Which viral mechanic it uses
- A confidence score (1-10) for viral potential
- Whether an image would boost it

Present the options in a clear numbered format.

### Step 6: Preview and confirm

**CRITICAL**: Show the user ALL generated content before posting anything.

Present each option with:
- The full post text
- Target platform
- Viral mechanic explanation
- Confidence score

Ask the user to:
1. Pick which variation(s) to post
2. Approve the exact text (they may want to edit)
3. Choose to post now or schedule for later

### Step 7: Post (only with explicit confirmation)

After the user picks and approves content:

**For X — immediate post:**
```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py create_tweet '<approved_text>'
```

**For X — quote tweet (if riffing on a specific trending tweet):**
```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py create_quote_tweet '<commentary>' <original_tweet_id>
```

**For X — scheduled:**
```bash
"$HOME/.goviral/venv/bin/python3" internal/platform/x/scripts/twikit_guest.py schedule_tweet '<approved_text>' <unix_timestamp>
```

**For LinkedIn — immediate post:**
```bash
echo '{"action": "create_post", "text": "<approved_text>", "visibility": "PUBLIC"}' | "$HOME/.goviral/venv/bin/python3" internal/platform/linkedin/scripts/linkitin_bridge.py
```

## Quick Mode

If the user just says "make me a viral post about X", you can streamline:
1. Skip persona analysis if you've already done it in this session
2. Search trending for the topic
3. Generate 3 variations
4. Present for approval
5. Post on confirmation

## Notes

- Always check that cookies exist before operations. If missing, direct user to `/x-setup` or `/linkedin-setup`
- The persona analysis only needs to happen once per session. Cache the results mentally for subsequent content generation
- For thread creation on X, split long content into tweet-sized chunks (under 280 chars each) and post as a reply chain
- LinkedIn posts can be longer (up to ~3000 chars). Optimize for readability with line breaks and formatting
