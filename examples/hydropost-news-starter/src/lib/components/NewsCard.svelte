<script lang="ts">
  // Single source of truth for how a news article renders as a card
  // anywhere in the app. Change the look here, not per-section.
  import { type NewsArticle, getMediaUrl, getCategoryBadge, slugify } from '#lib';

  interface Props {
    item: NewsArticle;
    /**
     * tile    — 16:9 image, white footer strip with the title (grids)
     * overlay — full-bleed image, title pinned over a dark gradient (hero side cards)
     * row     — small thumbnail beside the title (sidebars, "diğer başlıklar")
     */
    variant?: 'tile' | 'overlay' | 'row';
    rank?: number;
    /** Position within the list being rendered — pass the `{#each}` index
     * so sibling cards never repeat the same stand-in photo (see
     * getMediaUrl's `index` param). */
    index?: number;
    /** tile only — show a small colored category label above the title. */
    showCategory?: boolean;
  }

  let { item, variant = 'tile', rank, index, showCategory = false }: Props = $props();
  let badge = $derived(getCategoryBadge(item.category));

  let href = $derived(`/${slugify(item.category)}/${item.slug}`);
</script>

{#if variant === 'row'}
  <a href={href} class="group flex items-center gap-3">
    {#if rank}
      <span class="w-6 shrink-0 text-center font-mono font-bold text-xs text-[var(--hp-subtle)] group-hover:text-[var(--hp-primary)] transition-colors">{rank}</span>
    {/if}
    <div class="w-16 aspect-video shrink-0 rounded-lg overflow-hidden bg-[var(--hp-surface-sunken)]">
      <img src={getMediaUrl(item.cover_media_id, item.category, item.slug, index)} alt={item.title} class="w-full h-full object-cover" loading="lazy" />
    </div>
    <h4 class="min-w-0 text-xs sm:text-sm font-bold text-[var(--hp-ink)] line-clamp-2 leading-snug group-hover:text-[var(--hp-primary)] transition-colors">
      {item.title}
    </h4>
  </a>
{:else if variant === 'overlay'}
  <a href={href} class="group relative rounded-2xl overflow-hidden bg-[var(--hp-surface-sunken)] aspect-video sm:aspect-auto sm:h-full min-h-[9.5rem] block">
    <img src={getMediaUrl(item.cover_media_id, item.category, item.slug, index)} alt={item.title} class="absolute inset-0 w-full h-full object-cover transition-transform duration-500 group-hover:scale-[1.04]" loading="lazy" />
    <div class="absolute inset-0 bg-gradient-to-t from-black/85 via-black/10 to-transparent pointer-events-none"></div>
    <h3 class="absolute bottom-0 inset-x-0 p-4 text-sm sm:text-base font-bold text-white leading-snug line-clamp-2 group-hover:underline decoration-1 underline-offset-2">
      {item.title}
    </h3>
  </a>
{:else}
  <a href={href} class="group rounded-xl overflow-hidden bg-[var(--hp-surface)] border border-[var(--hp-border)] flex flex-col">
    <div class="relative aspect-video overflow-hidden bg-[var(--hp-surface-sunken)]">
      <img src={getMediaUrl(item.cover_media_id, item.category, item.slug, index)} alt={item.title} class="absolute inset-0 w-full h-full object-cover transition-transform duration-500 group-hover:scale-[1.04]" loading="lazy" />
    </div>
    <div class="px-3.5 py-3">
      {#if showCategory && item.category}
        <span class={`inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wide mb-1 ${badge.text}`}>
          <span class={`w-1.5 h-1.5 rounded-full ${badge.bg.replace('-100', '-600')}`}></span>
          {item.category}
        </span>
      {/if}
      <h3 class="text-sm font-bold text-[var(--hp-ink)] leading-snug line-clamp-2 group-hover:text-[var(--hp-primary)] transition-colors">
        {item.title}
      </h3>
    </div>
  </a>
{/if}
