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

// Ensure a LinkedIn tab exists; returns the tab id.
async function ensureLinkedInTab() {
  const tabs = await chrome.tabs.query({ url: "*://*.linkedin.com/*" });
  if (tabs.length > 0) {
    return tabs[0].id;
  }

  // Create a background tab and wait for it to finish loading
  const tab = await chrome.tabs.create({
    url: "https://www.linkedin.com/feed/",
    active: false,
  });

  await new Promise((resolve) => {
    function onUpdated(tabId, changeInfo) {
      if (tabId === tab.id && changeInfo.status === "complete") {
        chrome.tabs.onUpdated.removeListener(onUpdated);
        resolve();
      }
    }
    chrome.tabs.onUpdated.addListener(onUpdated);
  });

  return tab.id;
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
        const tabId = await ensureLinkedInTab();
        const result = await callLinkedInApi(tabId, "goviralFetchMyPosts", [
          message.count || 20,
        ]);
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
        const tabId = await ensureLinkedInTab();
        const result = await callLinkedInApi(tabId, "goviralFetchFeed", [
          message.count || 20,
        ]);
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
        const tabId = await ensureLinkedInTab();
        const result = await callLinkedInApi(tabId, "goviralSearchPosts", [
          message.keywords || "",
          message.count || 20,
        ]);
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
        const tabId = await ensureLinkedInTab();
        const result = await callLinkedInApi(tabId, "goviralFetchTrending", [
          message.keywords || "",
          message.count || 20,
        ]);
        sendResponse({ success: true, ...result });
      } catch (err) {
        sendResponse({ success: false, error: String(err) });
      }
    })();
    return true;
  }

  return false;
});
