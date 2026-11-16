<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
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
    if (!seconds || seconds <= 0) return t('dashboard.cards.online');
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    if (hrs > 24) {
      const days = Math.floor(hrs / 24);
      return t('dashboard.cards.uptimeDays', { days, hours: hrs % 24 });
    }
    if (hrs > 0) return t('dashboard.cards.uptimeHours', { hours: hrs, mins });
    return t('dashboard.cards.uptimeMins', { mins });
  }
</script>

<div class="grid grid-cols-1 gap-4 @xl/main:grid-cols-2 @5xl/main:grid-cols-4">
  <!-- Card 1: Pages Published -->
  <div class="rounded-xl border border-border bg-card p-5 text-card-foreground @container/card">
    <div class="flex items-start justify-between gap-2">
      <span class="text-xs font-medium text-muted-foreground">{t('dashboard.cards.pages')}</span>
      <span class="rounded border border-border bg-muted/50 px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground">
        {t('dashboard.cards.pagesBadge')}
      </span>
    </div>
    <div class="mt-2 text-2xl @[250px]/card:text-3xl font-medium tabular-nums text-foreground tracking-tight">
      {pagesCount}
    </div>
    <div class="mt-4 flex flex-col gap-0.5 text-xs">
      <div class="line-clamp-1 font-medium text-foreground">{t('dashboard.cards.pagesLine')}</div>
      <div class="text-muted-foreground">{t('dashboard.cards.pagesNote')}</div>
    </div>
  </div>

  <!-- Card 2: News Articles -->
  <div class="rounded-xl border border-border bg-card p-5 text-card-foreground @container/card">
    <div class="flex items-start justify-between gap-2">
      <span class="text-xs font-medium text-muted-foreground">{t('dashboard.cards.news')}</span>
      <span class="rounded border border-border bg-muted/50 px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground">
        {t('dashboard.cards.newsBadge')}
      </span>
    </div>
    <div class="mt-2 text-2xl @[250px]/card:text-3xl font-medium tabular-nums text-foreground tracking-tight">
      {newsCount}
    </div>
    <div class="mt-4 flex flex-col gap-0.5 text-xs">
      <div class="line-clamp-1 font-medium text-foreground">{t('dashboard.cards.newsLine')}</div>
      <div class="text-muted-foreground">{t('dashboard.cards.newsNote')}</div>
    </div>
  </div>

  <!-- Card 3: Members -->
  <div class="rounded-xl border border-border bg-card p-5 text-card-foreground @container/card">
    <div class="flex items-start justify-between gap-2">
      <span class="text-xs font-medium text-muted-foreground">{t('dashboard.cards.members')}</span>
      <span class="rounded border border-border bg-muted/50 px-1.5 py-0.5 font-mono text-[11px] tabular-nums text-muted-foreground">
        {t('dashboard.cards.membersActive', { count: metrics?.activeMembers ?? 0 })}
      </span>
    </div>
    <div class="mt-2 text-2xl @[250px]/card:text-3xl font-medium tabular-nums text-foreground tracking-tight">
      {(metrics?.totalMembers ?? 0).toLocaleString()}
    </div>
    <div class="mt-4 flex flex-col gap-0.5 text-xs">
      <div class="line-clamp-1 font-medium text-foreground">{t('dashboard.cards.membersVerified', { count: metrics?.verifiedMembers ?? 0 })}</div>
      <div class="text-muted-foreground">{t('dashboard.cards.membersNote')}</div>
    </div>
  </div>

  <!-- Card 4: API & Runtime -->
  <div class="rounded-xl border border-border bg-card p-5 text-card-foreground @container/card">
    <div class="flex items-start justify-between gap-2">
      <span class="text-xs font-medium text-muted-foreground">{t('dashboard.cards.api')}</span>
      <span class="rounded border border-border bg-muted/50 px-1.5 py-0.5 font-mono text-[11px] tabular-nums text-muted-foreground">
        {latency}
      </span>
    </div>
    <div class="mt-2 text-2xl @[250px]/card:text-3xl font-medium tabular-nums text-foreground tracking-tight">
      {requestsCount}
    </div>
    <div class="mt-4 flex flex-col gap-0.5 text-xs">
      <div class="line-clamp-1 font-medium text-foreground">{formatUptime(metrics?.uptimeSeconds)}</div>
      <div class="text-muted-foreground">{t('dashboard.cards.noteServer')}</div>
    </div>
  </div>
</div>
