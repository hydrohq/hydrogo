<script lang="ts">
  import { tick } from 'svelte';
  import { ui } from '../ui.svelte';
  import { search, sections } from '../docs';
  import { docsHref } from '../router.svelte';

  let query = $state('');
  let selected = $state(0);
  let input: HTMLInputElement | undefined = $state();
  let opener: HTMLElement | null = null;

  const hits = $derived(search(query));
  const suggestions = sections.map((s) => ({ section: s, page: s.pages[0] }));
  const count = $derived(query.trim() ? hits.length : suggestions.length);

  $effect(() => {
    if (ui.searchOpen) {
      opener = document.activeElement as HTMLElement | null;
      query = '';
      selected = 0;
      tick().then(() => input?.focus());
      document.body.style.overflow = 'hidden';
      return () => {
        document.body.style.overflow = '';
      };
    }
  });

  $effect(() => {
    query;
    selected = 0;
  });

  function close() {
    ui.searchOpen = false;
    opener?.focus?.();
  }

  function go(i: number) {
    const target = query.trim()
      ? hits[i] && docsHref(hits[i].section.id, hits[i].page.id, hits[i].anchor)
      : suggestions[i] && docsHref(suggestions[i].section.id, suggestions[i].page.id);
    if (!target) return;
    ui.searchOpen = false;
    window.location.hash = target.replace(/^#/, '');
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      close();
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      selected = count ? (selected + 1) % count : 0;
      scrollSelected();
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      selected = count ? (selected - 1 + count) % count : 0;
      scrollSelected();
    } else if (e.key === 'Enter') {
      e.preventDefault();
      go(selected);
    } else if (e.key === 'Tab') {
      e.preventDefault();
    }
  }

  function scrollSelected() {
    tick().then(() => document.getElementById(`search-opt-${selected}`)?.scrollIntoView({ block: 'nearest' }));
  }
</script>

{#if ui.searchOpen}
  <div class="fixed inset-0 z-[60] flex items-start justify-center px-4 pt-[10vh]" role="presentation">
    <button type="button" class="absolute inset-0 cursor-default bg-black/50 backdrop-blur-[2px]" aria-label="Close search" tabindex="-1" onclick={close}></button>
    <div
      class="relative flex max-h-[75vh] w-full max-w-xl flex-col overflow-hidden rounded-xl border border-line-strong bg-elevated shadow-2xl"
      role="dialog"
      aria-modal="true"
      aria-label="Search documentation"
      tabindex="-1"
      {onkeydown}
    >
      <div class="flex items-center gap-3 border-b border-line px-4">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" class="shrink-0 text-faint" aria-hidden="true"><circle cx="11" cy="11" r="7" /><line x1="21" y1="21" x2="16.65" y2="16.65" /></svg>
        <input
          bind:this={input}
          bind:value={query}
          type="text"
          role="combobox"
          aria-expanded="true"
          aria-controls="search-results"
          aria-activedescendant={count ? `search-opt-${selected}` : undefined}
          aria-label="Search documentation"
          placeholder="Search titles and text…"
          autocomplete="off"
          spellcheck="false"
          class="h-14 w-full bg-transparent text-base text-fg outline-none placeholder:text-faint"
        />
        <kbd class="rounded border border-line px-1.5 py-0.5 text-[11px] text-faint">Esc</kbd>
      </div>

      <ul id="search-results" role="listbox" aria-label="Results" class="thin-scroll min-h-0 flex-1 overflow-y-auto p-2">
        {#if query.trim() && hits.length === 0}
          <li class="px-3 py-8 text-center text-sm text-muted" role="presentation">No results for <strong class="text-fg">“{query}”</strong>. Try fewer or different words.</li>
        {:else if query.trim()}
          {#each hits as h, i}
            <li role="presentation">
              <a
                id="search-opt-{i}"
                role="option"
                aria-selected={selected === i}
                href={docsHref(h.section.id, h.page.id, h.anchor)}
                onclick={() => (ui.searchOpen = false)}
                onmousemove={() => (selected = i)}
                class="block rounded-lg px-3 py-2.5 no-underline {selected === i ? 'bg-accent-soft' : ''}"
              >
                <span class="flex items-center gap-2 text-xs text-faint">{h.section.title} <span aria-hidden="true">/</span> {h.page.title}</span>
                <span class="mt-0.5 block text-sm font-medium text-fg">{h.heading || h.page.title}</span>
                <span class="mt-0.5 line-clamp-2 block text-xs leading-5 text-muted">{h.snippet}</span>
              </a>
            </li>
          {/each}
        {:else}
          <li role="presentation" class="px-3 pb-1 pt-2 text-xs font-semibold uppercase tracking-wider text-faint">Jump to</li>
          {#each suggestions as s, i}
            <li role="presentation">
              <a
                id="search-opt-{i}"
                role="option"
                aria-selected={selected === i}
                href={docsHref(s.section.id, s.page.id)}
                onclick={() => (ui.searchOpen = false)}
                onmousemove={() => (selected = i)}
                class="flex items-center justify-between rounded-lg px-3 py-2 text-sm no-underline {selected === i ? 'bg-accent-soft' : ''}"
              >
                <span class="font-medium text-fg">{s.section.title}</span>
                <span class="text-xs text-faint">{s.section.pages.length} {s.section.pages.length === 1 ? 'page' : 'pages'}</span>
              </a>
            </li>
          {/each}
        {/if}
      </ul>

      <div class="flex items-center gap-4 border-t border-line px-4 py-2 text-[11px] text-faint">
        <span><kbd class="rounded border border-line px-1">↑</kbd> <kbd class="rounded border border-line px-1">↓</kbd> navigate</span>
        <span><kbd class="rounded border border-line px-1">Enter</kbd> open</span>
        <span class="ml-auto" aria-live="polite">{query.trim() ? `${hits.length} result${hits.length === 1 ? '' : 's'}` : ''}</span>
      </div>
    </div>
  </div>
{/if}
