<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { router } from '$lib/router.svelte';
  import { schema } from '$lib/stores/schema.svelte';
  import { sidebar } from '$lib/stores/sidebar.svelte';
  import { theme } from '$lib/stores/theme.svelte';
  import SearchCommandModal from '$lib/components/shell/SearchCommandModal.svelte';
  import NotificationPopover from '$lib/components/shell/NotificationPopover.svelte';
  import CopilotDrawer from '$lib/components/shell/CopilotDrawer.svelte';

  const path = $derived(router.path);
  let showSearchModal = $state(false);
  let showCopilotDrawer = $state(false);

  const pageTitle = $derived.by(() => {
    if (path === '/' || path === '/dashboard' || path === '/admin' || path === '/admin/' || path === '') {
      return t('nav.dashboard');
    }
    if (path.startsWith('/content/')) {
      const parts = path.split('/');
      const type = parts[2];
      const collection = schema.getCollection(type);
      const name = collection?.label || collection?.plural || type;
      if (parts[3] === 'new') return t('title.newItem', { name });
      if (parts[3]) return t('title.editItem', { name });
      return collection?.plural || collection?.label || name;
    }
    if (path.startsWith('/pages/new')) return t('nav.newPage');
    if (path.startsWith('/pages/')) return t('title.editPage');
    if (path.startsWith('/pages')) return t('nav.pages');
    if (path.startsWith('/news/new')) return t('nav.newArticle');
    if (path.startsWith('/news/')) return t('title.editArticle');
    if (path.startsWith('/news')) return t('nav.news');
    if (path.startsWith('/media')) return t('nav.media');
    if (path.startsWith('/members')) return t('nav.members');
    if (path.startsWith('/commerce/products/new')) return t('products.new');
    if (path.startsWith('/commerce/products')) return t('products.title');
    if (path.startsWith('/commerce/stock')) return t('stock.title');
    if (path.startsWith('/commerce/refunds-required')) return t('nav.refundsRequired');
    if (path.startsWith('/commerce/discount-codes')) return t('nav.discountCodes');
    if (path.startsWith('/commerce')) return t('nav.orders');
    if (path.startsWith('/site')) return t('title.siteSettings');
    if (path.startsWith('/users')) return t('title.users');
    if (path.startsWith('/roles')) return t('nav.roles');
    if (path.startsWith('/account')) return t('title.account');
    if (path.startsWith('/audit')) return t('title.audit');
    if (path.startsWith('/system')) return t('nav.telemetry');
    if (path.startsWith('/content-types')) return t('ct.title');
    if (path.startsWith('/console') || path.startsWith('/orchestrator')) return t('nav.console');
    return t('nav.documents');
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

<header class="flex h-12 shrink-0 items-center gap-2 border-b border-border bg-background">
  <div class="flex w-full items-center justify-between gap-2 px-4 lg:px-6">
    <!-- Left: SidebarTrigger, Vertical Separator, Page Title -->
    <div class="flex items-center min-w-0">
      <button
        type="button"
        onclick={handleToggleSidebar}
        class="-ml-1 flex size-8 items-center justify-center rounded-md hover:bg-accent text-muted-foreground hover:text-foreground transition-colors duration-150 cursor-pointer"
        title={t('header.toggleSidebar')}
        aria-label={t('header.toggleSidebar')}
      >
        <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <rect width="18" height="18" x="3" y="3" rx="2"></rect>
          <path d="M9 3v18"></path>
        </svg>
      </button>

      <div class="mx-2 h-4 w-px bg-border"></div>

      <h1 class="text-base font-medium text-foreground truncate">{pageTitle}</h1>
    </div>

    <!-- Right: Search, Live site, Status Indicator -->
    <div class="flex items-center gap-2">
      <!-- Search button (desktop) -->
      <button
        type="button"
        onclick={() => (showSearchModal = true)}
        class="hidden sm:flex items-center gap-2 h-8 w-44 lg:w-56 rounded-md bg-muted/40 border border-input px-2.5 text-xs text-muted-foreground hover:bg-muted/70 hover:text-foreground transition-colors duration-150 cursor-pointer"
        title={t('header.searchTip')}
      >
        <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="11" cy="11" r="8"></circle>
          <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        </svg>
        <span class="truncate flex-1 text-left">{t('header.search')}</span>
        <kbd class="px-1.5 py-0.5 text-[10px] font-mono bg-background text-muted-foreground border border-border rounded">Ctrl K</kbd>
      </button>

      <!-- Search button (mobile) -->
      <button
        type="button"
        onclick={() => (showSearchModal = true)}
        class="flex sm:hidden size-8 items-center justify-center rounded-md border border-input bg-background hover:bg-accent text-muted-foreground hover:text-foreground transition-colors duration-150 cursor-pointer"
        title={t('header.quickSearch')}
        aria-label={t('header.quickSearch')}
      >
        <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="11" cy="11" r="8"></circle>
          <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        </svg>
      </button>

      {#if schema.siteUrl}
      <a
        href={schema.siteUrl}
        target="_blank"
        rel="noopener noreferrer"
        class="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground hover:text-foreground h-8 px-2.5 rounded-md border border-input bg-background hover:bg-accent transition-colors duration-150 shadow-xs"
      >
        <span>{t('header.live')}</span>
        <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
          <polyline points="15 3 21 3 21 9"></polyline>
          <line x1="10" y1="14" x2="21" y2="3"></line>
        </svg>
      </a>
      {/if}

      <!-- Workspace AI Copilot Button -->
      {#if schema.modules['ai']}
      <button
        type="button"
        disabled
        aria-disabled="true"
        onclick={() => (showCopilotDrawer = true)}
        class="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground h-8 px-2.5 rounded-md border border-border bg-muted/40 opacity-75 cursor-not-allowed shadow-xs"
        title={`${t('copilot.title')} (${t('common.soon')})`}
      >
        <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M12 3l1.9 5.8a2 2 0 0 0 1.3 1.3L21 12l-5.8 1.9a2 2 0 0 0-1.3 1.3L12 21l-1.9-5.8a2 2 0 0 0-1.3-1.3L3 12l5.8-1.9a2 2 0 0 0 1.3-1.3L12 3z" />
        </svg>
        <span class="hidden sm:inline">{t('copilot.button')}</span>
        <span class="rounded bg-muted px-1 py-0.5 text-[10px] font-medium text-muted-foreground">{t('common.soon')}</span>
      </button>
      {/if}

      <!-- Notifications Popover -->
      <NotificationPopover />

      <!-- Theme Toggle Button -->
      <button
        type="button"
        onclick={() => theme.toggle()}
        class="flex size-8 items-center justify-center rounded-md border border-input bg-background hover:bg-accent text-muted-foreground hover:text-foreground transition-colors duration-150 cursor-pointer shadow-xs"
        title={theme.isDark ? t('header.toLight') : t('header.toDark')}
        aria-label={t('header.toggleTheme')}
      >
        {#if theme.isDark}
          <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
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
          <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"></path>
          </svg>
        {/if}
      </button>
    </div>
  </div>
</header>

<SearchCommandModal bind:open={showSearchModal} onclose={() => (showSearchModal = false)} />
<CopilotDrawer bind:open={showCopilotDrawer} onclose={() => (showCopilotDrawer = false)} />
