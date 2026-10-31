<script lang="ts">
  import { roleLabel } from '$lib/i18n/labels';
  import { t } from '$lib/i18n/index.svelte';
  import { auth } from '$lib/stores/auth.svelte';
  import { schema } from '$lib/stores/schema.svelte';
  import { router } from '$lib/router.svelte';
  import { sidebar } from '$lib/stores/sidebar.svelte';
  import { theme } from '$lib/stores/theme.svelte';
  import BrandLogo from '$lib/components/ui/BrandLogo.svelte';

  // A module that is not enabled adds no navigation. Until the schema has
  // loaded everything shows, so the menu does not flash empty.
  const moduleOn = (name: string) => !schema.schema || schema.modules[name] === true;
  // Hide what the signed-in user cannot open; the server still enforces it.
  const can = (permission: string) => auth.hasPermission(permission);
  const isAdmin = (auth.user?.role === 'admin');

  const currentPath = $derived(router.path);
  let showQuickCreate = $state(false);
  let showUserMenu = $state(false);
  let activeDocDropdown = $state<string | null>(null);

  function isActive(target: string): boolean {
    if (target === '/' || target === '/dashboard') {
      return (
        currentPath === '/' ||
        currentPath === '/dashboard' ||
        currentPath === '/admin' ||
        currentPath === '/admin/' ||
        currentPath === ''
      );
    }
    return currentPath.startsWith(target);
  }

  function handleNavigate(path: string) {
    showQuickCreate = false;
    showUserMenu = false;
    activeDocDropdown = null;
    sidebar.closeMobile();
    router.navigate(path);
  }
</script>

<svelte:window onclick={(e) => {
  const target = e.target as HTMLElement;
  if (!target.closest('[data-dropdown]')) {
    showQuickCreate = false;
    showUserMenu = false;
    activeDocDropdown = null;
  }
}} />

<!-- Mobile Overlay Backdrop -->
{#if sidebar.mobileOpen}
  <button
    type="button"
    class="fixed inset-0 z-40 bg-black/80 backdrop-blur-xs md:hidden"
    onclick={() => sidebar.closeMobile()}
    aria-label={t('nav.closeOverlay')}
  ></button>
{/if}

<aside
  class="fixed inset-y-0 left-0 z-50 flex h-full flex-col bg-sidebar text-sidebar-foreground transition-all duration-200 ease-in-out md:static md:z-0 {sidebar.open ? 'w-64' : 'w-16'} {sidebar.mobileOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}"
>
  <!-- SidebarHeader: Brand (Matches official shadcn AppSidebar SidebarHeader) -->
  <div class="flex h-12 shrink-0 items-center border-b border-sidebar-border px-3">
    <a
      href="/admin/dashboard"
      onclick={() => handleNavigate('/dashboard')}
      class="flex w-full items-center gap-2.5 rounded-md p-1.5 text-left text-sm font-semibold transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
    >
      <BrandLogo size="md" showText={sidebar.open} />
    </a>
  </div>

  <!-- SidebarContent (Matches official NavMain, NavDocuments, NavSecondary) -->
  <div class="flex flex-1 flex-col gap-2 overflow-y-auto px-2 py-2">
    <!-- Quick Create Button & Inbox (official nav-main.tsx layout) -->
    <div class="flex items-center gap-2 pt-1 pb-1 relative" data-dropdown="quick-create">
      {#if can('admin.content.write') || can('admin.media.manage')}
      <button
        type="button"
        onclick={() => { showQuickCreate = !showQuickCreate; }}
        class="flex flex-1 items-center justify-center gap-2 h-9 rounded-md bg-primary text-primary-foreground text-xs font-medium hover:bg-primary/90 transition-colors shadow-xs cursor-pointer px-3 {sidebar.open ? '' : 'justify-center !px-0'}"
        title={t('nav.quickCreate')}
      >
        <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="12" y1="8" x2="12" y2="16"></line>
          <line x1="8" y1="12" x2="16" y2="12"></line>
        </svg>
        {#if sidebar.open}
          <span class="truncate">{t('nav.quickCreate')}</span>
        {/if}
      </button>
      {/if}

      {#if sidebar.open && isAdmin}
        <button
          type="button"
          onclick={() => handleNavigate('/audit')}
          class="size-9 shrink-0 flex items-center justify-center rounded-md border border-sidebar-border bg-background hover:bg-sidebar-accent text-sidebar-foreground transition-colors cursor-pointer shadow-xs"
          title={t('nav.auditLog')}
        >
          <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect width="20" height="16" x="2" y="4" rx="2"></rect>
            <path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7"></path>
          </svg>
          <span class="sr-only">{t('nav.audit')}</span>
        </button>
      {/if}

      <!-- Quick Create Dropdown Menu -->
      {#if showQuickCreate}
        <div class="absolute left-0 top-11 w-48 rounded-lg border border-border bg-popover p-1 text-popover-foreground shadow-md z-50 text-xs">
          <div class="px-2 py-1.5 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
            {t('nav.createNew')}
          </div>
          {#if moduleOn('pages')}
          <button
            type="button"
            class="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-md hover:bg-accent text-left transition-colors cursor-pointer"
            onclick={() => handleNavigate('/pages/new')}
          >
            <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path><polyline points="14 2 14 8 20 8"></polyline></svg>
            <span>{t('nav.newPage')}</span>
          </button>
          {/if}
          {#if moduleOn('news')}
          <button
            type="button"
            class="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-md hover:bg-accent text-left transition-colors cursor-pointer"
            onclick={() => handleNavigate('/news/new')}
          >
            <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"></path></svg>
            <span>{t('nav.newArticle')}</span>
          </button>
          {/if}
          {#if moduleOn('media') && can('admin.media.manage')}
          <button
            type="button"
            class="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-md hover:bg-accent text-left transition-colors cursor-pointer"
            onclick={() => handleNavigate('/media')}
          >
            <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"></rect><circle cx="9" cy="9" r="2"></circle><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"></path></svg>
            <span>{t('nav.uploadMedia')}</span>
          </button>
          {/if}
          {#each schema.collections as col}
            <button
              type="button"
              class="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-md hover:bg-accent text-left transition-colors cursor-pointer"
              onclick={() => handleNavigate(`/content/${col.id}/new`)}
            >
              <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg>
              <span>{t('nav.newOf', { label: col.label })}</span>
            </button>
          {/each}
        </div>
      {/if}
    </div>

    <!-- NavMain (Matches official nav-main.tsx) -->
    <div class="flex flex-col gap-0.5">
      <!-- Dashboard -->
      <a
        href="/admin/dashboard"
        onclick={() => handleNavigate('/dashboard')}
        class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/dashboard') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
        title={t('nav.dashboard')}
      >
        <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect width="7" height="9" x="3" y="3" rx="1"></rect>
          <rect width="7" height="5" x="14" y="3" rx="1"></rect>
          <rect width="7" height="9" x="14" y="12" rx="1"></rect>
          <rect width="7" height="5" x="3" y="16" rx="1"></rect>
        </svg>
        {#if sidebar.open}
          <span class="truncate">{t('nav.dashboard')}</span>
        {/if}
      </a>

      {#if moduleOn('pages')}
      <!-- Pages -->
      <a
        href="/admin/pages"
        onclick={() => handleNavigate('/pages')}
        class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/pages') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
        title={t('nav.pages')}
      >
        <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2Z"></path>
        </svg>
        {#if sidebar.open}
          <span class="truncate">{t('nav.pages')}</span>
        {/if}
      </a>
      {/if}

      {#if moduleOn('news')}
      <!-- News -->
      <a
        href="/admin/news"
        onclick={() => handleNavigate('/news')}
        class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/news') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
        title={t('nav.news')}
      >
        <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path>
          <polyline points="14 2 14 8 20 8"></polyline>
          <line x1="16" y1="13" x2="8" y2="13"></line>
          <line x1="16" y1="17" x2="8" y2="17"></line>
          <line x1="10" y1="9" x2="8" y2="9"></line>
        </svg>
        {#if sidebar.open}
          <span class="truncate">{t('nav.news')}</span>
        {/if}
      </a>
      {/if}

      {#if moduleOn('media') && can('admin.media.manage')}
      <!-- Media -->
      <a
        href="/admin/media"
        onclick={() => handleNavigate('/media')}
        class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/media') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
        title={t('nav.media')}
      >
        <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect width="18" height="18" x="3" y="3" rx="2" ry="2"></rect>
          <circle cx="9" cy="9" r="2"></circle>
          <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"></path>
        </svg>
        {#if sidebar.open}
          <span class="truncate">{t('nav.media')}</span>
        {/if}
      </a>
      {/if}

      {#if moduleOn('members') && can('admin.members.manage')}
      <!-- Members -->
      <a
        href="/admin/members"
        onclick={() => handleNavigate('/members')}
        class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/members') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
        title={t('nav.members')}
      >
        <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"></path>
          <circle cx="9" cy="7" r="4"></circle>
          <path d="M22 21v-2a4 4 0 0 0-3-3.87"></path>
          <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
        </svg>
        {#if sidebar.open}
          <span class="truncate">{t('nav.members')}</span>
        {/if}
      </a>
      {/if}

      {#if moduleOn('commerce') && can('admin.commerce.read')}
      <!-- Commerce orders -->
      <a
        href="/admin/commerce"
        onclick={() => handleNavigate('/commerce')}
        class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/commerce') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
        title={t('nav.orders')}
      >
        <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M6 2 3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4Z"></path>
          <path d="M3 6h18"></path>
          <path d="M16 10a4 4 0 0 1-8 0"></path>
        </svg>
        {#if sidebar.open}
          <span class="truncate">{t('nav.orders')}</span>
        {/if}
      </a>
      {/if}
    </div>

    <!-- NavDocuments (Matches official nav-documents.tsx) -->
    <div class="mt-3 flex flex-col gap-1">
      {#if sidebar.open}
        <div class="px-2 py-1 flex items-center justify-between text-xs font-medium text-sidebar-foreground/60 tracking-wider">
          <span>{t('nav.documents')}</span>
          {#if can('admin.content.write')}
          <button
            type="button"
            onclick={() => handleNavigate('/content-types')}
            class="size-5 rounded hover:bg-sidebar-accent flex items-center justify-center text-sidebar-foreground/70 hover:text-sidebar-foreground hover:rotate-90 transition-all duration-200 cursor-pointer"
            title={t('nav.contentTypes')}
          >
            <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <line x1="12" y1="5" x2="12" y2="19"></line>
              <line x1="5" y1="12" x2="19" y2="12"></line>
            </svg>
          </button>
          {/if}
        </div>
      {/if}
        <div class="flex flex-col gap-0.5">
          {#each schema.collections as collection}
            <div class="relative group flex items-center" data-dropdown="doc-{collection.id}">
              <a
                href="/admin/content/{collection.id}"
                onclick={() => handleNavigate(`/content/${collection.id}`)}
                class="flex flex-1 items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive(`/content/${collection.id}`) ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
                title={collection.plural || collection.label}
              >
                <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <ellipse cx="12" cy="5" rx="9" ry="3"></ellipse>
                  <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"></path>
                  <path d="M3 12c0 1.66 4 3 9 3s9-1.34 9-3"></path>
                </svg>
                {#if sidebar.open}
                  <span class="truncate">{collection.plural || collection.label}</span>
                {/if}
              </a>

              {#if sidebar.open}
                <button
                  type="button"
                  onclick={(e) => {
                    e.stopPropagation();
                    activeDocDropdown = activeDocDropdown === collection.id ? null : collection.id;
                  }}
                  class="absolute right-1 size-6 rounded-sm opacity-0 group-hover:opacity-100 hover:bg-sidebar-accent flex items-center justify-center text-sidebar-foreground/70 hover:text-sidebar-foreground transition-all duration-150"
                  title={t('nav.options')}
                >
                  <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="1"></circle>
                    <circle cx="19" cy="12" r="1"></circle>
                    <circle cx="5" cy="12" r="1"></circle>
                  </svg>
                </button>

                {#if activeDocDropdown === collection.id}
                  <div class="absolute right-0 top-7 w-28 rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-md z-50 text-xs animate-modal-in">
                    <button
                      type="button"
                      class="w-full flex items-center gap-1.5 px-2 py-1 rounded-sm hover:bg-accent text-left transition-colors"
                      onclick={() => handleNavigate(`/content/${collection.id}`)}
                    >
                      <svg class="size-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2Z"></path></svg>
                      <span>{t('nav.open')}</span>
                    </button>
                    <button
                      type="button"
                      class="w-full flex items-center gap-1.5 px-2 py-1 rounded-sm hover:bg-accent text-left transition-colors"
                      onclick={() => handleNavigate(`/content/${collection.id}/new`)}
                    >
                      <svg class="size-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
                      <span>{t('nav.create')}</span>
                    </button>
                  </div>
                {/if}
              {/if}
            </div>
          {/each}
        </div>
      </div>

    <!-- NavSecondary (Matches official nav-secondary.tsx - pinned to mt-auto) -->
    <div class="mt-auto flex flex-col gap-0.5 pt-4">
      {#if can('admin.site.manage')}
      <a
        href="/admin/site"
        onclick={() => handleNavigate('/site')}
        class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/site') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
        title={t('nav.settings')}
      >
        <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"></path>
          <circle cx="12" cy="12" r="3"></circle>
        </svg>
        {#if sidebar.open}
          <span class="truncate">{t('nav.settings')}</span>
        {/if}
      </a>
      {/if}

      {#if isAdmin && can('admin.users.manage')}
      <a
        href="/admin/users"
        onclick={() => handleNavigate('/users')}
        class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/users') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
        title={t('nav.users')}
      >
        <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"></path>
        </svg>
        {#if sidebar.open}
          <span class="truncate">{t('nav.users')}</span>
        {/if}
      </a>
      <a
        href="/admin/roles"
        onclick={() => handleNavigate('/roles')}
        class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/roles') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
        title="Roles & Permissions"
      >
        <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect width="18" height="18" x="3" y="3" rx="2" />
          <path d="M7 7h10" /><path d="M7 12h10" /><path d="M7 17h10" />
        </svg>
        {#if sidebar.open}
          <span class="truncate">Roles</span>
        {/if}
      </a>
      {/if}

      {#if isAdmin}
      <a
        href="/admin/console"
        onclick={() => handleNavigate('/console')}
        class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/console') || isActive('/orchestrator') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
        title="Mağaza Fabrikası"
      >
        <svg class="size-4 shrink-0 text-amber-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
        </svg>
        {#if sidebar.open}
          <span class="truncate">Mağaza Fabrikası</span>
        {/if}
      </a>
      {/if}

      <a
        href="/admin/system"
        onclick={() => handleNavigate('/system')}
        class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/system') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
        title={t('nav.telemetry')}
      >
        <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="18" y1="20" x2="18" y2="10"></line>
          <line x1="12" y1="20" x2="12" y2="4"></line>
          <line x1="6" y1="20" x2="6" y2="14"></line>
        </svg>
        {#if sidebar.open}
          <span class="truncate">{t('nav.telemetry')}</span>
        {/if}
      </a>
    </div>
  </div>

  <!-- SidebarFooter: NavUser (Matches official nav-user.tsx) -->
  <div class="border-t border-sidebar-border p-2 relative" data-dropdown="user-menu">
    <button
      type="button"
      onclick={() => { showUserMenu = !showUserMenu; }}
      class="flex w-full items-center gap-2.5 rounded-lg p-1.5 text-left text-sm hover:bg-sidebar-accent hover:text-sidebar-accent-foreground transition-colors cursor-pointer"
    >
      <div class="size-8 rounded-lg bg-muted border border-border flex items-center justify-center font-medium text-xs text-foreground uppercase shrink-0">
        {auth.user?.email?.[0] || 'A'}
      </div>
      {#if sidebar.open}
        <div class="grid flex-1 text-left text-xs leading-tight min-w-0">
          <span class="truncate font-medium text-foreground">{auth.user?.email || 'admin'}</span>
          <span class="truncate text-muted-foreground capitalize">{roleLabel(auth.user?.role || 'admin')}</span>
        </div>
        <svg class="size-4 text-muted-foreground ml-auto shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="1"></circle>
          <circle cx="12" cy="5" r="1"></circle>
          <circle cx="12" cy="19" r="1"></circle>
        </svg>
      {/if}
    </button>

    <!-- User Popover Menu -->
    {#if showUserMenu}
      <div class="absolute bottom-12 left-2 right-2 rounded-lg border border-border bg-popover p-1 text-popover-foreground shadow-lg z-50 text-xs">
        <div class="flex items-center gap-2 p-2 border-b border-border/80">
          <div class="size-7 rounded-lg bg-muted flex items-center justify-center font-medium text-xs text-foreground uppercase shrink-0">
            {auth.user?.email?.[0] || 'A'}
          </div>
          <div class="grid flex-1 text-left text-xs leading-tight min-w-0">
            <span class="truncate font-medium text-foreground">{auth.user?.email || 'admin'}</span>
            <span class="truncate text-muted-foreground capitalize">{roleLabel(auth.user?.role || 'admin')}</span>
          </div>
        </div>
        <div class="py-1">
          <a
            href="/admin/account"
            onclick={() => handleNavigate('/account')}
            class="flex items-center gap-2 px-2.5 py-1.5 rounded-md hover:bg-accent text-left transition-colors"
          >
            <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg>
            <span>{t('nav.account')}</span>
          </a>
          <a
            href="/admin/system"
            onclick={() => handleNavigate('/system')}
            class="flex items-center gap-2 px-2.5 py-1.5 rounded-md hover:bg-accent text-left transition-colors"
          >
            <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline></svg>
            <span>{t('nav.telemetry')}</span>
          </a>
          <button
            type="button"
            onclick={() => theme.toggle()}
            class="w-full flex items-center justify-between px-2.5 py-1.5 rounded-md hover:bg-accent text-left transition-colors cursor-pointer"
          >
            <div class="flex items-center gap-2">
              {#if theme.isDark}
                <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="4"></circle><path d="M12 2v2"></path><path d="M12 20v2"></path><path d="m4.93 4.93 1.41 1.41"></path><path d="m17.66 17.66 1.41 1.41"></path><path d="M2 12h2"></path><path d="M20 12h2"></path><path d="m6.34 17.66-1.41 1.41"></path><path d="m19.07 4.93-1.41 1.41"></path></svg>
              {:else}
                <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"></path></svg>
              {/if}
              <span>{t('nav.theme')}</span>
            </div>
            <span class="text-[11px] text-muted-foreground capitalize">{theme.isDark ? t('nav.dark') : t('nav.light')}</span>
          </button>
        </div>
        <div class="border-t border-border/80 pt-1">
          <button
            type="button"
            onclick={() => auth.logout()}
            class="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-md text-destructive hover:bg-destructive/10 text-left transition-colors cursor-pointer"
          >
            <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path><polyline points="16 17 21 12 16 7"></polyline><line x1="21" y1="12" x2="9" y2="12"></line></svg>
            <span>{t('nav.logout')}</span>
          </button>
        </div>
      </div>
    {/if}
  </div>
</aside>
