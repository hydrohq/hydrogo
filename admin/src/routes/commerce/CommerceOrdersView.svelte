<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { onMount } from 'svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { auth } from '$lib/stores/auth.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';

  type PaymentStatus = 'pending' | 'paid' | 'failed' | 'refunded';
  type FulfillmentStatus = 'unfulfilled' | 'preparing' | 'shipped' | 'delivered' | 'cancelled';

  interface Order {
    id: string;
    member_id?: string;
    guest_name?: string;
    guest_email?: string;
    guest_phone?: string;
    guest_address?: string;
    guest_city?: string;
    guest_district?: string;
    guest_country?: string;
    guest_postal_code?: string;
    currency: string;
    subtotal_cents: number;
    discount_code?: string;
    discount_cents: number;
    tax_cents: number;
    shipping_cents: number;
    total_cents: number;
    payment_status: PaymentStatus;
    fulfillment_status: FulfillmentStatus;
    shipping_carrier?: string;
    shipping_tracking_number?: string;
    shipping_tracking_url?: string;
    shipped_at?: string;
    created_at: string;
  }

  interface OrderItem {
    variant_id: string;
    sku: string;
    title: string;
    quantity: number;
    unit_price_cents: number;
    total_cents: number;
  }

  type OrderDetail = Order & { items: OrderItem[] };

  const PAGE_SIZE = 25;
  const PAYMENT_FILTERS: Array<'all' | PaymentStatus> = ['all', 'pending', 'paid', 'failed', 'refunded'];

  let orders = $state<Order[]>([]);
  let loading = $state(true);
  let loadingMore = $state(false);
  let hasMore = $state(false);
  let listError = $state<string | null>(null);
  let search = $state('');
  let paymentFilter = $state<'all' | PaymentStatus>('all');

  let selected = $state<OrderDetail | null>(null);
  let detailOpen = $state(false);
  let detailLoading = $state(false);
  let actionLoading = $state(false);
  let feedback = $state<{ type: 'success' | 'error'; text: string } | null>(null);

  const canWrite = $derived(auth.hasPermission('admin.commerce.write'));

  const visibleOrders = $derived.by(() => {
    const q = search.trim().toLowerCase();
    if (!q) return orders;
    return orders.filter((o) => o.id.toLowerCase().includes(q) || (o.guest_email ?? '').toLowerCase().includes(q) || (o.guest_name ?? '').toLowerCase().includes(q));
  });

  function errorText(err: unknown, fallback: string): string {
    if (err instanceof ApiClientError) {
      const suffix = err.requestId ? ` (${t('commerce.requestId')}: ${err.requestId})` : '';
      return `${err.message || fallback}${suffix}`;
    }
    return fallback;
  }

  function money(cents: number, currency: string): string {
    return `${(cents / 100).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })} ${currency}`;
  }

  function paymentLabel(status: PaymentStatus): string {
    return t(`commerce.payment.${status}` as Parameters<typeof t>[0]);
  }

  function fulfillmentLabel(status: FulfillmentStatus): string {
    return t(`commerce.fulfillment.${status}` as Parameters<typeof t>[0]);
  }

  function paymentVariant(status: PaymentStatus): 'success' | 'warning' | 'destructive' | 'secondary' {
    if (status === 'paid') return 'success';
    if (status === 'pending') return 'warning';
    if (status === 'failed') return 'destructive';
    return 'secondary';
  }

  function fulfillmentVariant(status: FulfillmentStatus): 'success' | 'info' | 'destructive' | 'secondary' {
    if (status === 'delivered') return 'success';
    if (status === 'shipped' || status === 'preparing') return 'info';
    if (status === 'cancelled') return 'destructive';
    return 'secondary';
  }

  async function fetchPage(offset: number): Promise<{ data: Order[]; hasMore: boolean }> {
    const params: Record<string, string | number> = { limit: PAGE_SIZE, offset };
    if (paymentFilter !== 'all') params.payment_status = paymentFilter;
    const res = await api.get<{ data: Order[]; metadata?: { has_more?: boolean } }>('/api/v1/admin/commerce/orders', { params });
    return { data: Array.isArray(res?.data) ? res.data : [], hasMore: !!res?.metadata?.has_more };
  }

  async function loadOrders() {
    loading = true;
    listError = null;
    try {
      const page = await fetchPage(0);
      orders = page.data;
      hasMore = page.hasMore;
    } catch (err) {
      orders = [];
      hasMore = false;
      listError = errorText(err, t('commerce.orders.loadFailed'));
    } finally {
      loading = false;
    }
  }

  async function loadMore() {
    loadingMore = true;
    try {
      const page = await fetchPage(orders.length);
      orders = [...orders, ...page.data];
      hasMore = page.hasMore;
    } catch (err) {
      listError = errorText(err, t('commerce.orders.loadFailed'));
    } finally {
      loadingMore = false;
    }
  }

  async function openOrder(order: Order) {
    selected = { ...order, items: [] };
    detailOpen = true;
    detailLoading = true;
    feedback = null;
    try {
      const res = await api.get<{ data: OrderDetail }>(`/api/v1/admin/commerce/orders/${encodeURIComponent(order.id)}`);
      if (res?.data) selected = res.data;
    } catch (err) {
      feedback = { type: 'error', text: errorText(err, t('commerce.order.loadFailed')) };
    } finally {
      detailLoading = false;
    }
  }

  const canShipSelected = $derived(
    !!selected && selected.payment_status === 'paid' && (selected.fulfillment_status === 'unfulfilled' || selected.fulfillment_status === 'preparing')
  );

  async function createShipment() {
    if (!selected) return;
    actionLoading = true;
    feedback = null;
    try {
      const res = await api.post<{ data: Order }>(`/api/v1/admin/commerce/orders/${encodeURIComponent(selected.id)}/shipment`);
      if (res?.data) {
        selected = { ...selected, ...res.data };
        orders = orders.map((o) => (o.id === res.data.id ? { ...o, ...res.data } : o));
        feedback = { type: 'success', text: t('commerce.order.shipmentDone', { tracking: res.data.shipping_tracking_number ?? '' }) };
      }
    } catch (err) {
      if (err instanceof ApiClientError && err.status === 503) {
        feedback = { type: 'error', text: t('commerce.order.shipmentNoProvider') };
      } else if (err instanceof ApiClientError && err.status === 409) {
        feedback = { type: 'error', text: `${t('commerce.order.notShippableUnpaid')}${err.requestId ? ` (${t('commerce.requestId')}: ${err.requestId})` : ''}` };
      } else {
        feedback = { type: 'error', text: errorText(err, t('commerce.order.loadFailed')) };
      }
    } finally {
      actionLoading = false;
    }
  }

  function filterLabel(filter: 'all' | PaymentStatus): string {
    return filter === 'all' ? t('commerce.orders.filter.all') : paymentLabel(filter);
  }

  function selectFilter(filter: 'all' | PaymentStatus) {
    if (paymentFilter === filter) return;
    paymentFilter = filter;
    loadOrders();
  }

  function addressLines(o: Order): string[] {
    const cityLine = [o.guest_district, o.guest_city].filter(Boolean).join(' / ');
    return [o.guest_address, cityLine, [o.guest_postal_code, o.guest_country].filter(Boolean).join(' ')].filter((line) => !!line && line.trim() !== '') as string[];
  }

  onMount(() => {
    loadOrders();
  });
</script>

<Modal
  open={detailOpen}
  title={selected?.id ?? t('commerce.order.detail')}
  description={selected ? `${selected.member_id ? t('commerce.order.member') : t('commerce.order.guest')} · ${new Date(selected.created_at).toLocaleString()}` : ''}
  size="xl"
  onclose={() => { detailOpen = false; }}
>
  {#if selected}
    <div class="space-y-4">
      {#if feedback}
        <div class="p-3 rounded-lg text-xs {feedback.type === 'success' ? 'bg-emerald-500/10 border border-emerald-500/20 text-emerald-500' : 'bg-destructive/10 border border-destructive/20 text-destructive'} flex items-center justify-between gap-3">
          <span>{feedback.text}</span>
          <button type="button" class="hover:underline cursor-pointer shrink-0" onclick={() => { feedback = null; }}>{t('common.dismiss')}</button>
        </div>
      {/if}

      <div class="flex flex-wrap items-center gap-2">
        <Badge variant={paymentVariant(selected.payment_status)} size="sm">{paymentLabel(selected.payment_status)}</Badge>
        <Badge variant={fulfillmentVariant(selected.fulfillment_status)} size="sm">{fulfillmentLabel(selected.fulfillment_status)}</Badge>
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <div class="p-4 rounded-lg bg-muted/50 border border-border space-y-1 text-xs">
          <div class="text-muted-foreground font-medium mb-1">{t('commerce.orders.col.customer')}</div>
          <div class="text-foreground font-medium">{selected.guest_name || '—'}</div>
          <div class="text-foreground">{selected.guest_email || '—'}</div>
          {#if selected.guest_phone}<div class="text-muted-foreground">{selected.guest_phone}</div>{/if}
        </div>
        <div class="p-4 rounded-lg bg-muted/50 border border-border space-y-1 text-xs">
          <div class="text-muted-foreground font-medium mb-1">{t('commerce.order.shippingAddress')}</div>
          {#each addressLines(selected) as line}
            <div class="text-foreground">{line}</div>
          {:else}
            <div class="text-muted-foreground">—</div>
          {/each}
        </div>
      </div>

      <div class="rounded-lg border border-border overflow-hidden">
        <div class="px-4 py-2 bg-muted/50 border-b border-border text-xs font-medium text-muted-foreground">{t('commerce.order.items')}</div>
        {#if detailLoading}
          <div class="p-4 text-xs text-muted-foreground">{t('commerce.orders.loading')}</div>
        {:else}
          <table class="w-full text-left text-xs">
            <tbody class="divide-y divide-border">
              {#each selected.items as item (item.variant_id)}
                <tr>
                  <td class="px-4 py-2">
                    <div class="text-foreground font-medium">{item.title || item.variant_id}</div>
                    {#if item.sku}<div class="text-muted-foreground">{item.sku}</div>{/if}
                  </td>
                  <td class="px-4 py-2 text-muted-foreground whitespace-nowrap">{item.quantity} × {money(item.unit_price_cents, selected.currency)}</td>
                  <td class="px-4 py-2 text-right text-foreground font-medium whitespace-nowrap">{money(item.total_cents, selected.currency)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      </div>

      <dl class="grid grid-cols-2 gap-y-1 text-xs max-w-xs ml-auto">
        <dt class="text-muted-foreground">{t('commerce.order.subtotal')}</dt><dd class="text-right">{money(selected.subtotal_cents, selected.currency)}</dd>
        {#if selected.discount_cents > 0}
          <dt class="text-muted-foreground">{t('commerce.order.discount')}{selected.discount_code ? ` (${selected.discount_code})` : ''}</dt><dd class="text-right">−{money(selected.discount_cents, selected.currency)}</dd>
        {/if}
        {#if selected.tax_cents > 0}
          <dt class="text-muted-foreground">{t('commerce.order.tax')}</dt><dd class="text-right">{money(selected.tax_cents, selected.currency)}</dd>
        {/if}
        {#if selected.shipping_cents > 0}
          <dt class="text-muted-foreground">{t('commerce.order.shipping')}</dt><dd class="text-right">{money(selected.shipping_cents, selected.currency)}</dd>
        {/if}
        <dt class="text-foreground font-semibold pt-1 border-t border-border">{t('commerce.order.total')}</dt><dd class="text-right font-semibold pt-1 border-t border-border">{money(selected.total_cents, selected.currency)}</dd>
      </dl>

      <div class="pt-3 border-t border-border space-y-3">
        {#if selected.shipping_tracking_number}
          <div class="p-4 rounded-lg bg-muted/50 border border-border text-xs space-y-1">
            <div class="flex justify-between gap-3"><span class="text-muted-foreground">{t('commerce.order.carrier')}</span><span class="text-foreground">{selected.shipping_carrier}</span></div>
            <div class="flex justify-between gap-3"><span class="text-muted-foreground">{t('commerce.order.tracking')}</span><span class="text-foreground font-mono">{selected.shipping_tracking_number}</span></div>
            {#if selected.shipped_at}
              <div class="flex justify-between gap-3"><span class="text-muted-foreground">{t('commerce.order.shippedAt')}</span><span class="text-foreground">{new Date(selected.shipped_at).toLocaleString()}</span></div>
            {/if}
            {#if selected.shipping_tracking_url}
              <a class="text-primary hover:underline" href={selected.shipping_tracking_url} target="_blank" rel="noopener noreferrer">{t('commerce.order.trackingLink')}</a>
            {/if}
          </div>
        {:else if canWrite}
          <p class="text-xs text-muted-foreground">{t('commerce.order.createShipmentHint')}</p>
          <Button variant="primary" size="sm" class="w-full" disabled={!canShipSelected} loading={actionLoading} onclick={createShipment}>
            {t('commerce.order.createShipment')}
          </Button>
        {/if}
      </div>
    </div>
  {/if}
  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => { detailOpen = false; }}>{t('common.close')}</Button>
  {/snippet}
</Modal>

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-2">
    <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 px-4 lg:px-6">
        <div class="inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground overflow-x-auto max-w-full">
          {#each PAYMENT_FILTERS as filter (filter)}
            <button
              type="button"
              class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {paymentFilter === filter ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
              onclick={() => selectFilter(filter)}
            >
              {filterLabel(filter)}
            </button>
          {/each}
        </div>

        <div class="relative w-full sm:w-64">
          <svg class="absolute left-2.5 top-2.5 size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"></circle>
            <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
          </svg>
          <input
            type="search"
            placeholder={t('commerce.orders.search')}
            bind:value={search}
            class="h-8 w-full rounded-md border border-input bg-background pl-8 pr-3 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
          />
        </div>
      </div>

      <div class="px-4 lg:px-6">
        <div class="overflow-hidden rounded-xl border border-border bg-card shadow-xs">
          {#if listError}
            <div class="p-4 text-xs text-destructive border-b border-border">{listError}</div>
          {/if}
          {#if loading}
            <div class="p-12 text-center text-xs text-muted-foreground">{t('commerce.orders.loading')}</div>
          {:else if visibleOrders.length === 0}
            <div class="p-12 text-center text-xs text-muted-foreground">{t('commerce.orders.empty')}</div>
          {:else}
            <div class="overflow-x-auto">
              <table class="w-full text-left text-sm text-foreground caption-bottom">
                <thead class="sticky top-0 z-10 bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
                  <tr>
                    <th class="h-10 px-4 font-medium">{t('commerce.orders.col.order')}</th>
                    <th class="h-10 px-4 font-medium">{t('commerce.orders.col.customer')}</th>
                    <th class="h-10 px-4 font-medium text-right">{t('commerce.orders.col.total')}</th>
                    <th class="h-10 px-4 font-medium">{t('commerce.orders.col.payment')}</th>
                    <th class="h-10 px-4 font-medium">{t('commerce.orders.col.fulfillment')}</th>
                    <th class="h-10 px-4 font-medium">{t('commerce.orders.col.date')}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-border">
                  {#each visibleOrders as o (o.id)}
                    <tr class="hover:bg-muted/50 transition-colors cursor-pointer" onclick={() => openOrder(o)}>
                      <td class="px-4 py-3 font-mono text-xs text-foreground">{o.id}</td>
                      <td class="px-4 py-3">
                        <div class="text-foreground font-medium text-xs">{o.guest_name || '—'}</div>
                        <div class="text-muted-foreground text-xs">{o.guest_email || ''}</div>
                      </td>
                      <td class="px-4 py-3 text-right text-xs font-medium whitespace-nowrap">{money(o.total_cents, o.currency)}</td>
                      <td class="px-4 py-3"><Badge variant={paymentVariant(o.payment_status)} size="sm">{paymentLabel(o.payment_status)}</Badge></td>
                      <td class="px-4 py-3"><Badge variant={fulfillmentVariant(o.fulfillment_status)} size="sm">{fulfillmentLabel(o.fulfillment_status)}</Badge></td>
                      <td class="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">{new Date(o.created_at).toLocaleString()}</td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
            <div class="flex items-center justify-between px-4 py-3 border-t border-border text-xs text-muted-foreground">
              <span>{t('commerce.orders.showing', { shown: visibleOrders.length })}</span>
              {#if hasMore}
                <Button variant="outline" size="sm" loading={loadingMore} onclick={loadMore}>{t('commerce.orders.loadMore')}</Button>
              {/if}
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>
