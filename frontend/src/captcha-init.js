import { loadFCaptcha } from "./lib/fcaptcha.js";

// Runs before the Svelte app is evaluated (main.js imports this module first).
// The host page should load fcaptcha.js synchronously before comment.js; this
// keeps a fallback that starts the download as early as possible so the
// interactive widget can render without waiting for the form to mount.
const target = document.getElementById("comments");
const siteKey = target?.dataset.fcaptchaSitekey;

if (siteKey) {
  loadFCaptcha(target.dataset.fcaptchaUrl).catch((error) => {
    console.warn("FCaptcha 预加载失败，提交时将重试:", error);
  });
}
