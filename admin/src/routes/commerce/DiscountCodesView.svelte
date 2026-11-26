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

  let popupAvailable = $state(false);
  let popupSaving = $state(false);
  let popupFeedback = $state('');
  let popup = $state({
    enabled: true,
    code: '',
    percent: '',
    trigger: 'scroll',
    delaySeconds: '5',
    dismissDays: '7',
    startsAt: '',
    endsAt: ''
  });

  const fail = (err: unknown, fallback: string) => (err instanceof ApiClientError ? `${err.message}${err.requestId ? ` (${err.requestId})` : ''}` : fallback);

  async function loadPopupSettings() {
    try {
      const res = await api.get<{ data: Array<{ key: string; value: string }> }>('/api/v1/admin/site/settings');
      const map: Record<string, string> = {};
      for (const item of res?.data || []) map[item.key] = item.value;
      popup = {
        enabled: map['storefront.popup_enabled'] !== 'false',
        code: map['storefront.popup_code'] || '',
        percent: map['storefront.popup_percent'] || '',
        trigger: map['storefront.popup_trigger'] || 'scroll',
        delaySeconds: map['storefront.popup_delay_seconds'] || '5',
        dismissDays: map['storefront.popup_dismiss_days'] ?? '7',
        startsAt: map['storefront.popup_starts_at'] || '',
        endsAt: map['storefront.popup_ends_at'] || ''
      };
      popupAvailable = true;
    } catch {
      popupAvailable = false;
    }
  }

  async function savePopupSettings(event: Event) {
    event.preventDefault();
    popupFeedback = '';
    popupSaving = true;
    try {
      await api.put('/api/v1/admin/site/settings', {
        settings: [
          { key: 'storefront.popup_enabled', value: popup.enabled ? 'true' : 'false' },
          { key: 'storefront.popup_code', value: popup.code.trim().toUpperCase() },
          { key: 'storefront.popup_percent', value: String(popup.percent || '').trim() },
          { key: 'storefront.popup_trigger', value: popup.trigger || 'scroll' },
          { key: 'storefront.popup_delay_seconds', value: String(popup.delaySeconds ?? '5').trim() },
          { key: 'storefront.popup_dismiss_days', value: String(popup.dismissDays ?? '7').trim() },
          { key: 'storefront.popup_starts_at', value: popup.startsAt || '' },
          { key: 'storefront.popup_ends_at', value: popup.endsAt || '' }
        ]
      });
      popup.code = popup.code.trim().toUpperCase();
      popupFeedback = t('commerce.discounts.popup.saved');
    } catch (err) {
      popupFeedback = fail(err, t('commerce.discounts.popup.saveFailed'));
    } finally {
      popupSaving = false;
    }
  }

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

  onMount(() => {
    void load();
    void loadPopupSettings();
  });
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

  {#if popupAvailable}
    <form onsubmit={savePopupSettings} class="space-y-4 rounded-lg border border-border bg-card p-5">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 class="text-base font-semibold tracking-tight">{t('commerce.discounts.popup.title')}</h2>
          <p class="mt-0.5 text-xs text-muted-foreground">{t('commerce.discounts.popup.subtitle')}</p>
        </div>
        <label class="flex cursor-pointer select-none items-center gap-2 rounded-md border border-border bg-background px-3 py-2 text-xs font-medium text-foreground">
          <input type="checkbox" bind:checked={popup.enabled} disabled={!canWrite} class="size-4 rounded text-primary" />
          <span>{t('commerce.discounts.popup.enabled')}</span>
        </label>
      </div>

      {#if popupFeedback}<p role="status" class="text-xs text-muted-foreground">{popupFeedback}</p>{/if}

      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <div class="flex w-full flex-col gap-1.5">
          <label for="popup-code" class="text-xs font-medium text-foreground/90">{t('commerce.discounts.popup.code')}</label>
          <input
            id="popup-code"
            list="discount-code-options"
            type="text"
            maxlength="40"
            disabled={!canWrite}
            bind:value={popup.code}
            placeholder="HOSGELDIN10"
            class="h-9 rounded-md border border-input bg-background px-3 text-sm uppercase placeholder:normal-case"
          />
          <datalist id="discount-code-options">
            {#each codes.filter((c) => c.active) as c (c.id)}
              <option value={c.code}></option>
            {/each}
          </datalist>
        </div>

        <Input
          label={t('commerce.discounts.popup.percent')}
          type="number"
          min="1"
          max="99"
          step="1"
          disabled={!canWrite}
          bind:value={popup.percent}
          placeholder="10"
        />

        <div class="flex w-full flex-col gap-1.5">
          <label for="popup-trigger" class="text-xs font-medium text-foreground/90">{t('commerce.discounts.popup.trigger')}</label>
          <select id="popup-trigger" disabled={!canWrite} bind:value={popup.trigger} class="h-9 rounded-md border border-input bg-background px-2 text-sm">
            <option value="scroll">{t('commerce.discounts.popup.triggerScroll')}</option>
            <option value="delay">{t('commerce.discounts.popup.triggerDelay')}</option>
            <option value="immediate">{t('commerce.discounts.popup.triggerImmediate')}</option>
          </select>
        </div>

        {#if popup.trigger === 'delay'}
          <Input
            label={t('commerce.discounts.popup.delaySeconds')}
            type="number"
            min="0"
            max="120"
            step="1"
            disabled={!canWrite}
            bind:value={popup.delaySeconds}
          />
        {/if}

        <Input
          label={t('commerce.discounts.popup.dismissDays')}
          type="number"
          min="0"
          max="365"
          step="1"
          hint={t('commerce.discounts.popup.dismissHint')}
          disabled={!canWrite}
          bind:value={popup.dismissDays}
        />

        <Input
          label={t('commerce.discounts.popup.startsAt')}
          type="date"
          disabled={!canWrite}
          bind:value={popup.startsAt}
        />

        <Input
          label={t('commerce.discounts.popup.endsAt')}
          type="date"
          disabled={!canWrite}
          bind:value={popup.endsAt}
        />
      </div>

      {#if canWrite}
        <div class="flex justify-end">
          <Button type="submit" disabled={popupSaving}>{popupSaving ? '…' : t('commerce.discounts.popup.save')}</Button>
        </div>
      {/if}
    </form>
  {/if}
</section>
