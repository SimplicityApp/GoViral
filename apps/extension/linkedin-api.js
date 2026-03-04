// GoViral — LinkedIn Voyager API helpers (injected into page MAIN world)
// All functions are self-contained (no imports). Use goviralXxx prefix to avoid conflicts.
// Endpoint paths and parsing logic aligned with the linkitin Python library.

function goviralGetCsrfToken() {
  const match = document.cookie.match(/JSESSIONID=([^;]+)/);
  if (!match) return null;
  return match[1].replace(/^"/, "").replace(/"$/, "");
}

function goviralLinkedInHeaders() {
  const token = goviralGetCsrfToken();
  return {
    "csrf-token": token || "",
    accept: "application/vnd.linkedin.normalized+json+2.1",
    "x-restli-protocol-version": "2.0.0",
  };
}

// --- Parsing (mirrors linkitin/feed.py _parse_feed_response) ---

const GOVIRAL_POST_TYPES = [
  "com.linkedin.voyager.feed.render.UpdateV2",
  "com.linkedin.voyager.feed.Update",
  "com.linkedin.voyager.dash.feed.Update",
  "com.linkedin.voyager.identity.profile.ProfileUpdate",
];

function goviralIsPostEntity(entityType) {
  return GOVIRAL_POST_TYPES.some((pt) => entityType.includes(pt));
}

function goviralExtractText(entity) {
  // commentary.text.text (most common)
  const commentary = entity.commentary;
  if (commentary && typeof commentary === "object") {
    const text = commentary.text;
    if (typeof text === "object" && text !== null) return text.text || "";
    if (typeof text === "string") return text;
  }
  // content → TextComponent
  const content = entity.content;
  if (content && typeof content === "object") {
    const tc =
      content["com.linkedin.voyager.feed.render.TextComponent"];
    if (tc && typeof tc === "object") {
      const t = tc.text;
      if (typeof t === "object" && t !== null) return t.text || "";
    }
  }
  // specificContent (older format)
  const specific = entity.specificContent;
  if (specific && typeof specific === "object") {
    const sc = specific["com.linkedin.ugc.ShareContent"];
    if (sc && typeof sc === "object") {
      const scText = sc.shareCommentary;
      if (scText && typeof scText === "object") return scText.text || "";
    }
  }
  // header.text.text
  const header = entity.header;
  if (header && typeof header === "object") {
    const ht = header.text;
    if (typeof ht === "object" && ht !== null) return ht.text || "";
  }
  return "";
}

function goviralExtractSocialCounts(urn, entity, socialCounts, socialDetails) {
  let likes = 0, comments = 0, reposts = 0, impressions = 0;

  // Try socialDetail on entity itself
  const sd = entity.socialDetail;
  if (sd && typeof sd === "object") {
    likes = sd.reactionSummary?.count || sd.totalReactionCount || 0;
    comments = sd.commentSummary?.count || sd.totalComments || 0;
    reposts = sd.totalShares || 0;
  }

  // Try SocialActivityCounts from included
  const countEntity = socialCounts[urn] || null;
  if (countEntity) {
    likes = likes || countEntity.numLikes || 0;
    comments = comments || countEntity.numComments || 0;
    reposts = reposts || countEntity.numShares || 0;
    impressions = countEntity.numImpressions || 0;
  }

  // Try SocialDetail from included
  const detailEntity = socialDetails[urn] || null;
  if (detailEntity) {
    likes = likes || detailEntity.reactionSummary?.count || 0;
    comments = comments || detailEntity.commentSummary?.count || 0;
    reposts = reposts || detailEntity.totalShares || 0;
  }

  return { likes, comments, reposts, impressions };
}

function goviralExtractAuthor(entity, profiles) {
  const actor = entity.actor;
  if (actor && typeof actor === "object") {
    let name = "";
    if (typeof actor.name === "object" && actor.name !== null) {
      name = actor.name.text || "";
    } else if (typeof actor.name === "string") {
      name = actor.name;
    }
    const actorUrn = actor.urn || "";
    const profile = profiles[actorUrn] || null;
    const username =
      (profile && profile.publicIdentifier) ||
      (actor.navigationUrl
        ? actor.navigationUrl.replace(/.*\/in\//, "").replace(/\/.*/, "")
        : null);
    return { author_name: name, author_username: username || null };
  }
  // Fallback: author URN reference
  const authorUrn = entity.author || "";
  if (typeof authorUrn === "string" && profiles[authorUrn]) {
    const p = profiles[authorUrn];
    return {
      author_name: [p.firstName, p.lastName].filter(Boolean).join(" "),
      author_username: p.publicIdentifier || null,
    };
  }
  return { author_name: null, author_username: null };
}

function goviralExtractCreatedAt(entity) {
  const ms =
    (entity.updateMetadata && entity.updateMetadata.publishedAt) ||
    entity.createdAt ||
    null;
  return ms ? new Date(ms).toISOString() : null;
}

// Main parser: iterate included entities like linkitin does
function goviralParseNormalizedPosts(data, limit) {
  const included = data.included || [];
  const profiles = {};
  const socialCounts = {};
  const socialDetails = {};

  // First pass: index profiles, social counts, social details
  for (const entity of included) {
    const etype = entity.$type || "";
    const eurn = entity.entityUrn || entity.urn || "";
    if (etype.includes("MiniProfile") || etype.includes("Profile")) {
      profiles[eurn] = entity;
    } else if (etype.includes("SocialActivityCounts")) {
      const parts = eurn.split("fsd_socialActivityCounts:");
      if (parts.length === 2) socialCounts[parts[1]] = entity;
      socialCounts[eurn] = entity;
    } else if (etype.includes("SocialDetail")) {
      const threadId = entity.threadId || eurn;
      socialDetails[threadId] = entity;
    }
  }

  // Second pass: extract posts
  const posts = [];
  for (const entity of included) {
    const etype = entity.$type || "";
    if (!goviralIsPostEntity(etype)) continue;

    const urn = entity.entityUrn || entity.urn || "";
    if (!urn) continue;

    const text = goviralExtractText(entity);
    if (!text) continue;

    const { likes, comments, reposts, impressions } =
      goviralExtractSocialCounts(urn, entity, socialCounts, socialDetails);
    const { author_name, author_username } =
      goviralExtractAuthor(entity, profiles);
    const posted_at = goviralExtractCreatedAt(entity);

    // Extract activity ID as platform post ID
    const activityMatch = urn.match(/activity:(\d+)/);
    const platform_post_id = activityMatch ? activityMatch[1] : urn;

    posts.push({
      platform_post_id,
      content: text,
      likes,
      reposts,
      comments,
      impressions,
      posted_at,
      author_name,
      author_username,
      niche_tags: [],
    });

    if (limit && posts.length >= limit) break;
  }

  return posts;
}

// --- API functions ---

async function goviralFetchMyPosts(count = 20) {
  try {
    const headers = goviralLinkedInHeaders();

    // Use the same endpoint as linkitin: /voyager/api/identity/profileUpdatesV2
    const url =
      `/voyager/api/identity/profileUpdatesV2` +
      `?q=memberShareFeed&moduleKey=member-shares:phone` +
      `&count=${Math.min(count, 50)}&start=0`;

    const resp = await fetch(url, { credentials: "include", headers });
    if (!resp.ok) {
      return { error: `Failed to fetch my posts: ${resp.status}` };
    }
    const data = await resp.json();
    return { posts: goviralParseNormalizedPosts(data, count) };
  } catch (err) {
    return { error: String(err) };
  }
}

async function goviralFetchFeed(count = 20) {
  try {
    const headers = goviralLinkedInHeaders();

    // Use the same endpoint as linkitin: /voyager/api/feed/dash/feedUpdates
    const url =
      `/voyager/api/feed/dash/feedUpdates` +
      `?q=DECORATED_FEED&count=${Math.min(count, 50)}&start=0`;

    const resp = await fetch(url, { credentials: "include", headers });
    if (!resp.ok) {
      return { error: `Failed to fetch feed: ${resp.status}` };
    }
    const data = await resp.json();
    return { posts: goviralParseNormalizedPosts(data, count) };
  } catch (err) {
    return { error: String(err) };
  }
}

async function goviralSearchPosts(keywords, count = 20) {
  try {
    const headers = goviralLinkedInHeaders();

    // Same endpoint as linkitin: /voyager/api/search/dash/clusters
    const url =
      `/voyager/api/search/dash/clusters` +
      `?q=all&keywords=${encodeURIComponent(keywords)}` +
      `&type=CONTENT&count=${count}`;

    const resp = await fetch(url, { credentials: "include", headers });
    if (!resp.ok) {
      return { error: `Failed to search posts: ${resp.status}` };
    }
    const data = await resp.json();
    return { posts: goviralParseNormalizedPosts(data, count) };
  } catch (err) {
    return { error: String(err) };
  }
}

async function goviralFetchTrending(keywords, count = 20) {
  try {
    const headers = goviralLinkedInHeaders();

    const url =
      `/voyager/api/search/dash/clusters` +
      `?q=all&keywords=${encodeURIComponent(keywords)}` +
      `&type=CONTENT&count=${count}&origin=FACETED_SEARCH`;

    const resp = await fetch(url, { credentials: "include", headers });
    if (!resp.ok) {
      return { error: `Failed to fetch trending: ${resp.status}` };
    }
    const data = await resp.json();

    const posts = goviralParseNormalizedPosts(data, count);
    const niche_tags = keywords
      ? keywords.split(/[\s,]+/).filter(Boolean)
      : [];
    for (const post of posts) {
      post.niche_tags = niche_tags;
    }
    return { posts };
  } catch (err) {
    return { error: String(err) };
  }
}
