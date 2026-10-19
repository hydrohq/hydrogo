<script lang="ts">
  import NewsCard from '$lib/components/NewsCard.svelte';
  import Seo from '$lib/components/Seo.svelte';
  import { localePrefix } from '$lib/config';

  let { data } = $props();
  const prefix = $derived(localePrefix(data.locale));
</script>

<Seo title="News · {data.siteName}" path="/news" locale={data.locale} siteOrigin={data.siteOrigin} />

<h1>News</h1>
<div class="grid">
  {#each data.news as { item, cover } (item.id)}
    <NewsCard {item} {cover} href="{prefix}/news/{item.slug}" />
  {:else}
    <p>Nothing here yet.</p>
  {/each}
</div>
<nav class="pager" aria-label="Pagination">
  {#if data.pageNumber > 1}<a href="?page={data.pageNumber - 1}" rel="prev">← Newer</a>{/if}
  {#if data.hasMore}<a href="?page={data.pageNumber + 1}" rel="next">Older →</a>{/if}
</nav>
