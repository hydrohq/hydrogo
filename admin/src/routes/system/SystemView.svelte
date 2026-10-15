<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
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

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-2">
    <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <!-- Toolbar -->
      <div class="flex items-center justify-between px-4 lg:px-6">
        <div>
          <h2 class="text-base font-semibold text-foreground">System Telemetry & Diagnostics</h2>
          <p class="text-xs text-muted-foreground">In-process Go runtime performance and health status.</p>
        </div>
        <button
          type="button"
          class="h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs inline-flex items-center gap-1.5"
          onclick={loadData}
        >
          <svg class="size-3.5 {loading ? 'animate-spin' : ''}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"></path>
            <path d="M3 3v5h5"></path>
            <path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"></path>
            <path d="M16 21h5v-5"></path>
          </svg>
          <span>Refresh</span>
        </button>
      </div>

      <!-- Health Banner -->
      <div class="px-4 lg:px-6">
        <div class="p-4 rounded-xl border border-border bg-card shadow-xs flex items-center justify-between">
          <div class="flex items-center gap-3">
            <span class="relative flex size-3">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full {health?.ok ? 'bg-emerald-400' : 'bg-destructive'} opacity-75"></span>
              <span class="relative inline-flex rounded-full size-3 {health?.ok ? 'bg-emerald-500' : 'bg-destructive'}"></span>
            </span>
            <div>
              <div class="text-sm font-semibold text-foreground">
                Cluster Health: {health?.ok ? 'Operational' : 'Degraded'}
              </div>
              {#if health?.error}
                <div class="text-xs text-destructive font-mono mt-0.5">{health.error}</div>
              {:else if health?.checked_at}
                <div class="text-xs text-muted-foreground">
                  Last verified at {new Date(health.checked_at).toLocaleTimeString()}
                </div>
              {/if}
            </div>
          </div>
          <Badge variant={health?.ok ? 'success' : 'destructive'} size="sm">
            {health?.ok ? 'Healthy' : 'Unhealthy'}
          </Badge>
        </div>
      </div>

      <!-- KPI Cards (matching shadcn SectionCards layout) -->
      <div class="px-4 lg:px-6 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div class="rounded-xl border border-border bg-card p-4 shadow-xs">
          <span class="text-xs font-medium text-muted-foreground">Goroutines</span>
          <div class="text-2xl font-bold font-mono tracking-tight text-foreground mt-2">
            {metrics?.Goroutines ?? '—'}
          </div>
          <p class="text-xs text-muted-foreground mt-1">{metrics?.CPUCount || 1} CPU cores allocated</p>
        </div>

        <div class="rounded-xl border border-border bg-card p-4 shadow-xs">
          <span class="text-xs font-medium text-muted-foreground">Heap Allocation</span>
          <div class="text-2xl font-bold font-mono tracking-tight text-foreground mt-2">
            {formatBytes(metrics?.HeapAllocBytes)}
          </div>
          <p class="text-xs text-muted-foreground mt-1">Sys reserved: {formatBytes(metrics?.HeapSysBytes)}</p>
        </div>

        <div class="rounded-xl border border-border bg-card p-4 shadow-xs">
          <span class="text-xs font-medium text-muted-foreground">Requests Served</span>
          <div class="text-2xl font-bold font-mono tracking-tight text-foreground mt-2">
            {metrics?.Requests?.toLocaleString() ?? '—'}
          </div>
          <p class="text-xs text-muted-foreground mt-1">Active in-flight: {metrics?.InFlight ?? 0}</p>
        </div>

        <div class="rounded-xl border border-border bg-card p-4 shadow-xs">
          <span class="text-xs font-medium text-muted-foreground">Average Latency</span>
          <div class="text-2xl font-bold font-mono tracking-tight text-foreground mt-2">
            {metrics?.AvgLatencyMS ?? 0} ms
          </div>
          <p class="text-xs text-muted-foreground mt-1">Total errors: {metrics?.Errors ?? 0}</p>
        </div>
      </div>

      <!-- Environment & Build Profile -->
      <div class="px-4 lg:px-6">
        <div class="rounded-xl border border-border bg-card p-5 shadow-xs space-y-4">
          <h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            Environment & Build Profile
          </h3>
          <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4 text-xs font-mono">
            <div class="p-3 rounded-lg bg-muted/40 border border-border">
              <span class="text-muted-foreground block font-sans text-xs">HydroGo Core</span>
              <span class="text-foreground font-semibold">{info?.version || 'dev'}</span>
            </div>
            <div class="p-3 rounded-lg bg-muted/40 border border-border">
              <span class="text-muted-foreground block font-sans text-xs">Go Compiler</span>
              <span class="text-foreground">{info?.go_version || 'unknown'}</span>
            </div>
            <div class="p-3 rounded-lg bg-muted/40 border border-border">
              <span class="text-muted-foreground block font-sans text-xs">SQL Storage Dialect</span>
              <span class="text-foreground font-semibold">{info?.dialect || 'sqlite'}</span>
            </div>
            <div class="p-3 rounded-lg bg-muted/40 border border-border">
              <span class="text-muted-foreground block font-sans text-xs">Capacity Tier</span>
              <span class="text-foreground">{info?.capacity_profile || 'standard'}</span>
            </div>
            <div class="p-3 rounded-lg bg-muted/40 border border-border">
              <span class="text-muted-foreground block font-sans text-xs">Development Mode</span>
              <span class="text-foreground">{info?.dev_mode ? 'Enabled' : 'Disabled'}</span>
            </div>
            <div class="p-3 rounded-lg bg-muted/40 border border-border">
              <span class="text-muted-foreground block font-sans text-xs">Safe Mode</span>
              <span class="text-foreground">{info?.safe_mode ? 'Enabled' : 'Disabled'}</span>
            </div>
          </div>

          {#if info?.modules && info.modules.length > 0}
            <div class="pt-3 border-t border-border">
              <span class="text-xs text-muted-foreground block mb-2 font-medium">Registered Active Modules:</span>
              <div class="flex flex-wrap gap-1.5">
                {#each info.modules as mod}
                  <span class="inline-flex items-center px-2.5 py-0.5 rounded-md border border-border text-xs font-mono bg-muted/50 text-foreground">
                    {mod}
                  </span>
                {/each}
              </div>
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>
