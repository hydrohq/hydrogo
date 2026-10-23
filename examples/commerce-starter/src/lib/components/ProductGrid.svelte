<script lang="ts">
  import type { Product } from '$lib/catalog';
  import ProductCard from './ProductCard.svelte';

  let { products, emptyMessage = 'Bu bölümde henüz ürün yok.' }: { products: Product[]; emptyMessage?: string } = $props();
</script>

{#if products.length === 0}
  <p class="empty-state">{emptyMessage}</p>
{:else}
  <div class="product-grid">
    {#each products as product (product.id)}
      <ProductCard {product} />
    {/each}
  </div>
{/if}
