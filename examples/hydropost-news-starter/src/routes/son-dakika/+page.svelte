<script lang="ts">
  import type { PageProps } from './$types';
  import { getMediaUrl, slugify, type NewsArticle } from '#lib';

  let { data }: PageProps = $props();

  function timeOf(item: NewsArticle): string {
    return item.created_at ? new Date(item.created_at).toLocaleTimeString('tr-TR', { hour: '2-digit', minute: '2-digit' }) : '--:--';
  }
  function hourMilestone(item: NewsArticle): string {
    return item.created_at ? `${new Date(item.created_at).getHours().toString().padStart(2, '0')}:00` : '--:00';
  }
  function dateOf(item: NewsArticle): string {
    return item.created_at ? new Date(item.created_at).toLocaleDateString('tr-TR', { day: 'numeric', month: 'long', year: 'numeric' }) : '';
  }

  // Group into rows of 3, with an hour milestone divider after each row.
  let rows = $derived.by(() => {
    const out: { items: NewsArticle[]; milestone: string }[] = [];
    for (let i = 0; i < data.timeline.length; i += 3) {
      const chunk = data.timeline.slice(i, i + 3);
      if (chunk.length === 0) continue;
      out.push({ items: chunk, milestone: hourMilestone(chunk[chunk.length - 1]) });
    }
    return out;
  });
</script>

<svelte:head>
  <title>Son Dakika — HydroPost</title>
  <meta name="description" content="Son dakika haberleri, en güncel gelişmeler." />
</svelte:head>

<div class="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-8 py-4">
  <div>
    <div class="flex items-center gap-3 mb-8">
      <svg class="w-7 h-7 text-[var(--hp-ink)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M3 12a9 9 0 1 0 3-6.7" stroke-linecap="round" /><path d="M3 4v5h5" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
      <h1 class="text-2xl sm:text-3xl font-black text-[var(--hp-ink)]">Son Dakika!</h1>
      <div class="flex-1 h-px bg-[var(--hp-border-strong)]"></div>
    </div>

    {#each rows as row}
      <div class="flex items-center gap-4 mb-5 text-xs font-bold text-[var(--hp-muted)]">
        {#each row.items as item, i}
          {#if i > 0}<div class="flex-1 h-px bg-[var(--hp-border-strong)]"></div>{/if}
          <span class="flex items-center gap-1.5 shrink-0 hp-tabular">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 3" stroke-linecap="round" /></svg>
            {timeOf(item)}
          </span>
        {/each}
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-3 gap-5 mb-8">
        {#each row.items as item, i (item.id)}
          <a href={`/${slugify(item.category)}/${item.slug}`} class="group">
            <div class="aspect-video rounded-lg overflow-hidden bg-[var(--hp-surface-sunken)] mb-2.5">
              <img src={getMediaUrl(item.cover_media_id, item.category, item.slug, i)} alt={item.title} class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105" loading="lazy" />
            </div>
            <h3 class="text-sm font-bold text-[var(--hp-ink)] leading-snug line-clamp-3 group-hover:text-[var(--hp-primary)] transition-colors">
              {item.title}
            </h3>
          </a>
        {/each}
      </div>

      <div class="flex items-center gap-3 mb-8">
        <span class="text-2xl font-black text-[var(--hp-primary)] hp-tabular shrink-0">{row.milestone}</span>
        <div class="flex-1 h-0.5 bg-[var(--hp-primary)] relative">
          <span class="absolute right-0 -top-[3px] w-2 h-2 bg-[var(--hp-primary)]"></span>
        </div>
      </div>
    {/each}
  </div>

  <aside class="space-y-5">
    <div class="rounded-xl overflow-hidden border border-[var(--hp-border)]">
      <div class="bg-[var(--hp-primary)] px-4 py-2.5 flex items-center gap-2">
        <svg class="w-4 h-4 text-white" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2c-1 4-5 6-5 10a5 5 0 0 0 10 0c0-1.5-.5-2.5-1.3-3.6.1 1.6-.6 2.3-1.2 2.3-1 0-1.2-1-.7-1.8.9-1.4 1-3.2-1.8-6.9Z" /></svg>
        <h2 class="text-white font-bold text-sm uppercase tracking-wide">Çok Okunanlar</h2>
      </div>
      <div class="divide-y divide-[var(--hp-border)]">
        {#each data.popular as item, i (item.id)}
          <a href={`/${slugify(item.category)}/${item.slug}`} class="group flex items-stretch gap-3 p-3 hover:bg-[var(--hp-surface-sunken)] transition-colors">
            <span class="w-7 shrink-0 bg-[var(--hp-primary)] text-white font-black text-sm flex items-center justify-center rounded">{i + 1}</span>
            {#if item.cover_media_id || item.category}
              <div class="w-12 h-12 rounded-md overflow-hidden shrink-0 bg-[var(--hp-surface-sunken)]">
                <img src={getMediaUrl(item.cover_media_id, item.category, item.slug, i)} alt={item.title} class="w-full h-full object-cover" loading="lazy" />
              </div>
            {/if}
            <h4 class="text-xs font-semibold text-[var(--hp-ink-soft)] leading-snug line-clamp-3 group-hover:text-[var(--hp-primary)] transition-colors">
              {item.title}
            </h4>
          </a>
        {/each}
      </div>
    </div>

    {#if data.spotlight}
      <a href={`/${slugify(data.spotlight.category)}/${data.spotlight.slug}`} class="group block rounded-xl overflow-hidden border border-[var(--hp-border)]">
        <div class="aspect-video overflow-hidden bg-[var(--hp-surface-sunken)]">
          <img src={getMediaUrl(data.spotlight.cover_media_id, data.spotlight.category, data.spotlight.slug)} alt={data.spotlight.title} class="w-full h-full object-cover" loading="lazy" />
        </div>
        <div class="p-3.5">
          <span class="flex items-center gap-1.5 text-[11px] font-bold text-[var(--hp-primary)] mb-1.5">
            <span class="w-1.5 h-1.5 rounded-full bg-[var(--hp-primary)]"></span>
            {dateOf(data.spotlight)}
          </span>
          <h4 class="text-sm font-bold text-[var(--hp-ink)] leading-snug group-hover:text-[var(--hp-primary)] transition-colors">
            {data.spotlight.title}
          </h4>
        </div>
      </a>
    {/if}
  </aside>
</div>
