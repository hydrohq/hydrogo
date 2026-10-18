<script lang="ts">
  import { type NewsArticle, getCategoryBadge, slugify } from '#lib';

  interface Props {
    items: NewsArticle[];
  }

  let { items = [] }: Props = $props();

  function dateOf(item: NewsArticle): string {
    return item.created_at ? new Date(item.created_at).toLocaleDateString('tr-TR') : '';
  }
</script>

{#if items.length > 0}
  <section class="my-10">
    <div class="flex items-end justify-between mb-5">
      <h2 class="text-lg md:text-xl font-bold text-[var(--hp-primary)]">Çok Okunan Haberler</h2>
      <a href="/gundem" class="text-xs font-bold text-[var(--hp-muted)] hover:text-[var(--hp-primary)] flex items-center gap-1.5 transition-colors">Tümünü Gör <span>→</span></a>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-5">
      {#each items as item, i}
        {@const badge = getCategoryBadge(item.category)}
        <a href={`/${slugify(item.category)}/${item.slug}`} class="group flex items-start gap-3.5">
          <span class="text-2xl font-black text-[var(--hp-border-strong)] group-hover:text-[var(--hp-primary)] transition-colors leading-none shrink-0 w-7 hp-tabular">
            {i + 1}
          </span>
          <div class="min-w-0">
            <span class={`inline-block px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider ${badge.bg} ${badge.text}`}>
              {item.category}
            </span>
            <h3 class="mt-1.5 text-sm font-bold text-[var(--hp-ink)] leading-snug group-hover:text-[var(--hp-primary)] transition-colors">
              {item.title}
            </h3>
            <span class="mt-1 block text-[11px] text-[var(--hp-muted)] hp-tabular">{dateOf(item)}</span>
          </div>
        </a>
      {/each}
    </div>
  </section>
{/if}
