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

  // Left: a small rotating carousel. Right: two fixed secondary stories.
  let heroItems = $derived(items.slice(0, 5));
  let sideItems = $derived(items.slice(5, 7));
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
  function prevSlide() {
    if (count <= 1) return;
    activeIndex = (activeIndex - 1 + count) % count;
  }
  function nextSlide() {
    if (count <= 1) return;
    activeIndex = (activeIndex + 1) % count;
  }

  function dateOf(item: NewsArticle): string {
    return item.created_at ? new Date(item.created_at).toLocaleDateString('tr-TR') : '';
  }
</script>

{#if count > 0}
  {@const active = heroItems[activeIndex]}
  {@const articleUrl = `/${slugify(active.category)}/${active.slug}`}

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
    <!-- Big rotating hero -->
    <div
      class="lg:col-span-2 rounded-2xl overflow-hidden bg-[var(--hp-ink)] flex flex-col select-none"
      onmouseenter={() => (isHovered = true)}
      onmouseleave={() => (isHovered = false)}
      role="region"
      aria-label="Öne çıkan haberler"
    >
      <!-- Image + headline: pagination lives in its own bar below, so the
           two never fight over the same bottom-left corner. -->
      <div class="relative aspect-[16/10] sm:aspect-[16/9] w-full">
        {#each heroItems as item, i}
          <div class="absolute inset-0 transition-opacity duration-500 ease-in-out {i === activeIndex ? 'opacity-100 z-10' : 'opacity-0 z-0 pointer-events-none'}">
            <img src={getMediaUrl(item.cover_media_id, item.category, item.slug, i)} alt={item.title} class="w-full h-full object-cover" />
            <div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/20 to-transparent pointer-events-none"></div>
          </div>
        {/each}

        <a href={articleUrl} class="absolute bottom-0 inset-x-0 z-20 p-5 sm:p-7 block group">
          <h2 class="text-xl sm:text-2xl lg:text-3xl font-extrabold text-white leading-tight tracking-tight group-hover:underline decoration-2 underline-offset-4">
            {active.title}
          </h2>
          {#if active.excerpt}
            <p class="mt-2 text-sm text-white/75 line-clamp-2 max-w-xl hidden sm:block">{active.excerpt}</p>
          {/if}
          <span class="mt-3 block text-xs text-white/60 hp-tabular">{dateOf(active)}</span>
        </a>

        {#if count > 1}
          <button type="button" onclick={prevSlide} aria-label="Önceki" class="absolute left-3 top-1/2 -translate-y-1/2 z-30 w-9 h-9 rounded-full bg-black/35 hover:bg-black/55 backdrop-blur-sm text-white flex items-center justify-center transition-colors cursor-pointer">
            <svg class="w-4.5 h-4.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" /></svg>
          </button>
          <button type="button" onclick={nextSlide} aria-label="Sonraki" class="absolute right-3 top-1/2 -translate-y-1/2 z-30 w-9 h-9 rounded-full bg-black/35 hover:bg-black/55 backdrop-blur-sm text-white flex items-center justify-center transition-colors cursor-pointer">
            <svg class="w-4.5 h-4.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" /></svg>
          </button>
        {/if}
      </div>

      {#if count > 1}
        <div class="flex items-stretch w-full bg-[var(--hp-surface)]">
          {#each heroItems as _, i}
            <button
              type="button"
              onclick={() => selectSlide(i)}
              aria-label={`Slayt ${i + 1}`}
              class="flex-1 h-10 sm:h-11 flex items-center justify-center text-sm sm:text-base font-bold hp-tabular transition-colors cursor-pointer border-r border-[var(--hp-border)] last:border-r-0 {i === activeIndex ? 'bg-[var(--hp-primary)] text-white' : 'bg-[var(--hp-surface)] text-[var(--hp-ink)] hover:bg-[var(--hp-surface-sunken)]'}"
            >{i + 1}</button>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Two fixed secondary stories -->
    <div class="grid grid-cols-1 gap-4">
      {#each sideItems as item, i (item.id)}
        <NewsCard {item} variant="overlay" index={i + heroItems.length} />
      {/each}
    </div>
  </div>
{/if}
