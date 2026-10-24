<script lang="ts">
  import type { PageData } from './$types';
  import ProductGrid from '$lib/components/ProductGrid.svelte';

  let { data }: { data: PageData } = $props();
</script>

<svelte:head><title>"{data.query}" için arama sonuçları</title></svelte:head>

<h1>Arama sonuçları</h1>
<p class="meta">"{data.query}" için {data.products.length} sonuç bulundu.</p>

<ProductGrid products={data.products} emptyMessage="Aramanızla eşleşen ürün bulunamadı." displayCurrency={data.displayCurrency} currencies={data.currencies} />
