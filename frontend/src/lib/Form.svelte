<script>
  import { getContext, onMount } from "svelte";
  import { renderMarkdown } from "./markdown.js";
  import { loadTurnstile } from "./turnstile.js";
  const config = getContext("config");

  let { pid = "", rid = "", comments = $bindable([]), count = $bindable(), formOpened = $bindable(true) } = $props();

  let showPreview = $state(false);
  let turnstileContainer;
  let turnstileWidgetId;
  let turnstileWidgetPromise;
  let turnstileWasExecuted = false;
  let turnstileChallengePromise;
  let turnstileChallengeResolve;
  let turnstileChallengeReject;
  let turnstileToken;
  let turnstileTokenExpiresAt = 0;
  let newComment = $state({
    uri: config.pageUri,
    author: "",
    email: "",
    website: "",
    content: "",
  });

  function prepareTurnstileWidget() {
    if (!config.turnstileSiteKey) {
      return Promise.reject(new Error("评论验证未配置 Site Key"));
    }
    if (turnstileWidgetPromise) return turnstileWidgetPromise;

    turnstileWidgetPromise = loadTurnstile()
      .then((turnstile) => {
        if (turnstileWidgetId === undefined) {
          turnstileWidgetId = turnstile.render(turnstileContainer, {
            sitekey: config.turnstileSiteKey,
            size: "invisible",
            execution: "execute",
            callback: (token) => {
              turnstileToken = token;
              turnstileTokenExpiresAt = Date.now() + 4 * 60 * 1000;
              turnstileChallengeResolve?.(token);
            },
            "error-callback": () => turnstileChallengeReject?.(new Error("Turnstile 验证失败，请重试")),
            "expired-callback": () => {
              turnstileToken = undefined;
              turnstileTokenExpiresAt = 0;
              turnstileChallengeReject?.(new Error("Turnstile 验证已过期，请重试"));
            },
          });
        }
        return turnstile;
      })
      .catch((error) => {
        turnstileWidgetPromise = undefined;
        throw error;
      });

    return turnstileWidgetPromise;
  }

  onMount(() => {
    if (!config.turnstileSiteKey) return;
    prepareTurnstileWidget().catch((error) => {
      console.warn("Turnstile 预加载失败，提交时将重试:", error);
    });
  });

  async function getTurnstileToken() {
    const token = await requestTurnstileToken();
    turnstileToken = undefined;
    turnstileTokenExpiresAt = 0;
    return token;
  }

  async function requestTurnstileToken() {
    if (turnstileToken && Date.now() < turnstileTokenExpiresAt) return turnstileToken;
    if (turnstileChallengePromise) return turnstileChallengePromise;

    const turnstile = await prepareTurnstileWidget();
    if (turnstileToken && Date.now() < turnstileTokenExpiresAt) return turnstileToken;
    if (turnstileChallengePromise) return turnstileChallengePromise;

    if (turnstileWasExecuted) turnstile.reset(turnstileWidgetId);

    turnstileWasExecuted = true;
    const challengePromise = new Promise((resolve, reject) => {
      const clearChallenge = () => {
        clearTimeout(timeout);
        turnstileChallengeResolve = undefined;
        turnstileChallengeReject = undefined;
        turnstileChallengePromise = undefined;
      };

      const timeout = setTimeout(() => {
        turnstileChallengeReject?.(new Error("Turnstile 验证超时，请重试"));
      }, 120000);
      turnstileChallengeResolve = (token) => {
        clearChallenge();
        resolve(token);
      };
      turnstileChallengeReject = (error) => {
        clearChallenge();
        reject(error);
      };
    });
    turnstileChallengePromise = challengePromise;

    try {
      turnstile.execute(turnstileWidgetId);
    } catch (error) {
      turnstileChallengeReject?.(error);
    }

    return challengePromise;
  }

  function warmTurnstileToken() {
    requestTurnstileToken().catch((error) => {
      console.warn("Turnstile 预验证失败，提交时将重试:", error);
    });
  }

  async function sendComment(e) {
    e.preventDefault();
    const form = e.currentTarget;
    const submitBtn = form.querySelector('button[type="submit"]');

    if (newComment.website && !/^https?:\/\//i.test(newComment.website)) {
      alert("网址必须以 http:// 或 https:// 开头");
      return;
    }

    submitBtn.disabled = true;
    try {
      const turnstileToken = await getTurnstileToken();
      const response = await fetch(config.url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json; charset=UTF-8",
        },
        body: JSON.stringify({ ...newComment, pid, rid, turnstileToken }),
      });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const data = await response.json();

      // Put new comment to page
      count++;
      if (pid === "") {
        comments = [
          {
            id: data.id,
            author: data.author,
            avatar: data.avatar,
            website: data.website,
            content: data.content,
            created: data.created,
            isMod: data.isMod,
            replies: [],
            new: true,
          },
          ...comments,
        ];
      } else {
        const index = comments.findIndex((c) => c.id === pid);
        if (index !== -1) {
          comments[index].replies = [
            ...comments[index].replies,
            {
              id: data.id,
              author: data.author,
              avatar: data.avatar,
              website: data.website,
              content: data.content,
              created: data.created,
              isMod: data.isMod,
              pid: data.pid,
              rid: data.rid,
              new: true,
            },
          ];
        }
      }
      newComment.content = "";
      formOpened = false;
    } catch (err) {
      console.error("评论发送失败:", err);
      alert(err instanceof Error ? err.message : "评论发送失败，请重试");
    } finally {
      submitBtn.disabled = false;
    }
  }
</script>

<form class="comment-form" onsubmit={sendComment}>
  <fieldset>
    <legend>添加评论</legend>
    <div class="comment-info">
    <label for="author">
      名字<span class="required" aria-hidden="true">*</span>
      <input type="text" name="author" id="author" autocomplete="username" bind:value={newComment.author} onfocus={warmTurnstileToken} required>
    </label>
    <label for="email">
      邮箱<span class="required" aria-hidden="true">*</span>
      <input type="email" name="email" id="email" autocomplete="email" bind:value={newComment.email} onfocus={warmTurnstileToken} required>
    </label>
    <label for="website">
      网址
      <input type="url" name="website" id="website" autocomplete="url" bind:value={newComment.website} onfocus={warmTurnstileToken}>
    </label>
    </div>
    <textarea name="content" placeholder="欢迎评论……（支持 Markdown 语法，电邮地址不会公开）" rows="8" bind:value={newComment.content} onfocus={warmTurnstileToken} required></textarea>
    <div bind:this={turnstileContainer}></div>
    {#if showPreview}
    <div class="comment-preview">
      {@html renderMarkdown(newComment.content)}
    </div>
    {/if}
    <button type="button" onclick={() => { showPreview = !showPreview; }}>
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-eye"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg> 预览
    </button
    ><button type="submit">
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-send"><line x1="22" y1="2" x2="11" y2="13"></line><polygon points="22 2 15 22 11 13 2 9 22 2"></polygon></svg> 发送
    </button>
  </fieldset>
</form>
