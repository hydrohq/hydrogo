<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';

  interface SystemInfo {
    version: string;
    go_version: string;
    dialect: string;
    capacity_profile: string;
    dev_mode: boolean;
    safe_mode: boolean;
    modules: string[];
    started_at?: string;
  }

  interface HealthStatus {
    ok: boolean;
    checked_at: string;
    error?: string;
  }

  interface RuntimeMetrics {
    Goroutines?: number;
    CPUCount?: number;
    HeapAllocBytes?: number;
    HeapSysBytes?: number;
    Requests?: number;
    Errors?: number;
    AvgLatencyMS?: number;
    InFlight?: number;
    ReadinessState?: boolean;
  }

  let info = $state<SystemInfo | null>(null);
  let health = $state<HealthStatus | null>(null);
  let metrics = $state<RuntimeMetrics | null>(null);
  let loading = $state(true);

  async function loadData() {
    loading = true;
    try {
      const [infoRes, healthRes, metricsRes] = await Promise.allSettled([
        api.get<ApiResponse<SystemInfo>>('/api/v1/admin/system/info'),
        api.get<ApiResponse<HealthStatus>>('/api/v1/admin/system/health'),
        api.get<ApiResponse<RuntimeMetrics>>('/api/v1/admin/system/metrics'),
      ]);

      if (infoRes.status === 'fulfilled' && infoRes.value?.data) {
        info = infoRes.value.data;
      }
      if (healthRes.status === 'fulfilled' && healthRes.value?.data) {
        health = healthRes.value.data;
      }
      if (metricsRes.status === 'fulfilled' && metricsRes.value?.data) {
        metrics = metricsRes.value.data;
      }
    } catch (e) {
      console.warn('System telemetry load error:', e);
    } finally {
      loading = false;
    }
  }

  function formatBytes(bytes?: number): string {
    if (!bytes) return '0 B';
    const mb = bytes / (1024 * 1024);
    return `${mb.toFixed(1)} MB`;
  }

  onMount(() => {
    loadData();
  });
</script>

<div class="space-y-6 max-w-5xl mx-auto">
  <!-- Header -->
  <div class="flex items-center justify-between pb-4 border-b border-slate-800">
    <div>
      <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2">
        <span>⚡</span>
        <span>System Diagnostics & Telemetry</span>
      </h2>
      <p class="text-xs text-slate-400 mt-1">
        Runtime environment, cluster health, and in-process execution metrics.
      </p>
    </div>
    <Button size="sm" variant="ghost" loading={loading} onclick={loadData}>
      Refresh
    </Button>
  </div>

  <!-- Health Banner -->
  <div class="p-4 rounded-xl border flex items-center justify-between {health?.ok ? 'bg-emerald-950/40 border-emerald-800/80' : 'bg-rose-950/40 border-rose-800/80'}">
    <div class="flex items-center gap-3">
      <div class="w-3 h-3 rounded-full {health?.ok ? 'bg-emerald-400 animate-pulse' : 'bg-rose-500'}"></div>
      <div>
        <div class="text-sm font-semibold text-slate-100">
          Instance Health: {health?.ok ? 'Operational' : 'Degraded'}
        </div>
        {#if health?.error}
          <div class="text-xs text-rose-300 font-mono mt-0.5">{health.error}</div>
        {:else if health?.checked_at}
          <div class="text-[11px] text-slate-400">
            Last probe check at {new Date(health.checked_at).toLocaleTimeString()}
          </div>
        {/if}
      </div>
    </div>
    <Badge variant={health?.ok ? 'success' : 'danger'}>
      {health?.ok ? 'Healthy' : 'Unhealthy'}
    </Badge>
  </div>

  <!-- Grid of Metrics -->
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
    <div class="p-4 rounded-xl border border-slate-800 bg-slate-900/60">
      <div class="text-xs font-semibold uppercase text-slate-400">Goroutines</div>
      <div class="text-2xl font-bold font-mono text-slate-100 mt-2">
        {metrics?.Goroutines ?? '—'}
      </div>
      <div class="text-[11px] text-slate-500 mt-1">{metrics?.CPUCount || 0} CPU Cores</div>
    </div>

    <div class="p-4 rounded-xl border border-slate-800 bg-slate-900/60">
      <div class="text-xs font-semibold uppercase text-slate-400">Heap Allocated</div>
      <div class="text-2xl font-bold font-mono text-sky-400 mt-2">
        {formatBytes(metrics?.HeapAllocBytes)}
      </div>
      <div class="text-[11px] text-slate-500 mt-1">Sys: {formatBytes(metrics?.HeapSysBytes)}</div>
    </div>

    <div class="p-4 rounded-xl border border-slate-800 bg-slate-900/60">
      <div class="text-xs font-semibold uppercase text-slate-400">Total Requests</div>
      <div class="text-2xl font-bold font-mono text-emerald-400 mt-2">
        {metrics?.Requests?.toLocaleString() ?? '—'}
      </div>
      <div class="text-[11px] text-slate-500 mt-1">In-Flight: {metrics?.InFlight ?? 0}</div>
    </div>

    <div class="p-4 rounded-xl border border-slate-800 bg-slate-900/60">
      <div class="text-xs font-semibold uppercase text-slate-400">Avg Latency</div>
      <div class="text-2xl font-bold font-mono text-amber-400 mt-2">
        {metrics?.AvgLatencyMS ?? 0} ms
      </div>
      <div class="text-[11px] text-slate-500 mt-1">Errors: {metrics?.Errors ?? 0}</div>
    </div>
  </div>

  <!-- Environment & Build Info -->
  <div class="rounded-xl border border-slate-800 bg-slate-900/60 p-6 space-y-4">
    <h3 class="text-sm font-semibold uppercase text-slate-300 tracking-wider">
      Environment & Build Profile
    </h3>
    <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4 text-xs font-mono">
      <div>
        <span class="text-slate-500 block font-sans text-[11px]">HydroGo Version</span>
        <span class="text-slate-200">{info?.version || 'dev'}</span>
      </div>
      <div>
        <span class="text-slate-500 block font-sans text-[11px]">Go Runtime</span>
        <span class="text-slate-200">{info?.go_version || 'unknown'}</span>
      </div>
      <div>
        <span class="text-slate-500 block font-sans text-[11px]">Database Dialect</span>
        <span class="text-sky-400 font-semibold">{info?.dialect || 'sqlite'}</span>
      </div>
      <div>
        <span class="text-slate-500 block font-sans text-[11px]">Capacity Profile</span>
        <span class="text-slate-200">{info?.capacity_profile || 'standard'}</span>
      </div>
      <div>
        <span class="text-slate-500 block font-sans text-[11px]">Development Mode</span>
        <span class={info?.dev_mode ? 'text-amber-400' : 'text-slate-400'}>
          {info?.dev_mode ? 'Enabled' : 'Disabled'}
        </span>
      </div>
      <div>
        <span class="text-slate-500 block font-sans text-[11px]">Safe Mode</span>
        <span class={info?.safe_mode ? 'text-rose-400' : 'text-slate-400'}>
          {info?.safe_mode ? 'Enabled' : 'Disabled'}
        </span>
      </div>
    </div>

    {#if info?.modules && info.modules.length > 0}
      <div class="pt-4 border-t border-slate-800">
        <span class="text-slate-400 text-xs block mb-2">Active Modules:</span>
        <div class="flex flex-wrap gap-1.5">
          {#each info.modules as mod}
            <Badge variant="default" size="sm">{mod}</Badge>
          {/each}
        </div>
      </div>
    {/if}
  </div>
</div>
