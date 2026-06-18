<script>
  import { setContext } from "svelte";
  import Form from "./lib/Form.svelte";
  import Entry from "./lib/Entry.svelte";

  const config = {
    url: document.getElementById("comments").dataset.url,
    pageUri: window.location.pathname,
  };

  setContext("config", config);
  let count = $state();
  let comments = $state([]);
  let currentPage = $state(1);
  let commentsPerPage = $state();

  let reqUrl = $derived(
    `${config.url}?uri=${encodeURIComponent(config.pageUri)}&page=${currentPage}`,
  );

  async function getComments() {
    try {
      const response = await fetch(reqUrl);
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const data = await response.json();
      count = data.count;
      comments = data.comments;
      commentsPerPage = data.commentsPerPage;
    } catch (err) {
      console.error("加载评论失败:", err);
      throw err;
    }
  }

  let promise = $state(getComments());

  function changePage(delta) {
    currentPage += delta;
    promise = getComments();
  }

  function retry() {
    promise = getComments();
  }
</script>

<Form bind:comments bind:count />

{#if count}
<div class="comment-header">
  <p>
    <strong><span>{count}</span>评论</strong>
  </p>
  <p class="comment-pagination">
    {#if currentPage > 1}
      <button onclick={() => changePage(-1)}>上一页</button>
    {/if}
    {#if comments.length >= commentsPerPage}
      <button onclick={() => changePage(1)}>下一页</button>
    {/if}
  </p>
</div>
{/if}

{#await promise}
  <p>🍵 评论加载中……</p>
{:catch error}
  <p style="color: red">🚧 评论加载失败</p>
  <button onclick={retry}>重试</button>
{/await}

{#if count === 0}
  <p>暂无评论</p>
{/if}

{#each comments as comment (comment.id)}
  <div class="comment-group">
    <Entry data={comment} pid={comment.id} bind:comments bind:count />
    {#if comment.replies !== null && comment.replies.length > 0}
      <div class="replies">
        {#each comment.replies as reply (reply.id)}
          <Entry data={reply} pid={comment.id} bind:comments bind:count />
        {/each}
      </div>
    {/if}
  </div>
{/each}

<p class="comment-pagination">
  {#if currentPage > 1}
    <button onclick={() => changePage(-1)}>上一页</button>
  {/if}
  {#if comments.length >= commentsPerPage}
    <button onclick={() => changePage(1)}>下一页</button>
  {/if}
</p>
