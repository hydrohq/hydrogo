<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { confirmAction } from '$lib/confirm';
  import type { ApiResponse } from '$lib/api/types';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import { statusLabel } from '$lib/i18n/labels';

  interface RevisionSummary {
    version: number;
    state?: string;
    actor?: string;
    created_at: string;
  }

  interface RevisionSnapshot extends RevisionSummary {
    fields: Record<string, string>;
  }

  interface Props {
    open: boolean;
    endpoint: string;
    recordId: string;
    currentVersion: number;
    onclose: () => void;
    onrestored: (record: Record<string, any>) => void;
  }

  let {
    open = false,
    endpoint,
    recordId,
    currentVersion,
    onclose,
    onrestored,
  }: Props = $props();

  let loading = $state(false);
  let restoringVersion = $state<number | null>(null);
  let error = $state<string | null>(null);
  let revisions = $state<RevisionSummary[]>([]);
  let preview = $state<RevisionSnapshot | null>(null);
  let previewLoading = $state(false);

  async function loadRevisions() {
    if (!endpoint || !recordId) return;
    loading = true;
    error = null;
    preview = null;
    try {
      const res = await api.get<ApiResponse<RevisionSummary[]>>(
        `${endpoint}/${encodeURIComponent(recordId)}/revisions`,
        { params: { limit: 20 } }
      );
      revisions = Array.isArray(res?.data) ? res.data : [];
    } catch (err) {
      revisions = [];
      error = err instanceof ApiClientError ? err.message : t('schema.err.load');
    } finally {
      loading = false;
    }
  }

  async function inspectRevision(version: number) {
    previewLoading = true;
    error = null;
    try {
      const res = await api.get<ApiResponse<RevisionSnapshot>>(
        `${endpoint}/${encodeURIComponent(recordId)}/revisions/${version}`
      );
      preview = res?.data ?? null;
    } catch (err) {
      error = err instanceof ApiClientError ? err.message : t('schema.err.load');
    } finally {
      previewLoading = false;
    }
  }

  async function restore(version: number) {
    const ok = await confirmAction({
      title: t('schema.revisions.restoreTitle', { version }),
      description: t('schema.revisions.restoreDesc', { version }),
      confirmLabel: t('schema.revisions.restoreConfirm'),
      cancelLabel: t('common.cancel'),
      variant: 'primary',
    });
    if (!ok) return;

    restoringVersion = version;
    error = null;
    try {
      const res = await api.post<ApiResponse<Record<string, any>>>(
        `${endpoint}/${encodeURIComponent(recordId)}/revisions/${version}/restore`,
        { version: currentVersion }
      );
      if (res?.data) onrestored(res.data);
      onclose();
    } catch (err) {
      error = err instanceof ApiClientError
        ? `${err.message}${err.requestId ? ` (${t('schema.requestId')}: ${err.requestId})` : ''}`
        : t('schema.err.saveItem');
    } finally {
      restoringVersion = null;
    }
  }

  function formatTimestamp(ts: string): string {
    if (!ts) return '';
    try {
      return new Date(ts).toLocaleString();
    } catch {
      return ts;
    }
  }

  $effect(() => {
    if (open && endpoint && recordId) {
      void loadRevisions();
    }
  });
</script>

<Modal
  {open}
  size="2xl"
  title={t('schema.revisions.title')}
  description={t('schema.revisions.description')}
  {onclose}
>
  {#if error}
    <div class="mb-3 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive" role="alert">
      {error}
    </div>
  {/if}

  {#if loading}
    <div class="py-10 flex items-center justify-center gap-2 text-xs text-muted-foreground">
      <div class="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
      <span>{t('schema.revisions.loading')}</span>
    </div>
  {:else if revisions.length === 0}
    <p class="rounded-lg border border-dashed border-border py-8 text-center text-xs text-muted-foreground">
      {t('schema.revisions.empty')}
    </p>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <ul class="rounded-lg border border-border divide-y divide-border max-h-80 overflow-y-auto">
        {#each revisions as rev (rev.version)}
          <li
            class="flex items-center justify-between gap-2 px-3 py-2.5 text-xs transition-colors {preview?.version === rev.version ? 'bg-accent/60' : 'hover:bg-muted/40'}"
          >
            <button
              type="button"
              class="flex-1 text-left cursor-pointer min-w-0"
              onclick={() => inspectRevision(rev.version)}
            >
              <div class="flex items-center gap-2">
                <span class="font-mono font-semibold text-foreground tabular-nums">
                  {t('schema.revisions.version', { version: rev.version })}
                </span>
                {#if rev.state}
                  <Badge variant={rev.state === 'published' ? 'success' : 'warning'} size="sm">
                    {statusLabel(rev.state)}
                  </Badge>
                {/if}
              </div>
              <div class="mt-0.5 text-[11px] text-muted-foreground truncate">
                {formatTimestamp(rev.created_at)}{rev.actor ? ` · ${rev.actor}` : ''}
              </div>
            </button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              loading={restoringVersion === rev.version}
              disabled={restoringVersion !== null}
              onclick={() => restore(rev.version)}
            >
              {t('schema.revisions.restore')}
            </Button>
          </li>
        {/each}
      </ul>

      <div class="rounded-lg border border-border bg-muted/20 p-3 max-h-80 overflow-y-auto">
        {#if previewLoading}
          <p class="text-xs text-muted-foreground">{t('common.loading')}</p>
        {:else if preview}
          <div class="mb-2 pb-2 border-b border-border flex items-center justify-between text-xs">
            <span class="font-mono font-semibold">{t('schema.revisions.version', { version: preview.version })}</span>
            <span class="text-[11px] text-muted-foreground">{formatTimestamp(preview.created_at)}</span>
          </div>
          <dl class="space-y-2 text-xs">
            {#each Object.entries(preview.fields || {}) as [fieldId, val] (fieldId)}
              <div>
                <dt class="font-mono text-[10px] text-muted-foreground">{fieldId}</dt>
                <dd class="mt-0.5 text-foreground break-words line-clamp-4">{val || '—'}</dd>
              </div>
            {/each}
          </dl>
        {:else}
          <p class="py-10 text-center text-xs text-muted-foreground">{t('schema.revisions.description')}</p>
        {/if}
      </div>
    </div>
  {/if}

  {#snippet footer()}
    <Button variant="ghost" size="sm" onclick={onclose}>
      {t('common.closeModal')}
    </Button>
  {/snippet}
</Modal>
