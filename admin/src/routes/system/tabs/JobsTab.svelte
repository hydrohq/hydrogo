<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from '$lib/i18n/index.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
  import { deleteDeadLetterJob, loadDeadLetterJobs, loadJobs, requeueJob, type JobSummary } from '../jobs';

  let jobs = $state<JobSummary[]>([]);
  let deadLetters = $state<JobSummary[]>([]);
  let loading = $state(true);
  let requeuingID = $state<string | null>(null);
  let deleteTarget = $state<JobSummary | null>(null);
  let deleting = $state(false);

  async function load() {
    loading = true;
    try {
      [jobs, deadLetters] = await Promise.all([loadJobs(), loadDeadLetterJobs()]);
    } finally {
      loading = false;
    }
  }

  async function handleRequeue(job: JobSummary) {
    requeuingID = job.id;
    try {
      await requeueJob(job.id);
      await load();
    } finally {
      requeuingID = null;
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    deleting = true;
    try {
      await deleteDeadLetterJob(deleteTarget.id);
      deleteTarget = null;
      await load();
    } finally {
      deleting = false;
    }
  }

  function when(value?: string): string {
    return value ? new Date(value).toLocaleString() : '-';
  }

  onMount(load);
</script>

<div class="space-y-6">
  <div class="space-y-2">
    <h3 class="text-sm font-semibold text-foreground">{t('system.jobs.pending')}</h3>
    <p class="text-xs text-muted-foreground">{t('system.jobs.pendingHelp')}</p>
    {#if !loading && jobs.length === 0}
      <p class="text-xs text-muted-foreground">{t('system.jobs.empty')}</p>
    {:else}
      <div class="space-y-2">
        {#each jobs as job (job.id)}
          <div class="rounded-xl border border-border bg-card p-4 shadow-xs space-y-2">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div class="grid min-w-0">
                <span class="font-semibold text-sm text-foreground">{job.id}</span>
                <span class="font-mono text-[11px] text-muted-foreground">{job.type}</span>
              </div>
              <span class="text-xs font-medium {job.lease_owner ? 'text-primary' : 'text-muted-foreground'}">
                {job.lease_owner ? t('system.jobs.leasedBy', { worker: job.lease_owner }) : t('system.jobs.claimable')}
              </span>
            </div>
            <dl class="grid grid-cols-2 md:grid-cols-3 gap-3 text-xs">
              <div><dt class="text-muted-foreground">{t('system.jobs.attempts')}</dt><dd class="font-mono">{job.attempts}</dd></div>
              <div><dt class="text-muted-foreground">{t('system.jobs.available')}</dt><dd class="font-mono">{when(job.available_at)}</dd></div>
              <div><dt class="text-muted-foreground">{t('system.jobs.created')}</dt><dd class="font-mono">{when(job.created_at)}</dd></div>
            </dl>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <div class="space-y-2">
    <h3 class="text-sm font-semibold text-foreground">{t('system.jobs.deadLetters')}</h3>
    <p class="text-xs text-muted-foreground">{t('system.jobs.deadLettersHelp')}</p>
    {#if !loading && deadLetters.length === 0}
      <p class="text-xs text-muted-foreground">{t('system.jobs.empty')}</p>
    {:else}
      <div class="space-y-2">
        {#each deadLetters as job (job.id)}
          <div class="rounded-xl border border-destructive/30 bg-card p-4 shadow-xs space-y-2">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div class="grid min-w-0">
                <span class="font-semibold text-sm text-foreground">{job.id}</span>
                <span class="font-mono text-[11px] text-muted-foreground">{job.type}</span>
              </div>
              <div class="flex items-center gap-2">
                <Button variant="outline" disabled={requeuingID === job.id} onclick={() => handleRequeue(job)}>
                  {requeuingID === job.id ? t('system.jobs.requeuing') : t('system.jobs.requeue')}
                </Button>
                <Button variant="outline" onclick={() => (deleteTarget = job)}>{t('system.jobs.delete')}</Button>
              </div>
            </div>
            <dl class="grid grid-cols-2 md:grid-cols-3 gap-3 text-xs">
              <div><dt class="text-muted-foreground">{t('system.jobs.attempts')}</dt><dd class="font-mono">{job.attempts}</dd></div>
              <div><dt class="text-muted-foreground">{t('system.jobs.created')}</dt><dd class="font-mono">{when(job.created_at)}</dd></div>
              <div class="col-span-2 md:col-span-1"><dt class="text-muted-foreground">{t('system.jobs.lastError')}</dt><dd class="font-mono truncate">{job.last_error || '-'}</dd></div>
            </dl>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<ConfirmDialog
  open={!!deleteTarget}
  title={t('system.jobs.deleteTitle')}
  description={t('system.jobs.deleteDescription', { id: deleteTarget?.id ?? '' })}
  confirmLabel={deleting ? t('system.jobs.deleting') : t('system.jobs.delete')}
  loading={deleting}
  onconfirm={handleDelete}
  oncancel={() => (deleteTarget = null)}
/>
