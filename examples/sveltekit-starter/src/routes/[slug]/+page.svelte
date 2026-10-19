<script lang="ts">
  let { data } = $props();
</script>

<svelte:head>
  <title>{data.page.title}</title>
  <link rel="canonical" href={data.canonical} />
</svelte:head>

<h1>{data.page.title}</h1>
{@html data.page.body}
