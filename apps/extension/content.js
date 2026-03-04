// GoViral Cookie Sync — Content Script
// Bridges page (window.postMessage) <-> extension (chrome.runtime.sendMessage)

const EXTENSION_VERSION = "1.0.0";

// Announce availability to the page
function announce() {
  window.postMessage(
    { type: "GOVIRAL_EXTENSION_AVAILABLE", version: EXTENSION_VERSION },
    window.location.origin
  );
}

// Announce on load
announce();

// Re-announce when tab becomes visible again
document.addEventListener("visibilitychange", () => {
  if (document.visibilityState === "visible") {
    announce();
  }
});

// Helper: send message to background and post result back to page.
// Handles chrome.runtime.lastError and undefined responses gracefully.
function forwardToBackground(msg, resultType, requestId) {
  chrome.runtime.sendMessage(msg, (response) => {
    const err = chrome.runtime.lastError;
    if (err) {
      console.warn("[GoViral] content.js: chrome.runtime.lastError for", msg.type, err.message);
      window.postMessage(
        { type: resultType, requestId, success: false, error: err.message },
        window.location.origin
      );
      return;
    }
    if (!response) {
      console.warn("[GoViral] content.js: undefined response for", msg.type);
      window.postMessage(
        { type: resultType, requestId, success: false, error: "No response from extension background" },
        window.location.origin
      );
      return;
    }
    window.postMessage(
      { type: resultType, requestId, ...response },
      window.location.origin
    );
  });
}

// Listen for messages from the page
window.addEventListener("message", (event) => {
  // Security: only accept messages from same origin and same window
  if (event.origin !== window.location.origin) return;
  if (event.source !== window) return;

  const { type, requestId } = event.data || {};

  if (type === "GOVIRAL_PING") {
    window.postMessage(
      { type: "GOVIRAL_PONG", version: EXTENSION_VERSION, requestId },
      window.location.origin
    );
    return;
  }

  if (type === "GOVIRAL_EXTRACT_COOKIES") {
    forwardToBackground(
      { type: "GOVIRAL_EXTRACT_COOKIES" },
      "GOVIRAL_COOKIES_RESULT",
      requestId
    );
  }

  if (type === "GOVIRAL_LINKEDIN_FETCH_POSTS") {
    forwardToBackground(
      { type: "GOVIRAL_LINKEDIN_FETCH_POSTS", count: event.data.count },
      "GOVIRAL_LINKEDIN_FETCH_POSTS_RESULT",
      requestId
    );
  }

  if (type === "GOVIRAL_LINKEDIN_FETCH_FEED") {
    forwardToBackground(
      { type: "GOVIRAL_LINKEDIN_FETCH_FEED", count: event.data.count },
      "GOVIRAL_LINKEDIN_FETCH_FEED_RESULT",
      requestId
    );
  }

  if (type === "GOVIRAL_LINKEDIN_SEARCH_POSTS") {
    forwardToBackground(
      {
        type: "GOVIRAL_LINKEDIN_SEARCH_POSTS",
        keywords: event.data.keywords,
        count: event.data.count,
      },
      "GOVIRAL_LINKEDIN_SEARCH_POSTS_RESULT",
      requestId
    );
  }

  if (type === "GOVIRAL_LINKEDIN_FETCH_TRENDING") {
    forwardToBackground(
      {
        type: "GOVIRAL_LINKEDIN_FETCH_TRENDING",
        niches: event.data.niches,
        period: event.data.period,
        keywords: event.data.keywords,
        count: event.data.count,
      },
      "GOVIRAL_LINKEDIN_FETCH_TRENDING_RESULT",
      requestId
    );
  }
});
