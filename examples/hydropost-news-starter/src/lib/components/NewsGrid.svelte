<script lang="ts">
  import { type NewsArticle, type Ad, getMediaUrl, safeUrl } from '#lib';
  import NewsCard from './NewsCard.svelte';

  interface Props {
    items: NewsArticle[];
    ads?: (Ad | null)[];
  }

  let { items = [], ads = [] }: Props = $props();
</script>

<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
  {#each items as item, i (item.id)}
    <NewsCard {item} variant="tile" index={i} />
  {/each}

  {#each ads as ad}
    <div class="relative rounded-xl overflow-hidden border border-dashed border-[var(--hp-border-strong)] bg-[var(--hp-surface-sunken)] aspect-[4/3]">
      {#if ad}
        <a href={safeUrl(ad.target_url)} target="_blank" rel="noopener noreferrer sponsored" class="absolute inset-0 flex items-center justify-center">
          <img src={getMediaUrl(ad.banner_image)} alt={ad.name} class="max-w-full max-h-full object-contain" loading="lazy" />
        </a>
      {:else}
        <div class="absolute inset-0 flex flex-col items-center justify-center gap-1.5 text-center px-4">
          <span class="px-1.5 py-0.5 rounded bg-amber-400 text-[9px] font-bold uppercase tracking-wider text-[var(--hp-ink)]">Reklam</span>
          <span class="text-xs text-[var(--hp-subtle)]">İçerik henüz yüklenmedi</span>
        </div>
      {/if}
    </div>
  {/each}
</div>
