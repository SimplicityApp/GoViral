// GoViral — Background Service Worker

const ALLOWED_ORIGIN_PATTERNS = [
  /^http:\/\/localhost(:\d+)?$/,
  /^https:\/\/[^/]+\.fly\.dev$/,
  /^https:\/\/[^/]+\.simple-tech\.app$/,
];

function isAllowedOrigin(origin) {
  return ALLOWED_ORIGIN_PATTERNS.some((p) => p.test(origin));
}

async function getCookie(domain, name) {
  try {
    const cookie = await chrome.cookies.get({ url: `https://${domain}`, name });
    return cookie ? cookie.value : null;
  } catch {
    return null;
  }
}

// Navigate a LinkedIn tab to a specific page for DOM extraction.
// Returns { tabId, created } — caller should close tab if created === true.
async function ensureLinkedInPage(url, pathMatch) {
  const tabs = await chrome.tabs.query({ url: "*://*.linkedin.com/*" });

  // Reuse an existing LinkedIn tab — navigate it to the target URL so
  // the page content is fresh (avoids stale DOM / expired CSRF tokens).
  let tab = null;
  let created = false;
  for (const t of tabs) {
    if (t.url && t.url.includes(pathMatch)) {
      tab = t;
      break;
    }
  }

  if (tab) {
    // Navigate the existing tab to the (possibly new) URL.
    await chrome.tabs.update(tab.id, { url });
  } else {
    // Create a background tab.
    tab = await chrome.tabs.create({ url, active: false });
    created = true;
  }

  // Wait for the tab to finish loading.
  await new Promise((resolve) => {
    function onUpdated(tabId, changeInfo) {
      if (tabId === tab.id && changeInfo.status === "complete") {
        chrome.tabs.onUpdated.removeListener(onUpdated);
        resolve();
      }
    }
    chrome.tabs.onUpdated.addListener(onUpdated);
  });

  // Extra delay for SPA client-side rendering.
  await new Promise((r) => setTimeout(r, 3000));

  return { tabId: tab.id, created };
}

// Scroll the LinkedIn page to trigger lazy-loading of additional posts.
async function scrollLinkedInPage(tabId, scrollCount) {
  if (!scrollCount || scrollCount <= 0) return;
  await chrome.scripting.executeScript({
    target: { tabId },
    world: "MAIN",
    func: async (numScrolls) => {
      for (var i = 0; i < numScrolls; i++) {
        window.scrollTo(0, document.body.scrollHeight);
        await new Promise(function (r) { setTimeout(r, 1500); });
      }
      await new Promise(function (r) { setTimeout(r, 500); });
    },
    args: [scrollCount],
  });
}

// Inject linkedin-api.js into the tab and call one of the goviralXxx functions.
async function callLinkedInApi(tabId, funcName, args) {
  // Inject the API helpers into the page's MAIN world
  await chrome.scripting.executeScript({
    target: { tabId },
    world: "MAIN",
    files: ["linkedin-api.js"],
  });

  // Call the requested function
  const results = await chrome.scripting.executeScript({
    target: { tabId },
    world: "MAIN",
    func: (name, fnArgs) => {
      // eslint-disable-next-line no-undef
      const fn = window[name];
      if (typeof fn !== "function") {
        return { error: `Function ${name} not found` };
      }
      return fn(...fnArgs);
    },
    args: [funcName, args],
  });

  return results[0].result;
}

function mapPeriodToLinkedInDateFilter(period) {
  switch (period) {
    case "24h": case "day": return "past-24h";
    case "7d":  case "week": return "past-week";
    case "30d": case "month": return "past-month";
    default: return "past-week";
  }
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  const { type } = message;

  // ── Cookie extraction (existing handler) ─────────────────────────────────
  if (type === "GOVIRAL_EXTRACT_COOKIES") {
    // Validate sender is a content script from an allowed origin
    if (!sender.tab || !sender.tab.url) {
      sendResponse({ success: false, error: "Invalid sender" });
      return false;
    }

    let origin;
    try {
      origin = new URL(sender.tab.url).origin;
    } catch {
      sendResponse({ success: false, error: "Invalid sender URL" });
      return false;
    }

    if (!isAllowedOrigin(origin)) {
      sendResponse({ success: false, error: "Origin not allowed" });
      return false;
    }

    (async () => {
      const [authToken, ct0, liAt, jsessionid] = await Promise.all([
        getCookie("x.com", "auth_token"),
        getCookie("x.com", "ct0"),
        getCookie("www.linkedin.com", "li_at"),
        getCookie("www.linkedin.com", "JSESSIONID"),
      ]);

      sendResponse({
        success: true,
        cookies: {
          x:
            authToken && ct0
              ? { auth_token: authToken, ct0 }
              : null,
          linkedin:
            liAt && jsessionid
              ? { li_at: liAt, jsessionid: jsessionid.replace(/^"/, "").replace(/"$/, "") }
              : null,
        },
      });
    })();

    return true; // async sendResponse
  }

  // ── LinkedIn Voyager API handlers ─────────────────────────────────────────
  if (type === "GOVIRAL_LINKEDIN_FETCH_POSTS") {
    (async () => {
      try {
        const count = message.count || 20;
        const { tabId, created } = await ensureLinkedInPage(
          "https://www.linkedin.com/in/me/recent-activity/all/",
          "/recent-activity"
        );
        const scrollCount = Math.min(10, Math.max(0, Math.ceil((count - 5) / 4)));
        if (scrollCount > 0) await scrollLinkedInPage(tabId, scrollCount);
        const result = await callLinkedInApi(tabId, "goviralFetchMyPosts", [
          count,
        ]);
        if (created) chrome.tabs.remove(tabId).catch(() => {});
        sendResponse({ success: true, ...result });
      } catch (err) {
        sendResponse({ success: false, error: String(err) });
      }
    })();
    return true;
  }

  if (type === "GOVIRAL_LINKEDIN_FETCH_FEED") {
    (async () => {
      try {
        const count = message.count || 20;
        const { tabId, created } = await ensureLinkedInPage(
          "https://www.linkedin.com/feed/",
          "/feed"
        );
        const scrollCount = Math.min(10, Math.max(0, Math.ceil((count - 5) / 4)));
        if (scrollCount > 0) await scrollLinkedInPage(tabId, scrollCount);
        const result = await callLinkedInApi(tabId, "goviralFetchFeed", [
          count,
        ]);
        if (created) chrome.tabs.remove(tabId).catch(() => {});
        sendResponse({ success: true, ...result });
      } catch (err) {
        sendResponse({ success: false, error: String(err) });
      }
    })();
    return true;
  }

  if (type === "GOVIRAL_LINKEDIN_SEARCH_POSTS") {
    (async () => {
      try {
        const count = message.count || 20;
        const keywords = message.keywords || "";
        const searchUrl =
          "https://www.linkedin.com/search/results/content/" +
          "?keywords=" + encodeURIComponent(keywords) +
          "&origin=GLOBAL_SEARCH_HEADER";
        const { tabId, created } = await ensureLinkedInPage(
          searchUrl,
          "/search/results/content"
        );
        const scrollCount = Math.min(10, Math.max(0, Math.ceil((count - 5) / 4)));
        if (scrollCount > 0) await scrollLinkedInPage(tabId, scrollCount);
        const result = await callLinkedInApi(tabId, "goviralSearchPosts", [
          keywords,
          count,
        ]);
        if (created) chrome.tabs.remove(tabId).catch(() => {});
        sendResponse({ success: true, ...result });
      } catch (err) {
        sendResponse({ success: false, error: String(err) });
      }
    })();
    return true;
  }

  if (type === "GOVIRAL_LINKEDIN_FETCH_TRENDING") {
    (async () => {
      try {
        const count = message.count || 20;
        const niches = message.niches || (message.keywords ? message.keywords.split(/[\s,]+/).filter(Boolean) : []);
        const period = message.period || "24h";
        const dateFilter = mapPeriodToLinkedInDateFilter(period);

        if (niches.length === 0) {
          sendResponse({ success: true, posts: [] });
          return;
        }

        const allPosts = [];
        const seen = {};
        let tabId = null;
        let created = false;

        for (let i = 0; i < niches.length; i++) {
          const niche = niches[i];
          const searchUrl =
            "https://www.linkedin.com/search/results/content/" +
            "?keywords=" + encodeURIComponent(niche) +
            "&sortBy=%22date_posted%22" +
            "&datePosted=%22" + dateFilter + "%22" +
            "&origin=FACETED_SEARCH";

          const nav = await ensureLinkedInPage(searchUrl, "/search/results/content");
          tabId = nav.tabId;
          if (nav.created) created = true;

          await scrollLinkedInPage(tabId, 5);

          const result = await callLinkedInApi(tabId, "goviralFetchTrending", [
            niche, count, dateFilter,
          ]);

          if (result && result.posts) {
            for (let j = 0; j < result.posts.length; j++) {
              const p = result.posts[j];
              if (!seen[p.platform_post_id]) {
                seen[p.platform_post_id] = true;
                allPosts.push(p);
              }
            }
          }
        }

        if (created && tabId) chrome.tabs.remove(tabId).catch(() => {});
        sendResponse({ success: true, posts: allPosts.slice(0, count) });
      } catch (err) {
        sendResponse({ success: false, error: String(err) });
      }
    })();
    return true;
  }

  return false;
});
