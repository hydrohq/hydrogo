<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
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

  let entries = $state<AuditEntry[]>([]);
  let total = $state(0);
  let limit = $state(25);
  let offset = $state(0);
  let loading = $state(true);

  // Filters
  let filterActor = $state('');
  let filterAction = $state('');
  let filterResource = $state('');
  let selectedEntry = $state<AuditEntry | null>(null);

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
        total = res.meta?.total || 0;
      }
    } catch (err: any) {
      console.warn('Failed to load audit logs:', err);
    } finally {
      loading = false;
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

  function actionBadgeVariant(action: string): 'default' | 'success' | 'warning' | 'danger' | 'info' {
    if (action.includes('delete') || action.includes('revoke')) return 'danger';
    if (action.includes('create') || action.includes('publish')) return 'success';
    if (action.includes('update') || action.includes('edit')) return 'warning';
    return 'default';
  }

  onMount(() => {
    loadLogs();
  });
</script>

<div class="space-y-6">
  <!-- Header -->
  <div class="flex items-center justify-between pb-4 border-b border-slate-800">
    <div>
      <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2">
        <span>📋</span>
        <span>Audit Trail</span>
      </h2>
      <p class="text-xs text-slate-400 mt-1">
        Immutable record of operational changes and administrative actions.
      </p>
    </div>
    <div class="text-xs text-slate-500 font-mono">
      Total events: {total}
    </div>
  </div>

  <!-- Filters -->
  <form onsubmit={handleFilter} class="p-4 rounded-xl bg-slate-900/60 border border-slate-800 grid grid-cols-1 sm:grid-cols-4 gap-3 items-end">
    <Input label="Actor" placeholder="Filter by email…" bind:value={filterActor} />
    <Input label="Action" placeholder="e.g. create, update…" bind:value={filterAction} />
    <Input label="Resource" placeholder="e.g. page, user…" bind:value={filterResource} />
    <div class="flex gap-2">
      <Button type="submit" variant="primary" size="md">Filter</Button>
      <Button
        type="button"
        variant="ghost"
        size="md"
        onclick={() => {
          filterActor = '';
          filterAction = '';
          filterResource = '';
          offset = 0;
          loadLogs();
        }}
      >
        Reset
      </Button>
    </div>
  </form>

  <!-- Table -->
  <div class="rounded-xl border border-slate-800 bg-slate-900/60 overflow-hidden shadow-sm">
    {#if loading}
      <div class="py-12 text-center text-xs text-slate-500">Loading audit log events…</div>
    {:else if entries.length === 0}
      <div class="py-12 text-center text-xs text-slate-500">No audit events found.</div>
    {:else}
      <table class="w-full text-left text-sm text-slate-300">
        <thead class="bg-slate-950/80 text-[11px] uppercase tracking-wider text-slate-400 border-b border-slate-800">
          <tr>
            <th class="px-4 py-3 font-semibold">Timestamp</th>
            <th class="px-4 py-3 font-semibold">Actor</th>
            <th class="px-4 py-3 font-semibold">Action</th>
            <th class="px-4 py-3 font-semibold">Resource</th>
            <th class="px-4 py-3 font-semibold">Target</th>
            <th class="px-4 py-3 font-semibold text-right">Details</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/60">
          {#each entries as entry (entry.id)}
            <tr class="hover:bg-slate-800/40 transition-colors">
              <td class="px-4 py-3 text-xs text-slate-400 whitespace-nowrap">{formatDate(entry.at)}</td>
              <td class="px-4 py-3 font-medium text-xs text-slate-200">{entry.actor}</td>
              <td class="px-4 py-3">
                <Badge variant={actionBadgeVariant(entry.action)} size="sm">{entry.action}</Badge>
              </td>
              <td class="px-4 py-3 font-mono text-xs text-sky-400">{entry.resource}</td>
              <td class="px-4 py-3 font-mono text-xs text-slate-400 truncate max-w-[140px]">{entry.target}</td>
              <td class="px-4 py-3 text-right">
                <button
                  type="button"
                  class="text-xs text-sky-400 hover:text-sky-300 font-medium"
                  onclick={() => { selectedEntry = entry; }}
                >
                  View JSON
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>

      <!-- Pagination -->
      <div class="flex items-center justify-between px-4 py-3 border-t border-slate-800 bg-slate-950/60 text-xs text-slate-400">
        <div>
          Showing {offset + 1}–{Math.min(offset + entries.length, total)} of {total}
        </div>
        <div class="flex gap-2">
          <Button
            size="sm"
            variant="ghost"
            disabled={offset === 0}
            onclick={() => { offset = Math.max(0, offset - limit); loadLogs(); }}
          >
            Previous
          </Button>
          <Button
            size="sm"
            variant="ghost"
            disabled={offset + entries.length >= total}
            onclick={() => { offset += limit; loadLogs(); }}
          >
            Next
          </Button>
        </div>
      </div>
    {/if}
  </div>

  <!-- Detail Modal -->
  {#if selectedEntry}
    <Modal
      open={true}
      title="Audit Event Details"
      onClose={() => { selectedEntry = null; }}
    >
      <div class="space-y-4 text-xs font-mono max-h-[70vh] overflow-y-auto">
        <div class="p-3 bg-slate-950 rounded-lg space-y-1">
          <div><span class="text-slate-500">ID:</span> <span class="text-slate-300">{selectedEntry.id}</span></div>
          <div><span class="text-slate-500">Action:</span> <span class="text-sky-400">{selectedEntry.action}</span></div>
          <div><span class="text-slate-500">Resource:</span> <span class="text-amber-400">{selectedEntry.resource}</span></div>
          {#if selectedEntry.request_id}
            <div><span class="text-slate-500">Request ID:</span> <span class="text-slate-300">{selectedEntry.request_id}</span></div>
          {/if}
        </div>

        {#if selectedEntry.before_json}
          <div>
            <div class="text-xs font-semibold text-rose-400 uppercase font-sans mb-1">State Before</div>
            <pre class="p-3 bg-slate-950 rounded-lg text-slate-300 overflow-x-auto whitespace-pre-wrap">{selectedEntry.before_json}</pre>
          </div>
        {/if}

        {#if selectedEntry.after_json}
          <div>
            <div class="text-xs font-semibold text-emerald-400 uppercase font-sans mb-1">State After</div>
            <pre class="p-3 bg-slate-950 rounded-lg text-slate-300 overflow-x-auto whitespace-pre-wrap">{selectedEntry.after_json}</pre>
          </div>
        {/if}
      </div>
    </Modal>
  {/if}
</div>
