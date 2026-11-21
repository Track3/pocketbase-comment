const DEFAULT_SERVER_URL = "https://fcaptcha.zak.ee";

let loadPromise;
let preconnectAdded = false;

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
