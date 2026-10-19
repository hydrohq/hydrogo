<script lang="ts">
  let { data } = $props();
</script>

<svelte:head>
  <title>My site</title>
  <meta name="description" content="Latest news" />
</svelte:head>

<h1>Latest news</h1>
{#each data.news as item (item.id)}
  <article>
    <h2><a href="/news/{item.slug}">{item.title}</a></h2>
  </article>
{:else}
  <p>No news yet. Publish something in the HydroGo admin.</p>
{/each}
