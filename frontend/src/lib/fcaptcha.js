const DEFAULT_SERVER_URL = "https://fcaptcha.zak.ee";

let loadPromise;
let preconnectAdded = false;
let sessionPromise;

function normalizeServerUrl(serverUrl) {
  return (serverUrl || DEFAULT_SERVER_URL).replace(/\/+$/, "");
}

function addPreconnect(serverUrl) {
  if (preconnectAdded) return;
  preconnectAdded = true;

  for (const rel of ["preconnect", "dns-prefetch"]) {
    const link = document.createElement("link");
    link.rel = rel;
    link.href = serverUrl;
    if (rel === "preconnect") link.crossOrigin = "anonymous";
    document.head.appendChild(link);
  }
}

export function loadFCaptcha(serverUrl) {
  const base = normalizeServerUrl(serverUrl);

  if (window.FCaptcha) {
    window.FCaptcha.configure({ serverUrl: base });
    return Promise.resolve(window.FCaptcha);
  }
  if (loadPromise) return loadPromise;

  addPreconnect(base);
  loadPromise = new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = `${base}/fcaptcha.js`;
    script.async = true;
    script.defer = true;
    script.onload = () => {
      if (!window.FCaptcha) {
        reject(new Error("FCaptcha 未能加载"));
        return;
      }
      window.FCaptcha.configure({ serverUrl: base });
      resolve(window.FCaptcha);
    };
    script.onerror = () => reject(new Error("FCaptcha 脚本加载失败"));
    document.head.appendChild(script);
  }).catch((error) => {
    loadPromise = undefined;
    throw error;
  });

  return loadPromise;
}

/**
 * One invisible session per page.
 *
 * The session must start as early as possible: FCaptcha measures the time
 * between session creation and the first pointer/key event, and can treat a
 * very short interval as CDP-driven input. Starting it here (before the Svelte
 * tree mounts) gives a real visitor time to react to the page. Sharing it also
 * means reply forms inherit the signals already collected instead of starting
 * a fresh session that is immediately followed by the visitor's click.
 *
 * Load `<server>/fcaptcha.js` synchronously before comment.js in the host page
 * so this does not have to wait for an async network round-trip; see README.
 */
export function getFCaptchaSession({ siteKey, serverUrl } = {}) {
  if (sessionPromise) return sessionPromise;
  if (!siteKey) return Promise.reject(new Error("评论验证未配置 Site Key"));

  sessionPromise = loadFCaptcha(serverUrl)
    .then((fcaptcha) =>
      fcaptcha.invisible({
        siteKey,
        autoScore: false,
      }),
    )
    .catch((error) => {
      sessionPromise = undefined;
      throw error;
    });

  return sessionPromise;
}
