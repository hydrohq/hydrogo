<script lang="ts">
  import { enhance } from '$app/forms';
  import type { PageData, ActionData } from './$types';
  import { formatPrice } from '$lib/catalog';
  import { mediaUrl } from '$lib/media';

  let { data, form }: { data: PageData; form: ActionData } = $props();
</script>

<svelte:head><title>Sepet</title></svelte:head>

<h1>Sepetim</h1>

{#if form?.message}<p class="form-error">{form.message}</p>{/if}

{#if data.lines.length === 0}
  <p class="empty-state">Sepetiniz boş. <a href="/">Alışverişe başlayın</a>.</p>
{:else}
  <table class="cart-table">
    <thead>
      <tr><th>Ürün</th><th>Birim fiyat</th><th>Adet</th><th>Toplam</th><th></th></tr>
    </thead>
    <tbody>
      {#each data.lines as line (line.itemId)}
        <tr>
          <td>
            <div style="display:flex; align-items:center; gap:.6rem;">
              <img src={mediaUrl(line.productImage)} alt={line.productName} style="width:3.2rem; height:3.2rem; object-fit:cover; border-radius:.3rem;" />
              <div>
                <a href={`/urun/${line.productSlug}`}>{line.productName}</a>
                {#if line.variantLabel}<div class="meta">{line.variantLabel}</div>{/if}
              </div>
            </div>
          </td>
          <td>{formatPrice(line.unitPriceCents, data.currency)}</td>
          <td>
            <form method="POST" action="?/updateQuantity" use:enhance class="qty-form">
              <input type="hidden" name="variant_id" value={line.variantId} />
              <input type="number" name="quantity" min="1" max="20" value={line.quantity} />
              <button class="btn-secondary" type="submit">Güncelle</button>
            </form>
          </td>
          <td>{formatPrice(line.totalCents, data.currency)}</td>
          <td>
            <form method="POST" action="?/remove" use:enhance>
              <input type="hidden" name="item_id" value={line.itemId} />
              <button class="btn-secondary" type="submit">Kaldır</button>
            </form>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>

  <div class="cart-summary">
    <span>Toplam:</span>
    <strong>{formatPrice(data.totalCents, data.currency)}</strong>
  </div>

  <a class="btn-primary" href="/odeme" style="display:inline-block; text-decoration:none;">Ödemeye geç</a>
{/if}
