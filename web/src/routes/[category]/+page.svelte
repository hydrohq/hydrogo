<script lang="ts">
  import type { PageProps } from './$types';
  import { getMediaUrl, getCategoryBadge, slugify } from '#lib';
  import AdSlot from '#lib/components/AdSlot.svelte';

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
  <nav class="flex items-center gap-2 text-xs text-zinc-500 mb-4 select-none">
    <a href="/" class="hover:text-zinc-300 transition-colors">Ana Sayfa</a>
    <span>/</span>
    <span class="text-zinc-300 font-semibold uppercase">{data.category}</span>
  </nav>

  <!-- Category Title & Accent Header -->
  <div class="flex items-center justify-between pb-3 mb-6 border-b-2 border-zinc-800">
    <div class="flex items-center gap-3">
      <span class={`w-3.5 h-8 rounded-full ${badge.bg}`}></span>
      <div>
        <h1 class="text-2xl sm:text-3xl font-extrabold text-white tracking-tight uppercase">
          {categoryTitle} Haberleri
        </h1>
        <p class="text-xs text-zinc-400 mt-0.5">
          Günün öne çıkan {data.category} başlıkları ve son dakika gelişmeleri
        </p>
      </div>
    </div>
    <span class="text-xs text-zinc-500 font-mono hidden sm:inline">
      {data.articles.length} Haber
    </span>
  </div>

  <AdSlot zone="header_top" class="mb-6" />

  {#if data.articles.length === 0}
    <div class="text-center py-16 bg-zinc-950 rounded-2xl border border-zinc-800/80 p-8 space-y-3">
      <div class="text-4xl">📰</div>
      <h3 class="text-lg font-bold text-white">Bu kategoride henüz haber bulunmuyor</h3>
      <p class="text-xs text-zinc-500 max-w-sm mx-auto">
        Yeni haberler eklendiğinde burada listelenecektir. Ana sayfaya dönerek güncel manşetleri inceleyebilirsiniz.
      </p>
      <a href="/" class="inline-block px-4 py-2 rounded-xl bg-red-600 hover:bg-red-500 text-white text-xs font-bold transition-colors">
        Ana Sayfaya Dön
      </a>
    </div>
  {:else}
    <!-- Top Highlight Card (if more than 1) -->
    {@const featured = data.articles[0]}
    <div class="mb-8">
      <a
        href={`/${slugify(featured.category || data.category)}/${featured.slug}`}
        class="group relative block rounded-2xl overflow-hidden bg-zinc-900 border border-zinc-800 shadow-2xl aspect-[16/9] md:aspect-[21/9] transition-all duration-300 hover:border-red-500/80 hover:shadow-red-950/40"
      >
        <img
          src={getMediaUrl(featured.cover_media_id, featured.category || data.category)}
          alt={featured.title}
          class="w-full h-full object-cover transition-transform duration-700 ease-out group-hover:scale-108"
        />
        <div class="absolute inset-0 bg-gradient-to-t from-black via-black/50 to-transparent pointer-events-none"></div>
        <div class="absolute bottom-0 inset-x-0 p-6 sm:p-8 space-y-2 max-w-3xl">
          <span class={`inline-block px-2.5 py-1 rounded ${badge.bg} ${badge.text} text-[10px] font-bold uppercase tracking-wider shadow`}>
            {featured.category || data.category}
          </span>
          <h2 class="text-xl sm:text-2xl md:text-4xl font-extrabold text-white leading-tight group-hover:text-red-400 transition-colors drop-shadow">
            {featured.title}
          </h2>
          {#if featured.excerpt}
            <p class="text-xs sm:text-sm text-zinc-300 line-clamp-2 max-w-2xl drop-shadow hidden sm:block">
              {featured.excerpt}
            </p>
          {/if}
        </div>
      </a>
    </div>

    <!-- Remaining Grid -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
      {#each data.articles.slice(1) as art}
        <a
          href={`/${slugify(art.category || data.category)}/${art.slug}`}
          class="group flex flex-col rounded-2xl overflow-hidden bg-zinc-900/60 border border-zinc-800/80 hover:border-red-500/80 hover:bg-zinc-900 transition-all duration-300 hover:-translate-y-1.5 hover:shadow-2xl hover:shadow-red-950/30"
        >
          <div class="aspect-video w-full overflow-hidden bg-zinc-800 relative">
            <img
              src={getMediaUrl(art.cover_media_id, art.category || data.category)}
              alt={art.title}
              class="w-full h-full object-cover transition-transform duration-500 ease-out group-hover:scale-115"
              loading="lazy"
            />
            <div class="absolute top-3 left-3">
              <span class={`px-2 py-0.5 rounded text-[9px] font-bold uppercase tracking-wider ${badge.bg} ${badge.text} shadow`}>
                {art.category || data.category}
              </span>
            </div>
          </div>

          <div class="p-4 flex-1 flex flex-col justify-between space-y-3">
            <div class="space-y-1.5">
              <h3 class="text-sm sm:text-base font-bold text-white line-clamp-2 leading-snug group-hover:text-red-400 transition-colors">
                {art.title}
              </h3>
              {#if art.excerpt}
                <p class="text-xs text-zinc-400 line-clamp-2 leading-relaxed">
                  {art.excerpt}
                </p>
              {/if}
            </div>

            <div class="pt-3 border-t border-zinc-800/80 flex items-center justify-between text-[11px] text-zinc-500">
              <span>{art.author_name || 'Haber Merkezi'}</span>
              <span>{new Date(art.created_at).toLocaleDateString('tr-TR')}</span>
            </div>
          </div>
        </a>
      {/each}
    </div>
  {/if}

  <AdSlot zone="footer" class="my-8" />
</div>
