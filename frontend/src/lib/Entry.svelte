<script>
  import Form from './Form.svelte';
  import { renderMarkdown } from './markdown.js';

  let { data, pid, comments = $bindable(), count = $bindable() } = $props();
  let fullDate = $derived(new Date(data.created));
  let displayDate = $derived(`${fullDate.getFullYear()}-${fullDate.getMonth()+1}-${fullDate.getDate()}`);
  let renderedContent = $derived(renderMarkdown(data.content));
</script>

<article id={data.id} class:comment-new={data.new}>
  <aside class="comment-avatar">
    <img
      src="https://seccdn.libravatar.org/avatar/{data.avatar}?d=retro"
      alt="{data.author}的头像"
      loading="lazy"
      onerror={(e) => e.currentTarget.style.visibility = 'hidden'}
    >
  </aside>
  <div class="comment-wrapper">
    <header>
      {#if data.website}
      <a href={data.website} target="_blank" rel="noreferrer noopener">{data.author}</a>
      {:else}
      <span>{data.author}</span>
      {/if}
      {#if data.isMod}
      <small title="MOD">🚩</small>
      {/if}
      <span> &#183; </span>
      <span class="comment-date" title={fullDate.toString()}>{displayDate}</span>
    </header>
    <main>
      {@html renderedContent}
    </main>
    <button
      class="reply-btn"
      type="button"
      aria-expanded={data.formOpened}
      onclick={() => { data.formOpened = !data.formOpened; }}
    >{data.formOpened ? "关闭" : "回复"}</button>
    {#if data.formOpened}
    <Form pid={pid} rid={data.id} bind:formOpened={data.formOpened} bind:comments={comments} bind:count={count}/>
    {/if}
  </div>
</article>
