<script lang="ts">
  import { getContext } from 'svelte';
  import snarkdown from 'snarkdown';
  import insane from 'insane';
  export let parentId = "";
  export let comments = [];
  export let count: number;
  export let formOpened = true;
  let showPreview = false;
  const reqUrl:string = getContext("reqUrl");

  let newComment = {
    uri: getContext("pageUri"),
    author: "",
    email: "",
    website: "",
    content: "",
    parent: parentId
  };

  async function sendComment() {
    const submitBtn = this.querySelector('button[type="submit"]');
    submitBtn.disabled = true;

    const response = await fetch(reqUrl, {
      method: "POST",
      headers: {
        "Content-Type": "application/json; charset=UTF-8",
      },
      body: JSON.stringify(newComment)
    });
    const data = await response.json();

    // Put new comment to page
    count++
    if (parentId === "") {
      comments = [{
        id: data.id,
        author: data.author,
        avatar: data.avatar,
        website: data.website,
        content: data.content,
        created: data.created,
        reply: []
      }, ...comments];
    } else {
      const index = comments.findIndex(e => e.id === parentId)
      comments[index].reply.push({
        id: data.id,
        author: data.author,
        avatar: data.avatar,
        website: data.website,
        content: data.content,
        created: data.created,
      });
    };
    newComment.content = "";
    submitBtn.disabled = false;
    formOpened=false;
  };
</script>

<form class="comment-form" on:submit|preventDefault={sendComment}>
  <textarea name="content" placeholder="欢迎评论……（支持 Markdown 语法）" rows="6" bind:value={newComment.content} required></textarea>
  {#if showPreview}
  <div class="comment-preview">
    {@html insane(snarkdown(newComment.content))}
  </div>
  {/if}
  <div class=form-wrapper>
    <label for="author">
      名字<span class="required" aria-hidden="true">*</span>
      <input type="text" name="author" id="author" autocomplete="username" placeholder="John Doe" bind:value={newComment.author} required>
    </label>
    <label for="email">
      邮箱<span class="required" aria-hidden="true">*</span>
      <input type="email" name="email" id="email" autocomplete="email" placeholder="someone@example.com" bind:value={newComment.email} required>
    </label>
  </div>
  <label for="website">
    网址
    <input type="url" name="website" id="website" autocomplete="url" placeholder="https://example.com" bind:value={newComment.website}>
  </label>
  <div class=form-wrapper>
    <button type="button" on:click={()=>{showPreview = !showPreview}}>
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-eye"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg> 预览
    </button>
    <button type="submit">
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-send"><line x1="22" y1="2" x2="11" y2="13"></line><polygon points="22 2 15 22 11 13 2 9 22 2"></polygon></svg> 发送
    </button>
  </div>
</form>
