<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import { auth } from '$lib/stores/auth.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Card from '$lib/components/ui/Card.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Button from '$lib/components/ui/Button.svelte';

  interface DashboardData {
    user: { id: string; email: string; role: string };
    modules: Record<string, boolean>;
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
    broken_links?: {
      active: boolean;
      job_id?: string;
    };
  }

  let data = $state<DashboardData | null>(null);
  let loading = $state(true);

  onMount(async () => {
    try {
      const res = await api.get<ApiResponse<DashboardData>>('/api/v1/admin/dashboard');
      if (res && res.data) {
        data = res.data;
      }
    } catch (e) {
      console.warn('Dashboard load error:', e);
    } finally {
      loading = false;
    }
  });

  function formatUptime(seconds?: number): string {
    if (!seconds || seconds <= 0) return '0s';
    const d = Math.floor(seconds / 86400);
    const h = Math.floor((seconds % 86400) / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    if (d > 0) return `${d}d ${h}h`;
    if (h > 0) return `${h}h ${m}m`;
    return `${m}m ${seconds % 60}s`;
  }
</script>

<div class="space-y-8 max-w-7xl mx-auto">
  <!-- Hero Section -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-6 border-b border-border">
    <div class="space-y-1">
      <div class="flex items-center gap-2.5">
        <h1 class="text-2xl font-bold tracking-tight text-foreground">
          Welcome back, <span class="text-primary">{auth.user?.email || 'Admin'}</span>
        </h1>
        <Badge variant="primary" class="capitalize font-mono text-[10px]">
          {auth.user?.role || 'editor'}
        </Badge>
      </div>
      <p class="text-sm text-muted-foreground">
        Real-time telemetry and content operations for your single-binary Go headless CMS.
      </p>
    </div>

    <!-- Quick action buttons -->
    <div class="flex items-center gap-2.5 flex-wrap">
      {#if schema.modules['pages'] !== false}
        <a href="#/pages/new">
          <Button variant="outline" size="sm" class="gap-1.5 shadow-sm">
            <svg class="w-3.5 h-3.5 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <line x1="12" y1="5" x2="12" y2="19"></line>
              <line x1="5" y1="12" x2="19" y2="12"></line>
            </svg>
            <span>New Page</span>
          </Button>
        </a>
      {/if}
      {#if schema.modules['news'] !== false}
        <a href="#/news/new">
          <Button variant="primary" size="sm" class="gap-1.5 shadow-sm">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <line x1="12" y1="5" x2="12" y2="19"></line>
              <line x1="5" y1="12" x2="19" y2="12"></line>
            </svg>
            <span>Write Article</span>
          </Button>
        </a>
      {/if}
    </div>
  </div>

  <!-- Broken Links Alert Notice (if triggered) -->
  {#if data?.broken_links?.active}
    <div class="rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 flex items-center justify-between text-amber-300">
      <div class="flex items-center gap-3">
        <svg class="w-5 h-5 text-amber-400 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="12" y1="8" x2="12" y2="12"></line>
          <line x1="12" y1="16" x2="12.01" y2="16"></line>
        </svg>
        <span class="text-xs sm:text-sm font-medium">Potential broken links detected during the last background scan.</span>
      </div>
      <a href="#/system" class="text-xs font-semibold underline underline-offset-2 hover:text-amber-200">
        Inspect Scan →
      </a>
    </div>
  {/if}

  <!-- 4-Column shadcn Blue Metric Cards -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
    <!-- Published Pages -->
    <Card class="hover:border-primary/40 transition-colors">
      <div class="flex items-center justify-between">
        <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">Published Pages</span>
        <div class="w-8 h-8 rounded-lg bg-blue-500/15 border border-blue-500/30 flex items-center justify-center text-primary">
          <svg class="w-4 h-4 text-current" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path>
            <polyline points="14 2 14 8 20 8"></polyline>
          </svg>
        </div>
      </div>
      <div class="mt-2">
        <span class="text-2xl font-bold tracking-tight text-foreground">
          {#if loading}
            <span class="animate-pulse text-muted-foreground">...</span>
          {:else}
            {data?.content?.pages_published ?? 0}
          {/if}
        </span>
        <p class="text-xs text-muted-foreground mt-1 flex items-center gap-1">
          <span class="text-emerald-400 font-medium">Live</span> on public API
        </p>
      </div>
    </Card>

    <!-- Published News -->
    <Card class="hover:border-primary/40 transition-colors">
      <div class="flex items-center justify-between">
        <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">News & Articles</span>
        <div class="w-8 h-8 rounded-lg bg-blue-500/15 border border-blue-500/30 flex items-center justify-center text-primary">
          <svg class="w-4 h-4 text-current" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"></path>
            <path d="M18 14h-8"></path>
            <path d="M15 18h-5"></path>
            <path d="M10 6h8v4h-8V6Z"></path>
          </svg>
        </div>
      </div>
      <div class="mt-2">
        <span class="text-2xl font-bold tracking-tight text-foreground">
          {#if loading}
            <span class="animate-pulse text-muted-foreground">...</span>
          {:else}
            {data?.content?.news_published ?? 0}
          {/if}
        </span>
        <p class="text-xs text-muted-foreground mt-1">
          Broadcast items released
        </p>
      </div>
    </Card>

    <!-- Total Members -->
    <Card class="hover:border-primary/40 transition-colors">
      <div class="flex items-center justify-between">
        <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">Community Members</span>
        <div class="w-8 h-8 rounded-lg bg-blue-500/15 border border-blue-500/30 flex items-center justify-center text-primary">
          <svg class="w-4 h-4 text-current" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"></path>
            <circle cx="9" cy="7" r="4"></circle>
            <path d="M22 21v-2a4 4 0 0 0-3-3.87"></path>
            <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
          </svg>
        </div>
      </div>
      <div class="mt-2">
        <span class="text-2xl font-bold tracking-tight text-foreground">
          {#if loading}
            <span class="animate-pulse text-muted-foreground">...</span>
          {:else}
            {data?.members?.total_members ?? 0}
          {/if}
        </span>
        <p class="text-xs text-muted-foreground mt-1 flex items-center gap-1.5">
          <span class="text-primary font-medium">{data?.members?.active_members ?? 0} active</span>
          <span>•</span>
          <span>{data?.members?.verified_members ?? 0} verified</span>
        </p>
      </div>
    </Card>

    <!-- Go Server Telemetry -->
    <Card class="hover:border-primary/40 transition-colors">
      <div class="flex items-center justify-between">
        <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">Go Daemon Uptime</span>
        <div class="w-8 h-8 rounded-lg bg-emerald-500/15 border border-emerald-500/30 flex items-center justify-center text-emerald-400">
          <svg class="w-4 h-4 text-current" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 12h-4l-3 9L9 3l-3 9H2"></path>
          </svg>
        </div>
      </div>
      <div class="mt-2">
        <span class="text-2xl font-bold tracking-tight text-emerald-400 font-mono">
          {#if loading}
            <span class="animate-pulse text-muted-foreground">...</span>
          {:else}
            {formatUptime(data?.runtime?.uptime_seconds)}
          {/if}
        </span>
        <p class="text-xs text-muted-foreground mt-1 flex items-center gap-1.5">
          <span>Latency: <strong class="text-foreground font-mono">{data?.runtime?.avg_latency_ms ?? 0}ms</strong></span>
          <span>•</span>
          <span class="text-emerald-400">0 critical</span>
        </p>
      </div>
    </Card>
  </div>

  <!-- Content Collections Grid & Quick Overview -->
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-base font-semibold tracking-tight text-foreground">Content Collections</h2>
        <p class="text-xs text-muted-foreground">Dynamic schema-driven resources managed through YAML definitions.</p>
      </div>
      <Badge variant="outline" class="font-mono text-xs">
        {schema.resources.length} active types
      </Badge>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each schema.resources as col (col.id)}
        <a
          href={`#/content/${col.id}`}
          class="rounded-xl border border-border bg-card p-5 hover:border-primary/60 hover:shadow-md transition-all flex flex-col justify-between group cursor-pointer"
        >
          <div>
            <div class="flex items-center justify-between mb-3">
              <span class="w-9 h-9 rounded-lg bg-muted border border-border/80 flex items-center justify-center text-lg">
                {col.icon || '📦'}
              </span>
              <span class="text-xs font-medium text-muted-foreground group-hover:text-primary transition-colors flex items-center gap-1">
                Open Collection
                <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="9 18 15 12 9 6"></polyline>
                </svg>
              </span>
            </div>
            <h3 class="text-base font-semibold text-foreground group-hover:text-primary transition-colors">
              {col.label || col.plural}
            </h3>
            <p class="text-xs text-muted-foreground mt-1 line-clamp-2">
              {col.description || `Manage and publish ${col.plural.toLowerCase()} content.`}
            </p>
          </div>

          <div class="flex items-center justify-between mt-5 pt-3 border-t border-border/60 text-xs text-muted-foreground">
            <span class="font-mono text-[11px]">{col.fields.length} schema fields</span>
            {#if col.workflow?.enabled}
              <Badge variant="success" size="sm">Drafts</Badge>
            {:else}
              <Badge variant="secondary" size="sm">Direct</Badge>
            {/if}
          </div>
        </a>
      {/each}
    </div>
  </div>

  <!-- Operational Telemetry Overview -->
  <Card title="System Telemetry & Architecture" description="Single-binary persistent daemon with pooled SQLite/Postgres/MySQL engine.">
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4 py-2">
      <div class="rounded-lg bg-muted/40 p-3.5 border border-border/60">
        <span class="text-xs text-muted-foreground uppercase font-mono">Total Requests</span>
        <p class="text-xl font-bold font-mono text-foreground mt-1">{data?.runtime?.requests ?? 0}</p>
      </div>
      <div class="rounded-lg bg-muted/40 p-3.5 border border-border/60">
        <span class="text-xs text-muted-foreground uppercase font-mono">Content Views</span>
        <p class="text-xl font-bold font-mono text-foreground mt-1">{data?.runtime?.views ?? 0}</p>
      </div>
      <div class="rounded-lg bg-muted/40 p-3.5 border border-border/60">
        <span class="text-xs text-muted-foreground uppercase font-mono">Server Errors</span>
        <p class="text-xl font-bold font-mono text-emerald-400 mt-1">{data?.runtime?.errors ?? 0}</p>
      </div>
      <div class="rounded-lg bg-muted/40 p-3.5 border border-border/60">
        <span class="text-xs text-muted-foreground uppercase font-mono">Avg Latency</span>
        <p class="text-xl font-bold font-mono text-primary mt-1">{data?.runtime?.avg_latency_ms ?? 0} ms</p>
      </div>
    </div>
  </Card>
</div>
