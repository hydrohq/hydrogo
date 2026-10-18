<script lang="ts">
  import { type Columnist, getMediaUrl } from '#lib';

  interface Props {
    columnists: Columnist[];
  }

  let { columnists = [] }: Props = $props();
</script>

{#if columnists.length > 0}
  <section class="p-5 sm:p-6 rounded-xl bg-[var(--hp-surface)] border border-[var(--hp-border)]">
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      {#each columnists as c}
        <article class="group p-4 rounded-lg bg-[var(--hp-surface-sunken)] cursor-pointer flex flex-col justify-between transition-colors hover:bg-[var(--hp-primary-light)]">
          <div class="flex items-center gap-3.5 mb-3">
            {#if c.avatar}
              <img
                src={getMediaUrl(c.avatar, 'kultur', c.name)}
                alt={c.name}
                class="w-12 h-12 rounded-full object-cover border border-[var(--hp-border-strong)] shrink-0"
                loading="lazy"
              />
            {:else}
              <div class="w-12 h-12 rounded-full bg-[var(--hp-surface)] text-[var(--hp-ink-soft)] flex items-center justify-center font-bold text-sm shrink-0 border border-[var(--hp-border-strong)]">
                {c.name.slice(0, 2).toUpperCase()}
              </div>
            {/if}
            <div class="min-w-0">
              <h3 class="text-sm font-bold text-[var(--hp-ink)] truncate">{c.name}</h3>
              {#if c.title}
                <p class="text-[11px] text-[var(--hp-muted)] truncate">{c.title}</p>
              {/if}
            </div>
          </div>

          {#if c.latest_article_title}
            <div class="pt-2.5 border-t border-[var(--hp-border)]">
              <h4 class="text-xs font-semibold text-[var(--hp-ink-soft)] line-clamp-2 italic group-hover:underline leading-relaxed">
                "{c.latest_article_title}"
              </h4>
            </div>
          {/if}
        </article>
      {/each}
    </div>
  </section>
{/if}
