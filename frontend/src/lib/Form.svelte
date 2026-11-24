<script>
  import { getContext, onDestroy, onMount } from "svelte";
  import { renderMarkdown } from "./markdown.js";
  import { loadFCaptcha } from "./fcaptcha.js";
  const config = getContext("config");

  let { pid = "", rid = "", comments = $bindable([]), count = $bindable(), formOpened = $bindable(true) } = $props();

  let showPreview = $state(false);
  let fcaptchaContainer;
  let fcaptchaWidgetId;
  let fcaptchaWidgetPromise;
  let fcaptchaToken = "";
  let fcaptchaHint = $state("");
  let newComment = $state({
    uri: config.pageUri,
    author: "",
    email: "",
    website: "",
    content: "",
  });

  function prepareFCaptchaWidget() {
    if (!config.fcaptchaSiteKey) {
      return Promise.reject(new Error("评论验证未配置 Site Key"));
    }
    if (fcaptchaWidgetId !== undefined) return Promise.resolve();
    if (fcaptchaWidgetPromise) return fcaptchaWidgetPromise;

    fcaptchaWidgetPromise = loadFCaptcha(config.fcaptchaUrl)
      .then((fcaptcha) => {
        fcaptchaWidgetId = fcaptcha.render(fcaptchaContainer, {
          siteKey: config.fcaptchaSiteKey,
          theme:
            window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches
              ? "dark"
              : "light",
          // We post the token ourselves; don't add a hidden form field.
          responseField: false,
          callback: (token) => {
            fcaptchaToken = token;
            fcaptchaHint = "";
          },
          expiredCallback: () => {
            fcaptchaToken = "";
            fcaptchaHint = "人机验证已过期，请重新勾选";
          },
          errorCallback: () => {
            fcaptchaToken = "";
            fcaptchaHint = "人机验证失败，请重新勾选";
          },
        });
      })
      .catch((error) => {
        fcaptchaWidgetPromise = undefined;
        fcaptchaHint = "人机验证加载失败，请刷新页面重试";
        throw error;
      });

    return fcaptchaWidgetPromise;
  }

  function resetFCaptchaWidget() {
    fcaptchaToken = "";
    fcaptchaHint = "";
    if (fcaptchaWidgetId !== undefined && window.FCaptcha) {
      window.FCaptcha.reset(fcaptchaWidgetId);
    }
  }

  onMount(() => {
    if (!config.fcaptchaSiteKey) {
      fcaptchaHint = "评论验证未配置 Site Key";
      return;
    }
    prepareFCaptchaWidget().catch((error) => {
      console.warn("FCaptcha 加载失败，提交时将重试:", error);
    });
  });

  onDestroy(() => {
    if (fcaptchaWidgetId !== undefined && window.FCaptcha) {
      window.FCaptcha.destroy(fcaptchaWidgetId);
    }
    fcaptchaWidgetId = undefined;
  });

  async function sendComment(e) {
    e.preventDefault();
    const form = e.currentTarget;
    const submitBtn = form.querySelector('button[type="submit"]');

    if (newComment.website && !/^https?:\/\//i.test(newComment.website)) {
      alert("网址必须以 http:// 或 https:// 开头");
      return;
    }

    if (!fcaptchaToken) {
      if (fcaptchaWidgetId === undefined) {
        fcaptchaHint = "人机验证正在加载，请稍候再试";
        prepareFCaptchaWidget().catch((error) => {
          console.warn("FCaptcha 加载失败:", error);
        });
      } else {
        fcaptchaHint = "请先勾选“我不是机器人”完成验证";
        fcaptchaContainer?.scrollIntoView({ block: "center", behavior: "smooth" });
      }
      return;
    }

    submitBtn.disabled = true;
    try {
      const response = await fetch(config.url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json; charset=UTF-8",
        },
        body: JSON.stringify({ ...newComment, pid, rid, fcaptchaToken }),
      });
      if (!response.ok) {
        let message = `HTTP ${response.status}`;
        try {
          const body = await response.json();
          if (body?.message) message = body.message;
        } catch {}
        if (response.status === 400 && /FCaptcha/i.test(message)) {
          resetFCaptchaWidget();
          throw new Error("人机验证失败或已过期，请重新勾选验证");
        }
        throw new Error(message);
      }
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
      resetFCaptchaWidget();
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
      <input type="text" name="author" id="author" autocomplete="username" bind:value={newComment.author} required>
    </label>
    <label for="email">
      邮箱<span class="required" aria-hidden="true">*</span>
      <input type="email" name="email" id="email" autocomplete="email" bind:value={newComment.email} required>
    </label>
    <label for="website">
      网址
      <input type="url" name="website" id="website" autocomplete="url" bind:value={newComment.website}>
    </label>
    </div>
    <textarea name="content" placeholder="欢迎评论……（支持 Markdown 语法，电邮地址不会公开）" rows="8" bind:value={newComment.content} required></textarea>
    {#if showPreview}
    <div class="comment-preview">
      {@html renderMarkdown(newComment.content)}
    </div>
    {/if}
    <div class="fcaptcha" style="margin: 1rem 0;" bind:this={fcaptchaContainer}></div>
    {#if fcaptchaHint}
    <p class="fcaptcha-hint" role="status" style="color: #b45309; font-size: 0.8em;">{fcaptchaHint}</p>
    {/if}
    <button type="button" onclick={() => { showPreview = !showPreview; }}>
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-eye"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg> 预览
    </button
    ><button type="submit">
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-send"><line x1="22" y1="2" x2="11" y2="13"></line><polygon points="22 2 15 22 11 13 2 9 22 2"></polygon></svg> 发送
    </button>
  </fieldset>
</form>
