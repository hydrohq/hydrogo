<script lang="ts">
  let { data } = $props();
</script>

<svelte:head><title>News</title></svelte:head>

<h1>News</h1>
<ul>
  {#each data.news as item (item.id)}
    <li><a href="/news/{item.slug}">{item.title}</a></li>
  {/each}
</ul>
