<script lang="ts">
  import { type Gallery, type VideoNews, getMediaUrl, safeUrl } from '#lib';

  interface Props {
    galleries?: Gallery[];
    videos?: VideoNews[];
  }

  let { galleries = [], videos = [] }: Props = $props();
</script>

{#if galleries.length > 0 || videos.length > 0}
  <section class="my-8 py-6 px-4 rounded-2xl bg-gradient-to-b from-zinc-950 to-zinc-900 border border-zinc-800">
    <div class="flex items-center justify-between pb-3 mb-4 border-b border-zinc-800">
      <h2 class="text-lg font-extrabold text-white uppercase tracking-wider flex items-center gap-2">
        <span class="w-2.5 h-2.5 rounded-full bg-red-500"></span>
        Fotoğraf & Video Galerileri
      </h2>
      <span class="text-xs text-zinc-500">Zengin Medya</span>
    </div>

    <!-- Media Cards Grid -->
    <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
      {#each videos as v}
        <a
          href={safeUrl(v.video_url)}
          target="_blank"
          rel="noopener noreferrer"
          class="group relative rounded-xl overflow-hidden bg-zinc-900 border border-zinc-800 aspect-video flex flex-col justify-end p-2.5 shadow"
        >
          {#if v.cover_image}
            <img
              src={getMediaUrl(v.cover_image)}
              alt={v.title}
              class="absolute inset-0 w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
              loading="lazy"
            />
          {/if}
          <div class="absolute inset-0 bg-black/40 group-hover:bg-black/20 transition-colors"></div>

          <!-- Play Icon Overlay -->
          <div class="absolute inset-0 flex items-center justify-center">
            <div class="w-10 h-10 rounded-full bg-red-600/90 text-white flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform">
              ▶
            </div>
          </div>

          <div class="relative z-10">
            <h3 class="text-xs font-bold text-white line-clamp-2 leading-tight drop-shadow">
              {v.title}
            </h3>
          </div>
        </a>
      {/each}

      {#each galleries as g}
        <a
          href={`/galeri/${g.slug}`}
          class="group relative rounded-xl overflow-hidden bg-zinc-900 border border-zinc-800 aspect-video flex flex-col justify-end p-2.5 shadow"
        >
          {#if g.cover_image}
            <img
              src={getMediaUrl(g.cover_image)}
              alt={g.title}
              class="absolute inset-0 w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
              loading="lazy"
            />
          {/if}
          <div class="absolute inset-0 bg-gradient-to-t from-black via-black/40 to-transparent"></div>

          <div class="absolute top-2 right-2 px-1.5 py-0.5 rounded bg-black/60 text-white text-[10px] font-semibold flex items-center gap-1 backdrop-blur-sm">
            📷 Galeri
          </div>

          <div class="relative z-10">
            <h3 class="text-xs font-bold text-white line-clamp-2 leading-tight drop-shadow">
              {g.title}
            </h3>
          </div>
        </a>
      {/each}
    </div>
  </section>
{/if}
