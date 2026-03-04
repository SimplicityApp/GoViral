// GoViral — LinkedIn Voyager API helpers (injected into page MAIN world)
// All functions are self-contained (no imports). Use goviralXxx prefix to avoid conflicts.

function goviralGetCsrfToken() {
  const match = document.cookie.match(/JSESSIONID=([^;]+)/);
  if (!match) return null;
  // Strip surrounding quotes if present
  return match[1].replace(/^"/, "").replace(/"$/, "");
}

function goviralLinkedInHeaders() {
  const token = goviralGetCsrfToken();
  return {
    "csrf-token": token || "",
    "accept": "application/vnd.linkedin.normalized+json+2.1",
    "x-restli-protocol-version": "2.0.0",
  };
}

// Parse LinkedIn normalized JSON: resolve included entities and extract posts.
function goviralParseNormalizedPosts(data) {
  const included = data.included || [];
  // Build a lookup map by entityUrn
  const entityMap = {};
  for (const entity of included) {
    if (entity.entityUrn) {
      entityMap[entity.entityUrn] = entity;
    }
  }

  const posts = [];

  // Elements may be in data.elements or data.data?.elements
  const elements =
    (data.data && data.data.elements) ||
    data.elements ||
    [];

  for (const el of elements) {
    try {
      // Resolve the actual post entity if needed
      const urn = el.updateMetadata?.urn || el.entityUrn || el["$id"] || null;

      // Extract content text
      let content = "";
      const commentary =
        el.commentary ||
        el.content?.commentary ||
        (el.value && el.value.com$linkedin$voyager$feed$render$UpdateV2 &&
          el.value.com$linkedin$voyager$feed$render$UpdateV2.commentary) ||
        null;
      if (commentary && commentary.text && commentary.text.text) {
        content = commentary.text.text;
      }

      // Extract social stats
      const socialDetail =
        el.socialDetail ||
        el.value?.com$linkedin$voyager$feed$render$UpdateV2?.socialDetail ||
        null;
      const likes =
        socialDetail?.reactionSummary?.count ||
        el.numLikes ||
        0;
      const reposts =
        socialDetail?.totalShares ||
        el.numShares ||
        0;
      const comments =
        socialDetail?.commentSummary?.count ||
        el.numComments ||
        0;

      // Impressions (not always available in Voyager)
      const impressions = el.impressionCount || 0;

      // Post time
      const postedAtMs =
        el.updateMetadata?.publishedAt ||
        el.createdAt ||
        null;
      const posted_at = postedAtMs
        ? new Date(postedAtMs).toISOString()
        : null;

      // Platform post id
      const platform_post_id = urn
        ? urn.replace(/^urn:li:activity:/, "")
        : null;

      // Author info (for feed/search)
      let author_username = null;
      let author_name = null;
      const actor =
        el.actor ||
        el.value?.com$linkedin$voyager$feed$render$UpdateV2?.actor ||
        null;
      if (actor) {
        author_name =
          actor.name?.text ||
          actor.name ||
          null;
        // LinkedIn doesn't expose usernames directly; use publicIdentifier if present
        const actorUrn = actor.urn || "";
        const profileEntity = entityMap[actorUrn] || null;
        author_username =
          (profileEntity && profileEntity.publicIdentifier) ||
          null;
      }

      // Niche tags placeholder (populated by caller when relevant)
      const niche_tags = [];

      if (content || platform_post_id) {
        posts.push({
          platform_post_id,
          content,
          likes,
          reposts,
          comments,
          impressions,
          posted_at,
          author_username,
          author_name,
          niche_tags,
        });
      }
    } catch (_) {
      // Skip malformed entries
    }
  }

  return posts;
}

async function goviralFetchMyPosts(count = 20) {
  try {
    const headers = goviralLinkedInHeaders();

    // Step 1: Get own profile URN
    const meResp = await fetch("/voyager/api/me", {
      credentials: "include",
      headers,
    });
    if (!meResp.ok) {
      return { error: `Failed to fetch /voyager/api/me: ${meResp.status}` };
    }
    const meData = await meResp.json();

    // Extract miniProfile URN from the normalized response
    const included = meData.included || [];
    let profileUrn = null;
    for (const entity of included) {
      if (
        entity.$type === "com.linkedin.voyager.identity.shared.MiniProfile" ||
        (entity.entityUrn && entity.entityUrn.startsWith("urn:li:fs_miniProfile:"))
      ) {
        profileUrn = entity.entityUrn;
        break;
      }
    }
    // Fallback: top-level miniProfile
    if (!profileUrn && meData.data && meData.data.miniProfile) {
      profileUrn = meData.data.miniProfile;
    }

    if (!profileUrn) {
      return { error: "Could not determine profile URN from /voyager/api/me" };
    }

    // Step 2: Fetch posts for this profile
    const postsUrl =
      `/voyager/api/identity/dash/profileUpdatesByMemberProfile` +
      `?memberProfile=${encodeURIComponent(profileUrn)}` +
      `&q=memberProfile&moduleKey=creator_profile_all_updates_tab` +
      `&count=${count}`;

    const postsResp = await fetch(postsUrl, {
      credentials: "include",
      headers,
    });
    if (!postsResp.ok) {
      return { error: `Failed to fetch posts: ${postsResp.status}` };
    }
    const postsData = await postsResp.json();

    const posts = goviralParseNormalizedPosts(postsData);
    return { posts };
  } catch (err) {
    return { error: String(err) };
  }
}

async function goviralFetchFeed(count = 20) {
  try {
    const headers = goviralLinkedInHeaders();

    const resp = await fetch(
      `/voyager/api/feed/updates?q=FEED_UPDATES&count=${count}`,
      { credentials: "include", headers }
    );
    if (!resp.ok) {
      return { error: `Failed to fetch feed: ${resp.status}` };
    }
    const data = await resp.json();

    const posts = goviralParseNormalizedPosts(data);
    return { posts };
  } catch (err) {
    return { error: String(err) };
  }
}

async function goviralSearchPosts(keywords, count = 20) {
  try {
    const headers = goviralLinkedInHeaders();

    const url =
      `/voyager/api/search/dash/clusters` +
      `?q=all&keywords=${encodeURIComponent(keywords)}&type=CONTENT&count=${count}`;

    const resp = await fetch(url, { credentials: "include", headers });
    if (!resp.ok) {
      return { error: `Failed to search posts: ${resp.status}` };
    }
    const data = await resp.json();

    // Search results use clusters → items structure
    const posts = goviralParseSearchResults(data);
    return { posts };
  } catch (err) {
    return { error: String(err) };
  }
}

async function goviralFetchTrending(keywords, count = 20) {
  try {
    const headers = goviralLinkedInHeaders();

    // Sort by recency/engagement via origin=FACETED_SEARCH or similar
    const url =
      `/voyager/api/search/dash/clusters` +
      `?q=all&keywords=${encodeURIComponent(keywords)}&type=CONTENT` +
      `&count=${count}&origin=FACETED_SEARCH`;

    const resp = await fetch(url, { credentials: "include", headers });
    if (!resp.ok) {
      return { error: `Failed to fetch trending: ${resp.status}` };
    }
    const data = await resp.json();

    const posts = goviralParseSearchResults(data);
    // Tag posts with niche keywords
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

// Parse LinkedIn search clusters response into a flat posts array.
function goviralParseSearchResults(data) {
  const posts = [];
  const included = data.included || [];
  const entityMap = {};
  for (const entity of included) {
    if (entity.entityUrn) {
      entityMap[entity.entityUrn] = entity;
    }
  }

  const elements =
    (data.data && data.data.elements) || data.elements || [];

  for (const cluster of elements) {
    const items =
      cluster.items ||
      (cluster.data && cluster.data.items) ||
      [];

    for (const item of items) {
      try {
        const entityUrn =
          item.entityUrn ||
          item.targetUrn ||
          item.template?.entityUrn ||
          null;

        const entity = entityUrn ? entityMap[entityUrn] : null;

        // Title / headline text
        let content =
          item.title?.text ||
          item.headline?.text ||
          (entity && entity.commentary && entity.commentary.text && entity.commentary.text.text) ||
          "";

        // Subtitle may contain snippet
        if (!content && item.subTitle) {
          content = item.subTitle.text || "";
        }

        const platform_post_id = entityUrn
          ? entityUrn.replace(/^urn:li:activity:/, "")
          : null;

        // Author
        let author_name = null;
        let author_username = null;
        if (item.target && item.target.actor) {
          author_name = item.target.actor.name?.text || null;
        }
        if (entity && entity.actor) {
          author_name = author_name || entity.actor.name?.text || null;
          const actorUrn = entity.actor.urn || "";
          const profileEntity = entityMap[actorUrn] || null;
          author_username =
            (profileEntity && profileEntity.publicIdentifier) || null;
        }

        if (content || platform_post_id) {
          posts.push({
            platform_post_id,
            content,
            likes: 0,
            reposts: 0,
            comments: 0,
            impressions: 0,
            posted_at: null,
            author_username,
            author_name,
            niche_tags: [],
          });
        }
      } catch (_) {
        // Skip malformed entries
      }
    }
  }

  return posts;
}
