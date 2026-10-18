<script lang="ts">
  import { type NewsArticle, getMediaUrl, getCategoryBadge } from '#lib';
  import { onMount } from 'svelte';

  interface Props {
    items: NewsArticle[];
  }

  let { items = [] }: Props = $props();
  let activeIndex = $state(0);
  let isHovered = $state(false);

  onMount(() => {
    if (items.length <= 1) return;
    const timer = setInterval(() => {
      if (!isHovered) {
        activeIndex = (activeIndex + 1) % items.length;
      }
    }, 5500);

    return () => clearInterval(timer);
  });

  function selectSlide(index: number) {
    activeIndex = index;
  }

  function prevSlide() {
    activeIndex = (activeIndex - 1 + items.length) % items.length;
  }

  function nextSlide() {
    activeIndex = (activeIndex + 1) % items.length;
  }
</script>

{#if items.length > 0}
  {@const active = items[activeIndex]}
  {@const activeBadge = getCategoryBadge(active.category)}
  <div
    class="relative w-full rounded-2xl overflow-hidden bg-zinc-950 border border-zinc-800 shadow-2xl"
    onmouseenter={() => (isHovered = true)}
    onmouseleave={() => (isHovered = false)}
    role="region"
    aria-label="Ana Manşet Haberleri"
  >
    <!-- Visual Image Canvas -->
    <div class="relative aspect-[16/9] md:aspect-[21/9] w-full overflow-hidden bg-zinc-900">
      {#each items as item, i}
        <div
          class="absolute inset-0 transition-opacity duration-700 ease-in-out {i === activeIndex ? 'opacity-100 z-10' : 'opacity-0 z-0 pointer-events-none'}"
        >
          <img
            src={getMediaUrl(item.cover_media_id, item.category)}
            alt={item.title}
            class="w-full h-full object-cover transform duration-1000 scale-100 hover:scale-105"
          />

          <!-- Dark Gradient Overlays for Readability -->
          <div class="absolute inset-0 bg-gradient-to-t from-black via-black/50 to-transparent"></div>
          <div class="absolute inset-0 bg-gradient-to-r from-black/80 via-transparent to-transparent hidden md:block"></div>
        </div>
      {/each}

      <!-- Floating Article Info -->
      <div class="absolute bottom-14 md:bottom-16 left-0 right-0 z-20 px-4 md:px-8 max-w-4xl space-y-2">
        {#if active.category}
          <span class={`inline-block px-2.5 py-1 rounded ${activeBadge.bg} ${activeBadge.text} text-[11px] font-bold uppercase tracking-wider shadow`}>
            {active.category}
          </span>
        {/if}

        <h2 class="text-xl sm:text-2xl md:text-4xl font-extrabold text-white leading-tight drop-shadow-md">
          <a href={`/${active.category || 'haber'}/${active.slug}`} class="hover:text-red-400 transition-colors">
            {active.title}
          </a>
        </h2>

        {#if active.excerpt}
          <p class="text-xs sm:text-sm text-zinc-300 line-clamp-2 max-w-2xl hidden sm:block drop-shadow">
            {active.excerpt}
          </p>
        {/if}
      </div>

      <!-- Arrow Controls -->
      {#if items.length > 1}
        <button
          type="button"
          onclick={prevSlide}
          aria-label="Önceki Haber"
          class="absolute left-3 top-1/2 -translate-y-1/2 z-20 w-10 h-10 rounded-full bg-black/50 hover:bg-black/80 text-white flex items-center justify-center backdrop-blur-sm transition-all border border-white/10"
        >
          ‹
        </button>
        <button
          type="button"
          onclick={nextSlide}
          aria-label="Sonraki Haber"
          class="absolute right-3 top-1/2 -translate-y-1/2 z-20 w-10 h-10 rounded-full bg-black/50 hover:bg-black/80 text-white flex items-center justify-center backdrop-blur-sm transition-all border border-white/10"
        >
          ›
        </button>
      {/if}
    </div>

    <!-- Numbered Navigation Bar (Turkish Manşet Style) -->
    {#if items.length > 1}
      <div class="absolute bottom-0 left-0 right-0 z-20 bg-black/80 backdrop-blur-md border-t border-zinc-800/80 px-2 py-1.5 flex items-center justify-center gap-1 sm:gap-1.5 overflow-x-auto scrollbar-none">
        {#each items as _, i}
          <button
            type="button"
            onclick={() => selectSlide(i)}
            class="h-7 min-w-7 px-2 rounded flex items-center justify-center text-xs font-bold transition-all {i === activeIndex ? 'bg-red-600 text-white shadow' : 'bg-zinc-800/80 text-zinc-400 hover:bg-zinc-700 hover:text-white'}"
          >
            {i + 1}
          </button>
        {/each}
      </div>
    {/if}
  </div>
{/if}
