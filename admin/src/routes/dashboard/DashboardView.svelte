<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import SectionCards from '$lib/components/dashboard/SectionCards.svelte';
  import ChartAreaInteractive from '$lib/components/dashboard/ChartAreaInteractive.svelte';
  import DataTable from '$lib/components/dashboard/DataTable.svelte';

  interface DashboardData {
    content?: {
      pages_published: number;
      news_published: number;
    };
    members?: {
      total_members: number;
      active_members: number;
      verified_members: number;
    };
    runtime?: {
      uptime_seconds: number;
      requests: number;
      views: number;
      errors: number;
      avg_latency_ms: number;
    };
    analytics?: {
      unique_visitors_today: number;
    };
  }

  let data = $state<DashboardData | null>(null);

  onMount(async () => {
    try {
      const res = await api.get<any>('/api/v1/admin/dashboard');
      if (res && res.data) {
        data = res.data;
      }
    } catch (e) {
      console.warn('Dashboard metrics fetch error:', e);
    }
  });

  const liveMetrics = $derived.by(() => {
    if (!data) return undefined;
    return {
      pagesPublished: data.content?.pages_published ?? 0,
      newsPublished: data.content?.news_published ?? 0,
      totalMembers: data.members?.total_members ?? 0,
      activeMembers: data.members?.active_members ?? 0,
      verifiedMembers: data.members?.verified_members ?? 0,
      totalRequests: Number(data.runtime?.requests ?? 0),
      avgLatencyMs: Number(data.runtime?.avg_latency_ms ?? 0),
      uptimeSeconds: Number(data.runtime?.uptime_seconds ?? 0),
      uniqueVisitorsToday: Number(data.analytics?.unique_visitors_today ?? 0),
    };
  });
</script>

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-2">
    <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <SectionCards metrics={liveMetrics} />
      <ChartAreaInteractive />
      <DataTable />
    </div>
  </div>
</div>
