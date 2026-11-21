import { getFCaptchaSession } from "./lib/fcaptcha.js";

// Runs before the Svelte app is evaluated (main.js imports this module first).
// When the host page loads fcaptcha.js synchronously this starts the invisible
// session before first paint, so the visitor's first pointer/key event is not
// "immediately after session creation" — the timing FCaptcha's anti-CDP rule
// keys on. If fcaptcha.js is not preloaded it is fetched asynchronously here,
// which is still earlier than waiting for the comment form to mount.
const target = document.getElementById("comments");
const siteKey = target?.dataset.fcaptchaSitekey;

if (siteKey) {
  getFCaptchaSession({ siteKey, serverUrl: target.dataset.fcaptchaUrl }).catch((error) => {
    console.warn("FCaptcha 预加载失败，提交时将重试:", error);
  });
}
