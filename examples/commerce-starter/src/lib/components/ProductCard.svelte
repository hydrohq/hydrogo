<script lang="ts">
  import type { CurrenciesConfig, Product } from '$lib/catalog';
  import { convertDisplayPrice, formatPrice } from '$lib/catalog';
  import { mediaUrl } from '$lib/media';

  let {
    product,
    displayCurrency,
    currencies
  }: { product: Product; displayCurrency?: string; currencies?: CurrenciesConfig } = $props();

  const baseCents = $derived(Math.round(product.base_price * 100));
  const shown = $derived(
    displayCurrency && currencies ? convertDisplayPrice(baseCents, displayCurrency, currencies) : { cents: baseCents, currency: product.currency }
  );
</script>

<article class="product-card">
  <a href={`/urun/${product.slug}`}>
    <img src={mediaUrl(product.cover_image)} alt={product.name} loading="lazy" />
  </a>
  <div class="body">
    {#if product.expanded?.category}<p class="meta">{product.expanded.category.name}</p>{/if}
    <h2><a href={`/urun/${product.slug}`}>{product.name}</a></h2>
    <p class="price">{formatPrice(shown.cents, shown.currency)}</p>
  </div>
</article>
