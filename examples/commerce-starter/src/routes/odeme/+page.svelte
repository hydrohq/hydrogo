<script lang="ts">
  import type { PageData, ActionData } from './$types';
  import { formatPrice } from '$lib/catalog';

  let { data, form }: { data: PageData; form: ActionData } = $props();

  const config = $derived(data.checkoutConfig);
  // An estimate only, without a discount (that needs server validation) --
  // the authoritative breakdown, discount included, is whatever the real
  // checkout response/order confirmation says.
  const estimate = $derived.by(() => {
    const subtotal = data.totalCents;
    let tax = 0;
    let afterTax = subtotal;
    if (config.tax_rate > 0) {
      if (config.tax_inclusive) {
        const preTax = Math.round(subtotal / (1 + config.tax_rate));
        tax = subtotal - preTax;
      } else {
        tax = Math.round(subtotal * config.tax_rate);
        afterTax = subtotal + tax;
      }
    }
    const shipping = config.shipping_flat_rate_cents > 0 && (config.shipping_free_above_cents <= 0 || subtotal < config.shipping_free_above_cents) ? config.shipping_flat_rate_cents : 0;
    return { subtotal, tax, taxInclusive: config.tax_inclusive, shipping, total: afterTax + shipping };
  });
</script>

<svelte:head><title>Ödeme</title></svelte:head>

<h1>Ödeme</h1>

<dl class="order-summary">
  <dt>Ara toplam</dt>
  <dd>{formatPrice(estimate.subtotal, data.currency)}</dd>
  {#if config.tax_rate > 0}
    <dt>KDV {estimate.taxInclusive ? '(fiyata dahil)' : ''}</dt>
    <dd>{formatPrice(estimate.tax, data.currency)}</dd>
  {/if}
  {#if config.shipping_flat_rate_cents > 0}
    <dt>Kargo</dt>
    <dd>{estimate.shipping > 0 ? formatPrice(estimate.shipping, data.currency) : 'Ücretsiz'}</dd>
  {/if}
  <dt>Tahmini toplam</dt>
  <dd><strong>{formatPrice(estimate.total, data.currency)}</strong></dd>
</dl>
<p class="meta">İndirim kodu girerseniz kesin tutar sipariş onayı sayfasında görünecektir.</p>

{#if form?.message}<p class="form-error">{form.message}</p>{/if}

<form method="POST" class="checkout-form">
  <div>
    <label for="guest_name">Ad Soyad</label>
    <input id="guest_name" name="guest_name" type="text" required value={form?.guestName ?? ''} />
  </div>
  <div>
    <label for="guest_email">E-posta</label>
    <input id="guest_email" name="guest_email" type="email" required value={form?.guestEmail ?? data.memberEmail} />
  </div>
  <div>
    <label for="guest_phone">Telefon</label>
    <input id="guest_phone" name="guest_phone" type="tel" value={form?.guestPhone ?? ''} />
  </div>
  <div>
    <label for="guest_address">Adres</label>
    <textarea id="guest_address" name="guest_address" required rows="3">{form?.guestAddress ?? ''}</textarea>
  </div>
  <div class="row">
    <div>
      <label for="guest_city">Şehir</label>
      <input id="guest_city" name="guest_city" type="text" required value={form?.guestCity ?? ''} />
    </div>
    <div>
      <label for="guest_district">İlçe</label>
      <input id="guest_district" name="guest_district" type="text" required value={form?.guestDistrict ?? ''} />
    </div>
  </div>
  <div>
    <label for="guest_postal_code">Posta kodu</label>
    <input id="guest_postal_code" name="guest_postal_code" type="text" value={form?.guestPostalCode ?? ''} />
  </div>
  <div>
    <label for="guest_country">Ülke</label>
    <input id="guest_country" name="guest_country" type="text" value={form?.guestCountry ?? 'Türkiye'} />
  </div>
  <div>
    <label for="discount_code">İndirim kodu (opsiyonel)</label>
    <input id="discount_code" name="discount_code" type="text" value={form?.discountCode ?? ''} />
  </div>

  <button class="btn-primary" type="submit">Siparişi tamamla</button>
</form>
