<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';

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
  let restoring = $state<string | null>(null);

  async function loadRevisions() {
    if (!pageId || pageId === 'new') return;
    loading = true;
    try {
      const res = await api.get<ApiResponse<Revision[]>>(`/api/v1/admin/pages/${pageId}/revisions`);
      if (res && res.data) {
        revisions = Array.isArray(res.data) ? res.data : [];
      }
    } catch (e) {
      console.warn('Revisions load error:', e);
    } finally {
      loading = false;
    }
  }

  async function handleRestore(revId: string, revNum: number) {
    if (!confirm(`Restore revision #${revNum}? Any unsaved changes in the current draft will be replaced.`)) return;
    restoring = revId;
    try {
      await api.post(`/api/v1/admin/pages/${pageId}/revisions/${revId}/restore`);
      onrestored();
      onclose();
    } catch (err: any) {
      alert(`Restore failed: ${err.message}`);
    } finally {
      restoring = null;
    }
  }

  $effect(() => {
    if (open && pageId) {
      loadRevisions();
    }
  });
</script>

<Modal
  {open}
  title="Revision History"
  description="View and restore previously saved versions of this page."
  {onclose}
>
  <div class="space-y-3">
    {#if loading}
      <p class="py-8 text-center text-xs text-slate-500">Loading revision history…</p>
    {:else if revisions.length === 0}
      <p class="py-8 text-center text-xs text-slate-500">No previous revisions recorded yet.</p>
    {:else}
      <div class="divide-y divide-slate-800/80 rounded-xl border border-slate-800 bg-slate-950/60 max-h-80 overflow-y-auto">
        {#each revisions as rev (rev.id)}
          <div class="p-3.5 flex items-center justify-between">
            <div>
              <div class="flex items-center gap-2">
                <Badge variant="info" size="sm">rev #{rev.revision}</Badge>
                <span class="text-xs font-semibold text-slate-200">{rev.title}</span>
              </div>
              <p class="text-[11px] text-slate-500 mt-1">
                {new Date(rev.created_at).toLocaleString()} • {rev.author_name || 'Author'}
              </p>
            </div>
            <Button
              variant="outline"
              size="sm"
              loading={restoring === rev.id}
              onclick={() => handleRestore(rev.id, rev.revision)}
            >
              Restore
            </Button>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={onclose}>Close</Button>
  {/snippet}
</Modal>
