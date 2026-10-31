<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import BrandLogo from '$lib/components/ui/BrandLogo.svelte';
  import ContentSkeleton, { type SkeletonKind } from './ContentSkeleton.svelte';

  interface Props {
    path?: string;
  }

  let { path }: Props = $props();

  const currentPath = $derived.by(() => {
    if (path) return path;
    if (typeof window !== 'undefined') {
      const p = window.location.pathname;
      return p.replace(/^\/admin/, '') || '/';
    }
    return '/';
  });

  const isConsoleMode = $derived.by(() => {
    if (typeof window !== 'undefined') {
      const host = window.location.hostname.toLowerCase();
      if (host.startsWith('console.') || host.includes('console.example.com')) return true;
    }
    return currentPath.startsWith('/console') || currentPath.startsWith('/orchestrator');
  });

  const kind = $derived.by<SkeletonKind>(() => {
    const p = currentPath.toLowerCase();
    if (p === '/login' || p.startsWith('/reset-password') || p.startsWith('/accept-invite')) {
      return 'auth';
    }
    if (isConsoleMode) {
      return 'console';
    }
    if (p === '/' || p === '/dashboard') {
      return 'dashboard';
    }
    if (p.startsWith('/media')) {
      return 'media';
    }
    if (p.startsWith('/site') || p.startsWith('/account')) {
      return 'settings';
    }
    if (
      (p.startsWith('/pages/') && p !== '/pages') ||
      (p.startsWith('/news/') && p !== '/news') ||
      p.includes('/new') ||
      p.includes('/edit')
    ) {
      return 'editor';
    }
    return 'table';
  });

  const pageTitle = $derived.by(() => {
    if (isConsoleMode) return 'Mağaza Fabrikası';
    const p = currentPath.toLowerCase();
    if (p.startsWith('/pages')) return t('nav.pages');
    if (p.startsWith('/news')) return t('nav.news');
    if (p.startsWith('/media')) return t('nav.media');
    if (p.startsWith('/members')) return t('nav.members');
    if (p.startsWith('/users')) return t('nav.users');
    if (p.startsWith('/site')) return t('title.siteSettings');
    if (p.startsWith('/content-types')) return t('nav.contentTypes');
    return t('nav.dashboard');
  });
</script>

{#if kind === 'auth'}
  <!-- Clean Auth Screen Skeleton without App Chrome -->
  <ContentSkeleton kind="auth" />
{:else}
  <!-- Full App Chrome Skeleton (Adaptive: Console vs CMS) -->
  <div class="flex h-screen w-full bg-sidebar text-sidebar-foreground overflow-hidden font-sans select-none">
    <!-- Left Sidebar Skeleton -->
    <aside class="hidden md:flex w-64 flex-col bg-sidebar border-r border-sidebar-border shrink-0">
      <!-- SidebarHeader -->
      <div class="flex h-12 shrink-0 items-center justify-between border-b border-sidebar-border px-3">
        <div class="flex w-full items-center gap-2 rounded-md p-1.5 text-left text-sm font-semibold">
          <BrandLogo size="md" showText={true} />
          {#if isConsoleMode}
            <span class="ml-auto text-[10px] font-medium tracking-wide px-1.5 py-0.5 rounded border border-sidebar-border bg-sidebar-accent text-muted-foreground">
              Console
            </span>
          {/if}
        </div>
      </div>

      {#if isConsoleMode}
        <!-- Console Sidebar Navigation -->
        <div class="flex-1 overflow-y-auto px-2 py-3 space-y-1">
          <div class="flex items-center gap-2.5 px-2.5 py-2 rounded-md bg-sidebar-accent text-sidebar-accent-foreground font-medium text-sm">
            <svg class="size-4 shrink-0 text-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect width="18" height="18" x="3" y="3" rx="2" />
              <path d="M3 9h18" />
              <path d="M9 21V9" />
            </svg>
            <span class="truncate">Mağaza Fabrikası</span>
          </div>

          <div class="flex items-center gap-2.5 px-2.5 py-2 rounded-md text-sidebar-foreground/70 text-sm">
            <svg class="size-4 shrink-0 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
              <circle cx="9" cy="7" r="4" />
            </svg>
            <span class="truncate">Yöneticiler</span>
          </div>
        </div>
      {:else}
        <!-- Standard CMS Sidebar Navigation -->
        <div class="px-2 pt-2">
          <div class="h-8 rounded-md bg-muted/40 border border-sidebar-border flex items-center px-2.5 gap-2 text-muted-foreground/60 text-xs">
            <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"></circle>
              <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
            </svg>
            <span>{t('skeleton.search')}</span>
            <span class="ml-auto text-[10px] font-mono bg-muted/80 px-1 rounded border border-border">Ctrl K</span>
          </div>
        </div>

        <div class="flex-1 overflow-y-auto px-2 py-3 space-y-4">
          <!-- Overview -->
          <div class="space-y-1">
            <div class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-xs {currentPath === '/' || currentPath === '/dashboard' ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium' : 'text-muted-foreground'}">
              <div class="size-3.5 rounded bg-muted/50"></div>
              <span>{t('nav.dashboard')}</span>
            </div>
          </div>

          <!-- Documents -->
          <div class="space-y-1">
            <div class="h-2.5 w-16 bg-muted/50 rounded mb-2 ml-2"></div>
            {#each [
              { name: t('nav.pages'), active: currentPath.startsWith('/pages') },
              { name: t('nav.news'), active: currentPath.startsWith('/news') },
              { name: t('nav.media'), active: currentPath.startsWith('/media') },
              { name: 'FAQ', active: currentPath.startsWith('/content/faq') }
            ] as item}
              <div class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-xs {item.active ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium' : 'text-muted-foreground'}">
                <div class="size-3.5 rounded bg-muted/50"></div>
                <span>{item.name}</span>
              </div>
            {/each}
          </div>

          <!-- System -->
          <div class="space-y-1">
            <div class="h-2.5 w-14 bg-muted/50 rounded mb-2 ml-2"></div>
            {#each [
              { name: t('nav.members'), active: currentPath.startsWith('/members') },
              { name: t('nav.users'), active: currentPath.startsWith('/users') || currentPath.startsWith('/roles') },
              { name: t('title.siteSettings'), active: currentPath.startsWith('/site') },
              { name: t('nav.telemetry'), active: currentPath.startsWith('/system') || currentPath.startsWith('/audit') }
            ] as item}
              <div class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-xs {item.active ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium' : 'text-muted-foreground'}">
                <div class="size-3.5 rounded bg-muted/50"></div>
                <span>{item.name}</span>
              </div>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Bottom User Profile Footer -->
      <div class="p-2 border-t border-sidebar-border mt-auto">
        <div class="flex items-center gap-2.5 p-1.5 rounded-lg bg-muted/20 border border-sidebar-border/50">
          <div class="size-7 rounded-full bg-muted/70"></div>
          <div class="flex flex-col gap-1 flex-1">
            <div class="h-2.5 w-20 bg-muted/60 rounded"></div>
            <div class="h-2 w-12 bg-muted/40 rounded"></div>
          </div>
        </div>
      </div>
    </aside>

    <!-- Right Main Area Skeleton -->
    <div class="relative flex w-full flex-1 flex-col bg-background md:m-2 md:ml-0 md:rounded-xl md:border md:border-border md:shadow-xs overflow-hidden">
      <!-- Top Header Bar -->
      <header class="flex h-12 shrink-0 items-center gap-2 border-b border-border bg-background transition-[width,height] ease-linear">
        <div class="flex w-full items-center justify-between gap-1 px-4 lg:gap-2 lg:px-6">
          <div class="flex items-center min-w-0">
            <div class="-ml-1 flex size-7 items-center justify-center rounded-md text-muted-foreground">
              <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect width="18" height="18" x="3" y="3" rx="2"></rect>
                <path d="M9 3v18"></path>
              </svg>
            </div>
            <div class="mx-2 h-4 w-px bg-border"></div>
            <h1 class="text-base font-medium text-foreground truncate">{pageTitle}</h1>
          </div>

          <div class="flex items-center gap-3">
            <div class="hidden sm:flex items-center gap-2 h-8 w-44 lg:w-56 rounded-md bg-muted/30 border border-input px-2.5 text-xs text-muted-foreground/60">
              <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"></circle>
                <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
              </svg>
              <span class="truncate flex-1 text-left">{t('header.search')}</span>
              <kbd class="px-1.5 py-0.5 text-[10px] font-mono bg-background text-muted-foreground border border-border rounded">Ctrl K</kbd>
            </div>

            <!-- Status Indicator: Starting HydroGo Admin... -->
            <div class="flex items-center gap-2 pl-2 border-l border-border select-none">
              <span class="relative flex size-2">
                <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-sky-400 opacity-75"></span>
                <span class="relative inline-flex rounded-full size-2 bg-sky-500"></span>
              </span>
              <span class="text-xs text-muted-foreground font-medium">{t('skeleton.starting')}</span>
            </div>
          </div>
        </div>
      </header>

      <!-- Main Content Area: Content-aware Skeleton -->
      <main class="flex-1 overflow-y-auto bg-background min-w-0">
        <ContentSkeleton {kind} />
      </main>
    </div>
  </div>
{/if}
