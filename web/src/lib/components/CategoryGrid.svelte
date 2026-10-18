<script lang="ts">
  import { type NewsArticle, getMediaUrl, getCategoryBadge, slugify } from '#lib';

  interface Props {
    title: string;
    categorySlug: string;
    items: NewsArticle[];
  }

  let { title, categorySlug, items = [] }: Props = $props();
  let primary = $derived(items[0]);
  let badge = $derived(getCategoryBadge(categorySlug));
</script>

{#if items.length > 0}
  <section class="my-8">
    <!-- Header -->
    <div class="flex items-center justify-between pb-2.5 mb-4 border-b-2 border-zinc-200">
      <h2 class="text-lg md:text-xl font-extrabold text-zinc-900 uppercase tracking-wider flex items-center gap-2.5">
        <span class={`w-3 h-3 rounded-md ${badge.bg}`}></span>
        {title}
      </h2>
      <a
        href={`/${slugify(categorySlug)}`}
        class="text-xs font-bold text-zinc-500 hover:text-red-600 flex items-center gap-1.5 transition-colors"
      >
        Tümünü Gör <span>→</span>
      </a>
    </div>

    <!-- Layout: 1 Primary Card on Left, 3 Compact Cards on Right -->
    <div class="grid grid-cols-1 md:grid-cols-12 gap-4">
      <!-- Primary Card -->
      <div class="md:col-span-7">
        <a
          href={`/${slugify(primary.category || categorySlug)}/${primary.slug}`}
          class="group block relative rounded-2xl overflow-hidden bg-zinc-100 border border-zinc-200 shadow-xs aspect-[16/10] cursor-pointer"
        >
          <img
            src={getMediaUrl(primary.cover_media_id, primary.category || categorySlug)}
            alt={primary.title}
            class="w-full h-full object-cover"
            loading="lazy"
          />
          <div class="absolute inset-0 bg-gradient-to-t from-black via-black/45 to-transparent pointer-events-none"></div>
          <div class="absolute bottom-0 inset-x-0 p-5 sm:p-6 space-y-2 pointer-events-auto">
            <span class={`inline-block px-2.5 py-0.5 rounded ${badge.bg} ${badge.text} text-[10px] font-bold uppercase tracking-wider shadow`}>
              {primary.category || categorySlug}
            </span>
            <h3 class="text-base sm:text-lg md:text-xl font-extrabold text-white leading-snug group-hover:underline drop-shadow">
              {primary.title}
            </h3>
            {#if primary.excerpt}
              <p class="text-xs text-zinc-200 line-clamp-2 hidden sm:block drop-shadow">
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
            href={`/${slugify(item.category || categorySlug)}/${item.slug}`}
            class="group flex items-center gap-3.5 p-3 rounded-2xl bg-white border border-zinc-200 shadow-xs cursor-pointer"
          >
            <div class="w-24 h-18 sm:w-28 sm:h-20 shrink-0 rounded-xl overflow-hidden bg-zinc-100 relative">
              <img
                src={getMediaUrl(item.cover_media_id, item.category || categorySlug)}
                alt={item.title}
                class="w-full h-full object-cover"
                loading="lazy"
              />
            </div>
            <div class="flex-1 min-w-0 space-y-1">
              <h4 class="text-xs sm:text-sm font-bold text-zinc-900 line-clamp-2 leading-snug group-hover:underline">
                {item.title}
              </h4>
              {#if item.created_at}
                <span class="text-[10px] text-zinc-500 block font-medium">
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
