<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import { auth } from '$lib/stores/auth.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Card from '$lib/components/ui/Card.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';

  interface DashboardData {
    stats?: Record<string, number>;
    recent_pages?: Array<{ id: string; title: string; updated_at: string; status: string }>;
    system?: {
      version: string;
      database: string;
      uptime_seconds: number;
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
</script>

<div class="space-y-8 max-w-6xl mx-auto">
  <!-- Greeting -->
  <div class="flex items-center justify-between pb-6 border-b border-slate-800/60">
    <div>
      <h2 class="text-2xl font-bold tracking-tight text-slate-100">
        Welcome back, <span class="text-sky-400">{auth.user?.email || 'Admin'}</span>
      </h2>
      <p class="text-sm text-slate-400 mt-1">Here is a quick overview of your content and system services.</p>
    </div>
    <div class="flex items-center gap-2">
      <Badge variant="success">System Operational</Badge>
    </div>
  </div>

  <!-- Metric Cards -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
    <Card class="p-4">
      <div class="flex items-center justify-between">
        <span class="text-xs font-medium text-slate-400 uppercase tracking-wider">Content Types</span>
        <span class="text-xl">📦</span>
      </div>
      <p class="text-2xl font-bold text-slate-100 mt-2">{data?.stats?.content_types_count ?? schema.resources.length}</p>
      <p class="text-xs text-slate-500 mt-1">{loading ? 'Loading…' : 'Active registered collections'}</p>
    </Card>

    <Card class="p-4">
      <div class="flex items-center justify-between">
        <span class="text-xs font-medium text-slate-400 uppercase tracking-wider">Active Modules</span>
        <span class="text-xl">⚙️</span>
      </div>
      <p class="text-2xl font-bold text-slate-100 mt-2">
        {Object.values(schema.modules).filter(Boolean).length}
      </p>
      <p class="text-xs text-slate-500 mt-1">Enabled system modules</p>
    </Card>

    <Card class="p-4">
      <div class="flex items-center justify-between">
        <span class="text-xs font-medium text-slate-400 uppercase tracking-wider">Languages</span>
        <span class="text-xl">🌐</span>
      </div>
      <p class="text-2xl font-bold text-slate-100 mt-2">{schema.locales.length}</p>
      <p class="text-xs text-slate-500 mt-1">Default: {schema.defaultLocale.toUpperCase()}</p>
    </Card>

    <Card class="p-4">
      <div class="flex items-center justify-between">
        <span class="text-xs font-medium text-slate-400 uppercase tracking-wider">Your Role</span>
        <span class="text-xl">🛡️</span>
      </div>
      <p class="text-2xl font-bold text-sky-400 mt-2 capitalize">{auth.user?.role || 'Admin'}</p>
      <p class="text-xs text-slate-500 mt-1">Full management access</p>
    </Card>
  </div>

  <!-- Content Collections Grid -->
  <div>
    <h3 class="text-base font-semibold text-slate-200 mb-4 flex items-center gap-2">
      <span>Content Collections</span>
    </h3>
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each schema.resources as col (col.id)}
        <a
          href={`#/content/${col.id}`}
          class="p-5 rounded-xl border border-slate-800 bg-slate-900/60 hover:bg-slate-900 hover:border-slate-700 transition-all duration-150 flex flex-col justify-between group"
        >
          <div>
            <div class="flex items-center justify-between">
              <span class="text-2xl mb-2">{col.icon || '📄'}</span>
              <span class="text-xs text-slate-500 group-hover:text-sky-400 transition-colors">Open →</span>
            </div>
            <h4 class="text-base font-semibold text-slate-100 group-hover:text-sky-400 transition-colors">
              {col.label || col.plural}
            </h4>
            <p class="text-xs text-slate-400 mt-1 line-clamp-2">
              {col.description || `Manage ${col.plural.toLowerCase()} content items.`}
            </p>
          </div>
          <div class="flex items-center gap-2 mt-4 pt-3 border-t border-slate-800/80 text-[11px] text-slate-500">
            <span>{col.fields.length} fields</span>
            {#if col.workflow?.enabled}
              <span>•</span>
              <span class="text-emerald-400">Drafts enabled</span>
            {/if}
          </div>
        </a>
      {/each}
    </div>
  </div>
</div>
