<script lang="ts">
  import { onMount } from 'svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { confirmAction } from '$lib/confirm';
  import { t } from '$lib/i18n/index.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Button from '$lib/components/ui/Button.svelte';

  export interface ShippingOrder {
    id: string;
    payment_status: 'pending' | 'paid' | 'failed' | 'refunded';
    fulfillment_status: 'unfulfilled' | 'preparing' | 'shipped' | 'delivered' | 'cancelled';
    shipping_carrier?: string;
    shipping_tracking_number?: string;
    shipping_tracking_url?: string;
    shipping_shipment_id?: string;
    shipping_label_url?: string;
    shipped_at?: string;
    delivered_at?: string;
  }

  interface CarrierInfo {
    key: string;
    supports_offers: boolean;
    supports_cancel: boolean;
  }

  interface CarriersResponse {
    configured: boolean;
    default_carrier: string;
    carriers: CarrierInfo[];
  }

  interface ParcelSpec {
    length_cm?: string;
    width_cm?: string;
    height_cm?: string;
    weight_kg?: string;
  }

  interface ShipmentOffer {
    id: string;
    carrier: string;
    carrier_name: string;
    service_code?: string;
    amount: string;
    currency: string;
    estimated_delivery?: string;
  }

  interface ShipmentOfferDraft {
    shipment_id: string;
    cheapest_id?: string;
    fastest_id?: string;
    offers: ShipmentOffer[];
  }

  type Feedback = { type: 'success' | 'error'; text: string } | null;

  interface Props {
    order: ShippingOrder;
    canWrite: boolean;
    onupdated: (updated: ShippingOrder) => void;
    onfeedback: (fb: Feedback) => void;
  }

  let { order, canWrite, onupdated, onfeedback }: Props = $props();

  let carriers = $state<CarrierInfo[]>([]);
  let mode = $state<'geliver' | 'manual'>('manual');

  let lengthCm = $state('');
  let widthCm = $state('');
  let heightCm = $state('');
  let weightKg = $state('');

  let draft = $state<ShipmentOfferDraft | null>(null);
  let selectedOfferId = $state('');
  let offersLoading = $state(false);
  let busy = $state(false);

  // Carrier booked outside the system (manual carrier): the operator types what the carrier's portal gave.
  let carrierName = $state('');
  let trackingNumber = $state('');

  const hasGeliver = $derived(carriers.some((c) => c.key === 'geliver'));
  const hasOtherCarrier = $derived(carriers.some((c) => c.key !== 'geliver'));
  const hasShipmentRecord = $derived(
    Boolean(order.shipping_tracking_number) || order.fulfillment_status === 'shipped' || order.fulfillment_status === 'delivered'
  );
  const canShip = $derived(
    order.payment_status === 'paid' && (order.fulfillment_status === 'unfulfilled' || order.fulfillment_status === 'preparing')
  );
  const canCancel = $derived(
    canWrite &&
      order.fulfillment_status === 'shipped' &&
      (Boolean(order.shipping_shipment_id) || carriers.some((c) => c.supports_cancel))
  );

  let lastOrderId = '';
  $effect(() => {
    if (order.id !== lastOrderId) {
      lastOrderId = order.id;
      draft = null;
      selectedOfferId = '';
    }
  });

  onMount(async () => {
    try {
      const res = await api.get<{ data: CarriersResponse }>('/api/v1/admin/commerce/shipping/carriers');
      const list = Array.isArray(res?.data?.carriers) ? res.data.carriers : [];
      carriers = list;
      mode = list.some((c) => c.key === 'geliver') ? 'geliver' : 'manual';
    } catch {
      carriers = [];
      mode = 'manual';
    }
  });

  function isValidTimestamp(v?: string): v is string {
    return Boolean(v && !v.startsWith('0001') && !Number.isNaN(new Date(v).getTime()));
  }

  function errorText(err: unknown, fallback: string): string {
    if (err instanceof ApiClientError) {
      const suffix = err.requestId ? ` (${t('commerce.requestId')}: ${err.requestId})` : '';
      return `${err.message || fallback}${suffix}`;
    }
    return fallback;
  }

  function shipmentErrorText(err: unknown): string {
    if (err instanceof ApiClientError && err.status === 503) {
      return t('commerce.order.shipmentNoProvider');
    }
    if (err instanceof ApiClientError && err.status === 409) {
      return `${t('commerce.order.notShippableUnpaid')}${err.requestId ? ` (${t('commerce.requestId')}: ${err.requestId})` : ''}`;
    }
    return errorText(err, t('commerce.order.loadFailed'));
  }

  function buildParcel(): ParcelSpec | undefined {
    const p: ParcelSpec = {};
    if (lengthCm.trim()) p.length_cm = lengthCm.trim();
    if (widthCm.trim()) p.width_cm = widthCm.trim();
    if (heightCm.trim()) p.height_cm = heightCm.trim();
    if (weightKg.trim()) p.weight_kg = weightKg.trim();
    return Object.keys(p).length > 0 ? p : undefined;
  }

  async function fetchOffers() {
    offersLoading = true;
    onfeedback(null);
    try {
      const parcel = buildParcel();
      const body: { carrier: string; parcel?: ParcelSpec } = { carrier: 'geliver' };
      if (parcel) body.parcel = parcel;
      const res = await api.post<{ data: ShipmentOfferDraft }>(
        `/api/v1/admin/commerce/orders/${encodeURIComponent(order.id)}/shipment/offers`,
        body
      );
      const nextDraft: ShipmentOfferDraft = res?.data ?? { shipment_id: '', offers: [] };
      draft = {
        ...nextDraft,
        offers: Array.isArray(nextDraft.offers) ? nextDraft.offers : [],
      };
      selectedOfferId = draft.cheapest_id || draft.offers[0]?.id || '';
    } catch (err) {
      onfeedback({ type: 'error', text: errorText(err, t('commerce.order.offersFailed')) });
    } finally {
      offersLoading = false;
    }
  }

  async function acceptSelectedOffer() {
    if (!draft || !selectedOfferId) return;
    busy = true;
    onfeedback(null);
    try {
      const parcel = buildParcel();
      const body: {
        carrier: string;
        offer_id: string;
        shipment_id: string;
        parcel?: ParcelSpec;
      } = {
        carrier: 'geliver',
        offer_id: selectedOfferId,
        shipment_id: draft.shipment_id,
      };
      if (parcel) body.parcel = parcel;
      const res = await api.post<{ data: ShippingOrder }>(
        `/api/v1/admin/commerce/orders/${encodeURIComponent(order.id)}/shipment`,
        body
      );
      if (res?.data) {
        draft = null;
        selectedOfferId = '';
        onupdated(res.data);
        onfeedback({
          type: 'success',
          text: t('commerce.order.shipmentDone', { tracking: res.data.shipping_tracking_number ?? '' }),
        });
      }
    } catch (err) {
      onfeedback({ type: 'error', text: shipmentErrorText(err) });
    } finally {
      busy = false;
    }
  }

  async function quickShipGeliver() {
    const ok = await confirmAction({
      title: t('commerce.confirm.shipTitle'),
      description: t('commerce.confirm.shipDesc'),
      confirmLabel: t('commerce.confirm.shipButton'),
      cancelLabel: t('commerce.confirm.keep'),
      variant: 'primary',
    });
    if (!ok) return;
    busy = true;
    onfeedback(null);
    try {
      const parcel = buildParcel();
      const body: { carrier: string; parcel?: ParcelSpec } = { carrier: 'geliver' };
      if (parcel) body.parcel = parcel;
      const res = await api.post<{ data: ShippingOrder }>(
        `/api/v1/admin/commerce/orders/${encodeURIComponent(order.id)}/shipment`,
        body
      );
      if (res?.data) {
        draft = null;
        selectedOfferId = '';
        onupdated(res.data);
        onfeedback({
          type: 'success',
          text: t('commerce.order.shipmentDone', { tracking: res.data.shipping_tracking_number ?? '' }),
        });
      }
    } catch (err) {
      onfeedback({ type: 'error', text: shipmentErrorText(err) });
    } finally {
      busy = false;
    }
  }

  async function createManualShipment() {
    const ok = await confirmAction({
      title: t('commerce.confirm.shipTitle'),
      description: t('commerce.confirm.shipDesc'),
      confirmLabel: t('commerce.confirm.shipButton'),
      cancelLabel: t('commerce.confirm.keep'),
      variant: 'primary',
    });
    if (!ok) return;
    busy = true;
    onfeedback(null);
    try {
      const body =
        carrierName.trim() || trackingNumber.trim()
          ? { carrier: carrierName.trim(), tracking_number: trackingNumber.trim() }
          : undefined;
      const res = await api.post<{ data: ShippingOrder }>(
        `/api/v1/admin/commerce/orders/${encodeURIComponent(order.id)}/shipment`,
        body
      );
      if (res?.data) {
        onupdated(res.data);
        onfeedback({
          type: 'success',
          text: t('commerce.order.shipmentDone', { tracking: res.data.shipping_tracking_number ?? '' }),
        });
      }
    } catch (err) {
      onfeedback({ type: 'error', text: shipmentErrorText(err) });
    } finally {
      busy = false;
    }
  }

  async function cancelShipment() {
    const ok = await confirmAction({
      title: t('commerce.confirm.cancelShipTitle'),
      description: t('commerce.confirm.cancelShipDesc'),
      confirmLabel: t('commerce.confirm.cancelShipButton'),
      cancelLabel: t('commerce.confirm.keep'),
      variant: 'destructive',
    });
    if (!ok) return;
    busy = true;
    onfeedback(null);
    try {
      const res = await api.post<{ data: ShippingOrder }>(
        `/api/v1/admin/commerce/orders/${encodeURIComponent(order.id)}/shipment/cancel`
      );
      if (res?.data) {
        draft = null;
        selectedOfferId = '';
        onupdated({
          ...res.data,
          shipping_carrier: res.data.shipping_carrier ?? '',
          shipping_tracking_number: res.data.shipping_tracking_number ?? '',
          shipping_tracking_url: res.data.shipping_tracking_url ?? '',
          shipping_shipment_id: res.data.shipping_shipment_id ?? '',
          shipping_label_url: res.data.shipping_label_url ?? '',
          shipped_at: res.data.shipped_at ?? '',
        });
        onfeedback({ type: 'success', text: t('commerce.order.cancelShipmentDone') });
      }
    } catch (err) {
      onfeedback({ type: 'error', text: errorText(err, t('commerce.order.cancelShipmentFailed')) });
    } finally {
      busy = false;
    }
  }
</script>

{#if hasShipmentRecord}
  <div class="p-4 rounded-lg bg-muted/50 border border-border text-xs space-y-2.5">
    <div class="space-y-1.5">
      <div class="flex justify-between gap-3">
        <span class="text-muted-foreground">{t('commerce.order.carrier')}</span>
        <span class="text-foreground font-medium">{order.shipping_carrier || '—'}</span>
      </div>
      <div class="flex justify-between gap-3">
        <span class="text-muted-foreground">{t('commerce.order.tracking')}</span>
        <span class="text-foreground font-mono">{order.shipping_tracking_number || '—'}</span>
      </div>
      {#if isValidTimestamp(order.shipped_at)}
        <div class="flex justify-between gap-3">
          <span class="text-muted-foreground">{t('commerce.order.shippedAt')}</span>
          <span class="text-foreground">{new Date(order.shipped_at).toLocaleString()}</span>
        </div>
      {/if}
      {#if isValidTimestamp(order.delivered_at)}
        <div class="flex justify-between gap-3">
          <span class="text-muted-foreground">{t('commerce.order.deliveredAt')}</span>
          <span class="text-foreground">{new Date(order.delivered_at).toLocaleString()}</span>
        </div>
      {/if}
    </div>

    {#if order.shipping_label_url || order.shipping_tracking_url || canCancel}
      <div class="flex flex-wrap items-center gap-2 pt-2 border-t border-border/60">
        {#if order.shipping_label_url}
          <a
            href={order.shipping_label_url}
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs transition-colors"
          >
            <svg class="size-3.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <polyline points="6 9 6 2 18 2 18 9"></polyline>
              <path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"></path>
              <rect x="6" y="14" width="12" height="8"></rect>
            </svg>
            {t('commerce.order.labelPdf')}
          </a>
        {/if}

        {#if order.shipping_tracking_url}
          <a
            href={order.shipping_tracking_url}
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md border border-input bg-background hover:bg-accent hover:text-accent-foreground text-xs font-medium shadow-xs transition-colors"
          >
            <svg class="size-3.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
              <polyline points="15 3 21 3 21 9"></polyline>
              <line x1="10" y1="14" x2="21" y2="3"></line>
            </svg>
            {t('commerce.order.trackingLink')}
          </a>
        {/if}

        {#if canCancel}
          <Button
            variant="outline"
            size="sm"
            class="text-destructive hover:text-destructive ml-auto"
            loading={busy}
            onclick={cancelShipment}
          >
            {t('commerce.order.cancelShipment')}
          </Button>
        {/if}
      </div>
    {/if}
  </div>
{:else if canWrite}
  <div class="space-y-3">
    {#if hasGeliver && hasOtherCarrier}
      <div class="inline-flex h-8 items-center rounded-lg bg-muted p-1 text-muted-foreground">
        <button
          type="button"
          class="inline-flex items-center justify-center rounded-md px-2.5 py-1 text-xs font-medium transition-colors cursor-pointer {mode === 'geliver' ? 'bg-background text-foreground shadow-xs' : 'hover:text-foreground'}"
          onclick={() => { mode = 'geliver'; }}
        >
          {t('commerce.order.modeGeliver')}
        </button>
        <button
          type="button"
          class="inline-flex items-center justify-center rounded-md px-2.5 py-1 text-xs font-medium transition-colors cursor-pointer {mode === 'manual' ? 'bg-background text-foreground shadow-xs' : 'hover:text-foreground'}"
          onclick={() => { mode = 'manual'; }}
        >
          {t('commerce.order.modeManual')}
        </button>
      </div>
    {/if}

    {#if hasGeliver && mode === 'geliver'}
      <div class="space-y-2.5">
        <p class="text-xs text-muted-foreground">{t('commerce.order.parcelHint')}</p>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
          <input
            type="text"
            inputmode="decimal"
            class="w-full rounded-md border border-input bg-background px-2.5 py-1.5 text-xs"
            placeholder={t('commerce.order.parcelLength')}
            aria-label={t('commerce.order.parcelLength')}
            bind:value={lengthCm}
          />
          <input
            type="text"
            inputmode="decimal"
            class="w-full rounded-md border border-input bg-background px-2.5 py-1.5 text-xs"
            placeholder={t('commerce.order.parcelWidth')}
            aria-label={t('commerce.order.parcelWidth')}
            bind:value={widthCm}
          />
          <input
            type="text"
            inputmode="decimal"
            class="w-full rounded-md border border-input bg-background px-2.5 py-1.5 text-xs"
            placeholder={t('commerce.order.parcelHeight')}
            aria-label={t('commerce.order.parcelHeight')}
            bind:value={heightCm}
          />
          <input
            type="text"
            inputmode="decimal"
            class="w-full rounded-md border border-input bg-background px-2.5 py-1.5 text-xs"
            placeholder={t('commerce.order.parcelWeight')}
            aria-label={t('commerce.order.parcelWeight')}
            bind:value={weightKg}
          />
        </div>

        <Button
          variant="secondary"
          size="sm"
          class="w-full"
          disabled={!canShip || busy}
          loading={offersLoading}
          onclick={fetchOffers}
        >
          {draft ? t('commerce.order.refreshOffers') : t('commerce.order.fetchOffers')}
        </Button>

        {#if draft}
          {#if draft.offers.length === 0}
            <p class="text-xs text-muted-foreground text-center py-2 border border-dashed border-border rounded-md">
              {t('commerce.order.offersEmpty')}
            </p>
          {:else}
            <div class="space-y-1.5">
              {#each draft.offers as offer (offer.id)}
                <label
                  class="flex items-center justify-between gap-3 rounded-lg border px-3 py-2 text-xs cursor-pointer transition-colors {selectedOfferId === offer.id ? 'border-primary bg-primary/5' : 'border-border bg-background hover:bg-muted/40'}"
                >
                  <div class="flex items-center gap-2.5 min-w-0">
                    <input
                      type="radio"
                      name="geliver-offer"
                      value={offer.id}
                      checked={selectedOfferId === offer.id}
                      onchange={() => { selectedOfferId = offer.id; }}
                      class="accent-primary shrink-0"
                    />
                    <div class="min-w-0 space-y-0.5">
                      <div class="flex flex-wrap items-center gap-1.5">
                        <span class="font-medium text-foreground truncate">{offer.carrier_name || offer.carrier}</span>
                        {#if offer.service_code}
                          <span class="text-muted-foreground font-mono text-[11px]">{offer.service_code}</span>
                        {/if}
                        {#if offer.id === draft.cheapest_id}
                          <Badge variant="success" size="sm">{t('commerce.order.offerCheapest')}</Badge>
                        {/if}
                        {#if offer.id === draft.fastest_id}
                          <Badge variant="info" size="sm">{t('commerce.order.offerFastest')}</Badge>
                        {/if}
                      </div>
                      {#if offer.estimated_delivery}
                        <div class="text-[11px] text-muted-foreground">{offer.estimated_delivery}</div>
                      {/if}
                    </div>
                  </div>
                  <div class="font-semibold text-foreground tabular-nums whitespace-nowrap shrink-0">
                    {offer.amount} {offer.currency}
                  </div>
                </label>
              {/each}
            </div>

            <Button
              variant="primary"
              size="sm"
              class="w-full"
              disabled={!canShip || !selectedOfferId || offersLoading}
              loading={busy}
              onclick={acceptSelectedOffer}
            >
              {t('commerce.order.acceptOffer')}
            </Button>
          {/if}
        {/if}

        <Button
          variant={draft ? 'outline' : 'primary'}
          size="sm"
          class="w-full"
          disabled={!canShip || offersLoading}
          loading={busy}
          onclick={quickShipGeliver}
        >
          {t('commerce.order.quickShipGeliver')}
        </Button>
      </div>
    {:else}
      <div class="space-y-2">
        <p class="text-xs text-muted-foreground">{t('commerce.order.createShipmentHint')}</p>
        <input
          class="w-full rounded-md border border-input bg-background px-3 py-1.5 text-sm"
          placeholder={t('commerce.order.carrierName')}
          aria-label={t('commerce.order.carrierName')}
          bind:value={carrierName}
        />
        <input
          class="w-full rounded-md border border-input bg-background px-3 py-1.5 text-sm font-mono"
          placeholder={t('commerce.order.trackingNumber')}
          aria-label={t('commerce.order.trackingNumber')}
          bind:value={trackingNumber}
        />
        <Button
          variant="primary"
          size="sm"
          class="w-full"
          disabled={!canShip}
          loading={busy}
          onclick={createManualShipment}
        >
          {t('commerce.order.createShipment')}
        </Button>
      </div>
    {/if}
  </div>
{/if}
