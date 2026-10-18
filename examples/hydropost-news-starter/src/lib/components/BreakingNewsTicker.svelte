<script lang="ts">
  import { type NewsArticle } from '#lib';

  interface Props {
    items: NewsArticle[];
  }

  let { items = [] }: Props = $props();

  function timeOf(item: NewsArticle): string {
    if (!item.created_at) return '';
    return new Date(item.created_at).toLocaleTimeString('tr-TR', { hour: '2-digit', minute: '2-digit' });
  }
</script>

{#if items.length > 0}
  <div class="w-full bg-[var(--hp-surface-sunken)] border-b border-[var(--hp-border)] select-none overflow-hidden">
    <div class="max-w-7xl mx-auto flex items-stretch">
      <div class="flex items-center gap-1.5 shrink-0 px-3 py-2 bg-[var(--hp-primary)] text-white font-extrabold text-[11px] uppercase tracking-wider z-10">
        <span class="w-1.5 h-1.5 rounded-full bg-white animate-pulse"></span>
        Son Dakika
      </div>

      <div class="flex-1 overflow-hidden flex items-center py-2">
        <div class="flex items-center gap-8 whitespace-nowrap hp-marquee">
          {#each [0, 1] as dup}
            <div class="flex items-center gap-8" aria-hidden={dup === 1}>
              {#each items as item}
                <a href={`/${item.category || 'haber'}/${item.slug}`} class="flex items-center gap-2 text-[12px] text-[var(--hp-ink-soft)] hover:text-[var(--hp-primary)] transition-colors px-2">
                  {#if timeOf(item)}<span class="hp-tabular text-[var(--hp-muted)] font-semibold">{timeOf(item)}</span>{/if}
                  <span class="font-medium">{item.title}</span>
                </a>
              {/each}
            </div>
          {/each}
        </div>
      </div>

      <a href="/son-dakika" class="shrink-0 flex items-center px-3 text-[12px] font-bold text-[var(--hp-primary)] hover:text-[var(--hp-primary-dark)] bg-[var(--hp-surface-sunken)] z-10">
        Tümü →
      </a>
    </div>
  </div>
{/if}

<style>
  .hp-marquee {
    animation: hp-marquee-scroll 32s linear infinite;
  }
  .hp-marquee:hover {
    animation-play-state: paused;
  }
  @keyframes hp-marquee-scroll {
    from {
      transform: translateX(0);
    }
    to {
      transform: translateX(-50%);
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .hp-marquee {
      animation: none;
    }
  }
</style>
