<script lang="ts">
  interface DashboardMetrics {
    pagesPublished?: number;
    newsPublished?: number;
    totalMembers?: number;
    activeMembers?: number;
    verifiedMembers?: number;
    totalRequests?: number;
    avgLatencyMs?: number;
    uptimeSeconds?: number;
    uniqueVisitorsToday?: number;
  }

  interface Props {
    metrics?: DashboardMetrics;
  }

  let { metrics }: Props = $props();

  const pagesCount = $derived(metrics?.pagesPublished ?? 0);
  const newsCount = $derived(metrics?.newsPublished ?? 0);
  const requestsCount = $derived((metrics?.totalRequests ?? 0).toLocaleString());
  const latency = $derived(metrics?.avgLatencyMs ? `${metrics.avgLatencyMs}ms` : '< 5ms');

  function formatUptime(seconds?: number): string {
    if (!seconds || seconds <= 0) return 'System Online';
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    if (hrs > 24) {
      const days = Math.floor(hrs / 24);
      return `${days}d ${hrs % 24}h uptime`;
    }
    if (hrs > 0) return `${hrs}h ${mins}m uptime`;
    return `${mins}m uptime`;
  }
</script>

<div class="grid grid-cols-1 gap-4 px-4 @xl/main:grid-cols-2 @5xl/main:grid-cols-4 lg:px-6">
  <!-- Card 1: Pages Published -->
  <div class="rounded-xl border border-border bg-gradient-to-t from-primary/5 to-card dark:bg-card p-6 shadow-xs text-card-foreground @container/card">
    <div class="relative flex flex-col space-y-1.5">
      <span class="text-sm font-normal text-muted-foreground">Published Pages</span>
      <span class="text-2xl @[250px]/card:text-3xl font-semibold tabular-nums text-foreground tracking-tight">
        {pagesCount}
      </span>
      <div class="absolute right-0 top-0">
        <span class="inline-flex items-center gap-1 rounded-lg border border-border bg-background px-2 py-0.5 text-xs font-normal text-foreground shadow-xs">
          <svg class="size-3 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
            <polyline points="17 6 23 6 23 12"></polyline>
          </svg>
          Active
        </span>
      </div>
    </div>
    <div class="flex flex-col items-start gap-1 text-sm pt-4">
      <div class="line-clamp-1 flex items-center gap-2 font-medium text-foreground">
        <span>Site routes & layouts</span>
        <svg class="size-4 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
          <polyline points="17 6 23 6 23 12"></polyline>
        </svg>
      </div>
      <div class="text-xs text-muted-foreground">Served via Headless REST API</div>
    </div>
  </div>

  <!-- Card 2: News Articles -->
  <div class="rounded-xl border border-border bg-gradient-to-t from-primary/5 to-card dark:bg-card p-6 shadow-xs text-card-foreground @container/card">
    <div class="relative flex flex-col space-y-1.5">
      <span class="text-sm font-normal text-muted-foreground">News & Stories</span>
      <span class="text-2xl @[250px]/card:text-3xl font-semibold tabular-nums text-foreground tracking-tight">
        {newsCount}
      </span>
      <div class="absolute right-0 top-0">
        <span class="inline-flex items-center gap-1 rounded-lg border border-border bg-background px-2 py-0.5 text-xs font-normal text-foreground shadow-xs">
          <svg class="size-3 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
            <polyline points="17 6 23 6 23 12"></polyline>
          </svg>
          Editorial
        </span>
      </div>
    </div>
    <div class="flex flex-col items-start gap-1 text-sm pt-4">
      <div class="line-clamp-1 flex items-center gap-2 font-medium text-foreground">
        <span>Published articles</span>
        <svg class="size-4 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
          <polyline points="17 6 23 6 23 12"></polyline>
        </svg>
      </div>
      <div class="text-xs text-muted-foreground">Categorized and tagged</div>
    </div>
  </div>

  <!-- Card 3: Content Categories -->
  <div class="rounded-xl border border-border bg-gradient-to-t from-primary/5 to-card dark:bg-card p-6 shadow-xs text-card-foreground @container/card">
    <div class="relative flex flex-col space-y-1.5">
      <span class="text-sm font-normal text-muted-foreground">Haber Kategorileri</span>
      <span class="text-2xl @[250px]/card:text-3xl font-semibold tabular-nums text-foreground tracking-tight">
        7 Aktif
      </span>
      <div class="absolute right-0 top-0">
        <span class="inline-flex items-center gap-1 rounded-lg border border-border bg-background px-2 py-0.5 text-xs font-normal text-foreground shadow-xs">
          <svg class="size-3 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
            <polyline points="17 6 23 6 23 12"></polyline>
          </svg>
          Yayında
        </span>
      </div>
    </div>
    <div class="flex flex-col items-start gap-1 text-sm pt-4">
      <div class="line-clamp-1 flex items-center gap-2 font-medium text-foreground">
        <span>Gündem, Ekonomi, Spor...</span>
        <svg class="size-4 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
          <polyline points="17 6 23 6 23 12"></polyline>
        </svg>
      </div>
      <div class="text-xs text-muted-foreground">Kategori vitrinleri ve akışlar</div>
    </div>
  </div>

  <!-- Card 4: API & Runtime -->
  <div class="rounded-xl border border-border bg-gradient-to-t from-primary/5 to-card dark:bg-card p-6 shadow-xs text-card-foreground @container/card">
    <div class="relative flex flex-col space-y-1.5">
      <span class="text-sm font-normal text-muted-foreground">API Traffic</span>
      <span class="text-2xl @[250px]/card:text-3xl font-semibold tabular-nums text-foreground tracking-tight">
        {requestsCount}
      </span>
      <div class="absolute right-0 top-0">
        <span class="inline-flex items-center gap-1 rounded-lg border border-border bg-background px-2 py-0.5 text-xs font-normal text-foreground shadow-xs">
          <svg class="size-3 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <circle cx="12" cy="12" r="10"></circle>
            <polyline points="12 6 12 12 16 14"></polyline>
          </svg>
          {latency}
        </span>
      </div>
    </div>
    <div class="flex flex-col items-start gap-1 text-sm pt-4">
      <div class="line-clamp-1 flex items-center gap-2 font-medium text-foreground">
        <span>{formatUptime(metrics?.uptimeSeconds)}</span>
        <svg class="size-4 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
          <polyline points="17 6 23 6 23 12"></polyline>
        </svg>
      </div>
      <div class="text-xs text-muted-foreground">Zero downtime Go server</div>
    </div>
  </div>
</div>
