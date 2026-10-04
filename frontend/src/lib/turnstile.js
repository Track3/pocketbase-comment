let loadPromise;
let preconnectAdded = false;

function addPreconnect() {
  if (preconnectAdded) return;
  preconnectAdded = true;

  for (const rel of ["preconnect", "dns-prefetch"]) {
    const link = document.createElement("link");
    link.rel = rel;
    link.href = "https://challenges.cloudflare.com";
    if (rel === "preconnect") link.crossOrigin = "anonymous";
    document.head.appendChild(link);
  }
}

export function loadTurnstile() {
  if (window.turnstile) return Promise.resolve(window.turnstile);
  if (loadPromise) return loadPromise;

  addPreconnect();
  loadPromise = new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";
    script.async = true;
    script.defer = true;
    script.onload = () => {
      if (window.turnstile) resolve(window.turnstile);
      else reject(new Error("Turnstile 未能加载"));
    };
    script.onerror = () => reject(new Error("Turnstile 脚本加载失败"));
    document.head.appendChild(script);
  }).catch((error) => {
    loadPromise = undefined;
    throw error;
  });

  return loadPromise;
}
