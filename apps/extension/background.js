// GoViral Cookie Sync — Background Service Worker

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

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type !== "GOVIRAL_EXTRACT_COOKIES") return false;

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

  // Extract cookies asynchronously
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

  // Return true to indicate we will call sendResponse asynchronously
  return true;
});
