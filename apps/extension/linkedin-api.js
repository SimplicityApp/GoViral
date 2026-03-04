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
    var activityMatch = urn.match(/activity:(\d+)/);
    var platform_post_id = activityMatch ? activityMatch[1] : urn;

    var posted_at = window.goviralExtractCreatedAt(entity) || window.goviralActivityIdToDate(platform_post_id);

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

// --- Simple hash for generating deterministic IDs from text ---

window.goviralSimpleHash = function (str) {
  var hash = 0;
  for (var i = 0; i < str.length; i++) {
    var ch = str.charCodeAt(i);
    hash = ((hash << 5) - hash) + ch;
    hash = hash & hash; // Convert to 32-bit integer
  }
  return Math.abs(hash).toString(36);
};

// --- Activity ID → timestamp (LinkedIn Snowflake-like IDs) ---

window.goviralActivityIdToDate = function (activityId) {
  var id = typeof activityId === 'string' ? parseInt(activityId, 10) : activityId;
  if (!id || isNaN(id)) return null;
  var timestampMs = Math.floor(id / 4194304);
  var d = new Date(timestampMs);
  if (isNaN(d.getTime()) || d.getFullYear() < 2000) return null;
  return d.toISOString();
};

// --- RSC / __como_rehydration__ extraction (LinkedIn 2025+ SPA) ---

window.goviralExtractComoRehydration = function (count) {
  // LinkedIn's new React Server Components store embeds all post data in
  // window.__como_rehydration__.  We extract activity URNs and match them
  // to rendered DOM cards for text/engagement.
  var como = window.__como_rehydration__;
  if (!como || typeof como !== "object") return [];

  console.log("[GoViral] Como: scanning __como_rehydration__ (" + Object.keys(como).length + " entries)");

  // 1. Collect unique activity URNs from the RSC payload.
  var str = "";
  var keyCount = Object.keys(como).length;
  // Build a string from entries — limit to first ~4MB to avoid perf issues
  var charBudget = 4000000;
  for (var i = 0; i < keyCount && charBudget > 0; i++) {
    var entry = como[i];
    if (!entry) continue;
    var s = typeof entry === "string" ? entry : JSON.stringify(entry);
    str += s;
    charBudget -= s.length;
  }

  var urnRegex = /urn:li:activity:(\d{15,})/g;
  var match;
  var activityIds = [];
  var seenIds = {};
  while ((match = urnRegex.exec(str)) !== null) {
    if (!seenIds[match[1]]) {
      seenIds[match[1]] = true;
      activityIds.push(match[1]);
    }
  }

  console.log("[GoViral] Como: found", activityIds.length, "unique activity IDs");
  if (activityIds.length === 0) return [];

  // 2. Extract post text from DOM cards (reaction buttons).
  //    We walk the DOM to get text/author/engagement, then zip with URNs.
  var selectors = [
    'button[aria-label="Reaction button state: no reaction"]',
    'button[aria-label*="React"]',
    'button[aria-label="Like"]',
    'button[aria-label*="like"]',
  ];
  var reactionBtns = [];
  for (var si = 0; si < selectors.length; si++) {
    reactionBtns = document.querySelectorAll(selectors[si]);
    if (reactionBtns.length > 0) break;
  }

  console.log("[GoViral] Como: found", reactionBtns.length, "reaction buttons to match with", activityIds.length, "URNs");

  var results = [];
  var usedIds = {};

  for (var bi = 0; bi < reactionBtns.length; bi++) {
    var btn = reactionBtns[bi];

    // Walk up to card container (increase limit to 30 levels, 800 chars).
    var card = btn;
    for (var j = 0; j < 30; j++) {
      card = card.parentElement;
      if (!card) break;
      if ((card.textContent || "").length > 800) break;
    }
    if (!card) continue;

    var fullText = (card.textContent || "").replace(/\s+/g, " ").trim();

    // Extract author name.
    var authorName = "";
    var profileLinks = card.querySelectorAll('a[href*="/in/"], a[href*="/company/"]');
    for (var pl = 0; pl < profileLinks.length; pl++) {
      var linkText = profileLinks[pl].textContent.trim();
      if (linkText && linkText.length > 1 && linkText.length < 80 && linkText.indexOf("Sign") < 0) {
        authorName = linkText.replace(/\d[\d,]*\s*followers?/i, "").replace(/\s*·\s*.*$/, "").trim();
        if (authorName) break;
      }
    }
    if (!authorName) {
      var followBtn = card.querySelector('button[aria-label^="Follow "]');
      if (followBtn) {
        authorName = (followBtn.getAttribute("aria-label") || "").replace("Follow ", "").trim();
      }
    }
    if (!authorName) authorName = "Unknown";

    // Extract post text — try structured selectors first, then full card text.
    var postText = "";
    var textEl = card.querySelector(
      ".feed-shared-update-v2__description, .update-components-text, " +
      ".feed-shared-text, .feed-shared-inline-show-more-text"
    );
    if (textEl) {
      postText = textEl.textContent.trim();
    }
    if (!postText || postText.length < 20) {
      // Fall back to time-marker or "Follow(ing)" based text extraction.
      var tm = fullText.match(/\d+[dhwmo]\s*[·•]?\s*/);
      if (tm) {
        postText = fullText.substring(fullText.indexOf(tm[0]) + tm[0].length).trim();
      } else {
        var fi = fullText.indexOf("Following");
        if (fi < 0) fi = fullText.indexOf("Follow");
        if (fi >= 0) postText = fullText.substring(fi + 9).trim();
      }
    }

    // Clean up post text.
    postText = postText.replace(/\d[\d,]*\s*(reaction|comment|repost).*$/i, "").trim();
    postText = postText.replace(/Like\s*(Comment|Repost|Send|Share|Celebrate|Support|Love|Insightful|Funny).*$/i, "").trim();
    postText = postText.replace(/[…\.]{1,3}\s*more\s*$/i, "").trim();
    postText = postText.replace(/\u2026more\s*$/g, "").replace(/Show less\s*$/g, "").trim();

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

    // Assign activity ID — use positional matching, then fallback to hash.
    var platform_post_id = "";
    if (bi < activityIds.length && !usedIds[activityIds[bi]]) {
      platform_post_id = activityIds[bi];
      usedIds[activityIds[bi]] = true;
    } else {
      // Fallback: generate deterministic ID from content.
      platform_post_id = "hash_" + window.goviralSimpleHash(postText.substring(0, 200));
    }

    results.push({
      platform_post_id: platform_post_id,
      content: postText.substring(0, 2000),
      likes: likes,
      comments: comments,
      reposts: reposts,
      impressions: 0,
      posted_at: window.goviralActivityIdToDate(platform_post_id),
      author_name: authorName,
      author_username: null,
      niche_tags: [],
    });

    if (count && results.length >= count) break;
  }

  // 3. If we found fewer DOM cards than URNs, the remaining URNs still represent
  //    posts (just without extracted text). Don't add them — we need text content.

  console.log("[GoViral] Como: extracted", results.length, "posts");
  return results;
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
  // Extract posts from search/feed pages using reaction button anchors.
  // Tries multiple selectors since LinkedIn frequently changes aria-labels.
  var selectors = [
    'button[aria-label="Reaction button state: no reaction"]',
    'button[aria-label*="React"]',
    'button[aria-label="Like"]',
    'button[aria-label*="like"]',
  ];
  var reactionBtns = [];
  for (var si = 0; si < selectors.length; si++) {
    reactionBtns = document.querySelectorAll(selectors[si]);
    if (reactionBtns.length > 0) {
      console.log("[GoViral] DOM extraction: matched selector", selectors[si], "→", reactionBtns.length, "buttons");
      break;
    }
  }
  if (reactionBtns.length === 0) {
    console.log("[GoViral] DOM extraction: no reaction buttons found with any selector");
  }

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

    // Dedup: use postUrn if available, else use card text hash.
    var fullText = (card.textContent || "").replace(/\s+/g, " ").trim();
    var dedupKey = postUrn || ("text:" + fullText.substring(0, 200));
    if (seen[dedupKey]) continue;
    seen[dedupKey] = true;

    // Extract author name — try multiple strategies.
    var authorName = "";

    // Strategy 1: "Feed post AuthorName · Following" pattern
    var m = fullText.match(/Feed post\s*(.+?)\s*[·•]\s*(?:Following|Promoted)/);
    if (m) {
      authorName = m[1].trim();
    }

    // Strategy 2: "Follow AuthorName" button
    if (!authorName) {
      var followBtn = card.querySelector('button[aria-label^="Follow "]');
      if (followBtn) {
        authorName = (followBtn.getAttribute("aria-label") || "").replace("Follow ", "").trim();
      }
    }

    // Strategy 3: .update-components-actor__name element
    if (!authorName) {
      var actorEl = card.querySelector(".update-components-actor__name span[aria-hidden='true']") ||
                    card.querySelector(".update-components-actor__name");
      if (actorEl) {
        authorName = actorEl.textContent.trim();
      }
    }

    // Strategy 4: Profile link text (/in/ or /company/ URLs)
    if (!authorName) {
      var profileLinks = card.querySelectorAll('a[href*="/in/"], a[href*="/company/"]');
      for (var pl = 0; pl < profileLinks.length; pl++) {
        var linkText = profileLinks[pl].textContent.trim();
        if (linkText && linkText.length > 1 && linkText.length < 80 && linkText.indexOf("Sign") < 0) {
          authorName = linkText.replace(/\d[\d,]*\s*followers?/i, "").replace(/\s*·\s*.*$/, "").trim();
          if (authorName) break;
        }
      }
    }

    // Don't skip posts without author — use "Unknown" as fallback.
    if (!authorName) authorName = "Unknown";

    // Extract post text after time marker.
    var postText = "";
    // Try structured text selectors first.
    var textEl = card.querySelector(
      ".feed-shared-update-v2__description, .update-components-text, " +
      ".feed-shared-text, .feed-shared-inline-show-more-text"
    );
    if (textEl) {
      postText = textEl.textContent.trim();
    }
    if (!postText || postText.length < 20) {
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

    // Use real activity ID if we found a URN, else generate hash-based ID.
    var platform_post_id;
    if (postUrn) {
      var activityMatch = postUrn.match(/activity:(\d+)/);
      platform_post_id = activityMatch ? activityMatch[1] : postUrn;
    } else {
      platform_post_id = "hash_" + window.goviralSimpleHash(postText.substring(0, 200));
    }

    results.push({
      platform_post_id: platform_post_id,
      content: postText.substring(0, 2000),
      likes: likes,
      comments: comments,
      reposts: reposts,
      impressions: 0,
      posted_at: window.goviralActivityIdToDate(platform_post_id),
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
      posted_at: window.goviralActivityIdToDate(platform_post_id),
      author_name: null,
      author_username: null,
      niche_tags: [],
    });

    if (count && results.length >= count) break;
  }
  return results;
};

// --- Broad search-card DOM extraction (last-resort DOM method) ---

window.goviralExtractSearchCardsDOM = function (count) {
  // Find post cards on search result pages by looking for links to
  // /feed/update/ or /posts/ — does NOT depend on reaction buttons.
  var allLinks = document.querySelectorAll(
    'a[href*="/feed/update/"], a[href*="/posts/"]'
  );
  console.log("[GoViral] SearchCards: found", allLinks.length, "post links on page");

  var results = [];
  var seen = {};

  for (var i = 0; i < allLinks.length; i++) {
    var link = allLinks[i];
    var href = link.getAttribute("href") || "";
    var hrefDecoded = href;
    try { hrefDecoded = decodeURIComponent(href); } catch(e) {}

    // Extract post URN from link.
    var postUrn = "";
    if (href.indexOf("/feed/update/") >= 0) {
      var hm = hrefDecoded.match(/(urn:li:[a-zA-Z0-9_]+:[^?&# ]+)/);
      if (hm) {
        var cUrn = hm[1];
        if (cUrn.indexOf("activity") >= 0 || cUrn.indexOf("ugcPost") >= 0 ||
            cUrn.indexOf("fsd_update") >= 0) {
          postUrn = cUrn;
        }
      }
    } else if (href.indexOf("/posts/") >= 0) {
      var am = hrefDecoded.match(/[^a-zA-Z]activity([0-9]{15,})/);
      if (am) {
        postUrn = "urn:li:activity:" + am[1];
      }
    }
    if (!postUrn) continue;
    if (seen[postUrn]) continue;
    seen[postUrn] = true;

    // Walk up to find a large-enough card container.
    var card = link;
    for (var j = 0; j < 25; j++) {
      card = card.parentElement;
      if (!card) break;
      // Also check for data-urn on parent.
      var dataUrn = (card.getAttribute && card.getAttribute("data-urn")) || "";
      if (!postUrn && dataUrn && dataUrn.indexOf("activity") >= 0) {
        postUrn = dataUrn;
      }
      if ((card.textContent || "").length > 200) break;
    }
    if (!card) continue;

    var fullText = (card.textContent || "").replace(/\s+/g, " ").trim();
    if (fullText.length < 30) continue;

    // Extract author — try multiple strategies.
    var authorName = "";
    var actorEl = card.querySelector(".update-components-actor__name span[aria-hidden='true']") ||
                  card.querySelector(".update-components-actor__name") ||
                  card.querySelector('[data-anonymize="person-name"]');
    if (actorEl) {
      authorName = actorEl.textContent.trim();
    }
    if (!authorName) {
      var profileLinks = card.querySelectorAll('a[href*="/in/"]');
      for (var pl = 0; pl < profileLinks.length; pl++) {
        var linkText = profileLinks[pl].textContent.trim();
        if (linkText && linkText.length > 1 && linkText.length < 80) {
          authorName = linkText;
          break;
        }
      }
    }
    if (!authorName) authorName = "Unknown";

    // Extract post text — try structured selectors first, then fall back to
    // stripping header/footer from the full card text.
    var postText = "";
    var textEl = card.querySelector(
      ".feed-shared-update-v2__description, " +
      ".update-components-text, " +
      ".feed-shared-text, " +
      ".feed-shared-inline-show-more-text, " +
      '[data-anonymize="content"]'
    );
    if (textEl) {
      postText = textEl.textContent.trim();
    }
    if (!postText || postText.length < 20) {
      // Fall back to time-marker-based text extraction from full card.
      var tm = fullText.match(/\d+[dhwmo]\s*[·•]?\s*/);
      if (tm) {
        postText = fullText.substring(fullText.indexOf(tm[0]) + tm[0].length).trim();
      }
    }
    // Remove engagement/action noise from end.
    postText = postText.replace(/\d[\d,]*\s*(reaction|comment|repost).*$/i, "").trim();
    postText = postText.replace(/Like\s*(Comment|Repost|Send|Share|Celebrate|Support|Love|Insightful|Funny).*$/i, "").trim();
    postText = postText.replace(/[…\.]{1,3}\s*more\s*$/i, "").trim();
    postText = postText.replace(/\u2026more\s*$/g, "").replace(/Show less\s*$/g, "").trim();

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
      posted_at: window.goviralActivityIdToDate(platform_post_id),
      author_name: authorName,
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

window.goviralSearchPosts = async function (keywords, count, postedBy) {
  count = count || 20;

  console.log("[GoViral] goviralSearchPosts called:", { keywords: keywords, count: count, postedBy: postedBy });

  // 0. Try __como_rehydration__ extraction (LinkedIn 2025+ RSC data store).
  try {
    var comoPosts = window.goviralExtractComoRehydration(count);
    console.log("[GoViral] SearchPosts Step 0 — como rehydration posts:", comoPosts.length);
    if (comoPosts.length > 0) return { posts: comoPosts };
  } catch (e) {
    console.log("[GoViral] SearchPosts Step 0 — error:", e);
  }

  // 1. Try <code> entity extraction (legacy).
  try {
    var codeEntities = window.goviralExtractCodeEntities();
    var hasUpdates = codeEntities.some(function (e) {
      return (e.$type || "").indexOf("Update") !== -1;
    });
    console.log("[GoViral] SearchPosts Step 1 — code entities:", codeEntities.length, "hasUpdates:", hasUpdates);
    if (hasUpdates) {
      var posts = window.goviralParseNormalizedPosts({ included: codeEntities }, count);
      console.log("[GoViral] SearchPosts Step 1 — parsed posts:", posts.length);
      if (posts.length > 0) return { posts: posts };
    }
  } catch (e) {
    console.log("[GoViral] SearchPosts Step 1 — error:", e);
  }

  // 2a. Try reaction-button DOM extraction (now works without URNs).
  try {
    var domPosts = window.goviralExtractSearchPostsDOM(count);
    console.log("[GoViral] SearchPosts Step 2a — DOM posts:", domPosts.length);
    if (domPosts.length > 0) return { posts: domPosts };
  } catch (e) {
    console.log("[GoViral] SearchPosts Step 2a — error:", e);
  }

  // 2b. Try [data-urn] DOM extraction (legacy).
  try {
    var urnPosts = window.goviralExtractActivityPostsDOM(count);
    console.log("[GoViral] SearchPosts Step 2b — data-urn DOM posts:", urnPosts.length);
    if (urnPosts.length > 0) return { posts: urnPosts };
  } catch (e) {
    console.log("[GoViral] SearchPosts Step 2b — error:", e);
  }

  // 2c. Try broad search-card DOM extraction.
  try {
    var cardPosts = window.goviralExtractSearchCardsDOM(count);
    console.log("[GoViral] SearchPosts Step 2c — search-card DOM posts:", cardPosts.length);
    if (cardPosts.length > 0) return { posts: cardPosts };
  } catch (e) {
    console.log("[GoViral] SearchPosts Step 2c — error:", e);
  }

  // 3. Fall back to REST API (may fail on newer LinkedIn — non-fatal).
  try {
    console.log("[GoViral] SearchPosts Step 3 — falling back to REST API");
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

      console.log("[GoViral] SearchPosts Step 3 — REST URL:", url);
      var resp = await fetch(url, { credentials: "include", headers: headers });
      if (!resp.ok) {
        if (allPosts.length > 0) break;
        var body = await resp.text().catch(function () { return ""; });
        console.log("[GoViral] SearchPosts Step 3 — REST error:", resp.status, body.slice(0, 200), "(non-fatal)");
        return { posts: [] };
      }
      var data = await resp.json();
      var pagePosts = window.goviralParseNormalizedPosts(data, 0);
      console.log("[GoViral] SearchPosts Step 3 — page", page, "posts:", pagePosts.length);
      if (pagePosts.length === 0) break;
      for (var pi = 0; pi < pagePosts.length; pi++) allPosts.push(pagePosts[pi]);
      if (allPosts.length >= count) break;
      start += pageSize;
    }
    return { posts: allPosts.slice(0, count) };
  } catch (err) {
    console.log("[GoViral] SearchPosts Step 3 — exception:", err, "(non-fatal)");
    return { posts: [] };
  }
};

window.goviralFetchTrending = async function (keywords, count, dateFilter, postedBy) {
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

  console.log("[GoViral] goviralFetchTrending called:", { keywords: keywords, count: count, dateFilter: dateFilter, postedBy: postedBy });

  // DOM diagnostic — confirms whether SPA rendered any post elements.
  console.log("[GoViral] DOM diagnostic:", {
    dataUrnCount: document.querySelectorAll("[data-urn]").length,
    feedUpdateLinks: document.querySelectorAll('a[href*="/feed/update/"]').length,
    postsLinks: document.querySelectorAll('a[href*="/posts/"]').length,
    reactionButtons: document.querySelectorAll('button[aria-label*="React"], button[aria-label="Like"]').length,
    bodyLength: (document.body && document.body.textContent || "").length,
    hasComo: !!window.__como_rehydration__,
  });

  // 0. Try __como_rehydration__ extraction (LinkedIn 2025+ RSC data store).
  //    This is the most reliable method on the new LinkedIn SPA.
  try {
    var comoPosts = window.goviralExtractComoRehydration(count);
    console.log("[GoViral] Step 0 — como rehydration posts:", comoPosts.length);
    if (comoPosts.length > 0) return { posts: tagPosts(comoPosts) };
  } catch (e) {
    console.log("[GoViral] Step 0 — error:", e);
  }

  // 1. Try <code> entity extraction (legacy LinkedIn pages).
  try {
    var codeEntities = window.goviralExtractCodeEntities();
    var hasUpdates = codeEntities.some(function (e) {
      return (e.$type || "").indexOf("Update") !== -1;
    });
    console.log("[GoViral] Step 1 — code entities:", codeEntities.length, "hasUpdates:", hasUpdates);
    if (hasUpdates) {
      var posts = window.goviralParseNormalizedPosts({ included: codeEntities }, count);
      console.log("[GoViral] Step 1 — parsed posts:", posts.length);
      if (posts.length > 0) return { posts: tagPosts(posts) };
    }
  } catch (e) {
    console.log("[GoViral] Step 1 — error:", e);
  }

  // 2a. Try reaction-button DOM extraction (now works without URNs via hash fallback).
  try {
    var domPosts = window.goviralExtractSearchPostsDOM(count);
    console.log("[GoViral] Step 2a — reaction-button DOM posts:", domPosts.length);
    if (domPosts.length > 0) return { posts: tagPosts(domPosts) };
  } catch (e) {
    console.log("[GoViral] Step 2a — error:", e);
  }

  // 2b. Try [data-urn] DOM extraction (legacy — works on older LinkedIn pages).
  try {
    var urnPosts = window.goviralExtractActivityPostsDOM(count);
    console.log("[GoViral] Step 2b — data-urn DOM posts:", urnPosts.length);
    if (urnPosts.length > 0) return { posts: tagPosts(urnPosts) };
  } catch (e) {
    console.log("[GoViral] Step 2b — error:", e);
  }

  // 2c. Try broad search-card DOM extraction (last DOM attempt).
  try {
    var cardPosts = window.goviralExtractSearchCardsDOM(count);
    console.log("[GoViral] Step 2c — search-card DOM posts:", cardPosts.length);
    if (cardPosts.length > 0) return { posts: tagPosts(cardPosts) };
  } catch (e) {
    console.log("[GoViral] Step 2c — error:", e);
  }

  // 3. Fall back to REST API (may fail on newer LinkedIn — non-fatal).
  try {
    console.log("[GoViral] Step 3 — falling back to REST API");
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

      console.log("[GoViral] Step 3 — REST URL:", url);
      var resp = await fetch(url, { credentials: "include", headers: headers });
      if (!resp.ok) {
        if (allPosts.length > 0) break;
        var body = await resp.text().catch(function () { return ""; });
        console.log("[GoViral] Step 3 — REST error:", resp.status, body.slice(0, 200), "(non-fatal)");
        // Don't return error — the REST API is deprecated on newer LinkedIn.
        // Return empty posts so the caller doesn't see a hard failure.
        return { posts: [] };
      }
      var data = await resp.json();
      var pagePosts = window.goviralParseNormalizedPosts(data, 0);
      console.log("[GoViral] Step 3 — page", page, "posts:", pagePosts.length);
      if (pagePosts.length === 0) break;
      for (var pi = 0; pi < pagePosts.length; pi++) allPosts.push(pagePosts[pi]);
      if (allPosts.length >= count) break;
      start += pageSize;
    }
    return { posts: tagPosts(allPosts.slice(0, count)) };
  } catch (err) {
    console.log("[GoViral] Step 3 — exception:", err, "(non-fatal)");
    return { posts: [] };
  }
};
