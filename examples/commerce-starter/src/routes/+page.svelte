<script lang="ts">
  import type { PageData } from './$types';
  import ProductGrid from '$lib/components/ProductGrid.svelte';

  let { data }: { data: PageData } = $props();
</script>

<section class="hero">
  <h1>El yapımı deri ürünler</h1>
  <p>Kemer, cüzdan ve çanta koleksiyonumuza göz atın.</p>
</section>

<div class="notice">
  <p>🚚 500 ₺ ve üzeri siparişlerde kargo ücretsiz.</p>
</div>

{#if data.sections.length > 0}
  {#each data.sections as section (section.category.id)}
    <div class="section-heading">
      <h2>{section.category.name}</h2>
      <a href={`/kategori/${section.category.slug}`}>Tümünü gör →</a>
    </div>
    <ProductGrid products={section.products} />
  {/each}
{:else}
  <h2 class="section-title">Yeni ürünler</h2>
  <ProductGrid products={data.latest} emptyMessage="Henüz ürün eklenmemiş." />
{/if}
