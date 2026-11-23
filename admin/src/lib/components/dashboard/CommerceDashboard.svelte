<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { router } from '$lib/router.svelte';
  import { t } from '$lib/i18n/index.svelte';

  interface Order {
    id: string;
    created_at: string;
    currency: string;
    total_cents: number;
    payment_status: string;
    fulfillment_status: string;
    guest_name?: string;
  }
  interface Props {
    members?: number;
    visitors?: number;
  }
  let { members = 0, visitors = 0 }: Props = $props();

  const DAY = 86_400_000;
  let orders = $state<Order[]>([]);
  let loading = $state(true);
  let partial = $state(false);
  let refunds = $state(0);
  let pendingReviews = $state(0);
  let openReturns = $state(0);
  let lowStock = $state<{ variant_id: string; available: number }[]>([]);

  async function load() {
    // Newest first; 5 pages of 100 cover the 14-day window for a shop this size.
    const all: Order[] = [];
    try {
      for (let offset = 0; offset < 500; offset += 100) {
        const res = await api.get<{ data: Order[]; metadata?: { has_more?: boolean } }>('/api/v1/admin/commerce/orders', { params: { limit: 100, offset } });
        all.push(...(res.data ?? []));
        const oldest = all[all.length - 1];
        if (!res.metadata?.has_more || (oldest && Date.now() - new Date(oldest.created_at).getTime() > 14 * DAY)) break;
      }
    } catch {
      partial = true;
    }
    orders = all;
    loading = false;
  }

  const safe = <T,>(p: Promise<T>, fallback: T) => p.catch(() => { partial = true; return fallback; });

  onMount(() => {
    void load();
    void safe(api.get<{ data: { count: number } }>('/api/v1/admin/commerce/refunds-required/count'), { data: { count: 0 } }).then((r) => (refunds = r.data.count));
    void safe(api.get<{ data: unknown[] }>('/api/v1/admin/content/reviews?status=draft&limit=100'), { data: [] }).then((r) => (pendingReviews = r.data.length));
    void safe(api.get<{ data: { variant_id: string; available: number }[] }>('/api/v1/admin/commerce/stock/low?threshold=3'), { data: [] }).then((r) => (lowStock = r.data));
    void safe(api.get<{ data: { status?: string }[] }>('/api/v1/admin/content/return_requests?limit=100'), { data: [] }).then(
      (r) => (openReturns = r.data.filter((x) => !x.status || x.status === 'beklemede' || x.status === 'onaylandi' || x.status === 'kargoda').length)
    );
  });

  const startOfToday = () => { const d = new Date(); d.setHours(0, 0, 0, 0); return d.getTime(); };
  const paid = $derived(orders.filter((o) => o.payment_status === 'paid'));
  const sum = (list: Order[]) => list.reduce((s, o) => s + o.total_cents, 0);
  const todayPaid = $derived(paid.filter((o) => new Date(o.created_at).getTime() >= startOfToday()));
  const weekPaid = $derived(paid.filter((o) => new Date(o.created_at).getTime() >= startOfToday() - 6 * DAY));
  const toShip = $derived(paid.filter((o) => o.fulfillment_status === 'unfulfilled' || o.fulfillment_status === 'preparing').length);
  const awaiting = $derived(orders.filter((o) => o.payment_status === 'pending').length);

  const days = $derived.by(() => {
    const base = startOfToday();
    return Array.from({ length: 14 }, (_, i) => {
      const from = base - (13 - i) * DAY;
      const total = sum(paid.filter((o) => { const at = new Date(o.created_at).getTime(); return at >= from && at < from + DAY; }));
      return { from, total };
    });
  });
  const max = $derived(Math.max(1, ...days.map((d) => d.total)));

  const money = (cents: number) => new Intl.NumberFormat('tr-TR', { style: 'currency', currency: 'TRY', maximumFractionDigits: 0 }).format(cents / 100);
  const day = (ms: number) => new Intl.DateTimeFormat('tr-TR', { day: 'numeric', month: 'short' }).format(ms);
  const shortId = (id: string) => id.replace('order_', '').slice(0, 8).toUpperCase();
  const statusLabel: Record<string, string> = { paid: 'Paid', pending: 'Pending', failed: 'Failed', refunded: 'Refunded' };
  const statusTone: Record<string, string> = { paid: 'text-emerald-600', pending: 'text-amber-600', failed: 'text-destructive', refunded: 'text-muted-foreground' };

  const go = (path: string) => router.navigate(path);
  const kpiCards = $derived([
    { label: t('dashboard.commerce.today'), value: money(sum(todayPaid)), note: t('dashboard.commerce.todayNote', { count: todayPaid.length }), to: '/commerce', alert: false },
    { label: t('dashboard.commerce.week'), value: money(sum(weekPaid)), note: t('dashboard.commerce.weekNote', { count: weekPaid.length }), to: '/commerce', alert: false },
    { label: t('dashboard.commerce.toShip'), value: String(toShip), note: t('dashboard.commerce.toShipNote'), to: '/commerce', alert: toShip > 0 },
    { label: t('dashboard.commerce.awaiting'), value: String(awaiting), note: t('dashboard.commerce.awaitingNote'), to: '/commerce', alert: false },
  ]);
  const queueItems = $derived([
    { label: t('dashboard.commerce.refunds'), value: String(refunds), note: t('dashboard.commerce.refundsNote'), to: '/commerce/refunds-required', alert: refunds > 0, tone: 'bg-destructive' },
    { label: t('dashboard.commerce.reviews'), value: String(pendingReviews), note: t('dashboard.commerce.reviewsNote'), to: '/content/reviews', alert: pendingReviews > 0, tone: 'bg-amber-500' },
    { label: t('dashboard.commerce.returns'), value: String(openReturns), note: t('dashboard.commerce.returnsNote'), to: '/content/return_requests', alert: openReturns > 0, tone: 'bg-amber-500' },
    { label: t('dashboard.commerce.lowStock'), value: String(lowStock.length), note: t('dashboard.commerce.lowStockNote'), to: '/commerce/stock', alert: lowStock.length > 0, tone: 'bg-amber-500' },
    { label: t('dashboard.commerce.members'), value: members.toLocaleString(), note: visitors > 0 ? `${visitors} visitors today` : t('dashboard.commerce.membersNote'), to: '/members', alert: false, tone: 'bg-muted-foreground' },
  ]);
  const periodTotal = $derived(days.reduce((acc, d) => acc + d.total, 0));
</script>

<div class="flex flex-col gap-6">
  <header>
    <h1 class="text-2xl font-medium tracking-tight text-foreground">{t('dashboard.commerce.title')}</h1>
    <p class="mt-1 text-sm text-muted-foreground">{t('dashboard.commerce.subtitle')}</p>
    {#if partial}<p role="status" class="mt-2 text-xs text-amber-500">{t('dashboard.commerce.loadFailed')}</p>{/if}
  </header>

  <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    {#each kpiCards as card (card.label)}
      <button type="button" onclick={() => go(card.to)} class="rounded-xl border border-border bg-card p-5 text-left transition-colors duration-150 hover:bg-muted/40 cursor-pointer">
        <span class="text-xs font-medium text-muted-foreground">{card.label}</span>
        <span class="mt-2 block text-2xl font-medium tabular-nums tracking-tight {card.alert ? 'text-amber-500' : 'text-foreground'}">{loading ? '…' : card.value}</span>
        <span class="mt-1.5 block text-xs text-muted-foreground">{card.note}</span>
      </button>
    {/each}
  </div>

  <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
    <section class="flex flex-col justify-between rounded-xl border border-border bg-card p-5 lg:col-span-2">
      <div class="flex items-baseline justify-between gap-4">
        <h2 class="text-sm font-medium text-foreground">{t('dashboard.commerce.chart')}</h2>
        <span class="text-xs tabular-nums text-muted-foreground">
          {t('dashboard.commerce.periodTotal')}: <span class="font-medium text-foreground">{loading ? '…' : money(periodTotal)}</span>
        </span>
      </div>
      <div class="mt-6 flex h-44 items-end gap-1.5" role="img" aria-label={t('dashboard.commerce.chart')}>
        {#each days as d, i (d.from)}
          {@const isToday = i === days.length - 1}
          <div class="flex h-full flex-1 flex-col items-center justify-end gap-1.5" title="{day(d.from)}: {money(d.total)}">
            <div
              class="w-full rounded-t-sm transition-colors duration-150 {isToday ? 'bg-primary' : 'bg-muted-foreground/25 hover:bg-primary/60'}"
              style="height: {Math.max(d.total > 0 ? 6 : 2, (d.total / max) * 100)}%"
            ></div>
            <span class="font-mono text-[10px] tabular-nums {isToday ? 'font-medium text-foreground' : 'text-muted-foreground'}">{new Date(d.from).getDate()}</span>
          </div>
        {/each}
      </div>
    </section>

    <section class="flex flex-col rounded-xl border border-border bg-card">
      <div class="border-b border-border px-5 py-3.5">
        <h2 class="text-sm font-medium text-foreground">{t('dashboard.commerce.operations')}</h2>
      </div>
      <div class="divide-y divide-border flex-1 flex flex-col justify-center">
        {#each queueItems as item (item.label)}
          <button
            type="button"
            onclick={() => go(item.to)}
            class="flex items-center justify-between gap-3 px-5 py-3 text-left transition-colors duration-150 hover:bg-muted/40 cursor-pointer"
          >
            <div class="min-w-0 flex items-center gap-2.5">
              <span class="size-1.5 shrink-0 rounded-full {item.alert ? item.tone : 'bg-border'}"></span>
              <div class="min-w-0">
                <div class="truncate text-xs font-medium text-foreground">{item.label}</div>
                <div class="truncate text-[11px] text-muted-foreground">{item.note}</div>
              </div>
            </div>
            <span class="font-mono text-sm font-medium tabular-nums {item.alert ? 'text-foreground' : 'text-muted-foreground'}">
              {loading ? '…' : item.value}
            </span>
          </button>
        {/each}
      </div>
    </section>
  </div>

  <section class="rounded-xl border border-border bg-card">
    <div class="flex items-center justify-between border-b border-border px-5 py-3.5">
      <h2 class="text-sm font-medium text-foreground">{t('dashboard.commerce.recent')}</h2>
      <button type="button" class="text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer" onclick={() => go('/commerce')}>{t('dashboard.commerce.viewAll')} →</button>
    </div>
    {#if orders.length === 0 && !loading}
      <p class="px-5 py-8 text-center text-sm text-muted-foreground">{t('dashboard.commerce.none')}</p>
    {:else}
      <div class="overflow-x-auto">
      <table class="w-full text-left text-sm">
        <thead class="text-xs text-muted-foreground">
          <tr>
            <th class="px-4 py-2.5 font-medium sm:px-5">{t('dashboard.commerce.col.order')}</th>
            <th class="hidden px-5 py-2.5 font-medium sm:table-cell">{t('dashboard.commerce.col.date')}</th>
            <th class="px-4 py-2.5 font-medium sm:px-5">{t('dashboard.commerce.col.status')}</th>
            <th class="px-4 py-2.5 text-right font-medium sm:px-5">{t('dashboard.commerce.col.total')}</th>
          </tr>
        </thead>
        <tbody>
          {#each orders.slice(0, 8) as o (o.id)}
            <tr
              class="border-t border-border hover:bg-muted/40 transition-colors cursor-pointer"
              onclick={() => go(`/commerce?order=${encodeURIComponent(o.id)}`)}
            >
              <td class="px-4 py-3 sm:px-5"><span class="font-mono text-xs text-foreground">#{shortId(o.id)}</span>{#if o.guest_name}<span class="ml-2 block text-muted-foreground sm:inline">{o.guest_name}</span>{/if}<span class="mt-0.5 block text-[11px] text-muted-foreground sm:hidden">{day(new Date(o.created_at).getTime())}</span></td>
              <td class="hidden px-5 py-3 text-muted-foreground sm:table-cell">{day(new Date(o.created_at).getTime())}</td>
              <td class="px-4 py-3 sm:px-5 {statusTone[o.payment_status] ?? ''}">{statusLabel[o.payment_status] ?? o.payment_status}</td>
              <td class="whitespace-nowrap px-4 py-3 text-right font-mono text-xs tabular-nums sm:px-5">{money(o.total_cents)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
      </div>
    {/if}
  </section>
</div>
