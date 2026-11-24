# PocketBase Comment

A simple comment module for your static site, powered by PocketBase and Svelte.

## Human verification (FCaptcha)

Comment submission is protected by a self-hosted [FCaptcha](https://github.com/WebDecoy/FCaptcha) instance using its **interactive checkbox widget** ("I'm not a robot"). The widget script, the signal collection endpoint and token verification all live on your FCaptcha server.

1. Deploy FCaptcha and note its URL and verify credential (`FCAPTCHA_VERIFY_SECRET`, which defaults to `FCAPTCHA_SECRET`). Either leave `FCAPTCHA_ALLOWED_HOSTNAMES` unset or add the hostname of the site that embeds the comment widget.
2. Configure the PocketBase server:
   - `FCAPTCHA_VERIFY_SECRET` (**required**) — the verify credential from step 1 (`FCAPTCHA_SECRET` is used as a fallback when this is unset). Never expose it in a page.
   - `FCAPTCHA_URL` (optional) — your FCaptcha origin, default `https://fcaptcha.zak.ee`.
3. Add the site key and server URL to the comments element on the host page:

   ```html
   <div
     id="comments"
     data-url="https://comments.example.com/api/comment"
     data-fcaptcha-sitekey="pocketbase-comment"
     data-fcaptcha-url="https://fcaptcha.zak.ee"
   ></div>
   <script type="module" src="https://comments.example.com/comment.js"></script>
   ```

   `data-fcaptcha-sitekey` is any label you choose (FCaptcha accepts any key unless `FCAPTCHA_SITE_KEYS` restricts them); it is not a secret. `data-fcaptcha-url` is optional and defaults to `https://fcaptcha.zak.ee`.

The frontend loads `<server>/fcaptcha.js` when the comment form mounts and renders an FCaptcha checkbox inside the main comment form and every reply form. Clicking it verifies the visitor and returns a single-use token; the token is sent as `fcaptchaToken` with the comment and the server validates it against `POST <server>/api/token/verify` before saving. The widget handles its own failure/retry UI and token expiry, and the app resets it after each successful comment so the next one can be verified. The server rejects submissions if the verify secret is missing, the token is invalid or already used, or verification is unavailable.

If the host page uses a Content Security Policy, allow the FCaptcha origin in `script-src` and `connect-src` (the widget posts signals to the captcha server).
