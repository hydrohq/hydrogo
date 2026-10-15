<script lang="ts">
  import { onMount } from 'svelte';
  import { fetchNewsBySlug, type NewsArticle } from './api';

  interface Props {
    slug: string;
    onback: () => void;
  }

  let { slug, onback }: Props = $props();

  let article = $state<NewsArticle | null>(null);
  let loading = $state(true);

  onMount(async () => {
    loading = true;
    try {
      article = await fetchNewsBySlug(slug);
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
    ← Back to Articles
  </button>

  {#if loading}
    <div class="py-12 text-center text-sm text-slate-500 animate-pulse">Loading article…</div>
  {:else if !article}
    <div class="p-8 rounded-xl border border-slate-800 bg-slate-900 text-center space-y-2">
      <h3 class="text-base font-semibold text-slate-100">Article Not Found</h3>
      <p class="text-xs text-slate-400">The requested article slug "{slug}" was not found or is unpublished.</p>
    </div>
  {:else}
    <article class="space-y-6">
      <div class="pb-6 border-b border-slate-800 space-y-3">
        <div class="flex items-center gap-2">
          {#if article.category}
            <span class="px-2 py-0.5 rounded text-[10px] font-semibold bg-sky-500/10 text-sky-400 uppercase">
              {article.category}
            </span>
          {/if}
          {#if article.published_at}
            <span class="text-xs text-slate-500">
              {new Date(article.published_at).toLocaleDateString()}
            </span>
          {/if}
          {#if article.author_name}
            <span class="text-xs text-slate-400">• by {article.author_name}</span>
          {/if}
        </div>

        <h1 class="text-3xl md:text-4xl font-extrabold text-white">{article.title}</h1>
        {#if article.excerpt}
          <p class="text-sm text-slate-300 font-medium">{article.excerpt}</p>
        {/if}
      </div>

      {#if article.cover_media_id}
        <div class="aspect-video rounded-xl bg-slate-900 border border-slate-800 overflow-hidden">
          <img
            src={`/media/${article.cover_media_id}`}
            alt={article.title}
            class="w-full h-full object-cover"
          />
        </div>
      {/if}

      {#if article.body}
        <div class="prose prose-invert max-w-none text-slate-300 text-sm leading-relaxed whitespace-pre-wrap">
          {article.body}
        </div>
      {/if}

      {#if article.tags && article.tags.length > 0}
        <div class="pt-6 border-t border-slate-800 flex items-center gap-2">
          <span class="text-xs text-slate-500">Tags:</span>
          {#each article.tags as tag}
            <span class="px-2 py-0.5 rounded bg-slate-800 text-slate-400 text-xs font-mono">
              #{tag}
            </span>
          {/each}
        </div>
      {/if}
    </article>
  {/if}
</div>
