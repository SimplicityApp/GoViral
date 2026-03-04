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
    chrome.runtime.sendMessage(
      { type: "GOVIRAL_EXTRACT_COOKIES" },
      (response) => {
        window.postMessage(
          { type: "GOVIRAL_COOKIES_RESULT", requestId, ...response },
          window.location.origin
        );
      }
    );
  }

  if (type === "GOVIRAL_LINKEDIN_FETCH_POSTS") {
    chrome.runtime.sendMessage(
      { type: "GOVIRAL_LINKEDIN_FETCH_POSTS", count: event.data.count },
      (response) => {
        window.postMessage(
          { type: "GOVIRAL_LINKEDIN_FETCH_POSTS_RESULT", requestId, ...response },
          window.location.origin
        );
      }
    );
  }

  if (type === "GOVIRAL_LINKEDIN_FETCH_FEED") {
    chrome.runtime.sendMessage(
      { type: "GOVIRAL_LINKEDIN_FETCH_FEED", count: event.data.count },
      (response) => {
        window.postMessage(
          { type: "GOVIRAL_LINKEDIN_FETCH_FEED_RESULT", requestId, ...response },
          window.location.origin
        );
      }
    );
  }

  if (type === "GOVIRAL_LINKEDIN_SEARCH_POSTS") {
    chrome.runtime.sendMessage(
      {
        type: "GOVIRAL_LINKEDIN_SEARCH_POSTS",
        keywords: event.data.keywords,
        count: event.data.count,
      },
      (response) => {
        window.postMessage(
          { type: "GOVIRAL_LINKEDIN_SEARCH_POSTS_RESULT", requestId, ...response },
          window.location.origin
        );
      }
    );
  }

  if (type === "GOVIRAL_LINKEDIN_FETCH_TRENDING") {
    chrome.runtime.sendMessage(
      {
        type: "GOVIRAL_LINKEDIN_FETCH_TRENDING",
        keywords: event.data.keywords,
        count: event.data.count,
      },
      (response) => {
        window.postMessage(
          { type: "GOVIRAL_LINKEDIN_FETCH_TRENDING_RESULT", requestId, ...response },
          window.location.origin
        );
      }
    );
  }
});
