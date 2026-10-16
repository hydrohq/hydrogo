<script lang="ts">
  import { router } from '$lib/router.svelte';
  import { schema } from '$lib/stores/schema.svelte';
  import { sidebar } from '$lib/stores/sidebar.svelte';
  import { theme } from '$lib/stores/theme.svelte';
  import SearchCommandModal from '$lib/components/shell/SearchCommandModal.svelte';
  import NotificationPopover from '$lib/components/shell/NotificationPopover.svelte';

  const path = $derived(router.path);
  let showSearchModal = $state(false);

  const pageTitle = $derived.by(() => {
    if (path === '/' || path === '/dashboard' || path === '/admin' || path === '/admin/' || path === '') {
      return 'Dashboard';
    }
    if (path.startsWith('/content/')) {
      const parts = path.split('/');
      const type = parts[2];
      const collection = schema.getCollection(type);
      const name = collection?.label || collection?.plural || type;
      if (parts[3] === 'new') return `New ${name}`;
      if (parts[3]) return `Edit ${name}`;
      return collection?.plural || collection?.label || name;
    }
    if (path.startsWith('/pages/new')) return 'New Page';
    if (path.startsWith('/pages/')) return 'Edit Page';
    if (path.startsWith('/pages')) return 'Pages';
    if (path.startsWith('/news/new')) return 'New Article';
    if (path.startsWith('/news/')) return 'Edit Article';
    if (path.startsWith('/news')) return 'News';
    if (path.startsWith('/media')) return 'Media Library';
    if (path.startsWith('/members')) return 'Members';
    if (path.startsWith('/site')) return 'Site Settings';
    if (path.startsWith('/users')) return 'Users & Access';
    if (path.startsWith('/account')) return 'My Account';
    if (path.startsWith('/audit')) return 'Audit Trail';
    if (path.startsWith('/system')) return 'Telemetry';
    return 'Documents';
  });

  function handleToggleSidebar() {
    if (window.innerWidth < 768) {
      sidebar.toggleMobile();
    } else {
      sidebar.toggle();
    }
  }
</script>

<svelte:window
  onkeydown={(e) => {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault();
      showSearchModal = true;
    }
  }}
/>

<header class="flex h-12 shrink-0 items-center gap-2 border-b border-border bg-background transition-[width,height] ease-linear">
  <div class="flex w-full items-center justify-between gap-1 px-4 lg:gap-2 lg:px-6">
    <!-- Left: SidebarTrigger, Vertical Separator, Page Title (exact shadcn SiteHeader layout) -->
    <div class="flex items-center min-w-0">
      <button
        type="button"
        onclick={handleToggleSidebar}
        class="-ml-1 flex size-7 items-center justify-center rounded-md hover:bg-accent text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
        title="Toggle Sidebar"
        aria-label="Toggle Sidebar"
      >
        <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect width="18" height="18" x="3" y="3" rx="2"></rect>
          <path d="M9 3v18"></path>
        </svg>
      </button>

      <div class="mx-2 h-4 w-px bg-border"></div>

      <h1 class="text-base font-medium text-foreground truncate">{pageTitle}</h1>
    </div>

    <!-- Right: Search, Live site, Status Indicator -->
    <div class="flex items-center gap-2.5">
      <!-- Search button (desktop) -->
      <button
        type="button"
        onclick={() => (showSearchModal = true)}
        class="hidden sm:flex items-center gap-2 h-8 w-44 lg:w-56 rounded-md bg-muted/40 border border-input px-2.5 text-xs text-muted-foreground hover:bg-muted/70 hover:text-foreground transition-colors cursor-pointer"
        title="Quick search (⌘K or Ctrl+K)"
      >
        <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="11" cy="11" r="8"></circle>
          <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        </svg>
        <span class="truncate flex-1 text-left">Search...</span>
        <kbd class="px-1.5 py-0.5 text-[10px] font-mono bg-background text-muted-foreground border border-border rounded">Ctrl K</kbd>
      </button>

      <!-- Search button (mobile) -->
      <button
        type="button"
        onclick={() => (showSearchModal = true)}
        class="flex sm:hidden size-8 items-center justify-center rounded-md border border-input bg-background hover:bg-accent text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
        title="Quick search"
        aria-label="Quick search"
      >
        <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"></circle>
          <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        </svg>
      </button>

      {#if schema.locales.length > 1}
        <span class="inline-flex items-center px-2 py-0.5 rounded-md border border-border text-[10px] font-mono uppercase font-medium text-muted-foreground bg-muted/30">
          {schema.defaultLocale}
        </span>
      {/if}

      <a
        href="/"
        target="_blank"
        rel="noopener noreferrer"
        class="group inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground hover:text-foreground h-8 px-2.5 rounded-md border border-input bg-background hover:bg-accent hover:border-primary/40 transition-all duration-150 shadow-xs active:scale-95"
      >
        <span>Live</span>
        <svg class="size-3 group-hover:translate-x-0.5 group-hover:-translate-y-0.5 transition-transform duration-150" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
          <polyline points="15 3 21 3 21 9"></polyline>
          <line x1="10" y1="14" x2="21" y2="3"></line>
        </svg>
      </a>

      <!-- Notifications Popover -->
      <NotificationPopover />

      <!-- Theme Toggle Button -->
      <button
        type="button"
        onclick={() => theme.toggle()}
        class="flex size-8 items-center justify-center rounded-md border border-input bg-background hover:bg-accent text-muted-foreground hover:text-foreground transition-all duration-200 cursor-pointer shadow-xs active:scale-90 hover:rotate-12"
        title={theme.isDark ? "Switch to light mode" : "Switch to dark mode"}
        aria-label="Toggle theme"
      >
        {#if theme.isDark}
          <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="4"></circle>
            <path d="M12 2v2"></path>
            <path d="M12 20v2"></path>
            <path d="m4.93 4.93 1.41 1.41"></path>
            <path d="m17.66 17.66 1.41 1.41"></path>
            <path d="M2 12h2"></path>
            <path d="M20 12h2"></path>
            <path d="m6.34 17.66-1.41 1.41"></path>
            <path d="m19.07 4.93-1.41 1.41"></path>
          </svg>
        {:else}
          <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"></path>
          </svg>
        {/if}
      </button>

      <div class="flex items-center gap-1.5 pl-2 border-l border-border select-none">
        <span class="relative flex size-2">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
          <span class="relative inline-flex rounded-full size-2 bg-emerald-500 pulse-dot"></span>
        </span>
        <span class="text-[11px] text-muted-foreground font-medium hidden md:inline">Online</span>
      </div>
    </div>
  </div>
</header>

<SearchCommandModal bind:open={showSearchModal} onclose={() => (showSearchModal = false)} />
