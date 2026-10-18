<script lang="ts">
  import { type NewsArticle, getMediaUrl } from '#lib';

  interface Props {
    items: NewsArticle[];
  }

  let { items = [] }: Props = $props();
</script>

{#if items.length > 0}
  <div class="grid grid-cols-2 md:grid-cols-4 gap-3 my-4">
    {#each items.slice(0, 4) as item}
      <a
        href={`/${item.category || 'haber'}/${item.slug}`}
        class="group relative rounded-xl overflow-hidden bg-zinc-900 border border-zinc-800 shadow aspect-[4/3] flex flex-col justify-end p-3 transition-transform duration-300 hover:-translate-y-1"
      >
        {#if item.cover_media_id}
          <img
            src={getMediaUrl(item.cover_media_id)}
            alt={item.title}
            class="absolute inset-0 w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
            loading="lazy"
          />
        {/if}

        <div class="absolute inset-0 bg-gradient-to-t from-black via-black/60 to-transparent"></div>

        <div class="relative z-10 space-y-1">
          {#if item.category}
            <span class="inline-block px-1.5 py-0.5 rounded text-[9px] font-bold uppercase tracking-wider bg-red-600/90 text-white">
              {item.category}
            </span>
          {/if}
          <h3 class="text-xs sm:text-sm font-bold text-white line-clamp-2 leading-snug group-hover:text-red-400 transition-colors">
            {item.title}
          </h3>
        </div>
      </a>
    {/each}
  </div>
{/if}
