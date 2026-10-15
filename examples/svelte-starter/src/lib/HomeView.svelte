<script lang="ts">
  import { onMount } from 'svelte';
  import { fetchPublishedPages, type Page } from './api';

  interface Props {
    onopenpage: (slug: string) => void;
    onopennews: (slug: string) => void;
  }

  let { onopenpage }: Props = $props();

  let pages = $state<Page[]>([]);

  onMount(async () => {
    try {
      pages = await fetchPublishedPages();
    } catch {
      pages = [];
    }
  });
</script>

<div class="flex flex-col items-center justify-between min-h-[calc(100vh-14rem)] py-16 px-4">
  <!-- Center Next.js style Hero -->
  <div class="flex flex-col items-center text-center max-w-2xl my-auto space-y-8">
    <!-- Clean Minimal Logo / Title -->
    <div class="flex items-center gap-3">
      <div class="w-10 h-10 rounded-xl bg-white/5 border border-white/10 flex items-center justify-center text-white shadow-inner">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
          <polyline points="2 17 12 22 22 17"></polyline>
          <polyline points="2 12 12 17 22 12"></polyline>
        </svg>
      </div>
      <span class="text-3xl font-bold tracking-tight text-white font-sans">
        Hydro<span class="text-sky-400">Go</span>
      </span>
    </div>

    <!-- Next.js Signature Prompt Box -->
    <div class="flex items-center gap-2 px-4 py-2 rounded-lg bg-white/5 border border-white/10 text-xs font-mono text-neutral-300">
      <span>Get started by editing</span>
      <code class="font-semibold text-white bg-white/10 px-1.5 py-0.5 rounded">examples/svelte-starter/src/App.svelte</code>
    </div>

    <!-- Action Buttons -->
    <div class="flex items-center gap-4">
      <a
        href="/admin"
        target="_blank"
        rel="noreferrer"
        class="inline-flex items-center justify-center h-10 px-6 rounded-full bg-white text-black font-medium text-xs hover:bg-neutral-200 transition-colors shadow-sm"
      >
        Open Admin Studio
      </a>
      <a
        href="/api/v1/pages?locale=en"
        target="_blank"
        rel="noreferrer"
        class="inline-flex items-center justify-center h-10 px-6 rounded-full bg-white/5 hover:bg-white/10 border border-white/10 text-white font-medium text-xs transition-colors"
      >
        API Reference
      </a>
    </div>
  </div>

  <!-- Next.js Signature 4 Cards Grid at Bottom -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 w-full max-w-5xl mt-16 pt-8 border-t border-white/5">
    <!-- Card 1: Docs -->
    <a
      href="https://github.com/ardaozyildirim/hydrogo"
      target="_blank"
      rel="noreferrer"
      class="group p-4 rounded-xl border border-transparent hover:border-white/10 hover:bg-white/[0.02] transition-colors"
    >
      <h3 class="text-sm font-semibold text-white flex items-center gap-1">
        Docs
        <span class="inline-block transition-transform group-hover:translate-x-1">→</span>
      </h3>
      <p class="text-xs text-neutral-400 mt-1.5 leading-relaxed">
        Find in-depth information about HydroGo features and REST API.
      </p>
    </a>

    <!-- Card 2: Admin Studio -->
    <a
      href="/admin"
      target="_blank"
      rel="noreferrer"
      class="group p-4 rounded-xl border border-transparent hover:border-white/10 hover:bg-white/[0.02] transition-colors"
    >
      <h3 class="text-sm font-semibold text-white flex items-center gap-1">
        Admin
        <span class="inline-block transition-transform group-hover:translate-x-1">→</span>
      </h3>
      <p class="text-xs text-neutral-400 mt-1.5 leading-relaxed">
        Manage pages, articles, and media in the embedded studio.
      </p>
    </a>

    <!-- Card 3: REST API -->
    <a
      href="/api/v1/pages?locale=en"
      target="_blank"
      rel="noreferrer"
      class="group p-4 rounded-xl border border-transparent hover:border-white/10 hover:bg-white/[0.02] transition-colors"
    >
      <h3 class="text-sm font-semibold text-white flex items-center gap-1">
        API
        <span class="inline-block transition-transform group-hover:translate-x-1">→</span>
      </h3>
      <p class="text-xs text-neutral-400 mt-1.5 leading-relaxed">
        Consume clean JSON endpoints for pages, news, and media.
      </p>
    </a>

    <!-- Card 4: Pages / Deploy -->
    {#if pages.length > 0}
      <button
        type="button"
        onclick={() => onopenpage(pages[0].slug)}
        class="group p-4 rounded-xl border border-transparent hover:border-white/10 hover:bg-white/[0.02] transition-colors text-left cursor-pointer"
      >
        <h3 class="text-sm font-semibold text-white flex items-center gap-1">
          Pages
          <span class="inline-block transition-transform group-hover:translate-x-1">→</span>
        </h3>
        <p class="text-xs text-neutral-400 mt-1.5 leading-relaxed">
          {pages.length} live page{pages.length === 1 ? '' : 's'} published. Click to view.
        </p>
      </button>
    {:else}
      <a
        href="https://github.com/ardaozyildirim/hydrogo"
        target="_blank"
        rel="noreferrer"
        class="group p-4 rounded-xl border border-transparent hover:border-white/10 hover:bg-white/[0.02] transition-colors"
      >
        <h3 class="text-sm font-semibold text-white flex items-center gap-1">
          Deploy
          <span class="inline-block transition-transform group-hover:translate-x-1">→</span>
        </h3>
        <p class="text-xs text-neutral-400 mt-1.5 leading-relaxed">
          Deploy instantly to shared hosting with FastCGI or VPS.
        </p>
      </a>
    {/if}
  </div>
</div>
