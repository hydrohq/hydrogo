<script lang="ts">
  import { type NewsArticle, getMediaUrl } from '#lib';

  interface Props {
    title: string;
    categorySlug: string;
    items: NewsArticle[];
  }

  let { title, categorySlug, items = [] }: Props = $props();
  let primary = $derived(items[0]);
</script>

{#if items.length > 0}
  <section class="my-8">
    <!-- Header -->
    <div class="flex items-center justify-between pb-2 mb-4 border-b-2 border-red-600">
      <h2 class="text-lg md:text-xl font-extrabold text-white uppercase tracking-wider flex items-center gap-2">
        <span class="w-2.5 h-2.5 rounded bg-red-600"></span>
        {title}
      </h2>
      <a
        href={`/${categorySlug}`}
        class="text-xs font-semibold text-zinc-400 hover:text-red-400 flex items-center gap-1 transition-colors"
      >
        Tümünü Gör <span>→</span>
      </a>
    </div>

    <!-- Layout: 1 Primary Card on Left, 3 Compact Cards on Right -->
    <div class="grid grid-cols-1 md:grid-cols-12 gap-4">
      <!-- Primary Card -->
      <div class="md:col-span-7">
        <a
          href={`/${primary.category || categorySlug}/${primary.slug}`}
          class="group block relative rounded-xl overflow-hidden bg-zinc-900 border border-zinc-800 shadow aspect-[16/10]"
        >
          {#if primary.cover_media_id}
            <img
              src={getMediaUrl(primary.cover_media_id)}
              alt={primary.title}
              class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
              loading="lazy"
            />
          {/if}
          <div class="absolute inset-0 bg-gradient-to-t from-black via-black/40 to-transparent"></div>
          <div class="absolute bottom-0 inset-x-0 p-4 space-y-1.5">
            <h3 class="text-base sm:text-lg md:text-xl font-extrabold text-white leading-snug group-hover:text-red-400 transition-colors">
              {primary.title}
            </h3>
            {#if primary.excerpt}
              <p class="text-xs text-zinc-300 line-clamp-2 hidden sm:block">
                {primary.excerpt}
              </p>
            {/if}
          </div>
        </a>
      </div>

      <!-- Secondary Compact List -->
      <div class="md:col-span-5 flex flex-col justify-between gap-3">
        {#each items.slice(1, 4) as item}
          <a
            href={`/${item.category || categorySlug}/${item.slug}`}
            class="group flex items-center gap-3 p-2 rounded-xl bg-zinc-900/60 border border-zinc-800/80 hover:border-zinc-700 transition-colors"
          >
            {#if item.cover_media_id}
              <div class="w-24 h-18 shrink-0 rounded-lg overflow-hidden bg-zinc-800">
                <img
                  src={getMediaUrl(item.cover_media_id)}
                  alt={item.title}
                  class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105"
                  loading="lazy"
                />
              </div>
            {/if}
            <div class="flex-1 min-w-0">
              <h4 class="text-xs sm:text-sm font-bold text-zinc-200 line-clamp-2 leading-snug group-hover:text-red-400 transition-colors">
                {item.title}
              </h4>
              {#if item.created_at}
                <span class="text-[10px] text-zinc-500 mt-1 block">
                  {new Date(item.created_at).toLocaleDateString('tr-TR')}
                </span>
              {/if}
            </div>
          </a>
        {/each}
      </div>
    </div>
  </section>
{/if}
