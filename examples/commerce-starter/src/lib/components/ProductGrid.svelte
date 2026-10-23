<script lang="ts">
  import type { CurrenciesConfig, Product } from '$lib/catalog';
  import ProductCard from './ProductCard.svelte';

  let {
    products,
    emptyMessage = 'Bu bölümde henüz ürün yok.',
    displayCurrency,
    currencies
  }: { products: Product[]; emptyMessage?: string; displayCurrency?: string; currencies?: CurrenciesConfig } = $props();
</script>

{#if products.length === 0}
  <p class="empty-state">{emptyMessage}</p>
{:else}
  <div class="product-grid">
    {#each products as product (product.id)}
      <ProductCard {product} {displayCurrency} {currencies} />
    {/each}
  </div>
{/if}
