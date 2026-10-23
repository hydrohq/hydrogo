<script lang="ts">
  import type { PageData } from './$types';
  import { formatPrice } from '$lib/catalog';

  let { data }: { data: PageData } = $props();

  const statusLabel: Record<string, string> = {
    pending: 'Ödeme bekleniyor',
    paid: 'Ödendi',
    failed: 'Ödeme başarısız',
    refunded: 'İade edildi'
  };
</script>

<svelte:head><title>Sipariş {data.order.id}</title></svelte:head>

<h1>Siparişiniz alındı</h1>

<div class="notice {data.order.payment_status === 'paid' ? 'ok' : 'pending'}">
  <p><strong>Durum:</strong> {statusLabel[data.order.payment_status] ?? data.order.payment_status}</p>
  {#if data.order.payment_status === 'pending'}
    <p>Ödemeniz onaylandığında bu sayfadaki durum güncellenecektir. Sayfayı yenileyerek kontrol edebilirsiniz.</p>
  {/if}
</div>

<dl class="order-summary">
  <dt>Sipariş numarası</dt>
  <dd>{data.order.id}</dd>
  <dt>Toplam</dt>
  <dd>{formatPrice(data.order.total_cents, data.order.currency)}</dd>
</dl>

{#if data.order.items?.length}
  <table class="cart-table">
    <thead><tr><th>Ürün</th><th>Adet</th><th>Toplam</th></tr></thead>
    <tbody>
      {#each data.order.items as item (item.sku)}
        <tr>
          <td>{item.title}</td>
          <td>{item.quantity}</td>
          <td>{formatPrice(item.total_cents, data.order.currency)}</td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}

<p><a href="/">Alışverişe devam et</a></p>
