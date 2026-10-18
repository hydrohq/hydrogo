<script lang="ts">
  import { type NewsArticle } from '#lib';
  import { onMount } from 'svelte';

  interface Props {
    items: NewsArticle[];
  }

  let { items = [] }: Props = $props();
  let currentIndex = $state(0);

  onMount(() => {
    if (items.length <= 1) return;
    const interval = setInterval(() => {
      currentIndex = (currentIndex + 1) % items.length;
    }, 4500);

    return () => clearInterval(interval);
  });
</script>

{#if items.length > 0}
  <div class="w-full bg-zinc-100 border-y border-zinc-200 py-2 px-4 select-none">
    <div class="max-w-7xl mx-auto flex items-center gap-3">
      <div class="flex items-center gap-1.5 shrink-0 px-2.5 py-1 rounded bg-red-600 text-white font-extrabold text-[11px] uppercase tracking-wider shadow-xs">
        <span class="w-1.5 h-1.5 rounded-full bg-white"></span>
        Son Dakika
      </div>

      <div class="relative overflow-hidden flex-1 h-6 flex items-center">
        {#each items as item, i}
          <div
            class="absolute inset-0 flex items-center transition-all duration-500 ease-in-out {i === currentIndex ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-4 pointer-events-none'}"
          >
            <a
              href={`/${item.category || 'haber'}/${item.slug}`}
              class="text-xs sm:text-sm font-semibold text-zinc-800 hover:underline truncate"
            >
              {item.title}
            </a>
          </div>
        {/each}
      </div>

      {#if items.length > 1}
        <div class="hidden sm:flex items-center gap-1 text-[11px] text-zinc-500 font-mono">
          <span>{currentIndex + 1}</span>
          <span>/</span>
          <span>{items.length}</span>
        </div>
      {/if}
    </div>
  </div>
{/if}
