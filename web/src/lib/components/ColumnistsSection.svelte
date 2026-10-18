<script lang="ts">
  import { type Columnist, getMediaUrl } from '#lib';

  interface Props {
    columnists: Columnist[];
  }

  let { columnists = [] }: Props = $props();
</script>

{#if columnists.length > 0}
  <section id="yazarlar" class="my-8 py-6 px-4 rounded-3xl bg-zinc-950 border border-zinc-800/80 shadow-lg">
    <div class="flex items-center justify-between pb-3 mb-4 border-b border-zinc-800">
      <h2 class="text-lg font-extrabold text-white uppercase tracking-wider flex items-center gap-2">
        <span class="w-2.5 h-2.5 rounded-full bg-amber-500"></span>
        Yazarlarımız & Görüşler
      </h2>
      <span class="text-xs text-zinc-500">Güncel Köşe Yazıları</span>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      {#each columnists as c}
        <article class="group p-4 rounded-2xl bg-zinc-900/60 border border-zinc-800/80 hover:border-amber-500/70 hover:bg-zinc-900 hover:-translate-y-1.5 hover:shadow-xl hover:shadow-amber-950/30 transition-all duration-300 cursor-pointer flex flex-col justify-between">
          <div class="flex items-center gap-3.5 mb-3">
            {#if c.avatar}
              <img
                src={getMediaUrl(c.avatar, 'kultur')}
                alt={c.name}
                class="w-12 h-12 rounded-full object-cover border-2 border-zinc-700 group-hover:border-amber-400 group-hover:scale-105 transition-all duration-300 shrink-0 shadow"
                loading="lazy"
              />
            {:else}
              <div class="w-12 h-12 rounded-full bg-gradient-to-tr from-zinc-800 to-zinc-700 text-amber-400 flex items-center justify-center font-bold text-sm shrink-0 border border-zinc-700 group-hover:border-amber-400 group-hover:scale-105 transition-all duration-300 shadow">
                {c.name.slice(0, 2).toUpperCase()}
              </div>
            {/if}
            <div class="min-w-0">
              <h3 class="text-sm font-bold text-white truncate group-hover:text-amber-300 transition-colors">{c.name}</h3>
              {#if c.title}
                <p class="text-[11px] text-zinc-400 truncate">{c.title}</p>
              {/if}
            </div>
          </div>

          {#if c.latest_article_title}
            <div class="pt-2.5 border-t border-zinc-800/80">
              <h4 class="text-xs font-semibold text-zinc-300 line-clamp-2 group-hover:text-amber-400 transition-colors leading-relaxed">
                "{c.latest_article_title}"
              </h4>
            </div>
          {/if}
        </article>
      {/each}
    </div>
  </section>
{/if}
