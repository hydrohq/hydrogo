<script lang="ts">
  import type { PageProps } from './$types';
  import { getMediaUrl, getCategoryBadge, slugify } from '#lib';
  import AdSlot from '#lib/components/AdSlot.svelte';
  import NewsCard from '#lib/components/NewsCard.svelte';

  let { data }: PageProps = $props();
  let badge = $derived(getCategoryBadge(data.category));
  let categoryTitle = $derived(
    data.category.charAt(0).toUpperCase() + data.category.slice(1)
  );
</script>

<svelte:head>
  <title>{categoryTitle} Haberleri — En Son Gelişmeler — HydroPost</title>
  <meta name="description" content={`${categoryTitle} kategorisindeki en son haberler, analizler ve sıcak gelişmeler.`} />
</svelte:head>

<div class="max-w-6xl mx-auto py-4">
  <!-- Breadcrumbs -->
  <nav class="flex items-center gap-2 text-xs text-[var(--hp-muted)] mb-4 select-none">
    <a href="/" class="hover:text-[var(--hp-ink-soft)] transition-colors">Ana Sayfa</a>
    <span>/</span>
    <span class="text-[var(--hp-ink-soft)] font-semibold uppercase">{data.category}</span>
  </nav>

  <!-- Category Title & Accent Header -->
  <div class="flex items-center justify-between pb-3 mb-6 border-b-2 border-[var(--hp-border)]">
    <div class="flex items-center gap-3">
      <span class={`w-3.5 h-8 rounded-full ${badge.bg}`}></span>
      <div>
        <h1 class="text-2xl sm:text-3xl font-extrabold text-[var(--hp-ink)] tracking-tight uppercase">
          {categoryTitle} Haberleri
        </h1>
        <p class="text-xs text-[var(--hp-muted)] mt-0.5">
          Günün öne çıkan {data.category} başlıkları ve son dakika gelişmeleri
        </p>
      </div>
    </div>
    <span class="text-xs text-[var(--hp-muted)] font-mono hidden sm:inline">
      {data.articles.length} Haber
    </span>
  </div>

  <AdSlot zone="header_top" class="mb-6" />

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
    <!-- Top Highlight Card (if more than 1) -->
    {@const featured = data.articles[0]}
    <div class="mb-8">
      <a
        href={`/${slugify(featured.category || data.category)}/${featured.slug}`}
        class="group relative block rounded-2xl overflow-hidden bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)] aspect-[16/9] md:aspect-[21/9]"
      >
        <img
          src={getMediaUrl(featured.cover_media_id, featured.category || data.category, featured.slug)}
          alt={featured.title}
          class="w-full h-full object-cover"
        />
        <div class="absolute inset-0 bg-gradient-to-t from-black via-black/50 to-transparent pointer-events-none"></div>
        <div class="absolute bottom-0 inset-x-0 p-6 sm:p-8 space-y-2 max-w-3xl">
          <span class={`inline-block px-2.5 py-1 rounded ${badge.bg} ${badge.text} text-[10px] font-bold uppercase tracking-wider shadow`}>
            {featured.category || data.category}
          </span>
          <h2 class="text-xl sm:text-2xl md:text-4xl font-extrabold text-white leading-tight group-hover:underline drop-shadow">
            {featured.title}
          </h2>
          {#if featured.excerpt}
            <p class="text-xs sm:text-sm text-[#e9e2d3] line-clamp-2 max-w-2xl drop-shadow hidden sm:block">
              {featured.excerpt}
            </p>
          {/if}
        </div>
      </a>
    </div>

    <!-- Remaining Grid -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
      {#each data.articles.slice(1) as art, i (art.id)}
        <NewsCard item={art} variant="tile" index={i} />
      {/each}
    </div>
  {/if}

  <AdSlot zone="footer" class="my-8" />
</div>
