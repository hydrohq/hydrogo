<script lang="ts">
  interface Metrics {
    revenue?: string;
    revenueTrend?: string;
    customers?: string;
    customersTrend?: string;
    accounts?: string;
    accountsTrend?: string;
    growth?: string;
    growthTrend?: string;
  }

  interface Props {
    metrics?: Metrics;
  }

  let { metrics }: Props = $props();

  const totalRevenue = $derived(metrics?.revenue || "$1,250.00");
  const newCustomers = $derived(metrics?.customers || "1,234");
  const activeAccounts = $derived(metrics?.accounts || "45,678");
  const growthRate = $derived(metrics?.growth || "4.5%");
</script>

<div class="grid grid-cols-1 gap-4 px-4 @xl/main:grid-cols-2 @5xl/main:grid-cols-4 lg:px-6">
  <!-- Card 1: Total Revenue -->
  <div class="rounded-xl border border-border bg-gradient-to-t from-primary/5 to-card dark:bg-card p-6 shadow-xs text-card-foreground @container/card">
    <div class="relative flex flex-col space-y-1.5">
      <span class="text-sm font-normal text-muted-foreground">Total Revenue</span>
      <span class="text-2xl @[250px]/card:text-3xl font-semibold tabular-nums text-foreground tracking-tight">
        {totalRevenue}
      </span>
      <div class="absolute right-0 top-0">
        <span class="inline-flex items-center gap-1 rounded-lg border border-border bg-background px-2 py-0.5 text-xs font-normal text-foreground shadow-xs">
          <svg class="size-3 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
            <polyline points="17 6 23 6 23 12"></polyline>
          </svg>
          +12.5%
        </span>
      </div>
    </div>
    <div class="flex flex-col items-start gap-1 text-sm pt-4">
      <div class="line-clamp-1 flex items-center gap-2 font-medium text-foreground">
        <span>Trending up this month</span>
        <svg class="size-4 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
          <polyline points="17 6 23 6 23 12"></polyline>
        </svg>
      </div>
      <div class="text-xs text-muted-foreground">Visitors for the last 6 months</div>
    </div>
  </div>

  <!-- Card 2: New Customers -->
  <div class="rounded-xl border border-border bg-gradient-to-t from-primary/5 to-card dark:bg-card p-6 shadow-xs text-card-foreground @container/card">
    <div class="relative flex flex-col space-y-1.5">
      <span class="text-sm font-normal text-muted-foreground">New Customers</span>
      <span class="text-2xl @[250px]/card:text-3xl font-semibold tabular-nums text-foreground tracking-tight">
        {newCustomers}
      </span>
      <div class="absolute right-0 top-0">
        <span class="inline-flex items-center gap-1 rounded-lg border border-border bg-background px-2 py-0.5 text-xs font-normal text-foreground shadow-xs">
          <svg class="size-3 text-rose-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="23 18 13.5 8.5 8.5 13.5 1 6"></polyline>
            <polyline points="17 18 23 18 23 12"></polyline>
          </svg>
          -20%
        </span>
      </div>
    </div>
    <div class="flex flex-col items-start gap-1 text-sm pt-4">
      <div class="line-clamp-1 flex items-center gap-2 font-medium text-foreground">
        <span>Down 20% this period</span>
        <svg class="size-4 text-rose-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="23 18 13.5 8.5 8.5 13.5 1 6"></polyline>
          <polyline points="17 18 23 18 23 12"></polyline>
        </svg>
      </div>
      <div class="text-xs text-muted-foreground">Acquisition needs attention</div>
    </div>
  </div>

  <!-- Card 3: Active Accounts -->
  <div class="rounded-xl border border-border bg-gradient-to-t from-primary/5 to-card dark:bg-card p-6 shadow-xs text-card-foreground @container/card">
    <div class="relative flex flex-col space-y-1.5">
      <span class="text-sm font-normal text-muted-foreground">Active Accounts</span>
      <span class="text-2xl @[250px]/card:text-3xl font-semibold tabular-nums text-foreground tracking-tight">
        {activeAccounts}
      </span>
      <div class="absolute right-0 top-0">
        <span class="inline-flex items-center gap-1 rounded-lg border border-border bg-background px-2 py-0.5 text-xs font-normal text-foreground shadow-xs">
          <svg class="size-3 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
            <polyline points="17 6 23 6 23 12"></polyline>
          </svg>
          +12.5%
        </span>
      </div>
    </div>
    <div class="flex flex-col items-start gap-1 text-sm pt-4">
      <div class="line-clamp-1 flex items-center gap-2 font-medium text-foreground">
        <span>Strong user retention</span>
        <svg class="size-4 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
          <polyline points="17 6 23 6 23 12"></polyline>
        </svg>
      </div>
      <div class="text-xs text-muted-foreground">Engagement exceed targets</div>
    </div>
  </div>

  <!-- Card 4: Growth Rate -->
  <div class="rounded-xl border border-border bg-gradient-to-t from-primary/5 to-card dark:bg-card p-6 shadow-xs text-card-foreground @container/card">
    <div class="relative flex flex-col space-y-1.5">
      <span class="text-sm font-normal text-muted-foreground">Growth Rate</span>
      <span class="text-2xl @[250px]/card:text-3xl font-semibold tabular-nums text-foreground tracking-tight">
        {growthRate}
      </span>
      <div class="absolute right-0 top-0">
        <span class="inline-flex items-center gap-1 rounded-lg border border-border bg-background px-2 py-0.5 text-xs font-normal text-foreground shadow-xs">
          <svg class="size-3 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
            <polyline points="17 6 23 6 23 12"></polyline>
          </svg>
          +4.5%
        </span>
      </div>
    </div>
    <div class="flex flex-col items-start gap-1 text-sm pt-4">
      <div class="line-clamp-1 flex items-center gap-2 font-medium text-foreground">
        <span>Steady performance</span>
        <svg class="size-4 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
          <polyline points="17 6 23 6 23 12"></polyline>
        </svg>
      </div>
      <div class="text-xs text-muted-foreground">Meets growth projections</div>
    </div>
  </div>
</div>
