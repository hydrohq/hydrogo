<script lang="ts">
  import { onMount } from 'svelte';
  import { fetchPublishedPages, fetchPublishedNews, type Page, type NewsArticle } from './api';

  interface Props {
    onopenpage: (slug: string) => void;
    onopennews: (slug: string) => void;
  }

  let { onopenpage, onopennews }: Props = $props();

  let pages = $state<Page[]>([]);
  let news = $state<NewsArticle[]>([]);
  let loading = $state(true);

  onMount(async () => {
    loading = true;
    try {
      const [pagesRes, newsRes] = await Promise.allSettled([
        fetchPublishedPages(),
        fetchPublishedNews(),
      ]);
      if (pagesRes.status === 'fulfilled') pages = pagesRes.value;
      if (newsRes.status === 'fulfilled') news = newsRes.value;
    } finally {
      loading = false;
    }
  });
</script>

<div class="space-y-12">
  <!-- Hero -->
  <section class="py-12 border-b border-slate-800 text-center space-y-4">
    <div class="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-sky-500/10 border border-sky-500/20 text-sky-400 text-xs font-mono">
      <span>●</span> Headless Svelte Client
    </div>
    <h1 class="text-4xl md:text-5xl font-extrabold tracking-tight text-white max-w-2xl mx-auto">
      Powered by <span class="text-sky-400">HydroGo</span> API
    </h1>
    <p class="text-slate-400 text-base max-w-xl mx-auto">
      This frontend consumes pure JSON endpoints served by HydroGo v2 with zero server-side HTML rendering.
    </p>
  </section>

  <!-- Published Pages -->
  <section class="space-y-4">
    <div class="flex items-center justify-between">
      <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2">
        <span>📑</span> Published Pages
      </h2>
      <span class="text-xs text-slate-500 font-mono">{pages.length} pages</span>
    </div>

    {#if loading}
      <div class="py-12 text-center text-sm text-slate-500 animate-pulse">Loading pages from API…</div>
    {:else if pages.length === 0}
      <div class="p-8 rounded-xl border border-slate-800 bg-slate-900/40 text-center text-slate-500 text-sm">
        No published pages found. Publish pages in HydroGo Admin to see them here.
      </div>
    {:else}
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {#each pages as page (page.id)}
          <button
            type="button"
            onclick={() => onopenpage(page.slug)}
            class="p-5 rounded-xl border border-slate-800 bg-slate-900/60 hover:bg-slate-900 hover:border-slate-700 transition-all text-left flex flex-col justify-between group cursor-pointer"
          >
            <div>
              <h3 class="font-semibold text-slate-100 group-hover:text-sky-400 transition-colors">
                {page.title}
              </h3>
              <p class="text-xs text-slate-500 font-mono mt-1">/{page.slug}</p>
            </div>
            <div class="mt-4 pt-3 border-t border-slate-800 text-xs text-sky-400 font-medium">
              Read Page →
            </div>
          </button>
        {/each}
      </div>
    {/if}
  </section>

  <!-- Latest News -->
  <section class="space-y-4">
    <div class="flex items-center justify-between">
      <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2">
        <span>📰</span> Latest News
      </h2>
      <span class="text-xs text-slate-500 font-mono">{news.length} articles</span>
    </div>

    {#if loading}
      <div class="py-12 text-center text-sm text-slate-500 animate-pulse">Loading news…</div>
    {:else if news.length === 0}
      <div class="p-8 rounded-xl border border-slate-800 bg-slate-900/40 text-center text-slate-500 text-sm">
        No news published yet.
      </div>
    {:else}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        {#each news as article (article.id)}
          <button
            type="button"
            onclick={() => onopennews(article.slug)}
            class="p-5 rounded-xl border border-slate-800 bg-slate-900/60 hover:bg-slate-900 hover:border-slate-700 transition-all text-left flex flex-col justify-between group cursor-pointer"
          >
            <div>
              <div class="flex items-center gap-2 mb-2">
                {#if article.category}
                  <span class="px-2 py-0.5 rounded text-[10px] font-semibold bg-sky-500/10 text-sky-400 uppercase">
                    {article.category}
                  </span>
                {/if}
                {#if article.author_name}
                  <span class="text-xs text-slate-500">by {article.author_name}</span>
                {/if}
              </div>
              <h3 class="font-semibold text-base text-slate-100 group-hover:text-sky-400 transition-colors">
                {article.title}
              </h3>
              {#if article.excerpt}
                <p class="text-xs text-slate-400 mt-2 line-clamp-2">{article.excerpt}</p>
              {/if}
            </div>
            <div class="mt-4 pt-3 border-t border-slate-800 text-xs text-sky-400 font-medium">
              Read Article →
            </div>
          </button>
        {/each}
      </div>
    {/if}
  </section>
</div>
