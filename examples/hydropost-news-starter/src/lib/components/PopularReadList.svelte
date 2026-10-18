<script lang="ts">
  import { type NewsArticle, getMediaUrl, slugify } from '#lib';

  interface Props {
    items: NewsArticle[];
  }

  let { items = [] }: Props = $props();
  let scroller = $state<HTMLDivElement | null>(null);

  function scrollBy(dir: 1 | -1) {
    scroller?.scrollBy({ left: dir * 280, behavior: 'smooth' });
  }
</script>

{#if items.length > 0}
  <section class="my-10">
    <div class="flex items-center gap-3 mb-5">
      <h2 class="text-lg md:text-xl font-extrabold text-[var(--hp-primary)]">Çok Okunanlar</h2>
      <div class="flex-1 h-px bg-[var(--hp-border)]"></div>
      <div class="flex items-center gap-1.5 shrink-0">
        <button type="button" onclick={() => scrollBy(-1)} aria-label="Geri" class="w-7 h-7 rounded-full border border-[var(--hp-border-strong)] text-[var(--hp-ink-soft)] hover:bg-[var(--hp-surface-sunken)] flex items-center justify-center transition-colors cursor-pointer">
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" /></svg>
        </button>
        <button type="button" onclick={() => scrollBy(1)} aria-label="İleri" class="w-7 h-7 rounded-full border border-[var(--hp-border-strong)] text-[var(--hp-ink-soft)] hover:bg-[var(--hp-surface-sunken)] flex items-center justify-center transition-colors cursor-pointer">
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" /></svg>
        </button>
      </div>
    </div>

    <div bind:this={scroller} class="flex gap-4 overflow-x-auto scrollbar-none scroll-smooth pb-1">
      {#each items as item, i (item.id)}
        <a
          href={`/${slugify(item.category)}/${item.slug}`}
          class="group shrink-0 w-44 sm:w-48 rounded-xl overflow-hidden bg-[var(--hp-surface)] border border-[var(--hp-border)]"
        >
          <div class="relative aspect-video overflow-hidden bg-[var(--hp-surface-sunken)]">
            <img src={getMediaUrl(item.cover_media_id, item.category, item.slug, i)} alt={item.title} class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-[1.04]" loading="lazy" />
            <span class="absolute top-2 left-2 w-6 h-6 rounded-full bg-[var(--hp-primary)] text-white text-xs font-black flex items-center justify-center hp-tabular">
              {i + 1}
            </span>
          </div>
          <h3 class="px-3 py-2.5 text-xs font-bold text-[var(--hp-ink)] leading-snug line-clamp-2 group-hover:text-[var(--hp-primary)] transition-colors">
            {item.title}
          </h3>
        </a>
      {/each}
    </div>
  </section>
{/if}
