<script lang="ts">
  import { onMount } from 'svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { auth } from '$lib/stores/auth.svelte';
  import { t } from '$lib/i18n/index.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import type { DiscountCode } from '$lib/api/types';
  import { confirmAction } from '$lib/confirm';

  let codes = $state<DiscountCode[]>([]);
  let loading = $state(true);
  let error = $state('');
  let feedback = $state('');
  let saving = $state(false);
  let form = $state({ code: '', kind: 'percent', value: '', min: '', max: '', startsAt: '', endsAt: '' });
  const canWrite = $derived(auth.hasPermission('admin.commerce.write'));

  const fail = (err: unknown, fallback: string) => (err instanceof ApiClientError ? `${err.message}${err.requestId ? ` (${err.requestId})` : ''}` : fallback);

  async function load() {
    loading = true;
    error = '';
    try {
      codes = (await api.get<{ data: DiscountCode[] }>('/api/v1/admin/commerce/discount-codes?limit=200')).data;
    } catch (err) {
      error = fail(err, t('commerce.discounts.loadFailed'));
    } finally {
      loading = false;
    }
  }

  // Percent: whole number 1-100. Fixed: entered in the major currency unit, stored in minor units. Dates: <input type=date> -> RFC3339 (end = end of that day).
  async function create(event: Event) {
    event.preventDefault();
    feedback = '';
    saving = true;
    const value = form.kind === 'fixed' ? Math.round(Number(form.value) * 100) : Math.round(Number(form.value));
    try {
      await api.post('/api/v1/admin/commerce/discount-codes', {
        code: form.code.trim().toUpperCase(),
        kind: form.kind,
        value,
        min_subtotal_cents: Math.round(Number(form.min || 0) * 100),
        max_redemptions: Math.round(Number(form.max || 0)),
        starts_at: form.startsAt ? `${form.startsAt}T00:00:00Z` : '',
        ends_at: form.endsAt ? `${form.endsAt}T23:59:59Z` : ''
      });
      feedback = t('commerce.discounts.created', { code: form.code.trim().toUpperCase() });
      form = { code: '', kind: 'percent', value: '', min: '', max: '', startsAt: '', endsAt: '' };
      await load();
    } catch (err) {
      feedback = fail(err, t('commerce.discounts.createFailed'));
    } finally {
      saving = false;
    }
  }

  async function toggle(item: DiscountCode) {
    feedback = '';
    if (item.active && !(await confirmAction({ title: t('commerce.confirm.disableTitle'), description: t('commerce.confirm.disableDesc', { code: item.code }), confirmLabel: t('commerce.confirm.disableButton'), cancelLabel: t('commerce.confirm.keep'), variant: 'destructive' }))) return;
    try {
      await api.post(`/api/v1/admin/commerce/discount-codes/${encodeURIComponent(item.id)}/active`, { active: !item.active });
      await load();
    } catch (err) {
      feedback = fail(err, t('commerce.discounts.toggleFailed'));
    }
  }

  onMount(() => { void load(); });
  const tl = (cents: number) => new Intl.NumberFormat('tr-TR', { style: 'currency', currency: 'TRY' }).format(cents / 100);
  const amount = (c: DiscountCode) => (c.kind === 'percent' ? `%${c.value}` : tl(c.value));
  const day = (v: string) => (!v || v.startsWith('0001') ? '—' : new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date(v)));
</script>

<section class="w-full space-y-6">
  <header>
    <p class="text-sm text-muted-foreground">{t('commerce.refundsRequired.commerce')}</p>
    <h1 class="text-2xl font-semibold tracking-tight">{t('commerce.discounts.title')}</h1>
    <p class="mt-1 text-sm text-muted-foreground">{t('commerce.discounts.subtitle')}</p>
  </header>
  {#if feedback}<p role="status" class="text-sm text-muted-foreground">{feedback}</p>{/if}
  {#if error}<p role="alert" class="text-sm text-destructive">{error}</p>{/if}

  {#if canWrite}
    <form onsubmit={create} class="grid gap-3 rounded-lg border border-border bg-card p-4 sm:grid-cols-2 lg:grid-cols-4">
      <Input label={t('commerce.discounts.col.code')} bind:value={form.code} required />
      <div class="flex w-full flex-col gap-1.5">
        <label for="dc-kind" class="text-xs font-medium text-foreground/90">{t('commerce.discounts.col.kind')}</label>
        <select id="dc-kind" bind:value={form.kind} class="h-9 rounded-md border border-input bg-background px-2 text-sm">
          <option value="percent">{t('commerce.discounts.kindPercent')}</option>
          <option value="fixed">{t('commerce.discounts.kindFixed')}</option>
        </select>
      </div>
      <Input label={form.kind === 'percent' ? t('commerce.discounts.valuePercent') : t('commerce.discounts.valueFixed')} type="number" min="1" max={form.kind === 'percent' ? '100' : undefined} step={form.kind === 'percent' ? '1' : '0.01'} bind:value={form.value} required />
      <Input label={t('commerce.discounts.col.min')} type="number" min="0" step="0.01" bind:value={form.min} />
      <Input label={t('commerce.discounts.col.max')} type="number" min="0" step="1" hint={t('commerce.discounts.maxHint')} bind:value={form.max} />
      <Input label={t('commerce.discounts.col.starts')} type="date" bind:value={form.startsAt} />
      <Input label={t('commerce.discounts.col.ends')} type="date" bind:value={form.endsAt} />
      <div class="flex items-end"><Button type="submit" disabled={saving}>{saving ? '…' : t('commerce.discounts.create')}</Button></div>
    </form>
  {/if}

  <div class="overflow-x-auto rounded-lg border border-border bg-card">
    <table class="w-full text-left text-sm">
      <thead class="border-b border-border text-xs text-muted-foreground">
        <tr>
          <th class="px-4 py-3 font-medium">{t('commerce.discounts.col.code')}</th>
          <th class="px-4 py-3 font-medium">{t('commerce.discounts.col.discount')}</th>
          <th class="px-4 py-3 font-medium">{t('commerce.discounts.col.min')}</th>
          <th class="px-4 py-3 font-medium">{t('commerce.discounts.col.used')}</th>
          <th class="px-4 py-3 font-medium">{t('commerce.discounts.col.starts')}</th>
          <th class="px-4 py-3 font-medium">{t('commerce.discounts.col.ends')}</th>
          <th class="px-4 py-3 font-medium">{t('commerce.discounts.col.status')}</th>
          <th class="px-4 py-3 font-medium">{t('commerce.refundsRequired.col.action')}</th>
        </tr>
      </thead>
      <tbody>
        {#if loading}
          <tr><td colspan="8" class="px-4 py-8 text-center text-muted-foreground">{t('common.loading')}</td></tr>
        {:else if codes.length === 0}
          <tr><td colspan="8" class="px-4 py-8 text-center text-muted-foreground">{t('commerce.discounts.empty')}</td></tr>
        {:else}
          {#each codes as c (c.id)}
            <tr class="border-b border-border last:border-0">
              <td class="px-4 py-3 font-mono text-xs">{c.code}</td>
              <td class="px-4 py-3">{amount(c)}</td>
              <td class="px-4 py-3">{c.min_subtotal_cents ? tl(c.min_subtotal_cents) : '—'}</td>
              <td class="px-4 py-3">{c.redeemed_count}{c.max_redemptions ? ` / ${c.max_redemptions}` : ''}</td>
              <td class="whitespace-nowrap px-4 py-3">{day(c.starts_at)}</td>
              <td class="whitespace-nowrap px-4 py-3">{day(c.ends_at)}</td>
              <td class="px-4 py-3">{c.active ? t('commerce.discounts.active') : t('commerce.discounts.inactive')}</td>
              <td class="px-4 py-3">
                {#if canWrite}<Button size="sm" variant="outline" onclick={() => toggle(c)}>{c.active ? t('commerce.discounts.disable') : t('commerce.discounts.enable')}</Button>{/if}
              </td>
            </tr>
          {/each}
        {/if}
      </tbody>
    </table>
  </div>
</section>
