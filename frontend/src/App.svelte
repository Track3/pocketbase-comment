<script lang="ts">
  import { setContext, getContext } from "svelte";
  import Form from './lib/Form.svelte';
  import Entry from './lib/Entry.svelte';

  const url = document.getElementById('comments').dataset.url;
  setContext("pageUri", window.location.pathname);
  setContext("reqUrl", `${url}/?uri=${encodeURIComponent(getContext("pageUri"))}`);
  let count: number;
  let comments = []

  async function getComments() {
    const response = await fetch(getContext("reqUrl"));
    const data = await response.json();
    count = data.count;
    comments = data.list.reverse();
  };

  let promise = getComments();
</script>

<h2><span>评论</span>{#if count}<sup class=comment-counter>{count}</sup>{/if}</h2>
<p>您的电子邮件地址不会被公开。</p>
<Form bind:comments={comments} bind:count={count}/>

{#await promise}
<p>🍵 评论加载中……</p>
{:catch error}
<p style="color: red">🚧 {error}</p>
{/await}

{#each comments as comment}
<div class="comment-group">
  <Entry data={comment} parentId={comment.id} bind:comments={comments} bind:count={count}/>
  {#if comment.reply.length > 0}
  <div class="replies">
    {#each comment.reply as reply}
    <Entry data={reply} parentId={comment.id} bind:comments={comments} bind:count={count}/>
    {/each}
  </div>
  {/if}
</div>
{/each}
