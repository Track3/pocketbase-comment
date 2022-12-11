<script lang="ts">
  import Form from './Form.svelte';
  import snarkdown from 'snarkdown';
  import insane from 'insane';
  export let parentId = "";
  export let data;
  export let comments = [];
  export let count: number;

  let fullDate: Date;
  let displayDate: string;
  $: fullDate = new Date(data.created);
  $: displayDate = `${fullDate.getFullYear()}-${fullDate.getMonth()+1}-${fullDate.getDate()}`;

</script>


<article id="{data.id}">
  <aside class="comment-avatar">
    <img src="https://gravatar.loli.net/avatar/{data.avatar}?d=mp" alt="{data.author}的头像" loading="lazy">
  </aside>
  <div class="comment-wrapper">
    <header>
      {#if data.website}
      <a href={data.website} target="_blank" rel="noreferrer noopener">{data.author}</a>
      {:else}
      <span>{data.author}</span>
      {/if}
      <span> &#183; </span>
      <span class="comment-date" title={fullDate.toString()}>{displayDate}</span>
    </header>
    <main>
      {@html insane(snarkdown(data.content))}
    </main>
    <button class="reply-btn" type="button" on:click={()=>{data.formOpened = !data.formOpened}}>{data.formOpened? "关闭" : "回复"}</button>
    {#if data.formOpened}
    <Form parentId={parentId} bind:formOpened={data.formOpened} bind:comments={comments} bind:count={count}/>
    {/if}
  </div>
</article>
