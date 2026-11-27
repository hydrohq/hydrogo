<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import PageSpinner from '$lib/components/ui/PageSpinner.svelte';

  export interface AuditStreamEntry {
    id: string;
    at: string;
    actor: string;
    action: string;
    resource: string;
    target: string;
    after_json?: string;
  }

  interface StreamMeta {
    ip?: string;
    method?: string;
    path?: string;
    status?: number;
  }

  interface Props {
    entries: AuditStreamEntry[];
    total: number;
    limit: number;
    offset: number;
    loading: boolean;
    filterActor: string;
    filterAction: string;
    filterResource: string;
    actionBadge: (action: string) => { variant: 'success' | 'destructive' | 'warning' | 'outline'; label: string };
    formatDate: (iso: string) => string;
    parseMetadata: (jsonStr?: string) => StreamMeta | null;
    onFilter: (e: SubmitEvent) => void;
    onReset: () => void;
    onInspect: (entry: AuditStreamEntry) => void;
    onPageChange: (newOffset: number) => void;
  }

  let {
    entries,
    total,
    limit,
    offset,
    loading,
    filterActor = $bindable(),
    filterAction = $bindable(),
    filterResource = $bindable(),
    actionBadge,
    formatDate,
    parseMetadata,
    onFilter,
    onReset,
    onInspect,
    onPageChange,
  }: Props = $props();

  const activityBuckets = $derived.by(() => {
    const count = 24;
    const buckets = Array.from({ length: count }, () => ({ total: 0, errors: 0 }));
    if (entries.length === 0) return buckets;
    const times = entries.map((e) => new Date(e.at).getTime()).filter((n) => !Number.isNaN(n));
    if (times.length === 0) return buckets;
    const min = Math.min(...times);
    const max = Math.max(...times);
    const span = Math.max(max - min, 1);
    for (const e of entries) {
      const ts = new Date(e.at).getTime();
      if (Number.isNaN(ts)) continue;
      const idx = Math.min(count - 1, Math.floor(((ts - min) / span) * count));
      buckets[idx].total++;
      if (e.action.includes('failed') || e.action.includes('rejected') || e.action.includes('delete')) {
        buckets[idx].errors++;
      }
    }
    return buckets;
  });

  const maxBucketTotal = $derived(Math.max(1, ...activityBuckets.map((b) => b.total)));
</script>

{#if entries.length > 0}
  <div class="rounded-xl border border-border bg-card px-4 py-3 shadow-xs">
    <div class="flex items-center justify-between mb-2">
      <span class="text-[11px] font-medium text-muted-foreground">{t('audit.distribution')}</span>
      <span class="font-mono text-[11px] text-muted-foreground">{t('audit.loadedOfTotal', { loaded: entries.length, total })}</span>
    </div>
    <div class="flex items-end gap-1 h-12" aria-hidden="true">
      {#each activityBuckets as bucket, i (i)}
        {@const heightPct = bucket.total === 0 ? 6 : Math.max(14, Math.round((bucket.total / maxBucketTotal) * 100))}
        <div
          class="flex-1 rounded-t-xs transition-all {bucket.errors > 0 ? 'bg-destructive/75' : bucket.total > 0 ? 'bg-primary/70' : 'bg-muted'}"
          style="height: {heightPct}%"
          title={t('audit.bucketTooltip', { total: bucket.total, errors: bucket.errors })}
        ></div>
      {/each}
    </div>
  </div>
{/if}

<form onsubmit={onFilter} class="p-3 rounded-xl border border-border bg-card shadow-xs grid grid-cols-1 sm:grid-cols-4 gap-3 items-end">
  <Input label={t('audit.filter.actor')} placeholder={t('audit.filter.actorPlaceholder')} bind:value={filterActor} />
  <Input label={t('audit.filter.action')} placeholder={t('audit.filter.actionPlaceholder')} bind:value={filterAction} />
  <Input label={t('audit.filter.resource')} placeholder={t('audit.filter.resourcePlaceholder')} bind:value={filterResource} />
  <div class="flex gap-2">
    <button
      type="submit"
      class="h-9 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs transition-colors cursor-pointer"
    >
      {t('audit.filter.apply')}
    </button>
    <button
      type="button"
      class="h-9 px-4 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer"
      onclick={onReset}
    >
      {t('audit.filter.reset')}
    </button>
  </div>
</form>

<div class="overflow-hidden rounded-xl border border-border bg-card shadow-xs">
  {#if loading}
    <PageSpinner compact />
  {:else if entries.length === 0}
    <div class="p-12 text-center text-xs text-muted-foreground">{t('audit.empty')}</div>
  {:else}
    <div class="divide-y divide-border">
      {#each entries as entry (entry.id)}
        {@const badgeInfo = actionBadge(entry.action)}
        {@const meta = parseMetadata(entry.after_json)}
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 px-4 py-2.5 hover:bg-muted/40 transition-colors">
          <div class="flex items-start gap-3 min-w-0">
            <div class="shrink-0 pt-0.5 w-32">
              <Badge variant={badgeInfo.variant} size="sm">{badgeInfo.label}</Badge>
            </div>
            <div class="min-w-0 flex-1">
              <div class="font-mono text-xs font-medium text-foreground truncate">
                {#if meta?.method && meta?.path}
                  {meta.method} {meta.path}
                {:else}
                  {entry.action} {entry.resource}{entry.target ? `/${entry.target}` : ''}
                {/if}
              </div>
              <div class="mt-1 flex flex-wrap items-center gap-1.5 font-mono text-[11px] text-muted-foreground">
                <span class="inline-flex items-center gap-1 rounded bg-muted px-1.5 py-0.5 text-foreground/90">
                  <span>{t('audit.meta.actor')}</span>
                  <span>{entry.actor}</span>
                </span>
                <span class="rounded bg-muted px-1.5 py-0.5">{t('audit.meta.resource')} {entry.resource}</span>
                {#if entry.target}
                  <span class="rounded bg-muted px-1.5 py-0.5 truncate max-w-[180px]" title={entry.target}>{t('audit.meta.target')} {entry.target}</span>
                {/if}
                {#if meta?.status}
                  <span class="rounded px-1.5 py-0.5 {meta.status >= 400 ? 'bg-destructive/15 text-destructive' : 'bg-muted'}">{t('audit.meta.status')} {meta.status}</span>
                {/if}
                {#if meta?.ip}
                  <span class="rounded bg-muted px-1.5 py-0.5">ip: {meta.ip}</span>
                {/if}
              </div>
            </div>
          </div>

          <div class="flex items-center justify-between sm:justify-end gap-4 shrink-0">
            <span class="font-mono text-[11px] text-muted-foreground whitespace-nowrap">{formatDate(entry.at)}</span>
            <button
              type="button"
              class="inline-flex items-center gap-1 text-xs text-primary hover:underline font-medium cursor-pointer"
              onclick={() => onInspect(entry)}
            >
              <span>{t('audit.inspect')}</span>
              <span aria-hidden="true">→</span>
            </button>
          </div>
        </div>
      {/each}
    </div>

    <div class="flex items-center justify-between px-4 py-3 border-t border-border text-xs text-muted-foreground">
      <div>
        {t('audit.showingRange', { from: offset + 1, to: Math.min(offset + entries.length, total), total })}
      </div>
      <div class="flex gap-2">
        <button
          type="button"
          class="h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground disabled:opacity-50 cursor-pointer transition-colors"
          disabled={offset === 0}
          onclick={() => onPageChange(Math.max(0, offset - limit))}
        >
          {t('common.previous')}
        </button>
        <button
          type="button"
          class="h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground disabled:opacity-50 cursor-pointer transition-colors"
          disabled={offset + limit >= total}
          onclick={() => onPageChange(offset + limit)}
        >
          {t('common.next')}
        </button>
      </div>
    </div>
  {/if}
</div>
