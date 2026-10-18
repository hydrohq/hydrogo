<script lang="ts">
  import type { PageProps } from './$types';
  import { getMediaUrl, sanitizeHtml } from '#lib';
  import AdSlot from '#lib/components/AdSlot.svelte';

  let { data }: PageProps = $props();
  let article = $derived(data.article);
</script>

<svelte:head>
  <title>{article.seo_title || article.title} — HydroPost</title>
  <meta name="description" content={article.seo_description || article.excerpt || article.title} />
  <!-- OpenGraph -->
  <meta property="og:title" content={article.title} />
  <meta property="og:description" content={article.excerpt || article.title} />
  <meta property="og:type" content="article" />
  {#if article.cover_media_id}
    <meta property="og:image" content={getMediaUrl(article.cover_media_id)} />
  {/if}
  <!-- Google News JSON-LD -->
  <script type="application/ld+json">
    {@html JSON.stringify(data.jsonLd)}
  </script>
</svelte:head>

<div class="max-w-4xl mx-auto py-4">
  <!-- Breadcrumbs -->
  <nav class="flex items-center gap-2 text-xs text-zinc-500 mb-4 select-none">
    <a href="/" class="hover:text-zinc-300 transition-colors">Ana Sayfa</a>
    <span>/</span>
    <a href={`/${article.category || 'haber'}`} class="hover:text-zinc-300 transition-colors uppercase">
      {article.category || 'Haber'}
    </a>
    <span>/</span>
    <span class="text-zinc-400 truncate max-w-xs">{article.title}</span>
  </nav>

  <!-- Article Header -->
  <header class="space-y-4 mb-6">
    {#if article.category}
      <span class="inline-block px-3 py-1 rounded bg-red-600 text-white text-xs font-bold uppercase tracking-wider">
        {article.category}
      </span>
    {/if}

    <h1 class="text-2xl sm:text-3xl md:text-5xl font-extrabold text-white leading-tight tracking-tight">
      {article.title}
    </h1>

    {#if article.excerpt}
      <p class="text-base sm:text-lg text-zinc-300 font-medium leading-relaxed border-l-4 border-red-600 pl-4 py-1">
        {article.excerpt}
      </p>
    {/if}

    <div class="flex items-center justify-between py-3 border-y border-zinc-800 text-xs text-zinc-400">
      <div class="flex items-center gap-3">
        {#if article.author_name}
          <span class="font-bold text-white flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full bg-red-500"></span>
            {article.author_name}
          </span>
        {/if}
        <span>Yayınlanma: {new Date(article.created_at).toLocaleDateString('tr-TR', { day: 'numeric', month: 'long', year: 'numeric' })}</span>
      </div>

      <div class="flex items-center gap-2">
        <button
          type="button"
          onclick={() => {
            if (navigator.share) {
              navigator.share({ title: article.title, url: window.location.href });
            } else {
              navigator.clipboard.writeText(window.location.href);
              alert('Haber bağlantısı kopyalandı!');
            }
          }}
          class="px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-semibold transition-colors cursor-pointer"
        >
          Paylaş ↗
        </button>
      </div>
    </div>
  </header>

  <!-- Cover Image -->
  {#if article.cover_media_id}
    <div class="rounded-2xl overflow-hidden bg-zinc-900 border border-zinc-800 mb-8 aspect-video shadow-2xl">
      <img
        src={getMediaUrl(article.cover_media_id)}
        alt={article.title}
        class="w-full h-full object-cover"
      />
    </div>
  {/if}

  <!-- In-Article Ad Slot -->
  <AdSlot zone="haber_ici" />

  <!-- Article Body -->
  <article class="prose prose-invert prose-red max-w-none text-zinc-300 leading-relaxed sm:text-lg">
    {@html sanitizeHtml(article.body)}
  </article>

  <!-- Tags -->
  {#if article.tags}
    {@const tagsList = article.tags.split(',').map((t: string) => t.trim()).filter(Boolean)}
    {#if tagsList.length > 0}
      <div class="mt-8 pt-6 border-t border-zinc-800 flex flex-wrap items-center gap-2">
        <span class="text-xs text-zinc-500 font-medium">Etiketler:</span>
        {#each tagsList as tag}
          <span class="px-2.5 py-1 rounded-lg bg-zinc-900 border border-zinc-800 text-xs font-medium text-zinc-300 hover:text-white transition-colors">
            #{tag}
          </span>
        {/each}
      </div>
    {/if}
  {/if}

  <!-- Bottom Ad Slot -->
  <AdSlot zone="footer" class="my-8" />

  <!-- Related News -->
  {#if data.related && data.related.length > 0}
    <section class="mt-12 pt-8 border-t border-zinc-800">
      <h3 class="text-lg font-bold text-white mb-4 flex items-center gap-2">
        <span class="w-2 h-2 rounded bg-red-600"></span>
        İlgili Haberler
      </h3>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        {#each data.related as r}
          <a
            href={`/${r.category || 'haber'}/${r.slug}`}
            class="group block p-3 rounded-xl bg-zinc-900/60 border border-zinc-800/80 hover:border-zinc-700 transition-colors"
          >
            {#if r.cover_media_id}
              <div class="aspect-video rounded-lg overflow-hidden bg-zinc-800 mb-2.5">
                <img
                  src={getMediaUrl(r.cover_media_id)}
                  alt={r.title}
                  class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                  loading="lazy"
                />
              </div>
            {/if}
            <h4 class="text-xs font-bold text-zinc-200 line-clamp-2 leading-snug group-hover:text-red-400 transition-colors">
              {r.title}
            </h4>
          </a>
        {/each}
      </div>
    </section>
  {/if}
</div>
