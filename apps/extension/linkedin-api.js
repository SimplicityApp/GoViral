// GoViral — LinkedIn Voyager API helpers (injected into page MAIN world)
// Uses window assignments so re-injection is safe (no const/let redeclaration errors).
// Endpoint paths and parsing logic aligned with the linkitin Python library.

window.goviralGetCsrfToken = function () {
  var match = document.cookie.match(/JSESSIONID=([^;]+)/);
  if (!match) return null;
  return match[1].replace(/^"/, "").replace(/"$/, "");
};

window.goviralLinkedInHeaders = function () {
  var token = window.goviralGetCsrfToken();
  var tz = Intl.DateTimeFormat().resolvedOptions().timeZone || "America/New_York";
  var offsetHours = -(new Date().getTimezoneOffset() / 60);
  return {
    "csrf-token": token || "",
    accept: "application/vnd.linkedin.normalized+json+2.1",
    "x-restli-protocol-version": "2.0.0",
    "x-li-lang": "en_US",
    "x-li-track": JSON.stringify({
      clientVersion: "1.13.8735",
      mpVersion: "1.13.8735",
      osName: "web",
      timezoneOffset: offsetHours,
      timezone: tz,
      deviceFormFactor: "DESKTOP",
      mpName: "voyager-web",
      displayDensity: window.devicePixelRatio || 2,
      displayWidth: window.screen.width || 1920,
      displayHeight: window.screen.height || 1080,
    }),
    "Accept-Language": "en-US,en;q=0.9",
  };
};

// --- Parsing (mirrors linkitin/feed.py _parse_feed_response) ---

window._GOVIRAL_POST_TYPES = [
  "com.linkedin.voyager.feed.render.UpdateV2",
  "com.linkedin.voyager.feed.Update",
  "com.linkedin.voyager.dash.feed.Update",
  "com.linkedin.voyager.identity.profile.ProfileUpdate",
];

window.goviralIsPostEntity = function (entityType) {
  return window._GOVIRAL_POST_TYPES.some(function (pt) {
    return entityType.indexOf(pt) !== -1;
  });
};

window.goviralExtractText = function (entity) {
  var commentary = entity.commentary;
  if (commentary && typeof commentary === "object") {
    var text = commentary.text;
    if (typeof text === "object" && text !== null) return text.text || "";
    if (typeof text === "string") return text;
  }
  var content = entity.content;
  if (content && typeof content === "object") {
    var tc = content["com.linkedin.voyager.feed.render.TextComponent"];
    if (tc && typeof tc === "object") {
      var t = tc.text;
      if (typeof t === "object" && t !== null) return t.text || "";
    }
  }
  var specific = entity.specificContent;
  if (specific && typeof specific === "object") {
    var sc = specific["com.linkedin.ugc.ShareContent"];
    if (sc && typeof sc === "object") {
      var scText = sc.shareCommentary;
      if (scText && typeof scText === "object") return scText.text || "";
    }
  }
  var header = entity.header;
  if (header && typeof header === "object") {
    var ht = header.text;
    if (typeof ht === "object" && ht !== null) return ht.text || "";
  }
  return "";
};

window.goviralExtractSocialCounts = function (urn, entity, socialCounts, socialDetails) {
  var likes = 0, comments = 0, reposts = 0, impressions = 0;

  var sd = entity.socialDetail;
  if (sd && typeof sd === "object") {
    likes = (sd.reactionSummary && sd.reactionSummary.count) || sd.totalReactionCount || 0;
    comments = (sd.commentSummary && sd.commentSummary.count) || sd.totalComments || 0;
    reposts = sd.totalShares || 0;
  }

  var countEntity = socialCounts[urn] || null;
  if (countEntity) {
    likes = likes || countEntity.numLikes || 0;
    comments = comments || countEntity.numComments || 0;
    reposts = reposts || countEntity.numShares || 0;
    impressions = countEntity.numImpressions || 0;
  }

  var detailEntity = socialDetails[urn] || null;
  if (detailEntity) {
    likes = likes || (detailEntity.reactionSummary && detailEntity.reactionSummary.count) || 0;
    comments = comments || (detailEntity.commentSummary && detailEntity.commentSummary.count) || 0;
    reposts = reposts || detailEntity.totalShares || 0;
  }

  return { likes: likes, comments: comments, reposts: reposts, impressions: impressions };
};

window.goviralExtractAuthor = function (entity, profiles) {
  var actor = entity.actor;
  if (actor && typeof actor === "object") {
    var name = "";
    if (typeof actor.name === "object" && actor.name !== null) {
      name = actor.name.text || "";
    } else if (typeof actor.name === "string") {
      name = actor.name;
    }
    var actorUrn = actor.urn || "";
    var profile = profiles[actorUrn] || null;
    var username =
      (profile && profile.publicIdentifier) ||
      (actor.navigationUrl
        ? actor.navigationUrl.replace(/.*\/in\//, "").replace(/\/.*/, "")
        : null);
    return { author_name: name, author_username: username || null };
  }
  var authorUrn = entity.author || "";
  if (typeof authorUrn === "string" && profiles[authorUrn]) {
    var p = profiles[authorUrn];
    return {
      author_name: [p.firstName, p.lastName].filter(Boolean).join(" "),
      author_username: p.publicIdentifier || null,
    };
  }
  return { author_name: null, author_username: null };
};

window.goviralExtractCreatedAt = function (entity) {
  var ms =
    (entity.updateMetadata && entity.updateMetadata.publishedAt) ||
    entity.createdAt ||
    null;
  return ms ? new Date(ms).toISOString() : null;
};

window.goviralParseNormalizedPosts = function (data, limit) {
  var included = data.included || [];
  var profiles = {};
  var socialCounts = {};
  var socialDetails = {};

  for (var i = 0; i < included.length; i++) {
    var e = included[i];
    var etype = e.$type || "";
    var eurn = e.entityUrn || e.urn || "";
    if (etype.indexOf("MiniProfile") !== -1 || etype.indexOf("Profile") !== -1) {
      profiles[eurn] = e;
    } else if (etype.indexOf("SocialActivityCounts") !== -1) {
      var parts = eurn.split("fsd_socialActivityCounts:");
      if (parts.length === 2) socialCounts[parts[1]] = e;
      socialCounts[eurn] = e;
    } else if (etype.indexOf("SocialDetail") !== -1) {
      var threadId = e.threadId || eurn;
      socialDetails[threadId] = e;
    }
  }

  var posts = [];
  for (var j = 0; j < included.length; j++) {
    var entity = included[j];
    var et = entity.$type || "";
    if (!window.goviralIsPostEntity(et)) continue;

    var urn = entity.entityUrn || entity.urn || "";
    if (!urn) continue;

    var text = window.goviralExtractText(entity);
    if (!text) continue;

    var counts = window.goviralExtractSocialCounts(urn, entity, socialCounts, socialDetails);
    var author = window.goviralExtractAuthor(entity, profiles);
    var posted_at = window.goviralExtractCreatedAt(entity);

    var activityMatch = urn.match(/activity:(\d+)/);
    var platform_post_id = activityMatch ? activityMatch[1] : urn;

    posts.push({
      platform_post_id: platform_post_id,
      content: text,
      likes: counts.likes,
      reposts: counts.reposts,
      comments: counts.comments,
      impressions: counts.impressions,
      posted_at: posted_at,
      author_name: author.author_name,
      author_username: author.author_username,
      niche_tags: [],
    });

    if (limit && posts.length >= limit) break;
  }

  return posts;
};

// --- API functions ---

window.goviralFetchMyPosts = async function (count) {
  count = count || 20;
  try {
    var headers = window.goviralLinkedInHeaders();
    var url =
      "/voyager/api/identity/profileUpdatesV2" +
      "?q=memberShareFeed&moduleKey=member-shares:phone" +
      "&count=" + Math.min(count, 50) + "&start=0";

    var resp = await fetch(url, { credentials: "include", headers: headers });
    if (!resp.ok) {
      var body = await resp.text().catch(function () { return ""; });
      return { error: "Failed to fetch my posts: " + resp.status + " " + body.slice(0, 300) };
    }
    var data = await resp.json();
    return { posts: window.goviralParseNormalizedPosts(data, count) };
  } catch (err) {
    return { error: String(err) };
  }
};

window.goviralFetchFeed = async function (count) {
  count = count || 20;
  try {
    var headers = window.goviralLinkedInHeaders();
    var url =
      "/voyager/api/feed/dash/feedUpdates" +
      "?q=DECORATED_FEED&count=" + Math.min(count, 50) + "&start=0";

    var resp = await fetch(url, { credentials: "include", headers: headers });
    if (!resp.ok) {
      var body = await resp.text().catch(function () { return ""; });
      return { error: "Failed to fetch feed: " + resp.status + " " + body.slice(0, 300) };
    }
    var data = await resp.json();
    return { posts: window.goviralParseNormalizedPosts(data, count) };
  } catch (err) {
    return { error: String(err) };
  }
};

window.goviralSearchPosts = async function (keywords, count) {
  count = count || 20;
  try {
    var headers = window.goviralLinkedInHeaders();
    var url =
      "/voyager/api/search/dash/clusters" +
      "?q=all&keywords=" + encodeURIComponent(keywords) +
      "&type=CONTENT&count=" + count;

    var resp = await fetch(url, { credentials: "include", headers: headers });
    if (!resp.ok) {
      var body = await resp.text().catch(function () { return ""; });
      return { error: "Failed to search posts: " + resp.status + " " + body.slice(0, 300) };
    }
    var data = await resp.json();
    return { posts: window.goviralParseNormalizedPosts(data, count) };
  } catch (err) {
    return { error: String(err) };
  }
};

window.goviralFetchTrending = async function (keywords, count) {
  count = count || 20;
  try {
    var headers = window.goviralLinkedInHeaders();
    var url =
      "/voyager/api/search/dash/clusters" +
      "?q=all&keywords=" + encodeURIComponent(keywords) +
      "&type=CONTENT&count=" + count + "&origin=FACETED_SEARCH";

    var resp = await fetch(url, { credentials: "include", headers: headers });
    if (!resp.ok) {
      var body = await resp.text().catch(function () { return ""; });
      return { error: "Failed to fetch trending: " + resp.status + " " + body.slice(0, 300) };
    }
    var data = await resp.json();

    var posts = window.goviralParseNormalizedPosts(data, count);
    var niche_tags = keywords
      ? keywords.split(/[\s,]+/).filter(Boolean)
      : [];
    for (var i = 0; i < posts.length; i++) {
      posts[i].niche_tags = niche_tags;
    }
    return { posts: posts };
  } catch (err) {
    return { error: String(err) };
  }
};
