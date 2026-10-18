<script lang="ts">
  import { type NewsArticle, getMediaUrl, getCategoryBadge } from '#lib';

  interface Props {
    items: NewsArticle[];
  }

  let { items = [] }: Props = $props();
</script>

{#if items.length > 0}
  <div class="grid grid-cols-2 md:grid-cols-4 gap-3.5 my-4">
    {#each items.slice(0, 4) as item}
      {@const badge = getCategoryBadge(item.category)}
      <a
        href={`/${item.category || 'haber'}/${item.slug}`}
        class="group relative rounded-xl overflow-hidden bg-zinc-900 border border-zinc-800 shadow-lg aspect-[4/3] flex flex-col justify-end p-3.5 transition-all duration-300 hover:-translate-y-1 hover:border-zinc-700 hover:shadow-2xl"
      >
        <img
          src={getMediaUrl(item.cover_media_id, item.category)}
          alt={item.title}
          class="absolute inset-0 w-full h-full object-cover transition-transform duration-700 group-hover:scale-110"
          loading="lazy"
        />

        <div class="absolute inset-0 bg-gradient-to-t from-black via-black/60 to-transparent"></div>

        <div class="relative z-10 space-y-1.5">
          {#if item.category}
            <span class={`inline-block px-2 py-0.5 rounded text-[9px] font-bold uppercase tracking-wider ${badge.bg} ${badge.text} shadow`}>
              {item.category}
            </span>
          {/if}
          <h3 class="text-xs sm:text-sm font-bold text-white line-clamp-2 leading-snug group-hover:text-red-400 transition-colors drop-shadow">
            {item.title}
          </h3>
        </div>
      </a>
    {/each}
  </div>
{/if}
