<script lang="ts">
  import { onMount } from 'svelte';
  import { fetchPageBySlug, type Page } from './api';

  interface Props {
    slug: string;
    onback: () => void;
  }

  let { slug, onback }: Props = $props();

  let page = $state<Page | null>(null);
  let loading = $state(true);

  onMount(async () => {
    loading = true;
    try {
      page = await fetchPageBySlug(slug);
    } finally {
      loading = false;
    }
  });
</script>

<div class="max-w-3xl mx-auto space-y-6">
  <button
    type="button"
    onclick={onback}
    class="text-xs text-sky-400 hover:text-sky-300 font-medium cursor-pointer flex items-center gap-1"
  >
    ← Back to Home
  </button>

  {#if loading}
    <div class="py-12 text-center text-sm text-slate-500 animate-pulse">Loading page...</div>
  {:else if !page}
    <div class="p-8 rounded-xl border border-slate-800 bg-slate-900 text-center space-y-2">
      <h3 class="text-base font-semibold text-slate-100">Page Not Found</h3>
      <p class="text-xs text-slate-400">The requested page slug "{slug}" was not found or is unpublished.</p>
    </div>
  {:else}
    <article class="space-y-6">
      <div class="pb-6 border-b border-slate-800">
        <h1 class="text-3xl font-extrabold text-white">{page.title}</h1>
        <p class="text-xs text-slate-500 font-mono mt-2">/{page.slug}</p>
      </div>

      {#if page.body}
        <div class="prose prose-invert max-w-none text-slate-300 text-sm leading-relaxed">
          {@html page.body}
        </div>
      {/if}

      {#if page.blocks && page.blocks.length > 0}
        <div class="space-y-4 pt-4 border-t border-slate-800">
          <h3 class="text-xs uppercase font-mono tracking-wider text-slate-500">Page Blocks</h3>
          {#each page.blocks as block, i (i)}
            <div class="p-4 rounded-xl border border-slate-800 bg-slate-900/60">
              <span class="text-[10px] font-mono px-2 py-0.5 rounded bg-sky-500/10 text-sky-400 uppercase">
                {block.type}
              </span>
              <pre class="mt-2 text-xs text-slate-400 overflow-x-auto">{JSON.stringify(block.payload, null, 2)}</pre>
            </div>
          {/each}
        </div>
      {/if}
    </article>
  {/if}
</div>
