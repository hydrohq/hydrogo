<script lang="ts">
  import { type Gallery, type VideoNews, getMediaUrl, safeUrl } from '#lib';

  interface Props {
    galleries?: Gallery[];
    videos?: VideoNews[];
  }

  let { galleries = [], videos = [] }: Props = $props();
</script>

{#if galleries.length > 0 || videos.length > 0}
  <section class="my-10 p-5 sm:p-6 rounded-xl bg-[var(--hp-surface)] border border-[var(--hp-border)]">
    <div class="flex items-end justify-between mb-5">
      <h2 class="text-lg md:text-xl font-bold text-[var(--hp-ink)]">
        Fotoğraf &amp; Video Galerileri
      </h2>
      <span class="text-xs text-[var(--hp-muted)]">Zengin Medya</span>
    </div>

    <!-- Media Cards Grid -->
    <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
      {#each videos as v}
        <a
          href={safeUrl(v.video_url)}
          target="_blank"
          rel="noopener noreferrer"
          class="group relative rounded-lg overflow-hidden bg-[var(--hp-surface-sunken)] aspect-video flex flex-col justify-end p-3 cursor-pointer"
        >
          <img
            src={getMediaUrl(v.cover_image, 'teknoloji', v.title)}
            alt={v.title}
            class="absolute inset-0 w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
            loading="lazy"
          />
          <div class="absolute inset-0 bg-black/40 pointer-events-none"></div>

          <!-- Play Icon Overlay -->
          <div class="absolute inset-0 flex items-center justify-center pointer-events-none">
            <div class="w-10 h-10 rounded-full bg-[var(--hp-primary)]/90 text-white flex items-center justify-center">
              <svg class="w-4 h-4 ml-0.5" viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z" /></svg>
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
          class="group relative rounded-lg overflow-hidden bg-[var(--hp-surface-sunken)] aspect-video flex flex-col justify-end p-3 cursor-pointer"
        >
          <img
            src={getMediaUrl(g.cover_image, 'dunya', g.title)}
            alt={g.title}
            class="absolute inset-0 w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
            loading="lazy"
          />
          <div class="absolute inset-0 bg-gradient-to-t from-black via-black/40 to-transparent pointer-events-none"></div>

          <div class="absolute top-2.5 right-2.5 px-2 py-0.5 rounded-full bg-black/70 text-white text-[10px] font-bold backdrop-blur-md">
            Galeri
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
