<script lang="ts">
  import { enhance } from '$app/forms';
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

<svelte:head><title>Hesabım</title></svelte:head>

<h1>Hesabım</h1>
<p class="meta">{data.member.email}</p>

<form method="POST" action="?/logout" use:enhance>
  <button class="btn-secondary" type="submit">Çıkış yap</button>
</form>

<h2 class="section-title">Siparişlerim</h2>
{#if data.orders.length === 0}
  <p class="empty-state">Henüz bir siparişiniz yok.</p>
{:else}
  <table class="cart-table">
    <thead><tr><th>Sipariş</th><th>Durum</th><th>Toplam</th></tr></thead>
    <tbody>
      {#each data.orders as order (order.id)}
        <tr>
          <td><a href={`/siparis/${order.id}`}>{order.id}</a></td>
          <td>{statusLabel[order.payment_status] ?? order.payment_status}</td>
          <td>{formatPrice(order.total_cents, order.currency)}</td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}
