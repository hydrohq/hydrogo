<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';

  interface AuditEntry {
    id: string;
    at: string;
    actor: string;
    actor_id?: string;
    action: string;
    resource: string;
    target: string;
    request_id?: string;
    before_json?: string;
    after_json?: string;
  }

  interface NotFoundHit {
    path: string;
    hits: number;
    first_seen: string;
    last_seen: string;
  }

  interface RequestMetadata {
    ip?: string;
    user_agent?: string;
    method?: string;
    path?: string;
    query?: string;
    rule?: string;
    status?: number;
    reason?: string;
    attempted_email?: string;
    user_id?: string;
    email?: string;
    role?: string;
    body?: unknown;
  }

  let activeTab = $state<'events' | 'security' | 'not_found'>('events');
  let entries = $state<AuditEntry[]>([]);
  let notFoundHits = $state<NotFoundHit[]>([]);
  let total = $state(0);
  let limit = $state(25);
  let offset = $state(0);
  let loading = $state(true);
  let notFoundLoading = $state(false);

  // Filters
  let filterActor = $state('');
  let filterAction = $state('');
  let filterResource = $state('');
  let filterSearch = $state('');
  let selectedEntry = $state<AuditEntry | null>(null);
  let copiedId = $state<string | null>(null);

  // Security stats derived from audit data
  const securityStats = $derived.by(() => {
    let logins = 0;
    let failedLogins = 0;
    let anomalies = 0;

    for (const e of entries) {
      if (e.action === 'auth.login') logins++;
      else if (e.action === 'auth.login_failed') failedLogins++;
      else if (e.action === 'security.request_rejected') anomalies++;
    }

    return {
      logins,
      failedLogins,
      anomalies,
      totalSecurity: logins + failedLogins + anomalies,
    };
  });

  // Filtered entries for Security Tab
  const securityEntries = $derived.by(() => {
    return entries.filter((e) =>
      e.action.startsWith('auth.') ||
      e.action.startsWith('security.') ||
      e.resource === 'auth' ||
      e.resource === 'security'
    );
  });

  // Filtered 404 paths
  const filteredNotFoundHits = $derived.by(() => {
    if (!filterSearch.trim()) return notFoundHits;
    const q = filterSearch.toLowerCase();
    return notFoundHits.filter((h) => h.path.toLowerCase().includes(q));
  });

  async function loadLogs() {
    loading = true;
    try {
      const params = new URLSearchParams({
        limit: String(limit),
        offset: String(offset),
      });
      if (filterActor.trim()) params.set('actor', filterActor.trim());
      if (filterAction.trim()) params.set('action', filterAction.trim());
      if (filterResource.trim()) params.set('resource', filterResource.trim());

      const res = await api.get<ApiResponse<AuditEntry[]>>(`/api/v1/admin/audit?${params.toString()}`);
      if (res && res.data) {
        entries = Array.isArray(res.data) ? res.data : [];
        total = res.metadata?.total ?? res.meta?.total ?? 0;
      }
    } catch (err: any) {
      console.warn('Failed to load audit logs:', err);
    } finally {
      loading = false;
    }
  }

  async function loadNotFound() {
    notFoundLoading = true;
    try {
      const res = await api.get<ApiResponse<NotFoundHit[]>>('/api/v1/admin/site/not-found?limit=200');
      if (res && res.data) {
        notFoundHits = Array.isArray(res.data) ? res.data : [];
      }
    } catch (err: any) {
      console.warn('Failed to load 404 probed routes:', err);
    } finally {
      notFoundLoading = false;
    }
  }

  async function convertToRedirect(path: string) {
    try {
      await api.post('/api/v1/admin/site/not-found/convert', { path, target: '/' });
      await loadNotFound();
    } catch (err: any) {
      console.error('Failed to convert 404 into redirect:', err);
    }
  }

  async function dismissNotFound(path: string) {
    try {
      await api.delete('/api/v1/admin/site/not-found', { body: JSON.stringify({ path }) });
      notFoundHits = notFoundHits.filter((h) => h.path !== path);
    } catch (err: any) {
      console.error('Failed to dismiss not-found route:', err);
    }
  }

  function handleFilter(e: SubmitEvent) {
    e.preventDefault();
    offset = 0;
    loadLogs();
  }

  function formatDate(iso: string): string {
    try {
      return new Date(iso).toLocaleString();
    } catch {
      return iso;
    }
  }

  function parseMetadata(jsonStr?: string): RequestMetadata | null {
    if (!jsonStr || jsonStr === '{}') return null;
    try {
      return JSON.parse(jsonStr);
    } catch {
      return null;
    }
  }

  function isScannerProbe(path: string): boolean {
    const p = path.toLowerCase();
    const probeTokens = [
      'wp-', 'wordpress', '.env', '.git', 'config.', 'phpmyadmin',
      '.php', 'xmlrpc', 'actuator', 'swagger', 'api/v', 'shell',
      'boaform', 'cgi-bin', 'setup', 'backup', 'dump', 'eval', 'login'
    ];
    return probeTokens.some((token) => p.includes(token));
  }

  function actionBadge(action: string) {
    if (action === 'auth.login') {
      return { variant: 'success' as const, label: t('audit.signInSuccess') };
    }
    if (action === 'auth.login_failed') {
      return { variant: 'destructive' as const, label: t('audit.signInFailed') };
    }
    if (action === 'auth.mfa_failed') {
      return { variant: 'warning' as const, label: t('audit.mfaFailed') };
    }
    if (action === 'security.request_rejected') {
      return { variant: 'destructive' as const, label: t('audit.blocked') };
    }
    if (action === 'admin.request') {
      return { variant: 'outline' as const, label: t('audit.adminRequest') };
    }
    if (action.includes('delete') || action.includes('revoke')) {
      return { variant: 'destructive' as const, label: action };
    }
    if (action.includes('create') || action.includes('publish')) {
      return { variant: 'success' as const, label: action };
    }
    if (action.includes('update') || action.includes('edit')) {
      return { variant: 'warning' as const, label: action };
    }
    return { variant: 'outline' as const, label: action };
  }

  async function copyText(text: string, id: string) {
    try {
      await navigator.clipboard.writeText(text);
      copiedId = id;
      setTimeout(() => {
        if (copiedId === id) copiedId = null;
      }, 2000);
    } catch (e) {
      console.warn('Copy failed:', e);
    }
  }

  onMount(() => {
    loadLogs();
    loadNotFound();
  });
</script>

<div class="flex flex-1 flex-col font-sans select-text">
  <div class="@container/main flex flex-1 flex-col gap-2">
    <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">

      <!-- Top Security Posture & Metrics Overview Cards -->
      <div class="grid grid-cols-1 gap-4 px-4 sm:grid-cols-2 lg:grid-cols-4 lg:px-6">
        <!-- Posture Card -->
        <div class="rounded-xl border border-border bg-gradient-to-t from-primary/5 to-card dark:bg-card p-5 shadow-xs text-card-foreground relative">
          <div class="flex items-center justify-between pb-2">
            <span class="text-xs font-medium text-muted-foreground">Security Posture</span>
            <span class="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 border border-emerald-500/20 px-2 py-0.5 text-[10px] font-medium text-emerald-600 dark:text-emerald-400">
              <span class="size-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
              Protected
            </span>
          </div>
          <div class="space-y-1">
            <div class="text-xl font-bold tracking-tight text-foreground">Zero Compromise</div>
            <p class="text-xs text-muted-foreground">Rate limiters, tarpit and anomaly firewall active</p>
          </div>
        </div>

        <!-- Logins Metric -->
        <div class="rounded-xl border border-border bg-card p-5 shadow-xs text-card-foreground">
          <div class="flex items-center justify-between pb-2">
            <span class="text-xs font-medium text-muted-foreground">Admin Sign-in Traffic</span>
            <span class="text-xs font-mono text-muted-foreground">{total} events</span>
          </div>
          <div class="flex items-baseline gap-2">
            <span class="text-xl font-bold text-foreground tabular-nums">{securityStats.logins}</span>
            <span class="text-xs text-emerald-600 font-medium">Valid</span>
            {#if securityStats.failedLogins > 0}
              <span class="text-xs text-muted-foreground">/</span>
              <span class="text-xs font-semibold text-destructive">{securityStats.failedLogins} Failed</span>
            {/if}
          </div>
          <p class="text-xs text-muted-foreground mt-1">Staff session lifecycle logs</p>
        </div>

        <!-- Blocked Anomalies Metric -->
        <div class="rounded-xl border border-border bg-card p-5 shadow-xs text-card-foreground">
          <div class="flex items-center justify-between pb-2">
            <span class="text-xs font-medium text-muted-foreground">Security Blocks</span>
            <svg class="size-4 text-amber-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>
            </svg>
          </div>
          <div class="text-xl font-bold text-foreground tabular-nums">{securityStats.anomalies}</div>
          <p class="text-xs text-muted-foreground mt-1">Suspicious requests & injection probes dropped</p>
        </div>

        <!-- 404 Probed Routes Metric -->
        <div class="rounded-xl border border-border bg-card p-5 shadow-xs text-card-foreground">
          <div class="flex items-center justify-between pb-2">
            <span class="text-xs font-medium text-muted-foreground">Probed 404 Routes</span>
            <span class="text-xs font-mono text-muted-foreground">{notFoundHits.length} routes</span>
          </div>
          <div class="text-xl font-bold text-foreground tabular-nums">
            {notFoundHits.reduce((acc, h) => acc + h.hits, 0)}
          </div>
          <p class="text-xs text-muted-foreground mt-1">Scanner bot hits on non-existent endpoints</p>
        </div>
      </div>

      <!-- Segmented Navigation Tabs -->
      <div class="px-4 lg:px-6">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-border pb-3">
          <div class="inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground">
            <button
              type="button"
              onclick={() => (activeTab = 'events')}
              class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {activeTab === 'events' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
            >
              {t('audit.activity', { count: total })}
            </button>
            <button
              type="button"
              onclick={() => (activeTab = 'security')}
              class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {activeTab === 'security' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
            >
              {t('audit.security', { count: securityEntries.length })}
            </button>
            <button
              type="button"
              onclick={() => (activeTab = 'not_found')}
              class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {activeTab === 'not_found' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
            >
              {t('audit.scanner', { count: notFoundHits.length })}
            </button>
          </div>

          <button
            type="button"
            onclick={() => { loadLogs(); loadNotFound(); }}
            class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer self-start sm:self-auto"
            title={t('audit.refreshTitle')}
          >
            <svg class="size-3.5 {loading || notFoundLoading ? 'animate-spin' : ''}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"></path>
              <path d="M3 3v5h5"></path>
              <path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"></path>
              <path d="M16 21h5v-5"></path>
            </svg>
            <span>{t('audit.refresh')}</span>
          </button>
        </div>
      </div>

      <!-- TAB 1: Activity Stream -->
      {#if activeTab === 'events'}
        <!-- Filters Row -->
        <div class="px-4 lg:px-6">
          <form onsubmit={handleFilter} class="p-3 rounded-xl border border-border bg-card shadow-xs grid grid-cols-1 sm:grid-cols-4 gap-3 items-end">
            <Input label={t('audit.filter.actor')} placeholder={t('audit.filter.actorPlaceholder')} bind:value={filterActor} />
            <Input label={t('audit.filter.action')} placeholder="e.g. create, update, delete..." bind:value={filterAction} />
            <Input label={t('audit.filter.resource')} placeholder="e.g. pages, auth, site..." bind:value={filterResource} />
            <div class="flex gap-2">
              <button
                type="submit"
                class="h-9 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs transition-colors cursor-pointer"
              >
                Filter
              </button>
              <button
                type="button"
                class="h-9 px-4 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer"
                onclick={() => {
                  filterActor = '';
                  filterAction = '';
                  filterResource = '';
                  offset = 0;
                  loadLogs();
                }}
              >
                Reset
              </button>
            </div>
          </form>
        </div>

        <!-- Table Container -->
        <div class="px-4 lg:px-6">
          <div class="overflow-hidden rounded-xl border border-border bg-card shadow-xs">
            {#if loading}
              <div class="p-12 text-center text-xs text-muted-foreground animate-pulse">{t('audit.loading')}</div>
            {:else if entries.length === 0}
              <div class="p-12 text-center text-xs text-muted-foreground">{t('audit.empty')}</div>
            {:else}
              <div class="overflow-x-auto">
                <table class="w-full text-left text-sm text-foreground caption-bottom">
                  <thead class="sticky top-0 z-10 bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
                    <tr>
                      <th class="h-10 px-4 font-medium">{t('audit.col.timestamp')}</th>
                      <th class="h-10 px-4 font-medium">{t('audit.col.actor')}</th>
                      <th class="h-10 px-4 font-medium">{t('audit.col.action')}</th>
                      <th class="h-10 px-4 font-medium">{t('audit.col.resource')}</th>
                      <th class="h-10 px-4 font-medium">{t('audit.col.target')}</th>
                      <th class="h-10 w-24 px-4 font-medium text-right">{t('audit.col.details')}</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-border">
                    {#each entries as entry (entry.id)}
                      {@const badgeInfo = actionBadge(entry.action)}
                      <tr class="hover:bg-muted/40 transition-colors">
                        <td class="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">{formatDate(entry.at)}</td>
                        <td class="px-4 py-3 font-medium text-xs text-foreground">{entry.actor}</td>
                        <td class="px-4 py-3">
                          <Badge variant={badgeInfo.variant} size="sm">{badgeInfo.label}</Badge>
                        </td>
                        <td class="px-4 py-3 font-mono text-xs text-muted-foreground">{entry.resource}</td>
                        <td class="px-4 py-3 font-mono text-xs text-muted-foreground truncate max-w-[160px]" title={entry.target}>
                          {entry.target}
                        </td>
                        <td class="px-4 py-3 text-right">
                          <button
                            type="button"
                            class="text-xs text-primary hover:underline font-medium cursor-pointer"
                            onclick={() => { selectedEntry = entry; }}
                          >
                            Inspect
                          </button>
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>

              <!-- Pagination -->
              <div class="flex items-center justify-between px-4 py-3 border-t border-border text-xs text-muted-foreground">
                <div>
                  Showing {offset + 1}–{Math.min(offset + entries.length, total)} of {total}
                </div>
                <div class="flex gap-2">
                  <button
                    type="button"
                    disabled={offset === 0}
                    onclick={() => { offset = Math.max(0, offset - limit); loadLogs(); }}
                    class="h-7 px-2.5 rounded-md border border-input bg-background text-xs font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:bg-accent transition-colors"
                  >
                    Previous
                  </button>
                  <button
                    type="button"
                    disabled={offset + entries.length >= total}
                    onclick={() => { offset += limit; loadLogs(); }}
                    class="h-7 px-2.5 rounded-md border border-input bg-background text-xs font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:bg-accent transition-colors"
                  >
                    Next
                  </button>
                </div>
              </div>
            {/if}
          </div>
        </div>
      {/if}

      <!-- TAB 2: Security & Sign-in Attempts -->
      {#if activeTab === 'security'}
        <div class="px-4 lg:px-6">
          <div class="overflow-hidden rounded-xl border border-border bg-card shadow-xs">
            <div class="p-4 border-b border-border bg-muted/20 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
              <div>
                <h3 class="text-sm font-semibold text-foreground">Security & Access Audit Stream</h3>
                <p class="text-xs text-muted-foreground">Monitoring login attempts, failed credentials, and blocked vulnerability probes</p>
              </div>
              <span class="text-xs font-mono text-muted-foreground">{securityEntries.length} security events</span>
            </div>

            {#if securityEntries.length === 0}
              <div class="p-12 text-center text-xs text-muted-foreground">
                No security anomalies or sign-in events recorded.
              </div>
            {:else}
              <div class="overflow-x-auto">
                <table class="w-full text-left text-sm text-foreground">
                  <thead class="bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
                    <tr>
                      <th class="h-10 px-4 font-medium">{t('audit.col.timestamp')}</th>
                      <th class="h-10 px-4 font-medium">Event / Outcome</th>
                      <th class="h-10 px-4 font-medium">Actor / Target</th>
                      <th class="h-10 px-4 font-medium">Client Info</th>
                      <th class="h-10 px-4 font-medium">Request ID</th>
                      <th class="h-10 w-24 px-4 font-medium text-right">Inspect</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-border">
                    {#each securityEntries as entry (entry.id)}
                      {@const meta = parseMetadata(entry.after_json)}
                      {@const badgeInfo = actionBadge(entry.action)}
                      <tr class="hover:bg-muted/40 transition-colors">
                        <td class="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">{formatDate(entry.at)}</td>
                        <td class="px-4 py-3">
                          <Badge variant={badgeInfo.variant} size="sm">{badgeInfo.label}</Badge>
                        </td>
                        <td class="px-4 py-3 text-xs">
                          {#if meta?.attempted_email}
                            <span class="font-medium text-destructive">{meta.attempted_email}</span>
                          {:else if meta?.email}
                            <span class="font-medium text-foreground">{meta.email}</span>
                          {:else}
                            <span class="font-mono text-muted-foreground">{entry.actor}</span>
                          {/if}
                        </td>
                        <td class="px-4 py-3 text-xs text-muted-foreground font-mono">
                          {#if meta?.ip}
                            <span>{meta.ip}</span>
                            {#if meta?.method}
                              <span class="ml-1 text-[10px] px-1 rounded bg-muted text-foreground uppercase">{meta.method}</span>
                            {/if}
                          {:else}
                            <span class="truncate max-w-[140px] inline-block">{entry.target}</span>
                          {/if}
                        </td>
                        <td class="px-4 py-3 font-mono text-[11px] text-muted-foreground">
                          {entry.request_id ? entry.request_id.slice(0, 12) + '…' : '—'}
                        </td>
                        <td class="px-4 py-3 text-right">
                          <button
                            type="button"
                            class="text-xs text-primary hover:underline font-medium cursor-pointer"
                            onclick={() => { selectedEntry = entry; }}
                          >
                            Details
                          </button>
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            {/if}
          </div>
        </div>
      {/if}

      <!-- TAB 3: 404 Scanner & Probing Log -->
      {#if activeTab === 'not_found'}
        <div class="px-4 lg:px-6 space-y-4">
          <!-- Search input -->
          <div class="flex flex-col sm:flex-row items-center justify-between gap-3">
            <div class="relative w-full max-w-sm">
              <input
                type="text"
                placeholder="Filter probed paths (e.g. /wp-admin, .env)..."
                bind:value={filterSearch}
                class="w-full h-9 pl-9 pr-3 rounded-md border border-input bg-background text-xs text-foreground placeholder:text-muted-foreground focus:outline-hidden focus:ring-1 focus:ring-ring"
              />
              <svg class="size-4 absolute left-3 top-2.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"></circle>
                <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
              </svg>
            </div>
            <span class="text-xs text-muted-foreground">
              Showing {filteredNotFoundHits.length} of {notFoundHits.length} scanned paths
            </span>
          </div>

          <div class="overflow-hidden rounded-xl border border-border bg-card shadow-xs">
            {#if notFoundLoading}
              <div class="p-12 text-center text-xs text-muted-foreground animate-pulse">Loading 404 probe logs...</div>
            {:else if filteredNotFoundHits.length === 0}
              <div class="p-12 text-center text-xs text-muted-foreground">
                No non-existent paths probed by visitors or bots.
              </div>
            {:else}
              <div class="overflow-x-auto">
                <table class="w-full text-left text-sm text-foreground">
                  <thead class="bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
                    <tr>
                      <th class="h-10 px-4 font-medium">Probed Path</th>
                      <th class="h-10 px-4 font-medium">Threat Classification</th>
                      <th class="h-10 px-4 font-medium">Hits</th>
                      <th class="h-10 px-4 font-medium">First Seen</th>
                      <th class="h-10 px-4 font-medium">Last Seen</th>
                      <th class="h-10 px-4 font-medium text-right">Protection Action</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-border">
                    {#each filteredNotFoundHits as hit}
                      {@const scanner = isScannerProbe(hit.path)}
                      <tr class="hover:bg-muted/40 transition-colors">
                        <td class="px-4 py-3 font-mono text-xs font-semibold text-foreground">
                          {hit.path}
                        </td>
                        <td class="px-4 py-3">
                          {#if scanner}
                            <Badge variant="destructive" size="sm">Vulnerability Scanner</Badge>
                          {:else}
                            <Badge variant="secondary" size="sm">404 Route Miss</Badge>
                          {/if}
                        </td>
                        <td class="px-4 py-3 font-mono text-xs tabular-nums font-semibold text-foreground">
                          {hit.hits}
                        </td>
                        <td class="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">{formatDate(hit.first_seen)}</td>
                        <td class="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">{formatDate(hit.last_seen)}</td>
                        <td class="px-4 py-3 text-right flex items-center justify-end gap-2">
                          <button
                            type="button"
                            onclick={() => convertToRedirect(hit.path)}
                            class="h-7 px-2.5 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer"
                            title="Redirect to safe home page"
                          >
                            Redirect /
                          </button>
                          <button
                            type="button"
                            onclick={() => dismissNotFound(hit.path)}
                            class="h-7 px-2 rounded-md hover:bg-destructive/10 text-xs text-destructive transition-colors cursor-pointer"
                            title="Dismiss record"
                          >
                            Dismiss
                          </button>
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            {/if}
          </div>
        </div>
      {/if}

    </div>
  </div>
</div>

<!-- Detailed Request Inspector Modal -->
{#if selectedEntry}
  {@const meta = parseMetadata(selectedEntry.after_json)}
  <Modal
    open={true}
    title={t('audit.inspector')}
    description={t('audit.inspectorHelp')}
    size="2xl"
    onclose={() => { selectedEntry = null; }}
  >
    <div class="space-y-5 text-xs font-sans">
      <!-- High-level Summary Banner -->
      <div class="p-4 rounded-xl border border-border bg-muted/30 grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <span class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider block">{t('audit.col.action')}</span>
          <span class="text-sm font-bold text-foreground font-mono">{selectedEntry.action}</span>
        </div>
        <div>
          <span class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider block">{t('audit.col.timestamp')}</span>
          <span class="text-xs text-foreground font-mono">{formatDate(selectedEntry.at)}</span>
        </div>
        <div>
          <span class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider block">{t('audit.actorIdentity')}</span>
          <span class="text-xs text-foreground font-semibold">
            {meta?.email || meta?.attempted_email || selectedEntry.actor}
          </span>
        </div>
        <div>
          <span class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider block">{t('audit.targetResource')}</span>
          <span class="text-xs font-mono text-muted-foreground">{selectedEntry.target}</span>
        </div>
      </div>

      <!-- Request Telemetry (IP, Method, Path, Agent) -->
      <div class="rounded-xl border border-border p-4 bg-card space-y-3">
        <div class="flex items-center justify-between border-b border-border pb-2">
          <span class="text-xs font-semibold text-foreground uppercase tracking-wider">{t('audit.requestTelemetry')}</span>
          {#if selectedEntry.request_id}
            <button
              type="button"
              onclick={() => copyText(selectedEntry?.request_id || '', 'req-id')}
              class="inline-flex items-center gap-1 text-[11px] font-mono text-primary hover:underline cursor-pointer"
            >
              <span>{copiedId === 'req-id' ? '✓ ' : ''}{t('audit.copyRequestId')}</span>
            </button>
          {/if}
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
          <div>
            <span class="text-muted-foreground block text-[11px]">{t('audit.clientIp')}</span>
            <span class="font-mono text-foreground font-medium">
              {meta?.ip || (selectedEntry.target.includes('from') ? selectedEntry.target.split('from')[1].trim() : 'Server Internal / Loopback')}
            </span>
          </div>
          <div>
            <span class="text-muted-foreground block text-[11px]">{t('audit.httpRequest')}</span>
            <span class="font-mono text-foreground font-medium">
              {#if meta?.method}
                <span class="px-1.5 py-0.5 rounded bg-primary/10 text-primary font-bold text-[10px] mr-1">{meta.method}</span>
              {/if}
              {meta?.path || selectedEntry.resource}
            </span>
          </div>
          <div>
            <span class="text-muted-foreground block text-[11px]">{t('audit.status')}</span>
            <span class="font-medium text-foreground">
              {#if meta?.status}
                <Badge variant={meta.status < 400 ? 'success' : 'destructive'} size="sm">
                  {meta.status} {meta.reason || (meta.status === 200 ? 'OK' : 'Rejected')}
                </Badge>
              {:else}
                <Badge variant={actionBadge(selectedEntry.action).variant} size="sm">
                  {selectedEntry.action}
                </Badge>
              {/if}
            </span>
          </div>
          <div>
            <span class="text-muted-foreground block text-[11px]">{t('audit.trackingId')}</span>
            <span class="font-mono text-[11px] text-muted-foreground select-all break-all">
              {selectedEntry.request_id || selectedEntry.id}
            </span>
          </div>
        </div>

        {#if meta?.user_agent}
          <div class="pt-2 border-t border-border">
            <span class="text-muted-foreground block text-[11px] mb-1">{t('audit.userAgent')}</span>
            <div class="p-2 rounded-md bg-muted text-[11px] font-mono text-muted-foreground break-all select-all">
              {meta.user_agent}
            </div>
          </div>
        {/if}
      </div>

      <!-- State Changes (Before / After JSON) -->
      {#if selectedEntry.action === 'admin.request'}
        <div class="rounded-xl border border-border p-4 bg-card space-y-2">
          <span class="text-xs font-semibold text-foreground uppercase tracking-wider">{t('audit.requestContent')}</span>
          {#if meta?.body !== undefined}
            <pre class="p-3 bg-muted rounded-xl text-foreground font-mono text-[11px] overflow-x-auto whitespace-pre-wrap">{typeof meta.body === 'string' ? meta.body : JSON.stringify(meta.body, null, 2)}</pre>
          {:else}
            <p class="text-[11px] text-muted-foreground">{t('audit.requestContentOff')}</p>
          {/if}
        </div>
      {/if}

      {#if selectedEntry.before_json && selectedEntry.before_json !== '{}'}
        <div>
          <div class="flex items-center justify-between mb-1.5">
            <span class="text-xs font-semibold text-destructive uppercase">{t('audit.before')}</span>
            <button
              type="button"
              onclick={() => copyText(selectedEntry?.before_json || '', 'before-json')}
              class="text-[11px] text-muted-foreground hover:text-foreground cursor-pointer"
            >
              {copiedId === 'before-json' ? '✓ Copied' : 'Copy'}
            </button>
          </div>
          <pre class="p-3 bg-muted rounded-xl text-foreground font-mono text-[11px] overflow-x-auto whitespace-pre-wrap">{selectedEntry.before_json}</pre>
        </div>
      {/if}

      {#if selectedEntry.after_json && selectedEntry.after_json !== '{}'}
        <div>
          <div class="flex items-center justify-between mb-1.5">
            <span class="text-xs font-semibold text-emerald-500 uppercase">{t('audit.after')}</span>
            <button
              type="button"
              onclick={() => copyText(selectedEntry?.after_json || '', 'after-json')}
              class="text-[11px] text-muted-foreground hover:text-foreground cursor-pointer"
            >
              {copiedId === 'after-json' ? '✓ Copied' : 'Copy'}
            </button>
          </div>
          <pre class="p-3 bg-muted rounded-xl text-foreground font-mono text-[11px] overflow-x-auto whitespace-pre-wrap">{selectedEntry.after_json}</pre>
        </div>
      {/if}
    </div>
  </Modal>
{/if}
