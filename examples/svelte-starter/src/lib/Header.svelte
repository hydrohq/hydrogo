<script lang="ts">
  import Logo from './components/Logo.svelte';
  import ThemeToggle from './components/ThemeToggle.svelte';
  import { ui } from './ui.svelte';
  import { router } from './router.svelte';

  const onDocs = $derived(router.current.name === 'docs');
  const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/i.test(navigator.platform);
</script>

<header class="sticky top-0 z-40 border-b border-line bg-bg/85 backdrop-blur supports-[backdrop-filter]:bg-bg/70">
  <div class="mx-auto flex h-14 max-w-[90rem] items-center gap-2 px-4 sm:px-6">
    {#if onDocs}
      <button
        type="button"
        class="-ml-2 inline-flex size-9 items-center justify-center rounded-md text-muted hover:bg-subtle hover:text-fg lg:hidden"
        aria-label="Open navigation menu"
        aria-expanded={ui.drawerOpen}
        aria-controls="docs-sidebar"
        onclick={() => (ui.drawerOpen = true)}
      >
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><line x1="3" y1="6" x2="21" y2="6" /><line x1="3" y1="12" x2="21" y2="12" /><line x1="3" y1="18" x2="21" y2="18" /></svg>
      </button>
    {/if}

    <a href="#/" class="rounded-md no-underline" aria-label="HydroGo home"><Logo /></a>

    <nav class="ml-4 hidden items-center gap-1 text-sm font-medium sm:flex" aria-label="Main">
      <a href="#/docs" class="rounded-md px-3 py-1.5 no-underline {onDocs ? 'text-fg' : 'text-muted hover:text-fg'}" aria-current={onDocs ? 'page' : undefined}>Docs</a>
      <a href="#/docs/rest-api/overview" class="rounded-md px-3 py-1.5 text-muted no-underline hover:text-fg">API</a>
      <a href="#/docs/reference/cli" class="rounded-md px-3 py-1.5 text-muted no-underline hover:text-fg">CLI</a>
      <a href="/admin/" class="rounded-md px-3 py-1.5 text-muted no-underline hover:text-fg">Admin</a>
    </nav>

    <div class="ml-auto flex items-center gap-1.5">
      <button
        type="button"
        onclick={() => (ui.searchOpen = true)}
        class="inline-flex h-9 items-center gap-2 rounded-md border border-line bg-subtle px-2.5 text-sm text-muted hover:border-line-strong hover:text-fg sm:w-56 sm:justify-between"
        aria-label="Search documentation"
        aria-keyshortcuts="Control+K Meta+K"
      >
        <span class="flex items-center gap-2">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><circle cx="11" cy="11" r="7" /><line x1="21" y1="21" x2="16.65" y2="16.65" /></svg>
          <span class="hidden sm:inline">Search docs</span>
        </span>
        <kbd class="hidden rounded border border-line bg-bg px-1.5 py-0.5 font-sans text-[11px] sm:inline">{isMac ? '⌘' : 'Ctrl'} K</kbd>
      </button>
      <a
        href="https://github.com/ozyildirims/hydrogo"
        target="_blank"
        rel="noopener noreferrer"
        class="hidden size-9 items-center justify-center rounded-md text-muted hover:bg-subtle hover:text-fg sm:inline-flex"
        aria-label="HydroGo on GitHub (opens in a new tab)"
      >
        <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 .5a11.5 11.5 0 0 0-3.64 22.41c.58.1.79-.25.79-.56v-2c-3.2.7-3.88-1.37-3.88-1.37-.52-1.33-1.28-1.69-1.28-1.69-1.05-.71.08-.7.08-.7 1.16.08 1.77 1.19 1.77 1.19 1.03 1.77 2.7 1.26 3.36.96.1-.75.4-1.26.73-1.55-2.55-.29-5.24-1.28-5.24-5.69 0-1.26.45-2.28 1.19-3.09-.12-.29-.52-1.46.11-3.05 0 0 .97-.31 3.17 1.18a11 11 0 0 1 5.77 0c2.2-1.49 3.17-1.18 3.17-1.18.63 1.59.23 2.76.11 3.05.74.81 1.19 1.83 1.19 3.09 0 4.42-2.69 5.39-5.25 5.68.41.36.78 1.06.78 2.14v3.17c0 .31.21.67.8.56A11.5 11.5 0 0 0 12 .5z" /></svg>
      </a>
      <ThemeToggle />
    </div>
  </div>
</header>
