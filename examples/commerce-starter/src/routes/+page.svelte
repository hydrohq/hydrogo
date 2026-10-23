<script lang="ts">
  import type { PageData } from './$types';
  import ProductGrid from '$lib/components/ProductGrid.svelte';
  import { formatPrice } from '$lib/catalog';

  let { data }: { data: PageData } = $props();
</script>

<section class="hero">
  <h1>El yapımı deri ürünler</h1>
  <p>Kemer, cüzdan ve çanta koleksiyonumuza göz atın.</p>
</section>

{#if data.freeShippingAboveCents > 0}
  <div class="notice">
    <p>🚚 {formatPrice(data.freeShippingAboveCents, data.currencies.default)} ve üzeri siparişlerde kargo ücretsiz.</p>
  </div>
{/if}

{#if data.sections.length > 0}
  {#each data.sections as section (section.category.id)}
    <div class="section-heading">
      <h2>{section.category.name}</h2>
      <a href={`/kategori/${section.category.slug}`}>Tümünü gör →</a>
    </div>
    <ProductGrid products={section.products} displayCurrency={data.displayCurrency} currencies={data.currencies} />
  {/each}
{:else}
  <h2 class="section-title">Yeni ürünler</h2>
  <ProductGrid products={data.latest} emptyMessage="Henüz ürün eklenmemiş." displayCurrency={data.displayCurrency} currencies={data.currencies} />
{/if}
