<script lang="ts">
  import { onMount } from 'svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { confirmAction } from '$lib/confirm';
  import { t } from '$lib/i18n/index.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Button from '$lib/components/ui/Button.svelte';

  export interface InvoiceOrder {
    id: string;
    payment_status: 'pending' | 'paid' | 'failed' | 'refunded';
    invoice_status?: 'none' | 'pending' | 'issued' | 'failed' | string;
    invoice_provider?: string;
    invoice_guid?: string;
    invoice_no?: string;
    invoice_url?: string;
    invoice_error?: string;
    invoiced_at?: string;
  }

  interface InvoicingStatusResponse {
    configured: boolean;
    provider: string;
    auto_issue_on_paid: boolean;
  }

  type Feedback = { type: 'success' | 'error'; text: string } | null;

  interface Props {
    order: InvoiceOrder;
    canWrite: boolean;
    onupdated: (updated: InvoiceOrder) => void;
    onfeedback: (fb: Feedback) => void;
  }

  let { order, canWrite, onupdated, onfeedback }: Props = $props();

  let configured = $state(false);
  let busy = $state(false);

  const status = $derived((order.invoice_status || 'none') as 'none' | 'pending' | 'issued' | 'failed');
  const shouldRender = $derived(configured || status !== 'none' || Boolean(order.invoice_guid) || Boolean(order.invoice_url) || Boolean(order.invoice_error));
  const canIssue = $derived(canWrite && configured && order.payment_status === 'paid' && status !== 'issued');

  function isValidTimestamp(value?: string): value is string {
    return Boolean(value && !value.startsWith('0001'));
  }

  function statusVariant(value: string): 'success' | 'warning' | 'destructive' | 'secondary' {
    switch (value) {
      case 'issued':
        return 'success';
      case 'pending':
        return 'warning';
      case 'failed':
        return 'destructive';
      default:
        return 'secondary';
    }
  }

  function statusLabel(value: string): string {
    switch (value) {
      case 'issued':
        return t('commerce.invoice.status.issued');
      case 'pending':
        return t('commerce.invoice.status.pending');
      case 'failed':
        return t('commerce.invoice.status.failed');
      default:
        return t('commerce.invoice.status.none');
    }
  }

  onMount(async () => {
    try {
      const res = await api.get<{ data: InvoicingStatusResponse }>('/api/v1/admin/commerce/invoicing/status');
      configured = Boolean(res?.data?.configured);
    } catch {
      configured = false;
    }
  });

  async function issueInvoice() {
    const ok = await confirmAction({
      title: t('commerce.confirm.invoiceTitle'),
      description: t('commerce.confirm.invoiceDesc'),
      confirmLabel: t('commerce.confirm.invoiceButton'),
      cancelLabel: t('commerce.confirm.keep'),
      variant: 'primary',
    });
    if (!ok) return;

    busy = true;
    onfeedback(null);
    try {
      const res = await api.post<{ data: InvoiceOrder }>(
        `/api/v1/admin/commerce/orders/${encodeURIComponent(order.id)}/invoice`
      );
      if (res?.data) {
        onupdated(res.data);
        onfeedback({
          type: 'success',
          text: t('commerce.invoice.issuedDone', { guid: res.data.invoice_guid ?? '' }),
        });
      }
    } catch (err) {
      const text = err instanceof ApiClientError && err.message ? err.message : t('commerce.invoice.issueFailed');
      onfeedback({ type: 'error', text });
    } finally {
      busy = false;
    }
  }
</script>

{#if shouldRender}
  <div class="p-4 rounded-lg bg-muted/50 border border-border text-xs space-y-2.5">
    <div class="flex items-center justify-between gap-3">
      <span class="font-medium text-foreground">{t('commerce.invoice.heading')}</span>
      <Badge variant={statusVariant(status)} size="sm">{statusLabel(status)}</Badge>
    </div>

    {#if order.invoice_provider || order.invoice_no || order.invoice_guid || isValidTimestamp(order.invoiced_at)}
      <div class="space-y-1.5">
        {#if order.invoice_provider}
          <div class="flex justify-between gap-3">
            <span class="text-muted-foreground">{t('commerce.invoice.provider')}</span>
            <span class="text-foreground font-medium">{order.invoice_provider}</span>
          </div>
        {/if}
        {#if order.invoice_no}
          <div class="flex justify-between gap-3">
            <span class="text-muted-foreground">{t('commerce.invoice.number')}</span>
            <span class="text-foreground font-mono">{order.invoice_no}</span>
          </div>
        {/if}
        {#if order.invoice_guid}
          <div class="flex justify-between gap-3">
            <span class="text-muted-foreground">{t('commerce.invoice.guid')}</span>
            <span class="text-foreground font-mono break-all text-right">{order.invoice_guid}</span>
          </div>
        {/if}
        {#if isValidTimestamp(order.invoiced_at)}
          <div class="flex justify-between gap-3">
            <span class="text-muted-foreground">{t('commerce.invoice.issuedAt')}</span>
            <span class="text-foreground">{new Date(order.invoiced_at).toLocaleString()}</span>
          </div>
        {/if}
      </div>
    {/if}

    {#if status === 'failed' && order.invoice_error}
      <div class="p-2.5 rounded-md bg-destructive/10 border border-destructive/20 text-destructive break-words">
        {order.invoice_error}
      </div>
    {/if}

    {#if order.invoice_url || canIssue}
      <div class="flex flex-wrap items-center gap-2 pt-2 border-t border-border/60">
        {#if order.invoice_url}
          <a
            href={order.invoice_url}
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs transition-colors"
          >
            {t('commerce.invoice.openDocument')}
          </a>
        {/if}

        {#if canIssue}
          <Button
            variant={status === 'failed' ? 'outline' : 'primary'}
            size="sm"
            loading={busy}
            onclick={issueInvoice}
          >
            {#if busy}
              {t('commerce.invoice.issuing')}
            {:else if status === 'failed'}
              {t('commerce.invoice.retryButton')}
            {:else}
              {t('commerce.invoice.issueButton')}
            {/if}
          </Button>
        {/if}
      </div>
    {/if}
  </div>
{/if}
