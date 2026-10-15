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

<div class="space-y-16 py-4">
  <!-- Next.js Style Hero Section -->
  <section class="relative pt-8 pb-12 text-center space-y-6 overflow-hidden">
    <!-- Ambient background gradient blur -->
    <div
      class="pointer-events-none absolute inset-0 -top-20 flex items-center justify-center opacity-40"
      aria-hidden="true"
    >
      <div class="h-72 w-96 rounded-full bg-gradient-to-tr from-sky-500/30 to-indigo-500/20 blur-3xl"></div>
    </div>

    <!-- Live Status Pill -->
    <div class="relative inline-flex items-center gap-2 px-3 py-1 rounded-full bg-slate-900 border border-slate-800 text-xs text-slate-300 font-mono shadow-sm">
      <span class="flex h-2 w-2 relative">
        <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
        <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
      </span>
      <span>HydroGo Headless Starter • Pure JSON API</span>
    </div>

    <!-- Main Title -->
    <h1 class="relative text-5xl md:text-6xl lg:text-7xl font-extrabold tracking-tight text-white max-w-4xl mx-auto leading-tight">
      Welcome to <span class="bg-gradient-to-r from-sky-400 via-teal-300 to-indigo-400 bg-clip-text text-transparent">HydroGo</span>
    </h1>

    <!-- Subtitle -->
    <p class="relative text-slate-400 text-base md:text-lg max-w-2xl mx-auto leading-relaxed">
      A single-binary headless Go CMS with a pure REST API, embedded Svelte 5 admin studio, and zero server-side HTML rendering.
    </p>

    <!-- Code Instruction Box (Next.js Signature Pattern) -->
    <div class="relative pt-2">
      <div class="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl border border-slate-800 bg-slate-900/80 backdrop-blur-sm text-xs text-slate-400 font-mono shadow-inner">
        <span>Get started by editing</span>
        <code class="font-semibold text-sky-400 bg-slate-950 px-2 py-0.5 rounded border border-slate-800">
          examples/svelte-starter/src/App.svelte
        </code>
      </div>
    </div>

    <!-- Quick Action CTAs -->
    <div class="relative flex flex-wrap items-center justify-center gap-3 pt-2">
      <a
        href="/admin"
        target="_blank"
        rel="noreferrer"
        class="inline-flex items-center gap-2 h-10 px-5 rounded-lg bg-sky-500 text-slate-950 font-semibold text-xs hover:bg-sky-400 transition-colors shadow-md hover:shadow-sky-500/20"
      >
        <span>Open Admin Studio</span>
        <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
          <polyline points="15 3 21 3 21 9"></polyline>
          <line x1="10" y1="14" x2="21" y2="3"></line>
        </svg>
      </a>

      <a
        href="/api/v1/pages?locale=en"
        target="_blank"
        rel="noreferrer"
        class="inline-flex items-center gap-2 h-10 px-4 rounded-lg border border-slate-800 bg-slate-900/60 text-slate-300 font-medium text-xs hover:bg-slate-800 hover:text-white transition-colors"
      >
        <span>Explore REST API</span>
        <span class="text-slate-500 font-mono text-[11px]">(/api/v1)</span>
      </a>
    </div>
  </section>

  <!-- 4 Feature Cards (Next.js Cards Grid Pattern) -->
  <section class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
    <!-- Card 1: Docs & API -->
    <a
      href="/api/v1/pages?locale=en"
      target="_blank"
      rel="noreferrer"
      class="group p-5 rounded-2xl border border-slate-800/80 bg-slate-900/40 hover:bg-slate-900/90 hover:border-slate-700 transition-all duration-200 flex flex-col justify-between"
    >
      <div class="space-y-2">
        <div class="flex items-center justify-between">
          <span class="text-xl">📖</span>
          <span class="text-slate-600 group-hover:text-sky-400 group-hover:translate-x-0.5 group-hover:-translate-y-0.5 transition-transform duration-200 text-sm">↗</span>
        </div>
        <h3 class="font-semibold text-slate-100 group-hover:text-sky-400 transition-colors">
          RESTful API
        </h3>
        <p class="text-xs text-slate-400 leading-relaxed">
          Clean JSON endpoints for pages, news, media, members and custom collections.
        </p>
      </div>
      <div class="mt-4 pt-3 border-t border-slate-800/60 text-[11px] font-mono text-slate-500">
        GET /api/v1/pages
      </div>
    </a>

    <!-- Card 2: Admin Studio -->
    <a
      href="/admin"
      target="_blank"
      rel="noreferrer"
      class="group p-5 rounded-2xl border border-slate-800/80 bg-slate-900/40 hover:bg-slate-900/90 hover:border-slate-700 transition-all duration-200 flex flex-col justify-between"
    >
      <div class="space-y-2">
        <div class="flex items-center justify-between">
          <span class="text-xl">⚡</span>
          <span class="text-slate-600 group-hover:text-sky-400 group-hover:translate-x-0.5 group-hover:-translate-y-0.5 transition-transform duration-200 text-sm">↗</span>
        </div>
        <h3 class="font-semibold text-slate-100 group-hover:text-sky-400 transition-colors">
          Admin Studio
        </h3>
        <p class="text-xs text-slate-400 leading-relaxed">
          Embedded Svelte 5 + shadcn control panel with drafts, revisions and dark mode.
        </p>
      </div>
      <div class="mt-4 pt-3 border-t border-slate-800/60 text-[11px] font-mono text-slate-500">
        SPA at /admin
      </div>
    </a>

    <!-- Card 3: Content Types -->
    <div
      class="p-5 rounded-2xl border border-slate-800/80 bg-slate-900/40 flex flex-col justify-between"
    >
      <div class="space-y-2">
        <div class="flex items-center justify-between">
          <span class="text-xl">🧩</span>
          <span class="text-slate-600 text-sm">YAML</span>
        </div>
        <h3 class="font-semibold text-slate-100">
          Schema Engine
        </h3>
        <p class="text-xs text-slate-400 leading-relaxed">
          Declare FAQ, team, or any custom content type in <code class="text-sky-400 font-mono">content-types/*.yaml</code> with zero Go code.
        </p>
      </div>
      <div class="mt-4 pt-3 border-t border-slate-800/60 text-[11px] font-mono text-slate-500">
        Auto CRUD & Forms
      </div>
    </div>

    <!-- Card 4: Single Binary & FastCGI -->
    <div
      class="p-5 rounded-2xl border border-slate-800/80 bg-slate-900/40 flex flex-col justify-between"
    >
      <div class="space-y-2">
        <div class="flex items-center justify-between">
          <span class="text-xl">🚀</span>
          <span class="text-slate-600 text-sm">Deploy</span>
        </div>
        <h3 class="font-semibold text-slate-100">
          Single Binary & FCGI
        </h3>
        <p class="text-xs text-slate-400 leading-relaxed">
          Zero external runtime dependencies. Runs standalone, in Docker, or via FastCGI/CGI.
        </p>
      </div>
      <div class="mt-4 pt-3 border-t border-slate-800/60 text-[11px] font-mono text-slate-500">
        --fastcgi | --cgi
      </div>
    </div>
  </section>

  <!-- Live Content: Published Pages -->
  <section class="space-y-4 pt-4 border-t border-slate-800/60">
    <div class="flex items-center justify-between">
      <div class="space-y-0.5">
        <h2 class="text-lg font-bold text-slate-100 flex items-center gap-2">
          <span>📑</span> Live Published Pages
        </h2>
        <p class="text-xs text-slate-500">Real data consumed from <code class="text-slate-400">/api/v1/pages</code></p>
      </div>
      <span class="text-xs text-slate-500 font-mono px-2 py-0.5 rounded bg-slate-900 border border-slate-800">
        {pages.length} pages
      </span>
    </div>

    {#if loading}
      <div class="py-12 text-center text-xs text-slate-500 animate-pulse flex items-center justify-center gap-2">
        <div class="size-3.5 rounded-full border-2 border-sky-400 border-t-transparent animate-spin"></div>
        <span>Fetching live pages from HydroGo API…</span>
      </div>
    {:else if pages.length === 0}
      <div class="p-8 rounded-xl border border-slate-800/80 bg-slate-900/30 text-center text-slate-500 text-xs">
        No published pages found. Publish pages in HydroGo Admin to see them appear live here.
      </div>
    {:else}
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {#each pages as page (page.id)}
          <button
            type="button"
            onclick={() => onopenpage(page.slug)}
            class="p-5 rounded-xl border border-slate-800 bg-slate-900/50 hover:bg-slate-900 hover:border-slate-700 transition-all text-left flex flex-col justify-between group cursor-pointer shadow-xs"
          >
            <div>
              <h3 class="font-semibold text-sm text-slate-100 group-hover:text-sky-400 transition-colors">
                {page.title}
              </h3>
              <p class="text-xs text-slate-500 font-mono mt-1">/{page.slug}</p>
            </div>
            <div class="mt-4 pt-3 border-t border-slate-800/80 text-xs text-sky-400 font-medium flex items-center justify-between">
              <span>Read Page</span>
              <span class="group-hover:translate-x-1 transition-transform">→</span>
            </div>
          </button>
        {/each}
      </div>
    {/if}
  </section>

  <!-- Live Content: Latest News -->
  <section class="space-y-4 pt-4 border-t border-slate-800/60">
    <div class="flex items-center justify-between">
      <div class="space-y-0.5">
        <h2 class="text-lg font-bold text-slate-100 flex items-center gap-2">
          <span>📰</span> Latest News & Articles
        </h2>
        <p class="text-xs text-slate-500">Real data consumed from <code class="text-slate-400">/api/v1/news</code></p>
      </div>
      <span class="text-xs text-slate-500 font-mono px-2 py-0.5 rounded bg-slate-900 border border-slate-800">
        {news.length} articles
      </span>
    </div>

    {#if loading}
      <div class="py-12 text-center text-xs text-slate-500 animate-pulse flex items-center justify-center gap-2">
        <div class="size-3.5 rounded-full border-2 border-sky-400 border-t-transparent animate-spin"></div>
        <span>Fetching news articles…</span>
      </div>
    {:else if news.length === 0}
      <div class="p-8 rounded-xl border border-slate-800/80 bg-slate-900/30 text-center text-slate-500 text-xs">
        No news published yet.
      </div>
    {:else}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        {#each news as article (article.id)}
          <button
            type="button"
            onclick={() => onopennews(article.slug)}
            class="p-5 rounded-xl border border-slate-800 bg-slate-900/50 hover:bg-slate-900 hover:border-slate-700 transition-all text-left flex flex-col justify-between group cursor-pointer shadow-xs"
          >
            <div>
              <div class="flex items-center gap-2 mb-2">
                {#if article.category}
                  <span class="px-2 py-0.5 rounded text-[10px] font-semibold bg-sky-500/10 text-sky-400 uppercase tracking-wider">
                    {article.category}
                  </span>
                {/if}
                {#if article.author_name}
                  <span class="text-xs text-slate-500">by {article.author_name}</span>
                {/if}
              </div>
              <h3 class="font-semibold text-sm text-slate-100 group-hover:text-sky-400 transition-colors">
                {article.title}
              </h3>
              {#if article.excerpt}
                <p class="text-xs text-slate-400 mt-2 line-clamp-2 leading-relaxed">{article.excerpt}</p>
              {/if}
            </div>
            <div class="mt-4 pt-3 border-t border-slate-800/80 text-xs text-sky-400 font-medium flex items-center justify-between">
              <span>Read Article</span>
              <span class="group-hover:translate-x-1 transition-transform">→</span>
            </div>
          </button>
        {/each}
      </div>
    {/if}
  </section>
</div>
