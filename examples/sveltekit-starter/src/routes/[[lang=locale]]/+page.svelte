<script lang="ts">
  import NewsCard from '$lib/components/NewsCard.svelte';
  import Seo from '$lib/components/Seo.svelte';
  import { localePrefix } from '$lib/config';

  let { data } = $props();
  const prefix = $derived(localePrefix(data.locale));
</script>

<Seo title={data.siteName} description={data.tagline} locale={data.locale} siteOrigin={data.siteOrigin} />

<section class="hero">
  <h1>{data.siteName}</h1>
  {#if data.tagline}<p>{data.tagline}</p>{/if}
</section>

<h2 class="section-title">Latest news</h2>
<div class="grid">
  {#each data.news as { item, cover } (item.id)}
    <NewsCard {item} {cover} href="{prefix}/news/{item.slug}" />
  {:else}
    <p>No news yet. Publish an article in the HydroGo admin and refresh.</p>
  {/each}
</div>
<p><a href="{prefix}/news">All news →</a></p>
