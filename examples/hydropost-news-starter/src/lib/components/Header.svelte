<script lang="ts">
  import { getWeather, type NavItem, type NewsArticle } from '#lib';
  import BreakingNewsTicker from './BreakingNewsTicker.svelte';

  interface Props {
    nav: NavItem[];
    breakingNews: NewsArticle[];
    siteName?: string;
  }

  let { nav = [], breakingNews = [], siteName = 'HydroPost' }: Props = $props();

  const currentDate = new Date().toLocaleDateString('tr-TR', {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });

  let weather = $derived(getWeather('İstanbul'));
  let menuOpen = $state(false);
  let searchOpen = $state(false);
  let searchQuery = $state('');
  let searchInput = $state<HTMLInputElement | null>(null);

  function openSearch() {
    searchOpen = true;
    requestAnimationFrame(() => searchInput?.focus());
  }

  function submitSearch(e: SubmitEvent) {
    e.preventDefault();
    const q = searchQuery.trim();
    window.location.href = q ? `/ara?q=${encodeURIComponent(q)}` : '/ara';
  }
</script>

<header class="w-full bg-[var(--hp-surface)] border-b border-[var(--hp-border)] select-none sticky top-0 z-30">
  <!-- Main nav row -->
  <div class="px-4 max-w-7xl mx-auto flex items-center gap-6 h-16">
    <a href="/" class="flex items-center gap-2 shrink-0">
      <div class="w-8 h-8 rounded-full bg-[var(--hp-primary)] flex items-center justify-center font-black text-white text-sm">
        H
      </div>
      <span class="hidden sm:inline font-extrabold text-[var(--hp-ink)] text-lg tracking-tight">
        {siteName}
      </span>
    </a>

    <nav class="hidden lg:flex items-center gap-5 overflow-x-auto scrollbar-none flex-1" aria-label="Ana kategoriler">
      {#each nav as item}
        <a
          href={item.href}
          class="text-[13px] font-semibold text-[var(--hp-ink-soft)] hover:text-[var(--hp-primary)] whitespace-nowrap transition-colors"
        >
          {item.label}
        </a>
      {/each}
    </nav>

    <div class="flex items-center gap-3 ml-auto">
      <div class="hidden md:flex items-center gap-1.5 text-xs text-[var(--hp-muted)] border-r border-[var(--hp-border)] pr-3">
        <span>{weather.icon}</span>
        <span class="font-bold text-[var(--hp-ink)] hp-tabular">{weather.temp}°C</span>
        <span class="hidden xl:inline">İstanbul</span>
      </div>
      <span class="hidden md:inline text-[11px] text-[var(--hp-muted)] border-r border-[var(--hp-border)] pr-3">{currentDate}</span>

      <div class="relative">
        <button
          type="button"
          aria-label="Ara"
          aria-expanded={searchOpen}
          onclick={() => (searchOpen ? (searchOpen = false) : openSearch())}
          class="w-9 h-9 flex items-center justify-center rounded-full transition-colors cursor-pointer {searchOpen ? 'bg-[var(--hp-surface-sunken)] text-[var(--hp-primary)]' : 'text-[var(--hp-ink-soft)] hover:bg-[var(--hp-surface-sunken)]'}"
        >
          <svg class="w-[18px] h-[18px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="7" /><path d="m21 21-4.3-4.3" stroke-linecap="round" />
          </svg>
        </button>

        {#if searchOpen}
          <div class="fixed inset-0 z-40" onclick={() => (searchOpen = false)} aria-hidden="true"></div>
          <div class="absolute right-0 top-full mt-2 z-50 w-72 sm:w-80 rounded-xl border border-[var(--hp-border)] bg-[var(--hp-surface)] shadow-lg p-3">
            <form onsubmit={submitSearch} class="flex items-center gap-2">
              <div class="relative flex-1">
                <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--hp-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="11" cy="11" r="7" /><path d="m21 21-4.3-4.3" stroke-linecap="round" />
                </svg>
                <input
                  bind:this={searchInput}
                  type="search"
                  bind:value={searchQuery}
                  placeholder="Haber ara…"
                  onkeydown={(e) => e.key === 'Escape' && (searchOpen = false)}
                  class="w-full pl-9 pr-3 py-2 rounded-lg border border-[var(--hp-border-strong)] bg-[var(--hp-surface)] text-sm text-[var(--hp-ink)] outline-none focus:border-[var(--hp-primary)] transition-colors"
                />
              </div>
              <button type="submit" class="shrink-0 px-3 py-2 rounded-lg bg-[var(--hp-primary)] hover:bg-[var(--hp-primary-dark)] text-white text-sm font-bold transition-colors cursor-pointer">
                Ara
              </button>
            </form>
          </div>
        {/if}
      </div>
      <button
        type="button"
        aria-label="Menü"
        aria-expanded={menuOpen}
        onclick={() => (menuOpen = !menuOpen)}
        class="lg:hidden w-9 h-9 flex items-center justify-center rounded-full text-[var(--hp-ink-soft)] hover:bg-[var(--hp-surface-sunken)] transition-colors cursor-pointer"
      >
        <svg class="w-[18px] h-[18px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <path d="M4 7h16M4 12h16M4 17h16" />
        </svg>
      </button>
    </div>
  </div>

  {#if menuOpen}
    <nav class="lg:hidden border-t border-[var(--hp-border)] px-4 py-3 flex flex-col gap-1" aria-label="Ana kategoriler (mobil)">
      {#each nav as item}
        <a
          href={item.href}
          class="px-2 py-2 text-sm font-semibold text-[var(--hp-ink-soft)] hover:text-[var(--hp-primary)] transition-colors"
          onclick={() => (menuOpen = false)}
        >
          {item.label}
        </a>
      {/each}
    </nav>
  {/if}

  <!-- Sitewide, continuously looping breaking-news strip -->
  <BreakingNewsTicker items={breakingNews} />
</header>
