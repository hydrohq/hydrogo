<script lang="ts">
  import { onMount } from 'svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { auth } from '$lib/stores/auth.svelte';
  import { t } from '$lib/i18n/index.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import type { RefundRequired } from '$lib/api/types';

  let refunds = $state<RefundRequired[]>([]);
  let loading = $state(true);
  let error = $state('');
  let notes = $state<Record<string, string>>({});
  let resolving = $state('');
  let feedback = $state('');
  const canWrite = $derived(auth.hasPermission('admin.commerce.write'));

  async function load() {
    loading = true;
    error = '';
    try {
      const result = await api.get<{ data: RefundRequired[] }>('/api/v1/admin/commerce/refunds-required?unresolved_only=false&limit=100');
      refunds = result.data;
    } catch (err) {
      error = err instanceof ApiClientError ? `${err.message}${err.requestId ? ` (${err.requestId})` : ''}` : t('commerce.refundsRequired.loadFailed');
    } finally {
      loading = false;
    }
  }

  async function resolve(item: RefundRequired) {
    resolving = item.id;
    feedback = '';
    try {
      await api.post(`/api/v1/admin/commerce/refunds-required/${encodeURIComponent(item.id)}/resolve`, { note: notes[item.id] ?? '' });
      feedback = t('commerce.refundsRequired.resolved', { orderId: item.order_id });
      window.dispatchEvent(new CustomEvent('refunds-required:changed'));
      await load();
    } catch (err) {
      feedback = err instanceof ApiClientError ? `${err.message}${err.requestId ? ` (${err.requestId})` : ''}` : t('commerce.refundsRequired.resolveFailed');
    } finally {
      resolving = '';
    }
  }

  onMount(() => { void load(); });
  const money = (cents: number, currency: string) => new Intl.NumberFormat(undefined, { style: 'currency', currency }).format(cents / 100);
  const date = (value: string) => new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value));
</script>

<section class="w-full space-y-6">
  <header>
    <p class="text-sm text-muted-foreground">{t('commerce.refundsRequired.commerce')}</p>
    <h1 class="text-2xl font-semibold tracking-tight">{t('commerce.refundsRequired.title')}</h1>
    <p class="mt-1 text-sm text-muted-foreground">{t('commerce.refundsRequired.subtitle')}</p>
  </header>
  {#if feedback}<p role="status" class="text-sm text-muted-foreground">{feedback}</p>{/if}
  {#if error}<p role="alert" class="text-sm text-destructive">{error}</p>{/if}
  <div class="overflow-x-auto rounded-lg border border-border bg-card">
    <table class="w-full text-left text-sm">
      <thead class="border-b border-border text-xs text-muted-foreground">
        <tr>
          <th class="whitespace-nowrap px-4 py-3 font-medium">{t('commerce.refundsRequired.col.orderId')}</th>
          <th class="whitespace-nowrap px-4 py-3 font-medium">{t('commerce.refundsRequired.col.provider')}</th>
          <th class="whitespace-nowrap px-4 py-3 font-medium">{t('commerce.refundsRequired.col.amount')}</th>
          <th class="whitespace-nowrap px-4 py-3 font-medium">{t('commerce.refundsRequired.col.currency')}</th>
          <th class="whitespace-nowrap px-4 py-3 font-medium">{t('commerce.refundsRequired.col.reason')}</th>
          <th class="whitespace-nowrap px-4 py-3 font-medium">{t('commerce.refundsRequired.col.date')}</th>
          <th class="whitespace-nowrap px-4 py-3 font-medium">{t('commerce.refundsRequired.col.status')}</th>
          <th class="whitespace-nowrap px-4 py-3 font-medium">{t('commerce.refundsRequired.col.action')}</th>
        </tr>
      </thead>
      <tbody>
        {#if loading}
          <tr><td colspan="8" class="px-4 py-8 text-center text-muted-foreground">{t('common.loading')}</td></tr>
        {:else if refunds.length === 0}
          <tr><td colspan="8" class="px-4 py-8 text-center text-muted-foreground">{t('commerce.refundsRequired.empty')}</td></tr>
        {:else}
          {#each refunds as item (item.id)}
            <tr class="border-b border-border last:border-0">
              <td class="whitespace-nowrap px-4 py-3 font-mono text-xs">{item.order_id}</td>
              <td class="px-4 py-3">{item.provider}</td>
              <td class="whitespace-nowrap px-4 py-3">{money(item.amount_cents, item.currency)}</td>
              <td class="px-4 py-3">{item.currency}</td>
              <td class="min-w-48 px-4 py-3">{item.reason}</td>
              <td class="whitespace-nowrap px-4 py-3">{date(item.created_at)}</td>
              <td class="px-4 py-3">{item.resolved_at ? t('commerce.refundsRequired.statusResolved') : t('commerce.refundsRequired.statusPending')}</td>
              <td class="min-w-56 px-4 py-3">
                {#if !item.resolved_at && canWrite}
                  <div class="flex items-center gap-2">
                    <Input aria-label={t('commerce.refundsRequired.noteAria', { orderId: item.order_id })} placeholder={t('commerce.refundsRequired.notePlaceholder')} bind:value={notes[item.id]} />
                    <Button size="sm" disabled={resolving === item.id} onclick={() => resolve(item)}>{resolving === item.id ? '…' : t('commerce.refundsRequired.resolveAction')}</Button>
                  </div>
                {:else if item.resolved_at}
                  <span class="text-muted-foreground">{item.resolved_at ? date(item.resolved_at) : ''}</span>
                {/if}
              </td>
            </tr>
          {/each}
        {/if}
      </tbody>
    </table>
  </div>
</section>