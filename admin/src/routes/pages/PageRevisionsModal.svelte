<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
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
      errorMessage = e?.message || t('pages.revisions.loadFailed');
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
      errorMessage = err?.message || t('pages.revisions.restoreFailed');
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
  title={t('pages.revisions.restoreTitle', { revision: restoreTarget?.revision ?? '' })}
  description={t('pages.revisions.restoreDescription', { revision: restoreTarget?.revision ?? '' })}
  confirmLabel={t('pages.revisions.restoreConfirm')}
  variant="primary"
  loading={restoring}
  onconfirm={confirmRestore}
  oncancel={() => { if (!restoring) restoreTarget = null; }}
/>

<Modal
  {open}
  title={t('pages.revisions.title')}
  description={t('pages.revisions.description')}
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
          {t('common.dismiss')}
        </button>
      </div>
    {/if}

    {#if loading}
      <div class="py-12 flex flex-col items-center justify-center text-muted-foreground text-xs gap-2">
        <div class="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
        <span>{t('pages.revisions.loading')}</span>
      </div>
    {:else if revisions.length === 0}
      <div class="py-12 text-center text-xs text-muted-foreground border border-dashed border-border rounded-xl">
        <p>{t('pages.revisions.empty')}</p>
        <p class="text-[11px] text-muted-foreground/70 mt-1">{t('pages.revisions.emptyHelp')}</p>
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
                <span>•</span>
                <span>{rev.author_name || t('pages.revisions.systemAdmin')}</span>
              </p>
            </div>

            <button
              type="button"
              class="h-7 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs shrink-0"
              onclick={() => { restoreTarget = rev; }}
            >
              {t('pages.revisions.restore')}
            </button>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={onclose}>{t('common.close')}</Button>
  {/snippet}
</Modal>
