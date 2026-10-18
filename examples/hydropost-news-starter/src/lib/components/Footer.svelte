<script lang="ts">
  import type { NavItem, SiteSettings } from '#lib';

  interface Props {
    settings: SiteSettings;
    pagesNav: NavItem[];
    socialNav: NavItem[];
  }

  let { settings, pagesNav = [], socialNav = [] }: Props = $props();
  const currentYear = new Date().getFullYear();
</script>

<footer class="mt-16 px-3 sm:px-4 pb-6">
  <div class="max-w-7xl mx-auto rounded-2xl border border-[var(--hp-border)] bg-[var(--hp-surface)] overflow-hidden">
    <div class="p-6 sm:p-8 grid grid-cols-1 md:grid-cols-[1fr_1fr_auto] gap-8">
      <div>
        <h4 class="font-bold text-[var(--hp-ink)] text-sm mb-3">Sayfalar</h4>
        <ul class="space-y-2 text-sm text-[var(--hp-muted)]">
          {#each pagesNav as item}
            <li><a href={item.href} class="hover:text-[var(--hp-primary)] transition-colors">{item.label}</a></li>
          {/each}
        </ul>
      </div>

      <div>
        <h4 class="font-bold text-[var(--hp-ink)] text-sm mb-3">Sosyal</h4>
        <ul class="space-y-2 text-sm text-[var(--hp-muted)]">
          {#each socialNav as item}
            <li><a href={item.href} target="_blank" rel="noopener noreferrer" class="hover:text-[var(--hp-primary)] transition-colors">{item.label}</a></li>
          {/each}
        </ul>
      </div>

      {#if settings.footer_highlight_title}
        <div class="md:w-80 rounded-xl bg-[var(--hp-surface-sunken)] p-5">
          <div class="w-9 h-9 rounded-full bg-[var(--hp-primary)] flex items-center justify-center font-black text-white text-sm mb-3">
            H
          </div>
          <p class="text-xs text-[var(--hp-muted)] leading-relaxed">
            {settings.footer_highlight_text}
          </p>
          {#if settings.footer_highlight_cta_text}
            <a href={settings.footer_highlight_cta_href || '/'} class="inline-block mt-3 text-sm font-bold text-[var(--hp-primary)] hover:text-[var(--hp-primary-dark)] transition-colors">
              {settings.footer_highlight_cta_text}
            </a>
          {/if}
        </div>
      {/if}
    </div>

    <div class="border-t border-[var(--hp-border)] py-4 px-6 text-center text-[var(--hp-muted)] text-xs">
      © {currentYear} {settings.site_name}{settings.copyright_text ? `. ${settings.copyright_text}` : ''}
    </div>
  </div>
</footer>
