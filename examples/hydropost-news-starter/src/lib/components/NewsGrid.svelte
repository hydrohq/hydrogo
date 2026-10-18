<script lang="ts">
  import { type NewsArticle, type Ad, type Poll, getMediaUrl, safeUrl } from '#lib';
  import NewsCard from './NewsCard.svelte';
  import PollWidget from './PollWidget.svelte';

  type ExtraCell = { kind: 'ad'; ad: Ad | null } | { kind: 'poll'; poll: Poll | null };
  type Cell = { kind: 'article'; item: NewsArticle; index: number } | ExtraCell;

  interface Props {
    items: NewsArticle[];
    /** Ad slots and/or a poll to weave in among the news cards — spread
     * evenly through the grid instead of clumped together at the end. */
    extraCells?: ExtraCell[];
    columns?: 3 | 4;
    showCategory?: boolean;
  }

  let { items = [], extraCells = [], columns = 3, showCategory = false }: Props = $props();

  // Spreads extraCells evenly through the article list (e.g. 6 articles +
  // 2 extras -> positions after article 2 and article 4) rather than
  // appending them all together at the end.
  let cells = $derived.by<Cell[]>(() => {
    const out: Cell[] = [];
    const step = (items.length + 1) / (extraCells.length + 1);
    let nextExtraAt = step;
    let extraIdx = 0;
    items.forEach((item, i) => {
      out.push({ kind: 'article', item, index: i });
      while (extraIdx < extraCells.length && i + 1 >= nextExtraAt) {
        out.push(extraCells[extraIdx]);
        extraIdx += 1;
        nextExtraAt += step;
      }
    });
    while (extraIdx < extraCells.length) {
      out.push(extraCells[extraIdx]);
      extraIdx += 1;
    }
    return out;
  });
</script>

<div class={`grid grid-cols-1 sm:grid-cols-2 gap-5 ${columns === 4 ? 'lg:grid-cols-4' : 'lg:grid-cols-3'}`}>
  {#each cells as cell, i (i)}
    {#if cell.kind === 'article'}
      <NewsCard item={cell.item} variant="tile" index={cell.index} {showCategory} />
    {:else if cell.kind === 'poll'}
      <div class="sm:col-span-2 lg:col-span-1">
        <PollWidget poll={cell.poll} />
      </div>
    {:else if cell.kind === 'ad'}
      <div class="relative rounded-xl overflow-hidden border border-dashed border-[var(--hp-border-strong)] bg-[var(--hp-surface-sunken)] aspect-[4/3]">
        {#if cell.ad}
          <a href={safeUrl(cell.ad.target_url)} target="_blank" rel="noopener noreferrer sponsored" class="absolute inset-0 flex items-center justify-center">
            <img src={getMediaUrl(cell.ad.banner_image)} alt={cell.ad.name} class="max-w-full max-h-full object-contain" loading="lazy" />
          </a>
        {:else}
          <div class="absolute inset-0 flex flex-col items-center justify-center gap-1.5 text-center px-4">
            <span class="px-1.5 py-0.5 rounded bg-amber-400 text-[9px] font-bold uppercase tracking-wider text-[var(--hp-ink)]">Reklam</span>
            <span class="text-xs text-[var(--hp-subtle)]">İçerik henüz yüklenmedi</span>
          </div>
        {/if}
      </div>
    {/if}
  {/each}
</div>
