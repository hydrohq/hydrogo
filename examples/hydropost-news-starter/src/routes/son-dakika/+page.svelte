<script lang="ts">
  import type { PageProps } from './$types';
  import { getMediaUrl, slugify, type NewsArticle } from '#lib';
  import NewsCard from '#lib/components/NewsCard.svelte';

  let { data }: PageProps = $props();

  function timeOf(item: NewsArticle): string {
    return item.created_at ? new Date(item.created_at).toLocaleTimeString('tr-TR', { hour: '2-digit', minute: '2-digit' }) : '--:--';
  }
  function hourOf(item: NewsArticle): string {
    return item.created_at ? `${new Date(item.created_at).getHours().toString().padStart(2, '0')}:00` : '--:00';
  }

  // Group by hour first (one milestone divider per distinct hour, not per
  // row) — demo data can share identical timestamps, which used to produce
  // the same "20:00" divider repeated after every 3-item chunk.
  let hourGroups = $derived.by(() => {
    const groups: { hour: string; items: NewsArticle[] }[] = [];
    for (const item of data.timeline) {
      const hour = hourOf(item);
      const last = groups[groups.length - 1];
      if (last && last.hour === hour) {
        last.items.push(item);
      } else {
        groups.push({ hour, items: [item] });
      }
    }
    return groups;
  });

  function chunk3(items: NewsArticle[]): NewsArticle[][] {
    const out: NewsArticle[][] = [];
    for (let i = 0; i < items.length; i += 3) out.push(items.slice(i, i + 3));
    return out;
  }
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

    {#each hourGroups as group, gi (group.hour + gi)}
      {#each chunk3(group.items) as row, ri}
        <div class="flex items-center gap-4 mb-5 text-xs font-bold text-[var(--hp-muted)]">
          {#each row as item, i}
            {#if i > 0}<div class="flex-1 h-px bg-[var(--hp-border-strong)]"></div>{/if}
            <span class="flex items-center gap-1.5 shrink-0 hp-tabular">
              <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 3" stroke-linecap="round" /></svg>
              {timeOf(item)}
            </span>
          {/each}
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-3 gap-5 mb-8">
          {#each row as item, i (item.id)}
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
      {/each}

      <div class="flex items-center gap-3 mb-8">
        <span class="text-2xl font-black text-[var(--hp-primary)] hp-tabular shrink-0">{group.hour}</span>
        <div class="flex-1 h-0.5 bg-[var(--hp-primary)] relative">
          <span class="absolute right-0 -top-[3px] w-2 h-2 bg-[var(--hp-primary)]"></span>
        </div>
      </div>
    {/each}
  </div>

  <aside class="space-y-5">
    <div class="rounded-xl border border-[var(--hp-border)] bg-[var(--hp-surface)] p-4">
      <div class="flex items-center gap-2 mb-4">
        <svg class="w-4 h-4 text-[var(--hp-primary)]" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2c-1 4-5 6-5 10a5 5 0 0 0 10 0c0-1.5-.5-2.5-1.3-3.6.1 1.6-.6 2.3-1.2 2.3-1 0-1.2-1-.7-1.8.9-1.4 1-3.2-1.8-6.9Z" /></svg>
        <h2 class="text-[var(--hp-ink)] font-bold text-sm">Çok Okunanlar</h2>
      </div>
      <div class="space-y-3.5">
        {#each data.popular as item, i (item.id)}
          <NewsCard {item} variant="row" index={i} />
        {/each}
      </div>
    </div>
  </aside>
</div>
