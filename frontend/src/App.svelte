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
    const response = await fetch(reqUrl);
    const data = await response.json();
    count = data.count;
    comments = data.comments;
    commentsPerPage = data.commentsPerPage;
  }

  let promise = getComments();
</script>

<Form bind:comments bind:count />

<div class="comment-header">
  <p>
    <strong
      >{#if count}<span>{count}</span>{/if}评论</strong
    >
  </p>
  <p>
    {#if currentPage !== 1}
      <button
        onclick={() => {
          currentPage--;
          getComments();
        }}>上一页</button
      >
    {/if}
    {#if comments.length >= commentsPerPage}
      <button
        onclick={() => {
          currentPage++;
          getComments();
        }}>下一页</button
      >
    {/if}
  </p>
</div>

{#await promise}
  <p>🍵 评论加载中……</p>
{:catch error}
  <p style="color: red">🚧 {error}</p>
{/await}

{#each comments as comment}
  <div class="comment-group">
    <Entry data={comment} pid={comment.id} bind:comments bind:count />
    {#if comment.replies != null && comment.replies.length > 0}
      <div class="replies">
        {#each comment.replies as reply}
          <Entry data={reply} pid={comment.id} bind:comments bind:count />
        {/each}
      </div>
    {/if}
  </div>
{/each}
