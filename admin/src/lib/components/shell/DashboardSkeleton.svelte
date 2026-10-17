<script lang="ts">
  import BrandLogo from '$lib/components/ui/BrandLogo.svelte';
</script>

<div class="flex h-screen w-full bg-sidebar text-sidebar-foreground overflow-hidden font-sans select-none">
  <!-- Left Sidebar Skeleton (Exact match with Sidebar.svelte) -->
  <aside class="hidden md:flex w-64 flex-col bg-sidebar border-r border-sidebar-border shrink-0">
    <!-- SidebarHeader: Brand (h-12 border-b border-sidebar-border px-3) -->
    <div class="flex h-12 shrink-0 items-center border-b border-sidebar-border px-3">
      <div class="flex w-full items-center gap-2.5 rounded-md p-1.5 text-left text-sm font-semibold">
        <BrandLogo size="md" showText={true} />
      </div>
    </div>

    <!-- Quick Search Placeholder (p-2) -->
    <div class="px-2 pt-2">
      <div class="h-8 rounded-md bg-muted/40 border border-sidebar-border flex items-center px-2.5 gap-2 text-muted-foreground/60 text-xs">
        <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"></circle>
          <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        </svg>
        <span>Search modules...</span>
        <span class="ml-auto text-[10px] font-mono bg-muted/80 px-1 rounded border border-border">Ctrl K</span>
      </div>
    </div>

    <!-- Navigation Items Skeleton -->
    <div class="flex-1 overflow-y-auto px-2 py-3 space-y-4">
      <!-- Section 1: Overview -->
      <div class="space-y-1">
        <div class="h-2.5 w-14 bg-muted/50 rounded mb-2 ml-2"></div>
        <div class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg bg-sidebar-accent text-sidebar-accent-foreground font-medium text-xs">
          <svg class="size-4 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect width="7" height="9" x="3" y="3" rx="1"></rect>
            <rect width="7" height="5" x="14" y="3" rx="1"></rect>
            <rect width="7" height="9" x="14" y="12" rx="1"></rect>
            <rect width="7" height="5" x="3" y="16" rx="1"></rect>
          </svg>
          <span class="text-xs font-medium text-foreground">Dashboard</span>
        </div>
      </div>

      <!-- Section 2: Documents -->
      <div class="space-y-1">
        <div class="h-2.5 w-16 bg-muted/50 rounded mb-2 ml-2"></div>
        {#each ['Pages', 'News', 'Media', 'FAQ'] as name}
          <div class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-muted-foreground text-xs hover:bg-muted/30">
            <div class="size-3.5 rounded bg-muted/50"></div>
            <span class="text-xs text-muted-foreground">{name}</span>
          </div>
        {/each}
      </div>

      <!-- Section 3: System -->
      <div class="space-y-1">
        <div class="h-2.5 w-14 bg-muted/50 rounded mb-2 ml-2"></div>
        {#each ['Members', 'Users & Roles', 'Site Settings', 'Telemetry'] as name}
          <div class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-muted-foreground text-xs hover:bg-muted/30">
            <div class="size-3.5 rounded bg-muted/50"></div>
            <span class="text-xs text-muted-foreground">{name}</span>
          </div>
        {/each}
      </div>
    </div>

    <!-- Bottom User Profile Skeleton (Exact match with Sidebar.svelte footer) -->
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

  <!-- Right Main Area Skeleton (Exact match with AppLayout.svelte) -->
  <div class="relative flex w-full flex-1 flex-col bg-background md:m-2 md:ml-0 md:rounded-xl md:border md:border-border md:shadow-xs overflow-hidden">
    <!-- Top Header Bar (Exact match with Header.svelte: h-12) -->
    <header class="flex h-12 shrink-0 items-center gap-2 border-b border-border bg-background transition-[width,height] ease-linear">
      <div class="flex w-full items-center justify-between gap-1 px-4 lg:gap-2 lg:px-6">
        <!-- Left: SidebarToggle, Vertical Separator, Page Title -->
        <div class="flex items-center min-w-0">
          <div class="-ml-1 flex size-7 items-center justify-center rounded-md text-muted-foreground">
            <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect width="18" height="18" x="3" y="3" rx="2"></rect>
              <path d="M9 3v18"></path>
            </svg>
          </div>

          <div class="mx-2 h-4 w-px bg-border"></div>

          <h1 class="text-base font-medium text-foreground truncate">Dashboard</h1>
        </div>

        <!-- Right: Status Indicator (Clean, restrained, exact shadcn aesthetic) -->
        <div class="flex items-center gap-3">
          <div class="hidden sm:flex items-center gap-2 h-8 w-44 lg:w-56 rounded-md bg-muted/30 border border-input px-2.5 text-xs text-muted-foreground/60">
            <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"></circle>
              <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
            </svg>
            <span class="truncate flex-1 text-left">Search...</span>
            <kbd class="px-1.5 py-0.5 text-[10px] font-mono bg-background text-muted-foreground border border-border rounded">Ctrl K</kbd>
          </div>

          <div class="size-8 rounded-md border border-input bg-background flex items-center justify-center text-muted-foreground/60">
            <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="4"></circle>
              <path d="M12 2v2"></path>
            </svg>
          </div>

          <!-- Status Indicator: Starting HydroGo Admin... -->
          <div class="flex items-center gap-2 pl-2 border-l border-border select-none">
            <span class="relative flex size-2">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-sky-400 opacity-75"></span>
              <span class="relative inline-flex rounded-full size-2 bg-sky-500"></span>
            </span>
            <span class="text-xs text-muted-foreground font-medium">Starting HydroGo Admin…</span>
          </div>
        </div>
      </div>
    </header>

    <!-- Main Dashboard Scrollable Content (Exact match with DashboardView.svelte) -->
    <main class="flex-1 overflow-y-auto bg-background">
      <div class="flex flex-1 flex-col">
        <div class="@container/main flex flex-1 flex-col gap-2">
          <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
            <!-- 4 Section Metric Cards (Exact match with SectionCards.svelte) -->
            <div class="grid grid-cols-1 gap-4 px-4 @xl/main:grid-cols-2 @5xl/main:grid-cols-4 lg:px-6">
              {#each [
                { title: 'Published Pages', sub: 'Site routes & layouts' },
                { title: 'News Articles', sub: 'Editorial publications' },
                { title: 'Active Members', sub: 'Verified subscriber base' },
                { title: 'Requests & Latency', sub: 'System runtime online' }
              ] as card}
                <div class="rounded-xl border border-border bg-gradient-to-t from-primary/5 to-card dark:bg-card p-6 shadow-xs text-card-foreground @container/card relative">
                  <div class="relative flex flex-col space-y-1.5">
                    <span class="text-sm font-normal text-muted-foreground">{card.title}</span>
                    <div class="h-8 w-16 bg-muted/60 rounded-md animate-pulse"></div>
                    <div class="absolute right-0 top-0">
                      <span class="inline-flex items-center gap-1 rounded-lg border border-border bg-background px-2 py-0.5 text-xs font-normal text-muted-foreground shadow-xs">
                        <span class="size-1.5 rounded-full bg-sky-500"></span>
                        Sync
                      </span>
                    </div>
                  </div>
                  <div class="flex flex-col items-start gap-1 text-sm pt-4">
                    <span class="text-xs text-muted-foreground">{card.sub}</span>
                  </div>
                </div>
              {/each}
            </div>

            <!-- Chart Card Skeleton (Exact match with ChartAreaInteractive.svelte: mx-4 lg:mx-6) -->
            <div class="mx-4 lg:mx-6 rounded-xl border border-border bg-card text-card-foreground shadow-xs p-6 space-y-4">
              <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-border/60 pb-4">
                <div class="space-y-1">
                  <span class="text-sm font-semibold text-foreground">Traffic & Analytics</span>
                  <p class="text-xs text-muted-foreground">Showing request volume and visitor activity over time</p>
                </div>
                <div class="flex items-center gap-1 bg-muted/40 p-1 rounded-lg border border-border text-xs text-muted-foreground">
                  <span class="px-2.5 py-1 rounded-md bg-background text-foreground shadow-xs font-medium">30 Days</span>
                  <span class="px-2.5 py-1">7 Days</span>
                  <span class="px-2.5 py-1">90 Days</span>
                </div>
              </div>

              <!-- Refined Chart Area Line Graphic -->
              <div class="h-44 w-full flex flex-col justify-between py-2 relative overflow-hidden">
                <div class="border-b border-dashed border-border/50 w-full"></div>
                <div class="border-b border-dashed border-border/50 w-full"></div>
                <div class="border-b border-dashed border-border/50 w-full"></div>
                <div class="border-b border-border w-full"></div>

                <!-- Subtle smooth area graphic -->
                <svg class="absolute inset-0 h-full w-full opacity-25" preserveAspectRatio="none" viewBox="0 0 100 40">
                  <path
                    d="M0,35 Q15,10 30,22 T60,12 T85,18 T100,5 L100,40 L0,40 Z"
                    fill="currentColor"
                    class="text-sky-500"
                  ></path>
                </svg>
              </div>
            </div>

            <!-- Data Table Skeleton (Exact match with DataTable.svelte: mx-4 lg:mx-6) -->
            <div class="mx-4 lg:mx-6 rounded-xl border border-border bg-card text-card-foreground shadow-xs overflow-hidden">
              <div class="flex items-center justify-between p-4 border-b border-border">
                <div class="h-8 w-44 bg-muted/40 rounded-lg border border-border"></div>
                <div class="h-8 w-24 bg-muted/30 rounded-lg border border-border"></div>
              </div>
              <div class="divide-y divide-border">
                {#each [1, 2, 3] as _}
                  <div class="flex items-center justify-between p-3.5 px-4 gap-4">
                    <div class="flex items-center gap-3 flex-1">
                      <div class="size-4 rounded bg-muted/40"></div>
                      <div class="space-y-1.5 flex-1 max-w-sm">
                        <div class="h-3 w-40 bg-muted/60 rounded"></div>
                        <div class="h-2 w-24 bg-muted/40 rounded"></div>
                      </div>
                    </div>
                    <div class="h-4 w-16 bg-muted/40 rounded-full"></div>
                    <div class="h-3 w-20 bg-muted/40 rounded"></div>
                  </div>
                {/each}
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
</div>
