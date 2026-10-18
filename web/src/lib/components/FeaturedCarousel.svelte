<script lang="ts">
  import { type NewsArticle, getMediaUrl, slugify, splitMansetTitle, THEME_CONFIG } from '#lib';
  import { onMount } from 'svelte';

  interface Props {
    items: NewsArticle[];
  }

  let { items = [] }: Props = $props();
  let activeIndex = $state(0);
  let isHovered = $state(false);

  // Take up to 12 headline items for the classic Turkish portal manşet strip
  let displayItems = $derived(items.slice(0, 12));
  let count = $derived(displayItems.length);

  onMount(() => {
    if (count <= 1) return;
    const timer = setInterval(() => {
      if (!isHovered) {
        activeIndex = (activeIndex + 1) % count;
      }
    }, 4500);

    return () => clearInterval(timer);
  });

  function selectSlide(index: number) {
    if (index >= 0 && index < count) {
      activeIndex = index;
    }
  }

  function prevSlide() {
    if (count <= 1) return;
    activeIndex = (activeIndex - 1 + count) % count;
  }

  function nextSlide() {
    if (count <= 1) return;
    activeIndex = (activeIndex + 1) % count;
  }
</script>

{#if count > 0}
  {@const active = displayItems[activeIndex]}
  {@const headline = splitMansetTitle(active.title, active.category)}
  {@const articleUrl = `/${slugify(active.category)}/${active.slug}`}

  <div
    class="relative w-full rounded-2xl overflow-hidden bg-black border border-zinc-200 shadow-sm flex flex-col select-none"
    onmouseenter={() => (isHovered = true)}
    onmouseleave={() => (isHovered = false)}
    role="region"
    aria-label="Ana Manşet Haberleri"
  >
    <!-- Visual Image Canvas (16/9 cinematic proportion) -->
    <div class="relative aspect-[16/9] w-full overflow-hidden bg-black">
      {#each displayItems as item, i}
        <div
          class="absolute inset-0 transition-opacity duration-500 ease-in-out {i === activeIndex ? 'opacity-100 z-10' : 'opacity-0 z-0 pointer-events-none'}"
        >
          <img
            src={getMediaUrl(item.cover_media_id, item.category)}
            alt={item.title}
            class="w-full h-full object-cover"
          />

          <!-- Dark Gradient Overlay for optimal headline legibility -->
          <div class="absolute inset-0 bg-gradient-to-t from-black via-black/55 to-transparent pointer-events-none"></div>
        </div>
      {/each}

      <!-- Left / Right Chevron Navigation Arrows -->
      {#if count > 1}
        <button
          type="button"
          onclick={prevSlide}
          aria-label="Önceki Manşet"
          class="absolute left-2 sm:left-4 top-1/2 -translate-y-1/2 z-20 w-10 sm:w-12 h-14 sm:h-20 flex items-center justify-center text-white/60 hover:text-white bg-black/20 hover:bg-black/50 rounded-lg backdrop-blur-xs transition-colors cursor-pointer"
        >
          <svg class="w-7 h-7 sm:w-9 sm:h-9" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M15 19l-7-7 7-7" />
          </svg>
        </button>

        <button
          type="button"
          onclick={nextSlide}
          aria-label="Sonraki Manşet"
          class="absolute right-2 sm:right-4 top-1/2 -translate-y-1/2 z-20 w-10 sm:w-12 h-14 sm:h-20 flex items-center justify-center text-white/60 hover:text-white bg-black/20 hover:bg-black/50 rounded-lg backdrop-blur-xs transition-colors cursor-pointer"
        >
          <svg class="w-7 h-7 sm:w-9 sm:h-9" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M9 5l7 7-7 7" />
          </svg>
        </button>
      {/if}

      <!-- Center / Bottom Headline Overlay (Exactly like user reference image) -->
      <a
        href={articleUrl}
        class="absolute bottom-3 sm:bottom-6 left-0 right-0 z-20 px-4 sm:px-12 text-center block group hover:opacity-95 transition-opacity"
      >
        <!-- Line 1: White Bold Lead -->
        <div class="text-base sm:text-2xl md:text-3xl lg:text-4xl font-extrabold text-white tracking-tight leading-snug drop-shadow-md">
          {headline.lead}
        </div>

        <!-- Line 2: Heavy Vibrant Yellow Punchline -->
        <div class="text-lg sm:text-3xl md:text-4xl lg:text-5xl font-black text-[#ffe600] uppercase tracking-wide leading-tight mt-1 sm:mt-1.5 drop-shadow-[0_2px_10px_rgba(0,0,0,0.85)]">
          {headline.punchline}
        </div>
      </a>
    </div>

    <!-- Solid Numbered Navigation Bar Attached to Bottom (Exactly matching reference) -->
    <div class="flex items-stretch w-full border-t border-zinc-200 bg-white overflow-x-auto scrollbar-none">
      {#each displayItems as _, i}
        <button
          type="button"
          onmouseenter={() => selectSlide(i)}
          onclick={() => selectSlide(i)}
          aria-label={`Manşet ${i + 1}`}
          class="flex-1 min-w-[2.25rem] sm:min-w-[2.75rem] h-9 sm:h-11 flex items-center justify-center text-xs sm:text-sm md:text-base font-bold transition-colors cursor-pointer {
            i === activeIndex
              ? 'bg-red-600 text-white font-black'
              : 'bg-white hover:bg-zinc-100 text-zinc-900 border-r border-zinc-200'
          }"
        >
          {i + 1}
        </button>
      {/each}

      <!-- "T" Button for Tüm Manşetler -->
      <a
        href="/gundem"
        title="Tüm Manşetler"
        aria-label="Tüm Manşetler"
        class="flex-1 min-w-[2.25rem] sm:min-w-[2.75rem] h-9 sm:h-11 flex items-center justify-center text-xs sm:text-sm md:text-base font-black bg-white hover:bg-zinc-100 text-zinc-900 border-l border-zinc-200 transition-colors cursor-pointer"
      >
        T
      </a>
    </div>
  </div>
{/if}
