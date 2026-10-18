<script lang="ts">
  import { onMount } from 'svelte';
  import type { PageProps } from './$types';
  import NewsCard from '#lib/components/NewsCard.svelte';

  let { data }: PageProps = $props();
  let query = $state('');
  let inputEl = $state<HTMLInputElement | null>(null);

  $effect(() => {
    query = data.q;
  });

  onMount(() => inputEl?.focus());

  function onSubmit(e: SubmitEvent) {
    e.preventDefault();
    const q = query.trim();
    window.location.href = q ? `/ara?q=${encodeURIComponent(q)}` : '/ara';
  }
</script>

<svelte:head>
  <title>{data.q ? `"${data.q}" için arama sonuçları` : 'Haber Ara'} — HydroPost</title>
</svelte:head>

<div class="py-8">
  <form onsubmit={onSubmit} class="flex items-center gap-2 mb-8">
    <div class="relative flex-1">
      <svg class="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--hp-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="11" cy="11" r="7" /><path d="m21 21-4.3-4.3" stroke-linecap="round" />
      </svg>
      <input
        bind:this={inputEl}
        type="search"
        bind:value={query}
        placeholder="Haber, konu veya başlık ara…"
        class="w-full pl-10 pr-4 py-3 rounded-lg border border-[var(--hp-border-strong)] bg-[var(--hp-surface)] text-sm text-[var(--hp-ink)] outline-none focus:border-[var(--hp-primary)] transition-colors"
      />
    </div>
    <button type="submit" class="px-5 py-3 rounded-lg bg-[var(--hp-primary)] hover:bg-[var(--hp-primary-dark)] text-white text-sm font-bold transition-colors cursor-pointer">
      Ara
    </button>
  </form>

  {#if data.q}
    <p class="text-sm text-[var(--hp-muted)] mb-6">
      <strong class="text-[var(--hp-ink)]">"{data.q}"</strong> için {data.results.length} sonuç bulundu.
    </p>
  {/if}

  {#if data.q && data.results.length === 0}
    <div class="text-center py-16 rounded-xl bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)]">
      <p class="text-sm font-bold text-[var(--hp-ink)]">Sonuç bulunamadı</p>
      <p class="text-xs text-[var(--hp-muted)] mt-1">Farklı bir anahtar kelime deneyin.</p>
    </div>
  {:else if data.results.length > 0}
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
      {#each data.results as item (item.id)}
        <NewsCard {item} variant="tile" />
      {/each}
    </div>
  {/if}
</div>
