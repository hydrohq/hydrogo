<script lang="ts">
  import { type NewsArticle, getMediaUrl, getCategoryBadge, slugify } from '#lib';

  interface Props {
    items: NewsArticle[];
  }

  let { items = [] }: Props = $props();
</script>

{#if items.length > 0}
  <div class="grid grid-cols-2 md:grid-cols-4 gap-3.5 my-5">
    {#each items.slice(0, 4) as item}
      {@const badge = getCategoryBadge(item.category)}
      <a
        href={`/${slugify(item.category)}/${item.slug}`}
        class="group relative rounded-2xl overflow-hidden bg-zinc-100 border border-zinc-200 shadow-xs aspect-[4/3] flex flex-col justify-end p-4 cursor-pointer"
      >
        <img
          src={getMediaUrl(item.cover_media_id, item.category)}
          alt={item.title}
          class="absolute inset-0 w-full h-full object-cover"
          loading="lazy"
        />

        <div class="absolute inset-0 bg-gradient-to-t from-black via-black/60 to-transparent pointer-events-none"></div>

        <div class="relative z-10 space-y-1.5 pointer-events-auto">
          {#if item.category}
            <span class={`inline-block px-2 py-0.5 rounded text-[9px] font-bold uppercase tracking-wider ${badge.bg} ${badge.text} shadow-xs`}>
              {item.category}
            </span>
          {/if}
          <h3 class="text-xs sm:text-sm font-bold text-white line-clamp-2 leading-snug group-hover:underline drop-shadow">
            {item.title}
          </h3>
        </div>
      </a>
    {/each}
  </div>
{/if}
