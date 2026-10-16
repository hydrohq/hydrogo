<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import BackupsSecurityTab from './tabs/BackupsSecurityTab.svelte';

  interface SystemInfo {
    version: string;
    go_version: string;
    dialect: string;
    capacity_profile: string;
    dev_mode: boolean;
    safe_mode: boolean;
    modules: Record<string, boolean>;
    public_origin?: string;
    tenancy_enabled?: boolean;
    started_at?: string;
    support?: {
      enabled: boolean;
      agency_name?: string;
      email?: string;
      sla_badge?: string;
    };
  }

  interface HealthStatus {
    ok: boolean;
    checked_at: string;
    error?: string;
  }

  interface DatabaseStats {
    MaxOpenConnections: number;
    OpenConnections: number;
    InUseConnections: number;
    IdleConnections: number;
    WaitCount: number;
    WaitDuration: number;
  }

  interface CacheStats {
    Hits: number;
    Misses: number;
    Loads: number;
    LoadErrors: number;
    Evictions: number;
    Expirations: number;
  }

  interface JobStats {
    Depth: number;
    Leased: number;
    DeadLettered: number;
  }

  interface PathStatsSnapshot {
    Path: string;
    Requests: number;
    Views: number;
    Errors: number;
    AvgMS: number;
    P50MS: number;
    P95MS: number;
    P99MS: number;
    Status2xx: number;
    Status3xx: number;
    Status4xx: number;
    Status5xx: number;
  }

  interface RuntimeMetrics {
    StartedAt?: string;
    Uptime?: number;
    Requests?: number;
    InFlight?: number;
    Views?: number;
    Errors?: number;
    AvgLatencyMS?: number;
    UniqueVisitors?: number;
    Goroutines?: number;
    CPUCount?: number;
    GOMAXPROCS?: number;
    HeapAllocBytes?: number;
    HeapInUseBytes?: number;
    HeapSysBytes?: number;
    HeapObjects?: number;
    Database?: DatabaseStats;
    Cache?: CacheStats;
    Jobs?: JobStats;
    HasDatabase?: boolean;
    WorkerProcessed?: number;
    WorkerFailures?: number;
    WorkerRetries?: number;
    WorkerDeadLetters?: number;
    WorkerConcurrency?: number;
    ReadinessState?: number;
    ReadinessFailures?: number;
    TopPaths?: PathStatsSnapshot[];
  }

  let info = $state<SystemInfo | null>(null);
  let health = $state<HealthStatus | null>(null);
  let metrics = $state<RuntimeMetrics | null>(null);
  let loading = $state(true);
  let autoRefresh = $state(false);
  let refreshTimer: any = null;
  let activeTab = $state<'overview' | 'memory' | 'database' | 'jobs' | 'paths' | 'backups'>('overview');

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
    if (mb < 1024) return `${mb.toFixed(1)} MB`;
    return `${(mb / 1024).toFixed(2)} GB`;
  }

  function formatDuration(ns?: number): string {
    if (!ns) return '0s';
    const seconds = Math.floor(ns / 1_000_000_000);
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    if (days > 0) return `${days}d ${hours}h ${minutes}m`;
    if (hours > 0) return `${hours}h ${minutes}m ${secs}s`;
    if (minutes > 0) return `${minutes}m ${secs}s`;
    return `${secs}s`;
  }

  function toggleAutoRefresh() {
    autoRefresh = !autoRefresh;
    if (autoRefresh) {
      refreshTimer = setInterval(loadData, 5000);
    } else {
      if (refreshTimer) clearInterval(refreshTimer);
    }
  }

  onMount(() => {
    loadData();
  });

  onDestroy(() => {
    if (refreshTimer) clearInterval(refreshTimer);
  });
</script>

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-4 p-4 lg:p-6 max-w-6xl mx-auto w-full">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-border pb-4">
      <div>
        <h2 class="text-xl font-semibold tracking-tight text-foreground">System Telemetry & Runtime Diagnostics</h2>
        <p class="text-xs text-muted-foreground mt-0.5">In-process Go runtime performance, memory heap, connection pooling, and request metrics.</p>
      </div>

      <div class="flex items-center gap-2">
        <button
          type="button"
          onclick={toggleAutoRefresh}
          class="h-8 px-2.5 rounded-md border text-xs font-medium inline-flex items-center gap-1.5 transition-colors cursor-pointer {autoRefresh ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-600 dark:text-emerald-400' : 'border-input bg-background hover:bg-accent text-muted-foreground'}"
        >
          <span class="size-2 rounded-full {autoRefresh ? 'bg-emerald-500 animate-ping' : 'bg-muted-foreground'}"></span>
          <span>{autoRefresh ? 'Live (5s)' : 'Auto-refresh'}</span>
        </button>

        <Button variant="outline" onclick={loadData} disabled={loading}>
          <svg class="size-3.5 mr-1.5 {loading ? 'animate-spin' : ''}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"></path>
            <path d="M3 3v5h5"></path>
            <path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"></path>
            <path d="M16 21h5v-5"></path>
          </svg>
          Refresh
        </Button>
      </div>
    </div>

    <!-- Quick Metric Cards -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <!-- Card 1: Engine Health & Uptime -->
      <div class="rounded-xl border border-border bg-card p-4 shadow-xs space-y-2">
        <div class="flex items-center justify-between text-xs text-muted-foreground">
          <span class="font-medium">Engine Status</span>
          <span class="size-2 rounded-full {health?.ok ? 'bg-emerald-500' : 'bg-destructive'}"></span>
        </div>
        <div class="flex items-baseline gap-2">
          <span class="text-2xl font-bold tracking-tight text-foreground">
            {health?.ok ? 'Healthy' : 'Degraded'}
          </span>
          <span class="text-xs font-mono text-muted-foreground capitalize">
            {info?.dialect || 'sqlite'}
          </span>
        </div>
        <p class="text-[11px] text-muted-foreground">
          Uptime: <span class="font-mono text-foreground font-medium">{formatDuration(metrics?.Uptime)}</span>
        </p>
      </div>

      <!-- Card 2: Memory Heap -->
      <div class="rounded-xl border border-border bg-card p-4 shadow-xs space-y-2">
        <div class="flex items-center justify-between text-xs text-muted-foreground">
          <span class="font-medium">Memory Allocation</span>
          <span class="font-mono text-[10px]">{metrics?.HeapObjects?.toLocaleString() || 0} objects</span>
        </div>
        <div class="flex items-baseline gap-2">
          <span class="text-2xl font-bold tracking-tight text-foreground font-mono">
            {formatBytes(metrics?.HeapAllocBytes)}
          </span>
          <span class="text-xs text-muted-foreground font-mono">
            / {formatBytes(metrics?.HeapSysBytes)} sys
          </span>
        </div>
        <p class="text-[11px] text-muted-foreground">
          In-use: <span class="font-mono text-foreground font-medium">{formatBytes(metrics?.HeapInUseBytes)}</span>
        </p>
      </div>

      <!-- Card 3: Concurrency & Threads -->
      <div class="rounded-xl border border-border bg-card p-4 shadow-xs space-y-2">
        <div class="flex items-center justify-between text-xs text-muted-foreground">
          <span class="font-medium">Go Runtime Concurrency</span>
          <span class="font-mono text-[10px]">{metrics?.CPUCount || 1} CPU cores</span>
        </div>
        <div class="flex items-baseline gap-2">
          <span class="text-2xl font-bold tracking-tight text-foreground font-mono">
            {metrics?.Goroutines || 0}
          </span>
          <span class="text-xs text-muted-foreground">goroutines</span>
        </div>
        <p class="text-[11px] text-muted-foreground">
          GOMAXPROCS: <span class="font-mono text-foreground font-medium">{metrics?.GOMAXPROCS || 1}</span>
        </p>
      </div>

      <!-- Card 4: HTTP Latency & Throughput -->
      <div class="rounded-xl border border-border bg-card p-4 shadow-xs space-y-2">
        <div class="flex items-center justify-between text-xs text-muted-foreground">
          <span class="font-medium">Request Throughput</span>
          <span class="font-mono text-[10px]">{metrics?.InFlight || 0} in-flight</span>
        </div>
        <div class="flex items-baseline gap-2">
          <span class="text-2xl font-bold tracking-tight text-foreground font-mono">
            {metrics?.Requests?.toLocaleString() || 0}
          </span>
          <span class="text-xs text-muted-foreground font-mono">
            ~{metrics?.AvgLatencyMS || 0}ms avg
          </span>
        </div>
        <p class="text-[11px] text-muted-foreground">
          Errors: <span class="font-mono text-destructive font-medium">{metrics?.Errors || 0}</span>
        </p>
      </div>
    </div>

    <!-- Navigation Tabs -->
    <div class="flex items-center gap-1 border-b border-border overflow-x-auto pb-px">
      <button
        type="button"
        onclick={() => (activeTab = 'overview')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'overview' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Engine Overview
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'memory')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'memory' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Memory & Runtime
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'database')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'database' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Database & Cache
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'jobs')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'jobs' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Async Workers & Jobs
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'paths')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'paths' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Monitored Paths ({metrics?.TopPaths?.length || 0})
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'backups')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'backups' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Backups & Security
      </button>
    </div>

    <!-- Tab 1: Engine Overview -->
    {#if activeTab === 'overview'}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <!-- Configuration Card -->
        <div class="rounded-xl border border-border bg-card p-5 shadow-xs space-y-4">
          <h3 class="text-sm font-semibold text-foreground">Server Information</h3>
          <div class="divide-y divide-border/60 text-xs">
            <div class="py-2.5 flex justify-between">
              <span class="text-muted-foreground">HydroGo Core Version</span>
              <span class="font-mono font-medium text-foreground">{info?.version || 'dev'}</span>
            </div>
            <div class="py-2.5 flex justify-between">
              <span class="text-muted-foreground">Go Compiler Version</span>
              <span class="font-mono text-foreground">{info?.go_version || 'go1.24'}</span>
            </div>
            <div class="py-2.5 flex justify-between">
              <span class="text-muted-foreground">Database Engine Dialect</span>
              <Badge variant="outline" class="font-mono text-[10px] uppercase">{info?.dialect || 'sqlite'}</Badge>
            </div>
            <div class="py-2.5 flex justify-between">
              <span class="text-muted-foreground">Capacity Profile</span>
              <span class="font-mono text-foreground capitalize">{info?.capacity_profile || 'standard'}</span>
            </div>
            <div class="py-2.5 flex justify-between">
              <span class="text-muted-foreground">Development Hot Reload Mode</span>
              <Badge variant={info?.dev_mode ? 'primary' : 'outline'}>{info?.dev_mode ? 'Enabled' : 'Disabled'}</Badge>
            </div>
            <div class="py-2.5 flex justify-between">
              <span class="text-muted-foreground">Safe Mode</span>
              <span class="font-medium text-foreground">{info?.safe_mode ? 'Active' : 'Inactive'}</span>
            </div>
          </div>
        </div>

        <!-- Modules Card -->
        <div class="rounded-xl border border-border bg-card p-5 shadow-xs space-y-4">
          <h3 class="text-sm font-semibold text-foreground">Active Modules</h3>
          <p class="text-xs text-muted-foreground">Modules loaded into the binary for this installation.</p>
          <div class="flex flex-wrap gap-2 pt-2">
            {#if info?.modules}
              {#each Object.entries(info.modules) as [mod, enabled]}
                {#if enabled}
                  <span class="px-2.5 py-1 rounded-md bg-muted border border-border text-xs font-mono text-foreground flex items-center gap-1.5">
                    <span class="size-1.5 rounded-full bg-emerald-500"></span>
                    {mod}
                  </span>
                {/if}
              {/each}
            {:else}
              <span class="text-xs text-muted-foreground">Loading active modules...</span>
            {/if}
          </div>
        </div>
      </div>

    <!-- Tab 2: Memory & Runtime -->
    {:else if activeTab === 'memory'}
      <div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-6">
        <div>
          <h3 class="text-sm font-semibold text-foreground">Go Runtime Memory Heap</h3>
          <p class="text-xs text-muted-foreground mt-0.5">Real-time memory allocations recorded by the Go runtime collector.</p>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div class="p-4 rounded-lg border border-border bg-muted/20 space-y-1">
            <span class="text-xs text-muted-foreground block">Heap Alloc (Active Objects)</span>
            <span class="text-xl font-bold font-mono text-foreground">{formatBytes(metrics?.HeapAllocBytes)}</span>
          </div>
          <div class="p-4 rounded-lg border border-border bg-muted/20 space-y-1">
            <span class="text-xs text-muted-foreground block">Heap In-Use (Spanned Memory)</span>
            <span class="text-xl font-bold font-mono text-foreground">{formatBytes(metrics?.HeapInUseBytes)}</span>
          </div>
          <div class="p-4 rounded-lg border border-border bg-muted/20 space-y-1">
            <span class="text-xs text-muted-foreground block">Heap System (OS Reserved)</span>
            <span class="text-xl font-bold font-mono text-foreground">{formatBytes(metrics?.HeapSysBytes)}</span>
          </div>
        </div>

        <div class="p-4 rounded-lg border border-border bg-muted/10 flex items-center justify-between text-xs">
          <div>
            <span class="font-semibold text-foreground block">Garbage Collector Heap Objects</span>
            <span class="text-muted-foreground block mt-0.5">Total live pointer-managed objects tracked by Go's concurrent tri-color GC.</span>
          </div>
          <span class="text-base font-bold font-mono text-foreground">{metrics?.HeapObjects?.toLocaleString() || 0}</span>
        </div>
      </div>

    <!-- Tab 3: Database & Cache -->
    {:else if activeTab === 'database'}
      <div class="space-y-4">
        <!-- Database Pool -->
        <div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-4">
          <h3 class="text-sm font-semibold text-foreground">SQL Connection Pool ({info?.dialect || 'SQLite'})</h3>
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
            <div class="p-3 rounded-lg border border-border bg-muted/20">
              <span class="text-[11px] text-muted-foreground block">In-Use Connections</span>
              <span class="text-lg font-bold font-mono text-foreground">{metrics?.Database?.InUseConnections || 0}</span>
            </div>
            <div class="p-3 rounded-lg border border-border bg-muted/20">
              <span class="text-[11px] text-muted-foreground block">Idle in Pool</span>
              <span class="text-lg font-bold font-mono text-foreground">{metrics?.Database?.IdleConnections || 0}</span>
            </div>
            <div class="p-3 rounded-lg border border-border bg-muted/20">
              <span class="text-[11px] text-muted-foreground block">Max Configured</span>
              <span class="text-lg font-bold font-mono text-foreground">{metrics?.Database?.MaxOpenConnections || 'Default'}</span>
            </div>
            <div class="p-3 rounded-lg border border-border bg-muted/20">
              <span class="text-[11px] text-muted-foreground block">Lock Wait Count</span>
              <span class="text-lg font-bold font-mono text-foreground">{metrics?.Database?.WaitCount || 0}</span>
            </div>
          </div>
        </div>

        <!-- Cache -->
        <div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-4">
          <h3 class="text-sm font-semibold text-foreground">Internal In-Memory Cache</h3>
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
            <div class="p-3 rounded-lg border border-border bg-muted/20">
              <span class="text-[11px] text-muted-foreground block">Cache Hits</span>
              <span class="text-lg font-bold font-mono text-emerald-600 dark:text-emerald-400">{metrics?.Cache?.Hits || 0}</span>
            </div>
            <div class="p-3 rounded-lg border border-border bg-muted/20">
              <span class="text-[11px] text-muted-foreground block">Cache Misses</span>
              <span class="text-lg font-bold font-mono text-foreground">{metrics?.Cache?.Misses || 0}</span>
            </div>
            <div class="p-3 rounded-lg border border-border bg-muted/20">
              <span class="text-[11px] text-muted-foreground block">Evictions</span>
              <span class="text-lg font-bold font-mono text-foreground">{metrics?.Cache?.Evictions || 0}</span>
            </div>
            <div class="p-3 rounded-lg border border-border bg-muted/20">
              <span class="text-[11px] text-muted-foreground block">Loads & Refreshes</span>
              <span class="text-lg font-bold font-mono text-foreground">{metrics?.Cache?.Loads || 0}</span>
            </div>
          </div>
        </div>
      </div>

    <!-- Tab 4: Async Workers & Jobs -->
    {:else if activeTab === 'jobs'}
      <div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-4">
        <h3 class="text-sm font-semibold text-foreground">Background Queue & Job Processor</h3>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <div class="p-3 rounded-lg border border-border bg-muted/20">
            <span class="text-[11px] text-muted-foreground block">Queue Depth</span>
            <span class="text-lg font-bold font-mono text-foreground">{metrics?.Jobs?.Depth || 0}</span>
          </div>
          <div class="p-3 rounded-lg border border-border bg-muted/20">
            <span class="text-[11px] text-muted-foreground block">Leased / Running</span>
            <span class="text-lg font-bold font-mono text-primary">{metrics?.Jobs?.Leased || 0}</span>
          </div>
          <div class="p-3 rounded-lg border border-border bg-muted/20">
            <span class="text-[11px] text-muted-foreground block">Dead Lettered</span>
            <span class="text-lg font-bold font-mono text-destructive">{metrics?.Jobs?.DeadLettered || 0}</span>
          </div>
          <div class="p-3 rounded-lg border border-border bg-muted/20">
            <span class="text-[11px] text-muted-foreground block">Worker Processed</span>
            <span class="text-lg font-bold font-mono text-foreground">{metrics?.WorkerProcessed || 0}</span>
          </div>
        </div>
      </div>

    <!-- Tab 5: Monitored Paths -->
    {:else if activeTab === 'paths'}
      <div class="rounded-xl border border-border bg-card overflow-hidden shadow-xs">
        <div class="p-4 border-b border-border">
          <h3 class="text-sm font-semibold text-foreground">Top Instrumented Request Paths</h3>
        </div>
        <div class="divide-y divide-border">
          {#if !metrics?.TopPaths || metrics.TopPaths.length === 0}
            <div class="p-8 text-center text-xs text-muted-foreground">
              No top path samples collected yet. Request traffic will populate this table automatically.
            </div>
          {:else}
            {#each metrics.TopPaths as p}
              <div class="p-4 flex items-center justify-between gap-4 hover:bg-muted/30 transition-colors">
                <div class="grid text-left min-w-0">
                  <span class="font-mono text-xs font-semibold text-foreground truncate">{p.Path}</span>
                  <span class="text-[11px] text-muted-foreground mt-0.5 font-mono">
                    {p.Requests} requests • Avg: {p.AvgMS}ms • P95: {p.P95MS}ms
                  </span>
                </div>
                <div class="flex items-center gap-2 font-mono text-xs">
                  <span class="text-emerald-600 dark:text-emerald-400">{p.Status2xx} 2xx</span>
                  {#if p.Errors > 0}
                    <span class="text-destructive font-semibold">{p.Errors} err</span>
                  {/if}
                </div>
              </div>
            {/each}
          {/if}
        </div>
      </div>
    {:else if activeTab === 'backups'}
      <BackupsSecurityTab />
    {/if}
  </div>
</div>
