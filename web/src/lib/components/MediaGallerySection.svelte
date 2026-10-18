<script lang="ts">
  import { type Gallery, type VideoNews, getMediaUrl, safeUrl } from '#lib';

  interface Props {
    galleries?: Gallery[];
    videos?: VideoNews[];
  }

  let { galleries = [], videos = [] }: Props = $props();
</script>

{#if galleries.length > 0 || videos.length > 0}
  <section class="my-8 py-6 px-4 rounded-3xl bg-white border border-zinc-200 shadow-xs">
    <div class="flex items-center justify-between pb-3 mb-4 border-b border-zinc-200">
      <h2 class="text-lg font-extrabold text-zinc-900 uppercase tracking-wider flex items-center gap-2">
        <span class="w-2.5 h-2.5 rounded-full bg-red-600"></span>
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
          class="group relative rounded-2xl overflow-hidden bg-zinc-100 border border-zinc-200 aspect-video flex flex-col justify-end p-3 shadow-2xs cursor-pointer"
        >
          <img
            src={getMediaUrl(v.cover_image, 'teknoloji')}
            alt={v.title}
            class="absolute inset-0 w-full h-full object-cover"
            loading="lazy"
          />
          <div class="absolute inset-0 bg-black/40 pointer-events-none"></div>

          <!-- Play Icon Overlay -->
          <div class="absolute inset-0 flex items-center justify-center pointer-events-none">
            <div class="w-10 h-10 rounded-full bg-red-600/90 text-white flex items-center justify-center shadow-md">
              ▶
            </div>
          </div>

          <div class="relative z-10 pointer-events-auto">
            <h3 class="text-xs font-bold text-white line-clamp-2 leading-tight drop-shadow group-hover:underline">
              {v.title}
            </h3>
          </div>
        </a>
      {/each}

      {#each galleries as g}
        <a
          href={`/galeri/${g.slug}`}
          class="group relative rounded-2xl overflow-hidden bg-zinc-100 border border-zinc-200 aspect-video flex flex-col justify-end p-3 shadow-2xs cursor-pointer"
        >
          <img
            src={getMediaUrl(g.cover_image, 'dunya')}
            alt={g.title}
            class="absolute inset-0 w-full h-full object-cover"
            loading="lazy"
          />
          <div class="absolute inset-0 bg-gradient-to-t from-black via-black/40 to-transparent pointer-events-none"></div>

          <div class="absolute top-2.5 right-2.5 px-2 py-0.5 rounded-full bg-black/70 text-white text-[10px] font-bold flex items-center gap-1 backdrop-blur-md border border-white/10 shadow">
            📷 Galeri
          </div>

          <div class="relative z-10 pointer-events-auto">
            <h3 class="text-xs font-bold text-white line-clamp-2 leading-tight drop-shadow group-hover:underline">
              {g.title}
            </h3>
          </div>
        </a>
      {/each}
    </div>
  </section>
{/if}
