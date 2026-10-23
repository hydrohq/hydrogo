<script lang="ts">
  import type { PageData } from './$types';
  import ProductGrid from '$lib/components/ProductGrid.svelte';
  import { PRODUCT_SORTS } from '$lib/sorts';

  let { data }: { data: PageData } = $props();

  function sortHref(key: string): string {
    return key === 'newest' ? `/kategori/${data.category.slug}` : `/kategori/${data.category.slug}?sort=${key}`;
  }
</script>

<svelte:head><title>{data.category.name}</title></svelte:head>

<p class="breadcrumbs"><a href="/">Ana sayfa</a> / {data.category.name}</p>

<section class="hero">
  <h1>{data.category.name}</h1>
  {#if data.category.description}<p>{data.category.description}</p>{/if}
</section>

<div class="category-row">
  {#each Object.entries(PRODUCT_SORTS) as [key, sort] (key)}
    <a href={sortHref(key)} aria-current={data.sortKey === key ? 'true' : undefined}>{sort.label}</a>
  {/each}
</div>

<ProductGrid products={data.products} emptyMessage="Bu kategoride henüz ürün yok." />
