<script lang="ts">
  import { cart } from '$lib/stores/cart.svelte';

  export interface ProductItem {
    id: string;
    slug?: string;
    name: string;
    price: number;
    price_cents?: number;
    compare_at_price?: number;
    currency?: string;
    description?: string;
    image?: string;
    images?: string[];
    category?: string;
    in_stock?: boolean;
    sku?: string;
  }

  let { product = $bindable<ProductItem | null>(null) }: { product: ProductItem | null } = $props();

  let activeImage = $state<string>('');

  $effect(() => {
    if (product) {
      activeImage = product.image || (product.images && product.images[0]) || '';
    }
  });

  function close() {
    product = null;
  }

  function handleAddToCart() {
    if (!product) return;
    const priceCents = product.price_cents || Math.round(product.price * 100);
    cart.addItem({
      id: product.id,
      name: product.name,
      priceCents,
      currency: product.currency || 'TRY',
      image: activeImage || product.image,
      sku: product.sku,
    });
    close();
  }
</script>

{#if product}
  <div
    class="fixed inset-0 bg-black/60 backdrop-blur-xs z-50 flex items-center justify-center p-4 overflow-y-auto"
    onclick={(e) => {
      if (e.target === e.currentTarget) close();
    }}
    onkeydown={(e) => e.key === 'Escape' && close()}
    role="dialog"
    aria-modal="true"
    tabindex="0"
  >
    <div class="bg-card border border-border rounded-xl shadow-2xl max-w-2xl w-full overflow-hidden my-8">
      <!-- Modal Close Button -->
      <div class="p-4 border-b border-border flex items-center justify-between">
        <span class="text-xs uppercase tracking-wider text-muted-foreground font-semibold">
          {product.category || 'Ürün Detayı'}
        </span>
        <button
          type="button"
          class="text-muted-foreground hover:text-foreground p-1 rounded-md hover:bg-secondary transition-colors"
          onclick={close}
          aria-label="Kapat"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-6 p-6">
        <!-- Gallery -->
        <div class="space-y-3">
          <div class="aspect-square rounded-lg border border-border overflow-hidden bg-secondary/30">
            {#if activeImage}
              <img src={activeImage} alt={product.name} class="w-full h-full object-cover" />
            {:else}
              <div class="w-full h-full flex items-center justify-center text-muted-foreground">
                <svg class="w-12 h-12" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
              </div>
            {/if}
          </div>

          {#if product.images && product.images.length > 1}
            <div class="flex gap-2 overflow-x-auto pb-1">
              {#each product.images as img}
                <button
                  type="button"
                  class="w-14 h-14 rounded-md border shrink-0 overflow-hidden {activeImage === img ? 'border-primary ring-2 ring-primary/20' : 'border-border'}"
                  onclick={() => (activeImage = img)}
                >
                  <img src={img} alt="" class="w-full h-full object-cover" />
                </button>
              {/each}
            </div>
          {/if}
        </div>

        <!-- Details -->
        <div class="flex flex-col justify-between space-y-4">
          <div class="space-y-3">
            <h2 class="text-xl font-bold text-foreground tracking-tight">{product.name}</h2>

            <div class="flex items-baseline gap-2">
              <span class="text-2xl font-bold text-foreground font-mono">
                {(product.price || 0).toLocaleString('tr-TR', { style: 'currency', currency: product.currency || 'TRY' })}
              </span>
              {#if product.compare_at_price && product.compare_at_price > product.price}
                <span class="text-sm text-muted-foreground line-through font-mono">
                  {product.compare_at_price.toLocaleString('tr-TR', { style: 'currency', currency: product.currency || 'TRY' })}
                </span>
              {/if}
            </div>

            <div class="flex items-center gap-2 text-xs">
              {#if product.in_stock !== false}
                <span class="inline-flex items-center gap-1.5 text-emerald-500 font-medium">
                  <span class="w-2 h-2 rounded-full bg-emerald-500"></span>
                  Stokta Mevcut
                </span>
              {:else}
                <span class="inline-flex items-center gap-1.5 text-destructive font-medium">
                  <span class="w-2 h-2 rounded-full bg-destructive"></span>
                  Tükendi
                </span>
              {/if}
              {#if product.sku}
                <span class="text-muted-foreground font-mono">SKU: {product.sku}</span>
              {/if}
            </div>

            <div class="text-sm text-muted-foreground leading-relaxed pt-2 border-t border-border">
              {product.description || 'Özel tasarım butik ürün.'}
            </div>
          </div>

          <div class="pt-4 border-t border-border space-y-2">
            <button
              type="button"
              disabled={product.in_stock === false}
              class="w-full h-11 bg-primary text-primary-foreground font-medium rounded-lg hover:bg-primary/90 transition-colors flex items-center justify-center gap-2 text-sm disabled:opacity-50"
              onclick={handleAddToCart}
            >
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z" />
              </svg>
              <span>Sepete Ekle</span>
            </button>

            <p class="text-[11px] text-center text-muted-foreground">
              Aynı gün ücretsiz kargo ve kolay iade garantisi
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
{/if}
