<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import type { HookService } from '../services';

  let { services }: { services: HookService[] } = $props();

  function when(value?: string): string {
    return value ? new Date(value).toLocaleString() : t('system.services.never');
  }
</script>

<div class="space-y-4">
  <p class="text-xs text-muted-foreground">{t('system.services.note')}</p>
  {#each services as service (service.id)}
    <div class="rounded-xl border border-border bg-card p-4 shadow-xs space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="grid min-w-0">
          <span class="font-semibold text-sm text-foreground">{service.id}</span>
          <span class="font-mono text-[11px] text-muted-foreground truncate">{service.host}</span>
        </div>
        {#if service.calls === 0}
          <span class="text-xs text-muted-foreground">{t('system.services.noCalls')}</span>
        {:else if service.last_error}
          <span class="text-xs font-semibold text-destructive">{t('system.services.failing')}: {service.last_error}</span>
        {:else}
          <span class="text-xs font-semibold text-emerald-600 dark:text-emerald-400">{t('system.services.ok')}</span>
        {/if}
      </div>
      <dl class="grid grid-cols-2 md:grid-cols-4 gap-3 text-xs">
        <div><dt class="text-muted-foreground">{t('system.services.calls')}</dt><dd class="font-mono">{service.calls} ({service.failures} {t('system.services.failures')})</dd></div>
        <div><dt class="text-muted-foreground">{t('system.services.lastCall')}</dt><dd class="font-mono">{when(service.last_at)}</dd></div>
        <div><dt class="text-muted-foreground">{t('system.services.lastEvent')}</dt><dd class="font-mono">{service.last_event || '-'} {service.last_status ? `(${service.last_status})` : ''}</dd></div>
        <div><dt class="text-muted-foreground">{t('system.services.timeout')}</dt><dd class="font-mono">{service.timeout || '5s'}</dd></div>
      </dl>
      <div class="text-xs space-y-1">
        <div><span class="text-muted-foreground">{t('system.services.usedBy')}:</span> <span class="font-mono">{service.used_by.length ? service.used_by.join(', ') : '-'}</span></div>
        {#if service.schedules.length}
          <div><span class="text-muted-foreground">{t('system.services.schedules')}:</span>
            <span class="font-mono">{service.schedules.map((item) => `${item.name} (${item.cron})`).join(', ')}</span></div>
        {/if}
      </div>
    </div>
  {/each}
</div>
