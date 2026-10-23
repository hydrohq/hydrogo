<script lang="ts">
  import { enhance } from '$app/forms';
  import type { PageData, ActionData } from './$types';
  import { formatPrice, variantPriceCents, type Variant } from '$lib/catalog';
  import { mediaUrl } from '$lib/media';

  let { data, form }: { data: PageData; form: ActionData } = $props();

  // Seeded once from the initial load, not kept in sync with `data` --
  // correct here because SvelteKit creates a fresh component instance for
  // each distinct [slug], so there is no "later data update" to react to.
  let selectedVariantId = $state(data.variants[0]?.id ?? '');
  let quantity = $state(1);

  const hasVariants = $derived(data.variants.length > 0);
  const selectedVariant = $derived(data.variants.find((v: Variant) => v.id === selectedVariantId));
  const priceCents = $derived(selectedVariant ? variantPriceCents(selectedVariant, data.product) : Math.round(data.product.base_price * 100));
  const availableStock = $derived(selectedVariant ? data.stockByVariant[selectedVariant.id] ?? 0 : 1);
  const outOfStock = $derived(hasVariants && availableStock <= 0);
</script>

<svelte:head><title>{data.product.seo_title || data.product.name}</title></svelte:head>

<p class="breadcrumbs"><a href="/">Ana sayfa</a> / {data.product.name}</p>

<div class="product-detail">
  <div class="gallery">
    <img src={mediaUrl(data.product.cover_image)} alt={data.product.name} />
  </div>
  <div>
    <h1>{data.product.name}</h1>
    <p class="price">{formatPrice(priceCents, data.product.currency)}</p>
    {#if data.product.description}
      <div class="description">{@html data.product.description}</div>
    {/if}

    <form method="POST" action="?/addToCart" use:enhance>
      {#if hasVariants}
        <div class="variant-picker">
          <label for="variant">Seçenek</label>
          <select id="variant" name="variant_id" bind:value={selectedVariantId}>
            {#each data.variants as variant (variant.id)}
              <option value={variant.id}>{[variant.size, variant.color].filter(Boolean).join(' / ') || variant.sku}</option>
            {/each}
          </select>
        </div>
      {:else}
        <input type="hidden" name="variant_id" value={data.product.id} />
      {/if}
      <input type="hidden" name="unit_price_cents" value={priceCents} />

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
