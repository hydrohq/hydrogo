<script lang="ts">
  import type { PageData } from './$types';
  import { formatPrice } from '$lib/catalog';
  import { mediaUrl } from '$lib/media';

  let { data }: { data: PageData } = $props();
</script>

<section class="hero">
  <h1>El yapımı deri ürünler</h1>
  <p>Kemer, cüzdan ve çanta koleksiyonumuza göz atın.</p>
</section>

{#if data.products.length === 0}
  <p class="empty-state">Henüz ürün eklenmemiş.</p>
{:else}
  <div class="product-grid">
    {#each data.products as product (product.id)}
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
    {/each}
  </div>
{/if}
