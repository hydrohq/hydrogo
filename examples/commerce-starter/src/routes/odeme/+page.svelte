<script lang="ts">
  import type { PageData, ActionData } from './$types';
  import { formatPrice } from '$lib/catalog';

  let { data, form }: { data: PageData; form: ActionData } = $props();
</script>

<svelte:head><title>Ödeme</title></svelte:head>

<h1>Ödeme</h1>

<div class="order-summary">
  <dl>
    <dt>Sepetinizdeki ürün sayısı</dt>
    <dd>{data.lines.reduce((total, line) => total + line.quantity, 0)}</dd>
    <dt>Toplam</dt>
    <dd><strong>{formatPrice(data.totalCents, data.currency)}</strong></dd>
  </dl>
</div>

{#if form?.message}<p class="form-error">{form.message}</p>{/if}

<form method="POST" class="checkout-form">
  <div>
    <label for="guest_name">Ad Soyad</label>
    <input id="guest_name" name="guest_name" type="text" required value={form?.guestName ?? ''} />
  </div>
  <div>
    <label for="guest_email">E-posta</label>
    <input id="guest_email" name="guest_email" type="email" required value={form?.guestEmail ?? ''} />
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
      <label for="guest_postal_code">Posta kodu</label>
      <input id="guest_postal_code" name="guest_postal_code" type="text" value={form?.guestPostalCode ?? ''} />
    </div>
  </div>
  <div>
    <label for="guest_country">Ülke</label>
    <input id="guest_country" name="guest_country" type="text" value={form?.guestCountry ?? 'Türkiye'} />
  </div>

  <button class="btn-primary" type="submit">Siparişi tamamla</button>
</form>
