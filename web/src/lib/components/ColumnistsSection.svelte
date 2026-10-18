<script lang="ts">
  import { type Columnist, getMediaUrl } from '#lib';

  interface Props {
    columnists: Columnist[];
  }

  let { columnists = [] }: Props = $props();
</script>

{#if columnists.length > 0}
  <section id="yazarlar" class="my-8 py-6 px-4 rounded-3xl bg-white border border-zinc-200 shadow-xs">
    <div class="flex items-center justify-between pb-3 mb-4 border-b border-zinc-200">
      <h2 class="text-lg font-extrabold text-zinc-900 uppercase tracking-wider flex items-center gap-2">
        <span class="w-2.5 h-2.5 rounded-full bg-amber-600"></span>
        Yazarlarımız & Görüşler
      </h2>
      <span class="text-xs text-zinc-500">Güncel Köşe Yazıları</span>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      {#each columnists as c}
        <article class="group p-4 rounded-2xl bg-zinc-50 border border-zinc-200 cursor-pointer flex flex-col justify-between">
          <div class="flex items-center gap-3.5 mb-3">
            {#if c.avatar}
              <img
                src={getMediaUrl(c.avatar, 'kultur')}
                alt={c.name}
                class="w-12 h-12 rounded-full object-cover border border-zinc-300 shrink-0 shadow-2xs"
                loading="lazy"
              />
            {:else}
              <div class="w-12 h-12 rounded-full bg-zinc-200 text-zinc-700 flex items-center justify-center font-bold text-sm shrink-0 border border-zinc-300 shadow-2xs">
                {c.name.slice(0, 2).toUpperCase()}
              </div>
            {/if}
            <div class="min-w-0">
              <h3 class="text-sm font-bold text-zinc-900 truncate">{c.name}</h3>
              {#if c.title}
                <p class="text-[11px] text-zinc-500 truncate">{c.title}</p>
              {/if}
            </div>
          </div>

          {#if c.latest_article_title}
            <div class="pt-2.5 border-t border-zinc-200">
              <h4 class="text-xs font-semibold text-zinc-800 line-clamp-2 group-hover:underline leading-relaxed">
                "{c.latest_article_title}"
              </h4>
            </div>
          {/if}
        </article>
      {/each}
    </div>
  </section>
{/if}
