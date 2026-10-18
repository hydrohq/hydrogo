<script lang="ts">
  import type { PageProps } from './$types';
  import AdSlot from '#lib/components/AdSlot.svelte';
  import NewsCard from '#lib/components/NewsCard.svelte';
  import FeaturedCarousel from '#lib/components/FeaturedCarousel.svelte';

  let { data }: PageProps = $props();
  let categoryTitle = $derived(
    data.category.charAt(0).toUpperCase() + data.category.slice(1)
  );
</script>

<svelte:head>
  <title>{categoryTitle} Haberleri — En Son Gelişmeler — HydroPost</title>
  <meta name="description" content={`${categoryTitle} kategorisindeki en son haberler, analizler ve sıcak gelişmeler.`} />
</svelte:head>

<div class="py-4">
  <!-- Minimal category banner -->
  <div class="mb-6">
    <h1 class="text-2xl sm:text-3xl font-extrabold text-[var(--hp-ink)] tracking-tight">
      {categoryTitle} Haberleri
    </h1>
  </div>

  <AdSlot ad={data.headerTopAd} zone="header_top" class="mb-6" />

  {#if data.articles.length === 0}
    <div class="text-center py-16 bg-[var(--hp-surface-sunken)] rounded-2xl border border-[var(--hp-border)] p-8 space-y-3">
      <div class="text-4xl">📰</div>
      <h3 class="text-lg font-bold text-[var(--hp-ink)]">Bu kategoride henüz haber bulunmuyor</h3>
      <p class="text-xs text-[var(--hp-muted)] max-w-sm mx-auto">
        Yeni haberler eklendiğinde burada listelenecektir. Ana sayfaya dönerek güncel manşetleri inceleyebilirsiniz.
      </p>
      <a href="/" class="inline-block px-4 py-2 rounded-xl bg-[var(--hp-primary)] hover:bg-[var(--hp-primary-dark)] text-white text-xs font-bold transition-colors">
        Ana Sayfaya Dön
      </a>
    </div>
  {:else}
    <!-- Category manşet: the same rotating-hero pattern as the homepage,
         scoped to this category's own articles. -->
    <div class="mb-8">
      <FeaturedCarousel items={data.articles} />
    </div>

    <!-- Remaining Grid -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
      {#each data.articles.slice(14) as art, i (art.id)}
        <NewsCard item={art} variant="tile" index={i} />
      {/each}
    </div>
  {/if}

  <AdSlot ad={data.footerAd} zone="footer" class="my-8" />
</div>
