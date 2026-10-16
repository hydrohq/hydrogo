<script lang="ts">
</script>

<div class="flex h-screen w-full bg-sidebar text-sidebar-foreground overflow-hidden font-sans select-none">
  <!-- Left Sidebar Skeleton -->
  <aside class="hidden md:flex w-64 flex-col bg-sidebar border-r border-sidebar-border shrink-0">
    <!-- Brand Header -->
    <div class="flex h-14 items-center justify-between border-b border-sidebar-border px-4">
      <div class="flex items-center gap-2.5">
        <div class="flex size-7 items-center justify-center rounded-lg bg-primary text-primary-foreground font-bold shadow-xs">
          <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
            <polyline points="2 17 12 22 22 17"></polyline>
            <polyline points="2 12 12 17 22 12"></polyline>
          </svg>
        </div>
        <div class="flex flex-col">
          <span class="text-xs font-bold text-sidebar-foreground tracking-tight">Hydro<span class="text-sky-400">Go</span></span>
          <span class="text-[10px] text-muted-foreground font-mono">Admin Studio</span>
        </div>
      </div>
      <span class="inline-flex items-center rounded-full bg-muted/60 px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground border border-border">v2.0</span>
    </div>

    <!-- Quick Search Placeholder -->
    <div class="p-3">
      <div class="h-8 rounded-lg bg-muted/40 border border-sidebar-border flex items-center px-2.5 gap-2 text-muted-foreground/60 text-xs">
        <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"></circle>
          <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        </svg>
        <span>Search modules...</span>
        <span class="ml-auto text-[10px] font-mono bg-muted/80 px-1 rounded border border-border">⌘K</span>
      </div>
    </div>

    <!-- Navigation Items Skeleton -->
    <div class="flex-1 overflow-y-auto px-3 space-y-5 py-2">
      <!-- Section 1: Overview -->
      <div class="space-y-1">
        <div class="h-3 w-16 bg-muted/50 rounded mb-2 ml-2 animate-pulse"></div>
        <div class="flex items-center gap-2.5 px-2.5 py-2 rounded-lg bg-sidebar-accent text-sidebar-accent-foreground font-medium text-xs">
          <svg class="size-4 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect width="7" height="9" x="3" y="3" rx="1"></rect>
            <rect width="7" height="5" x="14" y="3" rx="1"></rect>
            <rect width="7" height="9" x="14" y="12" rx="1"></rect>
            <rect width="7" height="5" x="3" y="16" rx="1"></rect>
          </svg>
          <div class="h-3 w-20 bg-primary/20 rounded animate-pulse"></div>
        </div>
      </div>

      <!-- Section 2: Content Modules -->
      <div class="space-y-1">
        <div class="h-3 w-14 bg-muted/50 rounded mb-2 ml-2 animate-pulse"></div>
        {#each ['Pages', 'News', 'Media', 'FAQ'] as _}
          <div class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-muted-foreground text-xs hover:bg-muted/30">
            <div class="size-3.5 rounded bg-muted/60 animate-pulse"></div>
            <div class="h-3 w-16 bg-muted/40 rounded animate-pulse"></div>
          </div>
        {/each}
      </div>

      <!-- Section 3: System & Settings -->
      <div class="space-y-1">
        <div class="h-3 w-20 bg-muted/50 rounded mb-2 ml-2 animate-pulse"></div>
        {#each ['Members', 'Users & Roles', 'Site Settings', 'Telemetry'] as _}
          <div class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-muted-foreground text-xs hover:bg-muted/30">
            <div class="size-3.5 rounded bg-muted/60 animate-pulse"></div>
            <div class="h-3 w-20 bg-muted/40 rounded animate-pulse"></div>
          </div>
        {/each}
      </div>
    </div>

    <!-- Bottom User Profile Skeleton -->
    <div class="p-3 border-t border-sidebar-border mt-auto">
      <div class="flex items-center gap-2.5 p-1.5 rounded-lg bg-muted/20 border border-sidebar-border/50">
        <div class="size-7 rounded-full bg-primary/20 animate-pulse"></div>
        <div class="flex flex-col gap-1 flex-1">
          <div class="h-2.5 w-20 bg-muted/70 rounded animate-pulse"></div>
          <div class="h-2 w-14 bg-muted/40 rounded animate-pulse"></div>
        </div>
      </div>
    </div>
  </aside>

  <!-- Right Main Area Skeleton -->
  <div class="relative flex w-full flex-1 flex-col bg-background md:m-2 md:ml-0 md:rounded-xl md:border md:border-border md:shadow-xs overflow-hidden">
    <!-- Top Header Bar -->
    <header class="flex h-14 shrink-0 items-center justify-between border-b border-border bg-card/60 backdrop-blur px-4 lg:px-6">
      <div class="flex items-center gap-3">
        <div class="size-7 rounded-md bg-muted/40 border border-border flex items-center justify-center text-muted-foreground md:hidden">
          <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="3" y1="12" x2="21" y2="12"></line>
            <line x1="3" y1="6" x2="21" y2="6"></line>
            <line x1="3" y1="18" x2="21" y2="18"></line>
          </svg>
        </div>
        <div class="flex items-center gap-1.5 text-xs text-muted-foreground font-medium">
          <span>Admin Studio</span>
          <span>/</span>
          <span class="text-foreground font-semibold">Dashboard</span>
        </div>
      </div>

      <!-- Center Status Pill: Sleek, non-intrusive indicator -->
      <div class="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-primary/5 border border-primary/20 text-xs font-medium text-foreground shadow-xs">
        <span class="relative flex size-2">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-sky-400 opacity-75"></span>
          <span class="relative inline-flex rounded-full size-2 bg-sky-500"></span>
        </span>
        <span class="font-mono text-[11px] tracking-wide uppercase text-muted-foreground">Starting HydroGo Admin…</span>
      </div>

      <!-- Right Action Icons Placeholder -->
      <div class="flex items-center gap-2">
        <div class="size-7 rounded-md bg-muted/40 border border-border animate-pulse"></div>
        <div class="size-7 rounded-md bg-muted/40 border border-border animate-pulse"></div>
        <div class="size-7 rounded-full bg-primary/20 border border-border animate-pulse"></div>
      </div>
    </header>

    <!-- Main Dashboard Scrollable Content -->
    <main class="flex-1 overflow-y-auto bg-background p-4 lg:p-6 space-y-6">
      <!-- Welcome Header -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-2">
        <div class="space-y-1.5">
          <div class="h-6 w-48 bg-muted/70 rounded-md animate-pulse"></div>
          <div class="h-3.5 w-64 bg-muted/40 rounded animate-pulse"></div>
        </div>
        <div class="flex items-center gap-2">
          <div class="h-8 w-24 bg-muted/40 rounded-lg border border-border animate-pulse"></div>
          <div class="h-8 w-28 bg-primary/20 rounded-lg animate-pulse"></div>
        </div>
      </div>

      <!-- 4 Section Metric Cards -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {#each [
          { title: 'Published Pages', icon: 'pages' },
          { title: 'News Articles', icon: 'news' },
          { title: 'Active Members', icon: 'members' },
          { title: 'Cluster Health', icon: 'health' }
        ] as card}
          <div class="rounded-xl border border-border bg-card p-5 shadow-xs space-y-3 relative overflow-hidden">
            <div class="flex items-center justify-between">
              <span class="text-xs font-medium text-muted-foreground">{card.title}</span>
              <div class="size-6 rounded-md bg-muted/50 flex items-center justify-center animate-pulse"></div>
            </div>
            <div class="space-y-1">
              <div class="h-7 w-16 bg-foreground/15 rounded-md animate-pulse"></div>
              <div class="h-3 w-28 bg-muted/40 rounded animate-pulse"></div>
            </div>
          </div>
        {/each}
      </div>

      <!-- Chart Card Skeleton -->
      <div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-4">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-border/60 pb-4">
          <div class="space-y-1">
            <div class="h-4 w-36 bg-foreground/20 rounded animate-pulse"></div>
            <div class="h-3 w-56 bg-muted/40 rounded animate-pulse"></div>
          </div>
          <div class="flex items-center gap-1.5 bg-muted/40 p-1 rounded-lg border border-border">
            <div class="h-6 w-16 bg-muted/80 rounded animate-pulse"></div>
            <div class="h-6 w-16 bg-transparent rounded"></div>
            <div class="h-6 w-16 bg-transparent rounded"></div>
          </div>
        </div>

        <!-- Chart Grid Graphic -->
        <div class="h-48 w-full flex flex-col justify-between py-2 relative overflow-hidden">
          <div class="border-b border-dashed border-border/70 w-full"></div>
          <div class="border-b border-dashed border-border/70 w-full"></div>
          <div class="border-b border-dashed border-border/70 w-full"></div>
          <div class="border-b border-border w-full"></div>
          <!-- Shimmering Line / Bar graphic -->
          <div class="absolute inset-x-0 bottom-2 flex items-end gap-3 h-32 px-4 opacity-40">
            {#each [40, 65, 50, 80, 60, 95, 75, 90, 85, 70, 80, 100] as h}
              <div
                class="flex-1 rounded-t bg-gradient-to-t from-primary/30 to-primary/60 animate-pulse"
                style="height: {h}%"
              ></div>
            {/each}
          </div>
        </div>
      </div>

      <!-- Data Table Skeleton -->
      <div class="rounded-xl border border-border bg-card shadow-xs overflow-hidden">
        <!-- Table Toolbar -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 border-b border-border">
          <div class="flex items-center gap-2">
            <div class="h-8 w-48 bg-muted/40 rounded-lg border border-border animate-pulse"></div>
            <div class="h-8 w-24 bg-muted/30 rounded-lg border border-border animate-pulse"></div>
          </div>
          <div class="flex items-center gap-2">
            <div class="h-8 w-20 bg-muted/30 rounded-lg border border-border animate-pulse"></div>
          </div>
        </div>

        <!-- Table Rows -->
        <div class="divide-y divide-border">
          {#each [1, 2, 3, 4] as _}
            <div class="flex items-center justify-between p-3.5 px-4 gap-4">
              <div class="flex items-center gap-3 flex-1">
                <div class="size-4 rounded bg-muted/50 animate-pulse"></div>
                <div class="space-y-1.5 flex-1 max-w-sm">
                  <div class="h-3.5 w-3/4 bg-foreground/15 rounded animate-pulse"></div>
                  <div class="h-2.5 w-1/2 bg-muted/40 rounded animate-pulse"></div>
                </div>
              </div>
              <div class="hidden sm:block h-5 w-16 bg-muted/50 rounded-full animate-pulse"></div>
              <div class="hidden md:block h-3.5 w-24 bg-muted/40 rounded animate-pulse"></div>
              <div class="size-6 rounded bg-muted/40 animate-pulse"></div>
            </div>
          {/each}
        </div>
      </div>
    </main>
  </div>
</div>
