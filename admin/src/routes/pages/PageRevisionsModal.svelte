<script lang="ts">
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
      <p class="py-8 text-center text-xs text-muted-foreground">Loading revision history…</p>
    {:else if revisions.length === 0}
      <p class="py-8 text-center text-xs text-muted-foreground">No previous revisions recorded yet.</p>
    {:else}
      <div class="divide-y divide-border rounded-xl border border-border bg-card max-h-80 overflow-y-auto">
        {#each revisions as rev (rev.id)}
          <div class="p-3.5 flex items-center justify-between hover:bg-muted/30 transition-colors">
            <div>
              <div class="flex items-center gap-2">
                <Badge variant="outline" size="sm">rev #{rev.revision}</Badge>
                <span class="text-xs font-semibold text-foreground">{rev.title}</span>
              </div>
              <p class="text-[11px] text-muted-foreground mt-1">
                {new Date(rev.created_at).toLocaleString()} • {rev.author_name || 'Author'}
              </p>
            </div>
            <button
              type="button"
              class="h-7 px-2.5 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs"
              disabled={restoring === rev.id}
              onclick={() => handleRestore(rev.id, rev.revision)}
            >
              {restoring === rev.id ? 'Restoring…' : 'Restore'}
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
