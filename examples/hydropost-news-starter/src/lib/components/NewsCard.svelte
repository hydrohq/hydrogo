<script lang="ts">
  // Single source of truth for how a news article renders as a card
  // anywhere in the app. Change the look here, not per-section.
  import { type NewsArticle, getMediaUrl, slugify } from '#lib';

  interface Props {
    item: NewsArticle;
    /**
     * tile    — image card, white footer strip with the date (grids)
     * overlay — full-bleed image, title+date pinned over a dark gradient (hero side cards)
     * row     — small square thumbnail beside the title (sidebars, "diğer başlıklar")
     */
    variant?: 'tile' | 'overlay' | 'row';
    rank?: number;
    /** Position within the list being rendered — pass the `{#each}` index
     * so sibling cards never repeat the same stand-in photo (see
     * getMediaUrl's `index` param). */
    index?: number;
  }

  let { item, variant = 'tile', rank, index }: Props = $props();

  let href = $derived(`/${slugify(item.category)}/${item.slug}`);
  let dateText = $derived(item.created_at ? new Date(item.created_at).toLocaleDateString('tr-TR') : '');
</script>

{#if variant === 'row'}
  <a href={href} class="group flex items-center gap-3">
    {#if rank}
      <span class="w-6 shrink-0 text-center font-mono font-bold text-xs text-[var(--hp-subtle)] group-hover:text-[var(--hp-primary)] transition-colors">{rank}</span>
    {/if}
    <div class="w-16 h-12 shrink-0 rounded-lg overflow-hidden bg-[var(--hp-surface-sunken)]">
      <img src={getMediaUrl(item.cover_media_id, item.category, item.slug, index)} alt={item.title} class="w-full h-full object-cover" loading="lazy" />
    </div>
    <div class="min-w-0">
      <h4 class="text-xs sm:text-sm font-bold text-[var(--hp-ink)] line-clamp-2 leading-snug group-hover:text-[var(--hp-primary)] transition-colors">
        {item.title}
      </h4>
      {#if dateText}<span class="text-[10px] text-[var(--hp-muted)] hp-tabular">{dateText}</span>{/if}
    </div>
  </a>
{:else if variant === 'overlay'}
  <a href={href} class="group relative rounded-2xl overflow-hidden bg-[var(--hp-surface-sunken)] aspect-[16/10] sm:aspect-auto sm:h-full min-h-[9.5rem] block">
    <img src={getMediaUrl(item.cover_media_id, item.category, item.slug, index)} alt={item.title} class="absolute inset-0 w-full h-full object-cover transition-transform duration-500 group-hover:scale-[1.04]" loading="lazy" />
    <div class="absolute inset-0 bg-gradient-to-t from-black/85 via-black/10 to-transparent pointer-events-none"></div>
    <div class="absolute bottom-0 inset-x-0 p-4">
      <h3 class="text-sm sm:text-base font-bold text-white leading-snug line-clamp-2 group-hover:underline decoration-1 underline-offset-2">
        {item.title}
      </h3>
      {#if dateText}<span class="mt-1.5 block text-[11px] text-white/60 hp-tabular">{dateText}</span>{/if}
    </div>
  </a>
{:else}
  <a href={href} class="group rounded-xl overflow-hidden bg-[var(--hp-surface)] border border-[var(--hp-border)] flex flex-col">
    <div class="relative aspect-[4/3] overflow-hidden bg-[var(--hp-surface-sunken)]">
      <img src={getMediaUrl(item.cover_media_id, item.category, item.slug, index)} alt={item.title} class="absolute inset-0 w-full h-full object-cover transition-transform duration-500 group-hover:scale-[1.04]" loading="lazy" />
    </div>
    <div class="px-3.5 py-3 flex-1 flex flex-col">
      <h3 class="text-sm font-bold text-[var(--hp-ink)] leading-snug line-clamp-2 group-hover:text-[var(--hp-primary)] transition-colors">
        {item.title}
      </h3>
      {#if dateText}<span class="mt-2 text-[11px] text-[var(--hp-muted)] hp-tabular">{dateText}</span>{/if}
    </div>
  </a>
{/if}
