<script lang="ts">
  import { type Columnist, getMediaUrl } from '#lib';

  interface Props {
    columnists: Columnist[];
  }

  let { columnists = [] }: Props = $props();
</script>

{#if columnists.length > 0}
  <section class="my-8 py-6 px-4 rounded-2xl bg-zinc-950 border border-zinc-800/80">
    <div class="flex items-center justify-between pb-3 mb-4 border-b border-zinc-800">
      <h2 class="text-lg font-extrabold text-white uppercase tracking-wider flex items-center gap-2">
        <span class="w-2.5 h-2.5 rounded-full bg-amber-500"></span>
        Yazarlarımız & Görüşler
      </h2>
      <span class="text-xs text-zinc-500">Güncel Köşe Yazıları</span>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      {#each columnists as c}
        <article class="p-3 rounded-xl bg-zinc-900/60 border border-zinc-800/60 hover:border-zinc-700 transition-colors flex flex-col justify-between">
          <div class="flex items-center gap-3 mb-2.5">
            {#if c.avatar}
              <img
                src={getMediaUrl(c.avatar)}
                alt={c.name}
                class="w-12 h-12 rounded-full object-cover border-2 border-zinc-700 shrink-0"
                loading="lazy"
              />
            {:else}
              <div class="w-12 h-12 rounded-full bg-zinc-800 text-zinc-400 flex items-center justify-center font-bold text-sm shrink-0 border border-zinc-700">
                {c.name.slice(0, 2).toUpperCase()}
              </div>
            {/if}
            <div class="min-w-0">
              <h3 class="text-sm font-bold text-white truncate">{c.name}</h3>
              {#if c.title}
                <p class="text-[11px] text-zinc-400 truncate">{c.title}</p>
              {/if}
            </div>
          </div>

          {#if c.latest_article_title}
            <div class="pt-2 border-t border-zinc-800/80">
              <h4 class="text-xs font-semibold text-zinc-200 line-clamp-2 hover:text-amber-400 transition-colors">
                "{c.latest_article_title}"
              </h4>
            </div>
          {/if}
        </article>
      {/each}
    </div>
  </section>
{/if}
