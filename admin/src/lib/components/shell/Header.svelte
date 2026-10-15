<script lang="ts">
  import { router } from '$lib/router.svelte';
  import { schema } from '$lib/stores/schema.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';

  const path = $derived(router.path);

  const breadcrumbs = $derived.by(() => {
    if (path === '/' || path === '/dashboard') {
      return [{ label: 'Dashboard', href: '#/dashboard' }];
    }
    if (path.startsWith('/content/')) {
      const parts = path.split('/');
      const type = parts[2];
      const collection = schema.getCollection(type);
      const name = collection?.label || collection?.plural || type;
      const list = [{ label: 'Content', href: '#/dashboard' }];
      if (parts[3] === 'new') {
        list.push({ label: name, href: `#/content/${type}` });
        list.push({ label: `New ${collection?.singular || 'Item'}`, href: path });
      } else if (parts[3]) {
        list.push({ label: name, href: `#/content/${type}` });
        list.push({ label: 'Edit', href: path });
      } else {
        list.push({ label: name, href: `#/content/${type}` });
      }
      return list;
    }
    if (path.startsWith('/pages')) return [{ label: 'Modules', href: '#/dashboard' }, { label: 'Pages', href: '#/pages' }];
    if (path.startsWith('/news')) return [{ label: 'Modules', href: '#/dashboard' }, { label: 'News', href: '#/news' }];
    if (path.startsWith('/media')) return [{ label: 'Modules', href: '#/dashboard' }, { label: 'Media Library', href: '#/media' }];
    if (path.startsWith('/members')) return [{ label: 'Modules', href: '#/dashboard' }, { label: 'Members', href: '#/members' }];
    if (path.startsWith('/site')) return [{ label: 'Settings', href: '#/dashboard' }, { label: 'Site Settings', href: '#/site' }];
    if (path.startsWith('/users')) return [{ label: 'Administration', href: '#/dashboard' }, { label: 'Users & Roles', href: '#/users' }];
    if (path.startsWith('/audit')) return [{ label: 'Administration', href: '#/dashboard' }, { label: 'Audit Trail', href: '#/audit' }];
    if (path.startsWith('/system')) return [{ label: 'Administration', href: '#/dashboard' }, { label: 'System Metrics', href: '#/system' }];
    return [{ label: 'Admin', href: '#/dashboard' }];
  });
</script>

<header class="h-16 px-6 border-b border-border bg-card/80 backdrop-blur-md flex items-center justify-between flex-shrink-0 z-10">
  <!-- Left: Breadcrumb Navigation -->
  <nav class="flex items-center gap-1.5 text-xs text-muted-foreground" aria-label="Breadcrumb">
    {#each breadcrumbs as crumb, i}
      {#if i > 0}
        <svg class="w-3.5 h-3.5 text-muted-foreground/60 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="9 18 15 12 9 6"></polyline>
        </svg>
      {/if}
      {#if i === breadcrumbs.length - 1}
        <span class="font-semibold text-foreground tracking-tight text-sm">{crumb.label}</span>
      {:else}
        <a href={crumb.href} class="hover:text-foreground transition-colors">{crumb.label}</a>
      {/if}
    {/each}
  </nav>

  <!-- Right: Search & Quick Actions -->
  <div class="flex items-center gap-3">
    <!-- Quick Search Pill -->
    <div class="hidden md:flex items-center gap-2 px-3 py-1.5 rounded-lg border border-input bg-background/60 text-xs text-muted-foreground shadow-sm">
      <svg class="w-3.5 h-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="11" cy="11" r="8"></circle>
        <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
      </svg>
      <span>Quick search...</span>
      <kbd class="ml-2 px-1.5 py-0.5 text-[10px] font-mono bg-muted text-muted-foreground border border-border/80 rounded">⌘K</kbd>
    </div>

    <!-- Locale selector indicator -->
    {#if schema.locales.length > 1}
      <Badge variant="outline" class="font-mono text-[11px] gap-1 px-2.5 py-1">
        <span class="text-muted-foreground">LOCALE:</span>
        <span class="font-bold text-primary uppercase">{schema.defaultLocale}</span>
      </Badge>
    {/if}

    <!-- Live Site Link -->
    <a
      href="/"
      target="_blank"
      rel="noopener noreferrer"
      class="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground hover:text-foreground px-3 py-1.5 rounded-md border border-input bg-background/50 hover:bg-accent transition-all shadow-sm"
    >
      <span>Live Site</span>
      <svg class="w-3 h-3 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
        <polyline points="15 3 21 3 21 9"></polyline>
        <line x1="10" y1="14" x2="21" y2="3"></line>
      </svg>
    </a>

    <!-- Status dot -->
    <div class="flex items-center gap-1.5 pl-2 border-l border-border/60" title="API Status: Connected">
      <span class="relative flex h-2 w-2">
        <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
        <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
      </span>
      <span class="text-[11px] text-muted-foreground font-medium hidden sm:inline">Online</span>
    </div>
  </div>
</header>
