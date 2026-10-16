<script lang="ts">
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';

  interface Revision {
    id: string;
    revision: number;
    title: string;
    created_at: string;
    author_name?: string;
  }

  interface Props {
    pageId: string;
    open: boolean;
    onclose: () => void;
    onrestored: () => void;
  }

  let { pageId, open, onclose, onrestored }: Props = $props();

  let revisions = $state<Revision[]>([]);
  let loading = $state(false);
  let restoring = $state(false);
  let restoreTarget = $state<Revision | null>(null);
  let errorMessage = $state<string | null>(null);

  async function loadRevisions() {
    if (!pageId || pageId === 'new') return;
    loading = true;
    errorMessage = null;
    try {
      const res = await api.get<ApiResponse<Revision[]>>(`/api/v1/admin/pages/${pageId}/revisions`);
      if (res && res.data) {
        revisions = Array.isArray(res.data) ? res.data : [];
      }
    } catch (e: any) {
      console.warn('Revisions load error:', e);
      errorMessage = e?.message || 'Failed to load page revisions';
    } finally {
      loading = false;
    }
  }

  async function confirmRestore() {
    if (!restoreTarget) return;
    restoring = true;
    errorMessage = null;
    try {
      await api.post(`/api/v1/admin/pages/${pageId}/revisions/${restoreTarget.id}/restore`);
      restoreTarget = null;
      onrestored();
      onclose();
    } catch (err: any) {
      errorMessage = err?.message || 'Failed to restore revision';
    } finally {
      restoring = false;
    }
  }

  $effect(() => {
    if (open && pageId) {
      loadRevisions();
    }
  });
</script>

<ConfirmDialog
  open={!!restoreTarget}
  title={`Restore Revision #${restoreTarget?.revision}`}
  description={`Are you sure you want to restore revision #${restoreTarget?.revision}? Any unsaved edits in the current draft will be overwritten.`}
  confirmLabel="Restore Revision"
  variant="primary"
  loading={restoring}
  onconfirm={confirmRestore}
  oncancel={() => { if (!restoring) restoreTarget = null; }}
/>

<Modal
  {open}
  title="Revision History"
  description="Inspect and restore previous snapshots saved during page editing."
  {onclose}
>
  <div class="space-y-4">
    {#if errorMessage}
      <div class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive flex items-center justify-between">
        <span>{errorMessage}</span>
        <button
          type="button"
          class="text-xs hover:underline cursor-pointer"
          onclick={() => { errorMessage = null; }}
        >
          Dismiss
        </button>
      </div>
    {/if}

    {#if loading}
      <div class="py-12 flex flex-col items-center justify-center text-muted-foreground text-xs gap-2">
        <div class="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
        <span>Loading revision historyâ€¦</span>
      </div>
    {:else if revisions.length === 0}
      <div class="py-12 text-center text-xs text-muted-foreground border border-dashed border-border rounded-xl">
        <p>No previous revisions recorded yet for this page.</p>
        <p class="text-[11px] text-muted-foreground/70 mt-1">Revisions are automatically captured when saving changes.</p>
      </div>
    {:else}
      <div class="divide-y divide-border rounded-xl border border-border bg-card max-h-80 overflow-y-auto">
        {#each revisions as rev (rev.id)}
          <div class="p-3.5 flex items-center justify-between hover:bg-muted/30 transition-colors gap-3">
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <Badge variant="outline" size="sm" class="font-mono text-[10px]">
                  v{rev.revision}
                </Badge>
                <span class="text-xs font-semibold text-foreground truncate">{rev.title}</span>
              </div>
              <p class="text-[11px] text-muted-foreground mt-1 flex items-center gap-1.5">
                <span>{new Date(rev.created_at).toLocaleString()}</span>
                <span>â€¢</span>
                <span>{rev.author_name || 'System Admin'}</span>
              </p>
            </div>

            <button
              type="button"
              class="h-7 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs shrink-0"
              onclick={() => { restoreTarget = rev; }}
            >
              Restore
            </button>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={onclose}>Close</Button>
  {/snippet}
</Modal>
