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

// --- DOM extraction (adapted from linkitin/chrome_data.py) ---

window.goviralExtractCodeEntities = function () {
  // Extract Voyager entities from <code id="bpr-guid-*"> elements that
  // LinkedIn server-renders into the page HTML.
  var codes = document.querySelectorAll('code[id^="bpr-guid-"]');
  var entities = [];
  for (var i = 0; i < codes.length; i++) {
    try {
      var d = JSON.parse(codes[i].textContent);
      if (d.included) {
        for (var j = 0; j < d.included.length; j++) {
          entities.push(d.included[j]);
        }
      }
    } catch (e) { /* skip non-JSON code elements */ }
  }
  return entities;
};

window.goviralExtractSearchPostsDOM = function (count) {
  // Extract posts from search/feed pages using the "Reaction button" anchor.
  // Adapted from linkitin/chrome_data.py _JS_EXTRACT_POSTS_FROM_DOM.
  var reactionBtns = document.querySelectorAll(
    'button[aria-label="Reaction button state: no reaction"]'
  );
  var results = [];
  var seen = {};

  for (var i = 0; i < reactionBtns.length; i++) {
    var btn = reactionBtns[i];

    // Walk up to card container, collecting data-urn along the way.
    var card = btn;
    var postUrn = "";
    for (var j = 0; j < 20; j++) {
      card = card.parentElement;
      if (!card) break;
      var dataUrn = ((card.getAttribute && card.getAttribute("data-urn")) || "").replace(/[/]+$/, "");
      if (!postUrn && dataUrn &&
          (dataUrn.indexOf("activity") >= 0 || dataUrn.indexOf("ugcPost") >= 0 ||
           dataUrn.indexOf("fsd_update") >= 0)) {
        postUrn = dataUrn;
      }
      if ((card.textContent || "").length > 300) break;
    }
    if (!card) continue;

    // Second pass: look for /feed/update/ or /posts/ links inside the card.
    if (!postUrn) {
      var links = card.querySelectorAll("a[href]");
      for (var l = 0; l < links.length; l++) {
        var href = links[l].getAttribute("href") || "";
        var hrefDecoded = href;
        try { hrefDecoded = decodeURIComponent(href); } catch(e) {}

        if (href.indexOf("/feed/update/") >= 0) {
          var hm = hrefDecoded.match(/(urn:li:[a-zA-Z0-9_]+:[^?&# ]+)/);
          if (hm) {
            var cUrn = hm[1];
            if (cUrn.indexOf("activity") >= 0 || cUrn.indexOf("ugcPost") >= 0 ||
                cUrn.indexOf("fsd_update") >= 0) {
              postUrn = cUrn;
              break;
            }
          }
        } else if (href.indexOf("/posts/") >= 0) {
          var am = hrefDecoded.match(/[^a-zA-Z]activity([0-9]{15,})/);
          if (am) {
            postUrn = "urn:li:activity:" + am[1];
            break;
          }
        }
      }
    }

    if (!postUrn) continue;  // skip posts where we couldn't extract a URN

    var fullText = (card.textContent || "").replace(/\s+/g, " ").trim();

    // Extract author name.
    var authorName = "";
    var m = fullText.match(/Feed post\s*(.+?)\s*[·•]\s*Following/);
    if (m) {
      authorName = m[1].trim();
    } else {
      var followBtn = card.querySelector('button[aria-label^="Follow "]');
      if (followBtn) {
        authorName = (followBtn.getAttribute("aria-label") || "").replace("Follow ", "").trim();
      }
    }

    if (!authorName || seen[authorName]) continue;
    seen[authorName] = true;

    // Extract post text after time marker.
    var postText = "";
    var tm = fullText.match(/\d+[dhwmo]\s*[·•]?\s*/);
    if (tm) {
      postText = fullText.substring(fullText.indexOf(tm[0]) + tm[0].length).trim();
    } else {
      var fi = fullText.indexOf("Following");
      if (fi < 0) fi = fullText.indexOf("Follow");
      if (fi >= 0) {
        postText = fullText.substring(fi + 9).trim();
      }
    }

    // Remove engagement + action buttons from end.
    postText = postText.replace(/\d[\d,]*\s*(reaction|comment|repost).*$/i, "").trim();
    postText = postText.replace(/Like\s*(Comment|Repost|Send|Share|Celebrate|Support|Love|Insightful|Funny).*$/i, "").trim();
    postText = postText.replace(/[…\.]{1,3}\s*more\s*$/i, "").trim();

    if (postText.length < 10) continue;

    // Extract engagement metrics.
    var spans = card.querySelectorAll("span");
    var likes = 0, comments = 0, reposts = 0;
    for (var s = 0; s < spans.length; s++) {
      var t = spans[s].textContent.trim();
      var em;
      if ((em = t.match(/^([\d,]+)\s*reaction/i))) {
        likes = parseInt(em[1].replace(/,/g, ""), 10) || 0;
      } else if ((em = t.match(/^([\d,]+)\s*comment/i))) {
        comments = parseInt(em[1].replace(/,/g, ""), 10) || 0;
      } else if ((em = t.match(/^([\d,]+)\s*repost/i))) {
        reposts = parseInt(em[1].replace(/,/g, ""), 10) || 0;
      }
    }

    var activityMatch = postUrn.match(/activity:(\d+)/);
    var platform_post_id = activityMatch ? activityMatch[1] : postUrn;

    results.push({
      platform_post_id: platform_post_id,
      content: postText.substring(0, 2000),
      likes: likes,
      comments: comments,
      reposts: reposts,
      impressions: 0,
      posted_at: null,
      author_name: authorName,
      author_username: null,
      niche_tags: [],
    });

    if (count && results.length >= count) break;
  }
  return results;
};

window.goviralExtractActivityPostsDOM = function (count) {
  // Fallback: scrape [data-urn] elements on the activity page.
  // Adapted from linkitin/chrome_data.py _JS_EXTRACT_ACTIVITY_POSTS.

  // First, click all "…more" buttons to reveal full text.
  var moreButtons = document.querySelectorAll("button.see-more");
  for (var b = 0; b < moreButtons.length; b++) moreButtons[b].click();

  var postElements = document.querySelectorAll("[data-urn]");
  var results = [];
  for (var i = 0; i < postElements.length; i++) {
    var el = postElements[i];
    var urn = el.getAttribute("data-urn") || "";
    if (urn.indexOf("activity") < 0) continue;

    var textEl = el.querySelector(
      ".update-components-text, .feed-shared-inline-show-more-text"
    );
    var text = textEl ? textEl.textContent.trim() : "";
    if (!text) continue;
    // Strip residual "…more" / "Show less" button text.
    text = text.replace(/\u2026more\s*$/g, "").replace(/Show less\s*$/g, "").trim();

    var likes = 0, comments = 0, reposts = 0;
    var countSpans = el.querySelectorAll(
      ".social-details-social-counts span[aria-hidden='true']"
    );
    for (var s = 0; s < countSpans.length; s++) {
      var t = countSpans[s].textContent.trim();
      var m;
      if ((m = t.match(/^([\d,]+)\s*comment/i))) {
        comments = parseInt(m[1].replace(/,/g, ""), 10);
      } else if ((m = t.match(/^([\d,]+)\s*repost/i))) {
        reposts = parseInt(m[1].replace(/,/g, ""), 10);
      } else if ((m = t.match(/^([\d,]+)$/))) {
        likes = parseInt(m[1].replace(/,/g, ""), 10);
      }
    }
    if (likes === 0) {
      var reactBtn = el.querySelector(
        "button.social-details-social-counts__reactions-count"
      );
      if (reactBtn) {
        var rm = reactBtn.textContent.trim().match(/([\d,]+)/);
        if (rm) likes = parseInt(rm[1].replace(/,/g, ""), 10);
      }
    }

    var activityMatch = urn.match(/activity:(\d+)/);
    var platform_post_id = activityMatch ? activityMatch[1] : urn;

    results.push({
      platform_post_id: platform_post_id,
      content: text.substring(0, 2000),
      likes: likes,
      comments: comments,
      reposts: reposts,
      impressions: 0,
      posted_at: null,
      author_name: null,
      author_username: null,
      niche_tags: [],
    });

    if (count && results.length >= count) break;
  }
  return results;
};

// --- API functions ---

window.goviralFetchMyPosts = async function (count) {
  count = count || 20;

  // 1. Try <code> entity extraction (works when page has server-rendered data).
  try {
    var codeEntities = window.goviralExtractCodeEntities();
    var hasUpdates = codeEntities.some(function (e) {
      return (e.$type || "").indexOf("Update") !== -1;
    });
    if (hasUpdates) {
      var posts = window.goviralParseNormalizedPosts({ included: codeEntities }, count);
      if (posts.length > 0) return { posts: posts };
    }
  } catch (e) { /* fall through */ }

  // 2. Try [data-urn] DOM extraction (activity page with client-rendered posts).
  try {
    var domPosts = window.goviralExtractActivityPostsDOM(count);
    if (domPosts.length > 0) return { posts: domPosts };
  } catch (e) { /* fall through */ }

  // 3. Fall back to REST API with pagination (kept as last resort).
  try {
    var headers = window.goviralLinkedInHeaders();
    var allPosts = [];
    var pageSize = Math.min(count, 50);
    var start = 0;
    var maxPages = Math.ceil(count / pageSize);
    for (var page = 0; page < maxPages; page++) {
      var url =
        "/voyager/api/identity/profileUpdatesV2" +
        "?q=memberShareFeed&moduleKey=member-shares:phone" +
        "&count=" + pageSize + "&start=" + start;

      var resp = await fetch(url, { credentials: "include", headers: headers });
      if (!resp.ok) {
        if (allPosts.length > 0) break;
        var body = await resp.text().catch(function () { return ""; });
        return { error: "Failed to fetch my posts: " + resp.status + " " + body.slice(0, 300) };
      }
      var data = await resp.json();
      var pagePosts = window.goviralParseNormalizedPosts(data, 0);
      if (pagePosts.length === 0) break;
      for (var pi = 0; pi < pagePosts.length; pi++) allPosts.push(pagePosts[pi]);
      if (allPosts.length >= count) break;
      start += pageSize;
    }
    return { posts: allPosts.slice(0, count) };
  } catch (err) {
    return { error: String(err) };
  }
};

window.goviralFetchFeed = async function (count) {
  count = count || 20;

  // 1. Try <code> entity extraction.
  try {
    var codeEntities = window.goviralExtractCodeEntities();
    var hasUpdates = codeEntities.some(function (e) {
      return (e.$type || "").indexOf("Update") !== -1;
    });
    if (hasUpdates) {
      var posts = window.goviralParseNormalizedPosts({ included: codeEntities }, count);
      if (posts.length > 0) return { posts: posts };
    }
  } catch (e) { /* fall through */ }

  // 2. Try reaction-button DOM extraction.
  try {
    var domPosts = window.goviralExtractSearchPostsDOM(count);
    if (domPosts.length > 0) return { posts: domPosts };
  } catch (e) { /* fall through */ }

  // 3. Fall back to REST API with pagination.
  try {
    var headers = window.goviralLinkedInHeaders();
    var allPosts = [];
    var pageSize = Math.min(count, 50);
    var start = 0;
    var maxPages = Math.ceil(count / pageSize);
    for (var page = 0; page < maxPages; page++) {
      var url =
        "/voyager/api/feed/dash/feedUpdates" +
        "?q=DECORATED_FEED&count=" + pageSize + "&start=" + start;

      var resp = await fetch(url, { credentials: "include", headers: headers });
      if (!resp.ok) {
        if (allPosts.length > 0) break;
        var body = await resp.text().catch(function () { return ""; });
        return { error: "Failed to fetch feed: " + resp.status + " " + body.slice(0, 300) };
      }
      var data = await resp.json();
      var pagePosts = window.goviralParseNormalizedPosts(data, 0);
      if (pagePosts.length === 0) break;
      for (var pi = 0; pi < pagePosts.length; pi++) allPosts.push(pagePosts[pi]);
      if (allPosts.length >= count) break;
      start += pageSize;
    }
    return { posts: allPosts.slice(0, count) };
  } catch (err) {
    return { error: String(err) };
  }
};

window.goviralSearchPosts = async function (keywords, count) {
  count = count || 20;

  // 1. Try <code> entity extraction.
  try {
    var codeEntities = window.goviralExtractCodeEntities();
    var hasUpdates = codeEntities.some(function (e) {
      return (e.$type || "").indexOf("Update") !== -1;
    });
    if (hasUpdates) {
      var posts = window.goviralParseNormalizedPosts({ included: codeEntities }, count);
      if (posts.length > 0) return { posts: posts };
    }
  } catch (e) { /* fall through */ }

  // 2. Try reaction-button DOM extraction.
  try {
    var domPosts = window.goviralExtractSearchPostsDOM(count);
    if (domPosts.length > 0) return { posts: domPosts };
  } catch (e) { /* fall through */ }

  // 3. Fall back to REST API with pagination.
  try {
    var headers = window.goviralLinkedInHeaders();
    var allPosts = [];
    var pageSize = Math.min(count, 50);
    var start = 0;
    var maxPages = Math.ceil(count / pageSize);
    for (var page = 0; page < maxPages; page++) {
      var url =
        "/voyager/api/search/dash/clusters" +
        "?q=all&keywords=" + encodeURIComponent(keywords) +
        "&type=CONTENT&count=" + pageSize + "&start=" + start;

      var resp = await fetch(url, { credentials: "include", headers: headers });
      if (!resp.ok) {
        if (allPosts.length > 0) break;
        var body = await resp.text().catch(function () { return ""; });
        return { error: "Failed to search posts: " + resp.status + " " + body.slice(0, 300) };
      }
      var data = await resp.json();
      var pagePosts = window.goviralParseNormalizedPosts(data, 0);
      if (pagePosts.length === 0) break;
      for (var pi = 0; pi < pagePosts.length; pi++) allPosts.push(pagePosts[pi]);
      if (allPosts.length >= count) break;
      start += pageSize;
    }
    return { posts: allPosts.slice(0, count) };
  } catch (err) {
    return { error: String(err) };
  }
};

window.goviralFetchTrending = async function (keywords, count, dateFilter) {
  count = count || 20;
  var niche_tags = keywords
    ? keywords.split(/[\s,]+/).filter(Boolean)
    : [];

  function tagPosts(posts) {
    for (var i = 0; i < posts.length; i++) {
      posts[i].niche_tags = niche_tags;
    }
    return posts;
  }

  // 1. Try <code> entity extraction.
  try {
    var codeEntities = window.goviralExtractCodeEntities();
    var hasUpdates = codeEntities.some(function (e) {
      return (e.$type || "").indexOf("Update") !== -1;
    });
    if (hasUpdates) {
      var posts = window.goviralParseNormalizedPosts({ included: codeEntities }, count);
      if (posts.length > 0) return { posts: tagPosts(posts) };
    }
  } catch (e) { /* fall through */ }

  // 2. Try reaction-button DOM extraction.
  try {
    var domPosts = window.goviralExtractSearchPostsDOM(count);
    if (domPosts.length > 0) return { posts: tagPosts(domPosts) };
  } catch (e) { /* fall through */ }

  // 3. Fall back to REST API with pagination.
  try {
    var headers = window.goviralLinkedInHeaders();
    var allPosts = [];
    var pageSize = Math.min(count, 50);
    var start = 0;
    var maxPages = Math.ceil(count / pageSize);
    for (var page = 0; page < maxPages; page++) {
      var url =
        "/voyager/api/search/dash/clusters" +
        "?q=all&keywords=" + encodeURIComponent(keywords) +
        "&type=CONTENT&count=" + pageSize + "&start=" + start +
        "&origin=FACETED_SEARCH" +
        (dateFilter ? "&datePosted=%22" + dateFilter + "%22" : "");

      var resp = await fetch(url, { credentials: "include", headers: headers });
      if (!resp.ok) {
        if (allPosts.length > 0) break;
        var body = await resp.text().catch(function () { return ""; });
        return { error: "Failed to fetch trending: " + resp.status + " " + body.slice(0, 300) };
      }
      var data = await resp.json();
      var pagePosts = window.goviralParseNormalizedPosts(data, 0);
      if (pagePosts.length === 0) break;
      for (var pi = 0; pi < pagePosts.length; pi++) allPosts.push(pagePosts[pi]);
      if (allPosts.length >= count) break;
      start += pageSize;
    }
    return { posts: tagPosts(allPosts.slice(0, count)) };
  } catch (err) {
    return { error: String(err) };
  }
};
