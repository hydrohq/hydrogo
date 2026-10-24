<script lang="ts">
  import { enhance } from '$app/forms';
  import type { PageData, ActionData } from './$types';
  import { convertDisplayPrice, formatPrice, variantPriceCents, type Variant } from '$lib/catalog';
  import { mediaUrl } from '$lib/media';
  import ProductGrid from '$lib/components/ProductGrid.svelte';

  let { data, form }: { data: PageData; form: ActionData } = $props();

  // Seeded once from the initial load, not kept in sync with `data` --
  // correct here because SvelteKit creates a fresh component instance for
  // each distinct [slug], so there is no "later data update" to react to.
  let selectedVariantId = $state(data.variants[0]?.id ?? '');
  let quantity = $state(1);
  let activeImage = $state(data.variants[0]?.image ?? data.product.cover_image);

  const hasVariants = $derived(data.variants.length > 0);
  const selectedVariant = $derived(data.variants.find((v: Variant) => v.id === selectedVariantId));
  const priceCents = $derived(selectedVariant ? variantPriceCents(selectedVariant, data.product) : Math.round(data.product.base_price * 100));
  const availableStock = $derived(selectedVariant ? data.stockByVariant[selectedVariant.id] ?? 0 : 1);
  const outOfStock = $derived(hasVariants && availableStock <= 0);

  const thumbnails = $derived([data.product.cover_image, ...data.gallery.map((img) => img.image)]);
  // priceCents is the catalog's base-currency price, for display only:
  // HydroGo prices the cart line itself. Only the visible <p.price> text
  // below shows the converted display currency.
  const shownPrice = $derived(convertDisplayPrice(priceCents, data.displayCurrency, data.currencies));

  function selectVariant(id: string) {
    selectedVariantId = id;
    const variant = data.variants.find((v: Variant) => v.id === id);
    activeImage = variant?.image ?? data.product.cover_image;
  }
</script>

<svelte:head><title>{data.product.seo_title || data.product.name}</title></svelte:head>

<p class="breadcrumbs"><a href="/">Ana sayfa</a> / {data.product.name}</p>

<div class="product-detail">
  <div class="gallery">
    <img src={mediaUrl(activeImage)} alt={data.product.name} />
    {#if thumbnails.length > 1}
      <div class="gallery-thumbs">
        {#each thumbnails as thumb (thumb)}
          <button type="button" class="thumb" class:active={thumb === activeImage} onclick={() => (activeImage = thumb)}>
            <img src={mediaUrl(thumb)} alt="" />
          </button>
        {/each}
      </div>
    {/if}
  </div>
  <div>
    <h1>{data.product.name}</h1>
    <p class="price">{formatPrice(shownPrice.cents, shownPrice.currency)}</p>
    {#if data.product.description}
      <div class="description">{@html data.product.description}</div>
    {/if}

    <form method="POST" action="?/addToCart" use:enhance>
      {#if hasVariants}
        <div class="variant-picker">
          <label for="variant">Seçenek</label>
          <select id="variant" name="variant_id" value={selectedVariantId} onchange={(e) => selectVariant(e.currentTarget.value)}>
            {#each data.variants as variant (variant.id)}
              <option value={variant.id}>{[variant.size, variant.color].filter(Boolean).join(' / ') || variant.sku}</option>
            {/each}
          </select>
        </div>
      {:else}
        <input type="hidden" name="variant_id" value={data.product.id} />
      {/if}

      {#if hasVariants}
        {#if outOfStock}
          <p class="stock-note out">Bu seçenek stokta yok.</p>
        {:else}
          <p class="stock-note in">Stokta {availableStock} adet.</p>
        {/if}
      {/if}

      <div class="variant-picker">
        <label for="quantity">Adet</label>
        <input id="quantity" type="number" name="quantity" min="1" max="20" bind:value={quantity} />
      </div>

      {#if form?.message}<p class="form-error">{form.message}</p>{/if}
      {#if form?.added}<p class="stock-note in">Ürün sepete eklendi.</p>{/if}

      <button class="btn-primary" type="submit" disabled={outOfStock}>Sepete ekle</button>
    </form>
  </div>
</div>

{#if data.relatedProducts.length > 0}
  <h2 class="section-title">Benzer ürünler</h2>
  <ProductGrid products={data.relatedProducts} displayCurrency={data.displayCurrency} currencies={data.currencies} />
{/if}
