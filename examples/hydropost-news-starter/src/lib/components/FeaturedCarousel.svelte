<script lang="ts">
  import { type NewsArticle, getMediaUrl, slugify } from '#lib';
  import { onMount } from 'svelte';
  import NewsCard from './NewsCard.svelte';

  interface Props {
    items: NewsArticle[];
  }

  let { items = [] }: Props = $props();
  let activeIndex = $state(0);
  let isHovered = $state(false);

  // Left: a rotating carousel (classic Turkish portal "1..12" manşet strip).
  // Right: two fixed secondary stories.
  let heroItems = $derived(items.slice(0, 12));
  let sideItems = $derived(items.slice(12, 14));
  let count = $derived(heroItems.length);

  onMount(() => {
    if (count <= 1) return;
    const timer = setInterval(() => {
      if (!isHovered) {
        activeIndex = (activeIndex + 1) % count;
      }
    }, 5000);
    return () => clearInterval(timer);
  });

  function selectSlide(index: number) {
    if (index >= 0 && index < count) activeIndex = index;
  }
</script>

{#if count > 0}
  {@const active = heroItems[activeIndex]}
  {@const articleUrl = `/${slugify(active.category)}/${active.slug}`}

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
    <!-- Big rotating hero -->
    <div
      class="lg:col-span-2 rounded-2xl overflow-hidden bg-[var(--hp-surface)] border border-[var(--hp-border)] flex flex-col select-none"
      onmouseenter={() => (isHovered = true)}
      onmouseleave={() => (isHovered = false)}
      role="region"
      aria-label="Öne çıkan haberler"
    >
      <!-- Image only — the title lives in the white area below, like
           every other card on the site. -->
      <div class="relative aspect-[16/10] sm:aspect-[16/9] w-full bg-[var(--hp-ink)]">
        {#each heroItems as item, i}
          <div class="absolute inset-0 transition-opacity duration-500 ease-in-out {i === activeIndex ? 'opacity-100 z-10' : 'opacity-0 z-0 pointer-events-none'}">
            <img src={getMediaUrl(item.cover_media_id, item.category, item.slug, i)} alt={item.title} class="w-full h-full object-cover" />
          </div>
        {/each}
      </div>

      <a href={articleUrl} class="block p-4 sm:p-5 group">
        <h2 class="text-lg sm:text-xl lg:text-2xl font-extrabold text-[var(--hp-ink)] leading-snug group-hover:text-[var(--hp-primary)] transition-colors">
          {active.title}
        </h2>
        {#if active.excerpt}
          <p class="mt-1.5 text-sm text-[var(--hp-muted)] line-clamp-2 hidden sm:block">{active.excerpt}</p>
        {/if}
      </a>

      {#if count > 1}
        <div class="flex items-stretch w-full border-t border-[var(--hp-border)]">
          {#each heroItems as _, i}
            <button
              type="button"
              onclick={() => selectSlide(i)}
              onmouseenter={() => selectSlide(i)}
              aria-label={`Slayt ${i + 1}`}
              class="flex-1 h-10 sm:h-11 flex items-center justify-center text-xs sm:text-sm font-bold hp-tabular transition-colors duration-150 cursor-pointer border-r border-[var(--hp-border)] last:border-r-0 {i === activeIndex ? 'bg-[var(--hp-primary)] text-white' : 'bg-[var(--hp-surface)] text-[var(--hp-ink)] hover:bg-[var(--hp-surface-sunken)]'}"
            >{i + 1}</button>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Two fixed secondary stories -->
    <div class="grid grid-cols-1 gap-4">
      {#each sideItems as item, i (item.id)}
        <NewsCard {item} variant="tile" index={i + heroItems.length} />
      {/each}
    </div>
  </div>
{/if}
