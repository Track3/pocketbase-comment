# PocketBase Comment

A simple comment module for your static site, powered by PocketBase and Svelte.

## Human verification (FCaptcha)

Comment submission is protected by a self-hosted [FCaptcha](https://github.com/WebDecoy/FCaptcha) instance in invisible mode. The widget script, the signal collection endpoint and token verification all live on your FCaptcha server.

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
   <script src="https://fcaptcha.zak.ee/fcaptcha.js"></script>
   <script type="module" src="https://comments.example.com/comment.js"></script>
   ```

   `data-fcaptcha-sitekey` is any label you choose (FCaptcha accepts any key unless `FCAPTCHA_SITE_KEYS` restricts them); it is not a secret. `data-fcaptcha-url` is optional and defaults to `https://fcaptcha.zak.ee`.

   Load `fcaptcha.js` **synchronously before `comment.js`**. `comment.js` starts one invisible session for the page as soon as it is evaluated and shares it across the comment and reply forms. Having the FCaptcha script already present lets that happen before first paint; if it has to be fetched asynchronously, the session can start while the visitor is already moving the mouse, which trips FCaptcha's anti-CDP timing rule (see below).

The frontend opens one invisible session at page start so behavioural signals accumulate across the whole visit, including every reply form. On submission it requests a single-use token bound to the `comment` action and sends it as `fcaptchaToken`; the server verifies it against `POST <server>/api/token/verify` before saving the comment. The server rejects submissions if the verify secret is missing, the token is invalid or already used, or verification is unavailable.

If the host page uses a Content Security Policy, allow the FCaptcha origin in `script-src` and `connect-src` (the widget posts signals to the captcha server).
