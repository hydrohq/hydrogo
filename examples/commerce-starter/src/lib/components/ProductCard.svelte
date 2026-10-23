<script lang="ts">
  import type { Product } from '$lib/catalog';
  import { formatPrice } from '$lib/catalog';
  import { mediaUrl } from '$lib/media';

  let { product }: { product: Product } = $props();
</script>

<article class="product-card">
  <a href={`/urun/${product.slug}`}>
    <img src={mediaUrl(product.cover_image)} alt={product.name} loading="lazy" />
  </a>
  <div class="body">
    {#if product.expanded?.category}<p class="meta">{product.expanded.category.name}</p>{/if}
    <h2><a href={`/urun/${product.slug}`}>{product.name}</a></h2>
    <p class="price">{formatPrice(Math.round(product.base_price * 100), product.currency)}</p>
  </div>
</article>
