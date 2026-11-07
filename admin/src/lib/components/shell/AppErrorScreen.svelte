<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import Button from '$lib/components/ui/Button.svelte';

  interface Props {
    error: {
      message?: string;
      code?: string;
      requestId?: string;
      status?: number;
    } | null;
    onretry?: () => void;
  }

  let { error, onretry }: Props = $props();

  const isSuspended = $derived(error?.code === 'workspace_suspended' || error?.status === 503);
</script>

<div class="min-h-screen w-full flex flex-col items-center justify-center p-6 bg-muted/40 font-sans">
  <div class="w-full max-w-md rounded-2xl border border-border bg-card p-8 shadow-xs space-y-5 text-center">
    <div class="size-12 rounded-full bg-destructive/10 text-destructive flex items-center justify-center mx-auto">
      <svg class="size-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="8" x2="12" y2="12" />
        <line x1="12" y1="16" x2="12.01" y2="16" />
      </svg>
    </div>

    <div class="space-y-1.5">
      <h2 class="text-xl font-semibold tracking-tight text-foreground">
        {isSuspended ? t('error.suspendedTitle') : t('error.genericTitle')}
      </h2>
      <p class="text-xs text-muted-foreground leading-relaxed">
        {error?.message || t('error.genericSub')}
      </p>
    </div>

    {#if error?.code || error?.requestId}
      <div class="p-3 rounded-lg bg-muted/40 border border-border font-mono text-[11px] text-left space-y-1">
        {#if error.code}
          <div class="flex justify-between">
            <span class="text-muted-foreground">{t('error.code')}</span>
            <span class="text-foreground font-medium">{error.code}</span>
          </div>
        {/if}
        {#if error.requestId}
          <div class="flex justify-between">
            <span class="text-muted-foreground">{t('error.requestId')}</span>
            <span class="text-foreground select-all">{error.requestId}</span>
          </div>
        {/if}
      </div>
    {/if}

    <div class="pt-2 flex items-center justify-center gap-2.5">
      {#if onretry}
        <Button variant="outline" class="h-9 text-xs" onclick={onretry}>
          {t('error.retry')}
        </Button>
      {/if}
      <a
        href="https://console.example.com"
        class="inline-flex items-center justify-center h-9 px-4 rounded-md text-xs font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
      >
        Platform Console
      </a>
    </div>
  </div>
</div>
