<script>
  import { getContext, onMount } from "svelte";
  import { renderMarkdown } from "./markdown.js";
  import { getFCaptchaSession } from "./fcaptcha.js";
  const config = getContext("config");

  const FCAPTCHA_ACTION = "comment";
  const FCAPTCHA_TIMEOUT_MS = 60 * 1000;

  let { pid = "", rid = "", comments = $bindable([]), count = $bindable(), formOpened = $bindable(true) } = $props();

  let showPreview = $state(false);
  let newComment = $state({
    uri: config.pageUri,
    author: "",
    email: "",
    website: "",
    content: "",
  });

  function prepareFCaptchaSession() {
    // A single page-level session, started as early as possible (see main.js).
    return getFCaptchaSession({
      siteKey: config.fcaptchaSiteKey,
      serverUrl: config.fcaptchaUrl,
    });
  }

  function withTimeout(promise, ms, message) {
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error(message)), ms);
      promise.then(
        (value) => {
          clearTimeout(timer);
          resolve(value);
        },
        (error) => {
          clearTimeout(timer);
          reject(error);
        },
      );
    });
  }

  onMount(() => {
    if (!config.fcaptchaSiteKey) return;
    prepareFCaptchaSession().catch((error) => {
      console.warn("FCaptcha 预加载失败，提交时将重试:", error);
    });
  });

  function fcaptchaErrorMessage(result) {
    const corroborating = result?.experimental?.corroboratingCategories || [];
    if (corroborating.includes("cdp")) {
      return "人机验证触发了 FCaptcha 的 CDP 防自动化规则（疑似误判），请检查 FCaptcha 服务端策略";
    }
    switch (result?.reason) {
      case "hostname_not_allowed":
        return "人机验证未允许当前域名，请检查 FCaptcha 的 FCAPTCHA_ALLOWED_HOSTNAMES 配置";
      case "rate_limited":
        return "验证请求过于频繁，请稍后再试";
      case "pow_not_satisfied":
        return "人机验证未完成，请稍后重试";
      case "experimental_detection":
        return "人机验证未通过，请稍后重试";
      default:
        return typeof result?.score === "number"
          ? `人机验证未通过（评分 ${result.score.toFixed(2)}），请刷新页面后重试`
          : "人机验证未通过，请重试";
    }
  }

  async function getFCaptchaToken() {
    const session = await prepareFCaptchaSession();

    // Reuse the session across attempts: its behavioural and temporal
    // collectors keep accumulating, and FCaptcha already fetches a fresh PoW
    // challenge for every execute(). Destroying it after a rejection would
    // throw away exactly the signals a retry needs.
    let result;
    try {
      result = await withTimeout(
        session.execute(FCAPTCHA_ACTION),
        FCAPTCHA_TIMEOUT_MS,
        "人机验证超时，请重试",
      );
    } catch (error) {
      console.warn("FCaptcha 验证失败:", error);
      if (error?.code === "server_unavailable") {
        throw new Error("人机验证服务暂时不可用，请稍后重试");
      }
      throw error;
    }

    if (!result?.success || !result.token) {
      console.warn("FCaptcha 验证被拒绝:", result);
      throw new Error(fcaptchaErrorMessage(result));
    }

    return result.token;
  }

  function warmFCaptchaSession() {
    prepareFCaptchaSession().catch((error) => {
      console.warn("FCaptcha 预加载失败，提交时将重试:", error);
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
      const fcaptchaToken = await getFCaptchaToken();
      const response = await fetch(config.url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json; charset=UTF-8",
        },
        body: JSON.stringify({ ...newComment, pid, rid, fcaptchaToken }),
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
      <input type="text" name="author" id="author" autocomplete="username" bind:value={newComment.author} onfocus={warmFCaptchaSession} required>
    </label>
    <label for="email">
      邮箱<span class="required" aria-hidden="true">*</span>
      <input type="email" name="email" id="email" autocomplete="email" bind:value={newComment.email} onfocus={warmFCaptchaSession} required>
    </label>
    <label for="website">
      网址
      <input type="url" name="website" id="website" autocomplete="url" bind:value={newComment.website} onfocus={warmFCaptchaSession}>
    </label>
    </div>
    <textarea name="content" placeholder="欢迎评论……（支持 Markdown 语法，电邮地址不会公开）" rows="8" bind:value={newComment.content} onfocus={warmFCaptchaSession} required></textarea>
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
