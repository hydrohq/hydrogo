<script lang="ts">
  import { auth } from '$lib/stores/auth.svelte';
  import { schema } from '$lib/stores/schema.svelte';
  import { router } from '$lib/router.svelte';

  const currentPath = $derived(router.path);

  function isActive(target: string): boolean {
    if (target === '/' || target === '/dashboard') {
      return currentPath === '/' || currentPath === '/dashboard';
    }
    return currentPath.startsWith(target);
  }
</script>

<aside class="w-64 flex-shrink-0 bg-card border-r border-border flex flex-col h-full select-none">
  <!-- Brand Header -->
  <div class="h-16 px-5 flex items-center justify-between border-b border-border/80">
    <div class="flex items-center gap-3">
      <div class="w-9 h-9 rounded-lg bg-primary/15 border border-primary/30 flex items-center justify-center text-primary shadow-sm shadow-blue-500/10">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"></path>
        </svg>
      </div>
      <div class="flex flex-col">
        <span class="text-sm font-bold tracking-tight text-foreground flex items-center gap-1">
          Hydro<span class="text-primary font-black">Go</span>
        </span>
        <span class="text-[10px] text-muted-foreground font-mono tracking-wider">CMS STUDIO</span>
      </div>
    </div>
    <span class="text-[10px] bg-secondary text-secondary-foreground px-2 py-0.5 rounded font-mono border border-border/60">
      v2.0
    </span>
  </div>

  <!-- Navigation Links -->
  <div class="flex-1 overflow-y-auto px-3 py-4 space-y-6">
    <!-- Overview -->
    <div>
      <div class="px-3 mb-2 text-[11px] font-semibold tracking-wider text-muted-foreground uppercase">
        Overview
      </div>
      <nav class="space-y-1">
        <a
          href="#/dashboard"
          class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all {isActive('/dashboard') ? 'bg-primary/10 text-primary font-semibold border-l-2 border-primary pl-2.5' : 'text-muted-foreground hover:text-foreground hover:bg-accent/60'}"
        >
          <svg class="w-4 h-4 text-current" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect width="7" height="9" x="3" y="3" rx="1"></rect>
            <rect width="7" height="5" x="14" y="3" rx="1"></rect>
            <rect width="7" height="9" x="14" y="12" rx="1"></rect>
            <rect width="7" height="5" x="3" y="16" rx="1"></rect>
          </svg>
          <span>Dashboard</span>
        </a>
      </nav>
    </div>

    <!-- Schema Collections -->
    {#if schema.resources.length > 0}
      <div>
        <div class="px-3 mb-2 text-[11px] font-semibold tracking-wider text-muted-foreground uppercase flex items-center justify-between">
          <span>Content</span>
          <span class="text-[10px] bg-secondary text-muted-foreground px-1.5 py-0.5 rounded-full font-mono border border-border/60">
            {schema.resources.length}
          </span>
        </div>
        <nav class="space-y-1">
          {#each schema.resources as res (res.id)}
            <a
              href={`#/content/${res.id}`}
              class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all {isActive(`/content/${res.id}`) ? 'bg-primary/10 text-primary font-semibold border-l-2 border-primary pl-2.5' : 'text-muted-foreground hover:text-foreground hover:bg-accent/60'}"
            >
              <span class="text-sm">{res.icon || '📦'}</span>
              <span class="truncate">{res.label || res.plural}</span>
            </a>
          {/each}
        </nav>
      </div>
    {/if}

    <!-- Core CMS Modules -->
    <div>
      <div class="px-3 mb-2 text-[11px] font-semibold tracking-wider text-muted-foreground uppercase">
        Modules
      </div>
      <nav class="space-y-1">
        {#if schema.modules['pages'] !== false}
          <a
            href="#/pages"
            class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all {isActive('/pages') ? 'bg-primary/10 text-primary font-semibold border-l-2 border-primary pl-2.5' : 'text-muted-foreground hover:text-foreground hover:bg-accent/60'}"
          >
            <svg class="w-4 h-4 text-current" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path>
              <polyline points="14 2 14 8 20 8"></polyline>
              <line x1="16" y1="13" x2="8" y2="13"></line>
              <line x1="16" y1="17" x2="8" y2="17"></line>
              <line x1="10" y1="9" x2="8" y2="9"></line>
            </svg>
            <span>Pages</span>
          </a>
        {/if}
        {#if schema.modules['news'] !== false}
          <a
            href="#/news"
            class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all {isActive('/news') ? 'bg-primary/10 text-primary font-semibold border-l-2 border-primary pl-2.5' : 'text-muted-foreground hover:text-foreground hover:bg-accent/60'}"
          >
            <svg class="w-4 h-4 text-current" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"></path>
              <path d="M18 14h-8"></path>
              <path d="M15 18h-5"></path>
              <path d="M10 6h8v4h-8V6Z"></path>
            </svg>
            <span>News & Articles</span>
          </a>
        {/if}
        {#if schema.modules['media'] !== false}
          <a
            href="#/media"
            class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all {isActive('/media') ? 'bg-primary/10 text-primary font-semibold border-l-2 border-primary pl-2.5' : 'text-muted-foreground hover:text-foreground hover:bg-accent/60'}"
          >
            <svg class="w-4 h-4 text-current" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect width="18" height="18" x="3" y="3" rx="2" ry="2"></rect>
              <circle cx="9" cy="9" r="2"></circle>
              <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"></path>
            </svg>
            <span>Media Library</span>
          </a>
        {/if}
        {#if schema.modules['members'] !== false}
          <a
            href="#/members"
            class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all {isActive('/members') ? 'bg-primary/10 text-primary font-semibold border-l-2 border-primary pl-2.5' : 'text-muted-foreground hover:text-foreground hover:bg-accent/60'}"
          >
            <svg class="w-4 h-4 text-current" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"></path>
              <circle cx="9" cy="7" r="4"></circle>
              <path d="M22 21v-2a4 4 0 0 0-3-3.87"></path>
              <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
            </svg>
            <span>Members</span>
          </a>
        {/if}
        {#if schema.modules['site'] !== false}
          <a
            href="#/site"
            class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all {isActive('/site') ? 'bg-primary/10 text-primary font-semibold border-l-2 border-primary pl-2.5' : 'text-muted-foreground hover:text-foreground hover:bg-accent/60'}"
          >
            <svg class="w-4 h-4 text-current" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="3"></circle>
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
            </svg>
            <span>Site Settings</span>
          </a>
        {/if}
      </nav>
    </div>

    <!-- Management & System -->
    {#if auth.user?.role === 'admin'}
      <div>
        <div class="px-3 mb-2 text-[11px] font-semibold tracking-wider text-muted-foreground uppercase">
          Administration
        </div>
        <nav class="space-y-1">
          <a
            href="#/users"
            class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all {isActive('/users') ? 'bg-primary/10 text-primary font-semibold border-l-2 border-primary pl-2.5' : 'text-muted-foreground hover:text-foreground hover:bg-accent/60'}"
          >
            <svg class="w-4 h-4 text-current" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"></path>
            </svg>
            <span>Users & Access</span>
          </a>
          <a
            href="#/audit"
            class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all {isActive('/audit') ? 'bg-primary/10 text-primary font-semibold border-l-2 border-primary pl-2.5' : 'text-muted-foreground hover:text-foreground hover:bg-accent/60'}"
          >
            <svg class="w-4 h-4 text-current" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 8v4l3 3"></path>
              <circle cx="12" cy="12" r="10"></circle>
            </svg>
            <span>Audit Trail</span>
          </a>
          <a
            href="#/system"
            class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all {isActive('/system') ? 'bg-primary/10 text-primary font-semibold border-l-2 border-primary pl-2.5' : 'text-muted-foreground hover:text-foreground hover:bg-accent/60'}"
          >
            <svg class="w-4 h-4 text-current" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M22 12h-4l-3 9L9 3l-3 9H2"></path>
            </svg>
            <span>System Telemetry</span>
          </a>
        </nav>
      </div>
    {/if}
  </div>

  <!-- User Profile Footer -->
  <div class="p-3 border-t border-border/80 bg-card/60">
    <div class="flex items-center justify-between p-2 rounded-lg hover:bg-accent/50 transition-colors">
      <div class="flex items-center gap-2.5 min-w-0">
        <div class="w-8 h-8 rounded-full bg-primary/20 border border-primary/40 flex items-center justify-center font-bold text-xs text-primary flex-shrink-0">
          {auth.user?.email?.[0]?.toUpperCase() || 'A'}
        </div>
        <div class="min-w-0">
          <p class="text-xs font-semibold text-foreground truncate">{auth.user?.email || 'User'}</p>
          <span class="text-[10px] text-muted-foreground capitalize tracking-wider font-mono">{auth.user?.role || 'editor'}</span>
        </div>
      </div>
      <button
        type="button"
        onclick={() => auth.logout()}
        class="text-muted-foreground hover:text-destructive p-1.5 rounded-md hover:bg-accent transition-colors cursor-pointer"
        title="Log out"
        aria-label="Log out"
      >
        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
          <polyline points="16 17 21 12 16 7"></polyline>
          <line x1="21" y1="12" x2="9" y2="12"></line>
        </svg>
      </button>
    </div>
  </div>
</aside>
