<script lang="ts">
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

  function actionBadgeVariant(action: string): 'default' | 'success' | 'warning' | 'destructive' | 'outline' {
    if (action.includes('delete') || action.includes('revoke')) return 'destructive';
    if (action.includes('create') || action.includes('publish')) return 'success';
    if (action.includes('update') || action.includes('edit')) return 'warning';
    return 'outline';
  }

  onMount(() => {
    loadLogs();
  });
</script>

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-2">
    <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <!-- Filters row -->
      <div class="px-4 lg:px-6">
        <form onsubmit={handleFilter} class="p-3 rounded-xl border border-border bg-card shadow-xs grid grid-cols-1 sm:grid-cols-4 gap-3 items-end">
          <Input label="Actor" placeholder="Filter by email…" bind:value={filterActor} />
          <Input label="Action" placeholder="e.g. create, update…" bind:value={filterAction} />
          <Input label="Resource" placeholder="e.g. page, user…" bind:value={filterResource} />
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
            <div class="p-12 text-center text-xs text-muted-foreground">Loading audit log events…</div>
          {:else if entries.length === 0}
            <div class="p-12 text-center text-xs text-muted-foreground">No audit events found.</div>
          {:else}
            <div class="overflow-x-auto">
              <table class="w-full text-left text-sm text-foreground caption-bottom">
                <thead class="sticky top-0 z-10 bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
                  <tr>
                    <th class="h-10 px-4 font-medium">Timestamp</th>
                    <th class="h-10 px-4 font-medium">Actor</th>
                    <th class="h-10 px-4 font-medium">Action</th>
                    <th class="h-10 px-4 font-medium">Resource</th>
                    <th class="h-10 px-4 font-medium">Target</th>
                    <th class="h-10 w-24 px-4 font-medium text-right">Details</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-border">
                  {#each entries as entry (entry.id)}
                    <tr class="hover:bg-muted/50 transition-colors">
                      <td class="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">{formatDate(entry.at)}</td>
                      <td class="px-4 py-3 font-medium text-xs text-foreground">{entry.actor}</td>
                      <td class="px-4 py-3">
                        <Badge variant={actionBadgeVariant(entry.action)} size="sm">{entry.action}</Badge>
                      </td>
                      <td class="px-4 py-3 font-mono text-xs text-muted-foreground">{entry.resource}</td>
                      <td class="px-4 py-3 font-mono text-xs text-muted-foreground truncate max-w-[140px]">{entry.target}</td>
                      <td class="px-4 py-3 text-right">
                        <button
                          type="button"
                          class="text-xs text-primary hover:underline font-medium cursor-pointer"
                          onclick={() => { selectedEntry = entry; }}
                        >
                          View JSON
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
    </div>
  </div>
</div>

<!-- Detail Modal -->
{#if selectedEntry}
  <Modal
    open={true}
    title="Audit Event Details"
    onclose={() => { selectedEntry = null; }}
  >
    <div class="space-y-4 text-xs font-mono max-h-[70vh] overflow-y-auto">
      <div class="p-3 bg-muted rounded-lg space-y-1">
        <div><span class="text-muted-foreground">ID:</span> <span class="text-foreground">{selectedEntry.id}</span></div>
        <div><span class="text-muted-foreground">Action:</span> <span class="text-foreground font-semibold">{selectedEntry.action}</span></div>
        <div><span class="text-muted-foreground">Resource:</span> <span class="text-foreground">{selectedEntry.resource}</span></div>
        {#if selectedEntry.request_id}
          <div><span class="text-muted-foreground">Request ID:</span> <span class="text-foreground">{selectedEntry.request_id}</span></div>
        {/if}
      </div>

      {#if selectedEntry.before_json}
        <div>
          <div class="text-xs font-semibold text-destructive uppercase font-sans mb-1">State Before</div>
          <pre class="p-3 bg-muted rounded-lg text-foreground overflow-x-auto whitespace-pre-wrap">{selectedEntry.before_json}</pre>
        </div>
      {/if}

      {#if selectedEntry.after_json}
        <div>
          <div class="text-xs font-semibold text-emerald-500 uppercase font-sans mb-1">State After</div>
          <pre class="p-3 bg-muted rounded-lg text-foreground overflow-x-auto whitespace-pre-wrap">{selectedEntry.after_json}</pre>
        </div>
      {/if}
    </div>
  </Modal>
{/if}
