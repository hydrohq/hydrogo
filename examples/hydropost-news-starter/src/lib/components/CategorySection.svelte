<script lang="ts">
  import { type NewsArticle, getCategoryBadge, slugify } from '#lib';
  import NewsCard from './NewsCard.svelte';

  interface Props {
    category: string;
    items: NewsArticle[];
  }

  let { category, items = [] }: Props = $props();
  let badge = $derived(getCategoryBadge(category));
  let big = $derived(items[0]);
  let rest = $derived(items.slice(1, 5));
</script>

{#if big}
  <section class="my-10">
    <div class="flex items-center gap-3 mb-5">
      <span class={`w-2.5 h-2.5 rounded-full ${badge.bg.replace('-100', '-600')}`}></span>
      <h2 class="text-lg md:text-xl font-extrabold text-[var(--hp-ink)] uppercase tracking-wide">{category} Haberleri</h2>
      <div class="flex-1 h-px bg-[var(--hp-border)]"></div>
      <a
        href={`/${slugify(category)}`}
        class={`shrink-0 px-3 py-1.5 rounded-full text-xs font-bold transition-colors ${badge.bg} ${badge.text} hover:opacity-85`}
      >
        Tüm {category} Haberleri
      </a>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
      <NewsCard item={big} variant="overlay" index={0} />
      <div class="grid grid-cols-2 gap-4">
        {#each rest as item, i (item.id)}
          <NewsCard {item} variant="tile" index={i + 1} />
        {/each}
      </div>
    </div>
  </section>
{/if}
