<script lang="ts">
  import { onMount } from 'svelte';
  import { cart } from '$lib/stores/cart.svelte';
  import CartDrawer from './CartDrawer.svelte';
  import FastCheckoutModal from './FastCheckoutModal.svelte';
  import ProductDetailModal, { type ProductItem } from './ProductDetailModal.svelte';

  // Initial State from Server SSR or Defaults
  let initial = typeof window !== 'undefined' ? window.__HYDRO_DATA__ : undefined;

  let storeName = $state(initial?.store?.name || 'Hydro Boutique');
  let tagline = $state(initial?.store?.tagline || 'Zamansız parçalar, üstün kumaş kalitesi ve modern butik seçkisi.');
  let storePhone = $state(initial?.store?.phone || '05550000000');
  let announcement = $state(initial?.store?.announcement || 'Tüm siparişlerde ücretsiz kargo ve aynı gün teslimat');
  let products = $state<ProductItem[]>(initial?.products || []);

  let searchQuery = $state('');
  let selectedCategory = $state('Tümü');
  let selectedProduct = $state<ProductItem | null>(null);
  let checkoutOpen = $state(false);

  // Derive unique categories from products
  let categories = $derived.by(() => {
    const set = new Set<string>();
    for (const p of products) {
      if (p.category) set.add(p.category);
    }
    return ['Tümü', ...Array.from(set)];
  });

  // Filtered Products
  let filteredProducts = $derived.by(() => {
    let list = products;
    if (selectedCategory !== 'Tümü') {
      list = list.filter((p) => p.category === selectedCategory);
    }
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      list = list.filter((p) => p.name.toLowerCase().includes(q) || (p.description && p.description.toLowerCase().includes(q)));
    }
    return list;
  });

  onMount(async () => {
    // If no SSR products supplied, fetch from live API
    if (products.length === 0) {
      try {
        const res = await fetch('/api/v1/content/boutique_products');
        if (res.ok) {
          const json = await res.json();
          if (json.data && Array.isArray(json.data)) {
            products = json.data.map((item: any) => ({
              id: item.id,
              name: item.title || item.name || 'Özel Ürün',
              price: item.price || 0,
              price_cents: Math.round((item.price || 0) * 100),
              currency: item.currency || 'TRY',
              description: item.description || '',
              image: item.image || item.featured_image || '',
              category: item.category || 'Koleksiyon',
              in_stock: item.in_stock !== false,
            }));
          }
        }
      } catch {}
    }
  });

  function quickAdd(p: ProductItem, e: MouseEvent) {
    e.stopPropagation();
    const priceCents = p.price_cents || Math.round(p.price * 100);
    cart.addItem({
      id: p.id,
      name: p.name,
      priceCents,
      currency: p.currency || 'TRY',
      image: p.image,
      sku: p.sku,
    });
  }
</script>

<div class="min-h-screen bg-background text-foreground flex flex-col font-sans selection:bg-primary/20">
  <!-- Top Announcement Bar -->
  {#if announcement}
    <div class="bg-primary text-primary-foreground text-xs py-2 px-4 text-center font-medium tracking-wide">
      {announcement}
    </div>
  {/if}

  <!-- Header -->
  <header class="sticky top-0 z-40 bg-background/80 backdrop-blur-md border-b border-border">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between gap-4">
      <!-- Brand -->
      <a href="/" class="flex items-center gap-2 text-foreground font-bold text-lg tracking-tight hover:opacity-90 transition-opacity">
        <span class="w-2.5 h-2.5 rounded-full bg-primary inline-block"></span>
        <span>{storeName}</span>
      </a>

      <!-- Search Bar -->
      <div class="hidden md:flex items-center flex-1 max-w-xs relative">
        <input
          type="search"
          bind:value={searchQuery}
          placeholder="Koleksiyonda ara..."
          class="w-full h-9 pl-9 pr-3 rounded-full border border-input bg-secondary/40 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
        />
        <svg class="w-3.5 h-3.5 text-muted-foreground absolute left-3 top-2.5 pointer-events-none" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
      </div>

      <!-- Actions: Phone & Cart -->
      <div class="flex items-center gap-3">
        {#if storePhone}
          <a
            href="https://wa.me/{storePhone.replace(/[^0-9]/g, '')}"
            target="_blank"
            rel="noopener noreferrer"
            class="hidden sm:inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground font-medium px-3 py-1.5 rounded-full border border-border hover:bg-secondary transition-colors"
          >
            <svg class="w-3.5 h-3.5 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
            </svg>
            <span>WhatsApp Destek</span>
          </a>
        {/if}

        <button
          type="button"
          class="relative h-9 px-3 rounded-full border border-border bg-card hover:bg-secondary transition-colors flex items-center gap-2 text-xs font-medium"
          onclick={() => cart.toggle(true)}
          aria-label="Sepeti Görüntüle"
        >
          <svg class="w-4 h-4 text-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z" />
          </svg>
          <span class="hidden sm:inline">Sepet</span>
          {#if cart.totalCount > 0}
            <span class="w-5 h-5 rounded-full bg-primary text-primary-foreground font-mono font-bold flex items-center justify-center text-[10px]">
              {cart.totalCount}
            </span>
          {/if}
        </button>
      </div>
    </div>
  </header>

  <!-- Hero Section -->
  <section class="py-12 sm:py-16 border-b border-border bg-linear-to-b from-card to-background">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center max-w-3xl">
      <span class="text-[11px] uppercase tracking-widest font-mono text-muted-foreground font-semibold px-3 py-1 rounded-full border border-border bg-background inline-block mb-4">
        2026 Butik Koleksiyonu
      </span>
      <h1 class="text-3xl sm:text-4xl lg:text-5xl font-bold tracking-tight text-foreground">
        {storeName}
      </h1>
      <p class="mt-4 text-sm sm:text-base text-muted-foreground leading-relaxed">
        {tagline}
      </p>
    </div>
  </section>

  <!-- Main Content & Catalog -->
  <main class="flex-1 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 w-full">
    <!-- Category Tabs & Controls -->
    <div class="flex flex-col sm:flex-row items-center justify-between gap-4 mb-8">
      <div class="flex items-center gap-1.5 overflow-x-auto w-full sm:w-auto pb-2 sm:pb-0">
        {#each categories as cat}
          <button
            type="button"
            class="px-3 py-1.5 rounded-full text-xs font-medium whitespace-nowrap transition-colors {selectedCategory === cat ? 'bg-primary text-primary-foreground shadow-xs' : 'border border-border text-muted-foreground hover:text-foreground hover:bg-secondary/60'}"
            onclick={() => (selectedCategory = cat)}
          >
            {cat}
          </button>
        {/each}
      </div>

      <div class="text-xs text-muted-foreground self-end sm:self-auto font-mono">
        {filteredProducts.length} Ürün Listelendi
      </div>
    </div>

    <!-- Product Grid -->
    {#if filteredProducts.length === 0}
      <div class="text-center py-16 border border-dashed border-border rounded-xl">
        <p class="text-sm font-medium text-foreground">Aramanıza uygun ürün bulunamadı.</p>
        <p class="text-xs text-muted-foreground mt-1">Farklı bir kategori seçebilir veya filtreleri temizleyebilirsiniz.</p>
        <button
          type="button"
          class="mt-4 px-4 py-2 rounded-lg bg-secondary text-secondary-foreground text-xs font-medium hover:bg-secondary/80"
          onclick={() => {
            searchQuery = '';
            selectedCategory = 'Tümü';
          }}
        >
          Filtreleri Temizle
        </button>
      </div>
    {:else}
      <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4 sm:gap-6">
        {#each filteredProducts as product (product.id)}
          <div
            class="group rounded-xl border border-border bg-card overflow-hidden hover:border-primary/40 hover:shadow-lg transition-all duration-200 flex flex-col cursor-pointer"
            onclick={() => (selectedProduct = product)}
            role="button"
            tabindex="0"
            onkeydown={(e) => e.key === 'Enter' && (selectedProduct = product)}
          >
            <!-- Image Frame -->
            <div class="aspect-square relative bg-secondary/30 overflow-hidden">
              {#if product.image}
                <img
                  src={product.image}
                  alt={product.name}
                  loading="lazy"
                  class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                />
              {:else}
                <div class="w-full h-full flex items-center justify-center text-muted-foreground">
                  <svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                  </svg>
                </div>
              {/if}

              <!-- Stock Tag -->
              {#if product.in_stock === false}
                <span class="absolute top-2 left-2 px-2 py-0.5 rounded text-[10px] font-medium bg-destructive/90 text-destructive-foreground">
                  Tükendi
                </span>
              {/if}
            </div>

            <!-- Card Body -->
            <div class="p-3.5 flex-1 flex flex-col justify-between space-y-2">
              <div>
                <span class="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                  {product.category || 'Butik'}
                </span>
                <h3 class="text-xs sm:text-sm font-medium text-foreground line-clamp-2 mt-0.5 group-hover:text-primary transition-colors">
                  {product.name}
                </h3>
              </div>

              <div class="pt-2 border-t border-border/60 flex items-center justify-between gap-2">
                <span class="text-xs sm:text-sm font-bold text-foreground font-mono">
                  {(product.price || 0).toLocaleString('tr-TR', { style: 'currency', currency: product.currency || 'TRY' })}
                </span>

                <button
                  type="button"
                  disabled={product.in_stock === false}
                  class="h-8 px-2.5 rounded-md bg-secondary hover:bg-primary hover:text-primary-foreground text-foreground text-xs font-medium transition-colors flex items-center gap-1 shrink-0 disabled:opacity-40"
                  onclick={(e) => quickAdd(product, e)}
                  aria-label="Sepete Ekle"
                >
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
                  </svg>
                  <span class="hidden sm:inline">Ekle</span>
                </button>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </main>

  <!-- Footer -->
  <footer class="mt-16 border-t border-border bg-card py-10">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="grid grid-cols-1 md:grid-cols-4 gap-8 mb-8 text-xs">
        <div class="space-y-2">
          <div class="font-bold text-sm text-foreground">{storeName}</div>
          <p class="text-muted-foreground leading-relaxed">{tagline}</p>
        </div>

        <div>
          <div class="font-semibold text-foreground mb-2">Hızlı Bağlantılar</div>
          <ul class="space-y-1.5 text-muted-foreground">
            <li><a href="/urunler" class="hover:text-foreground">Koleksiyonlar</a></li>
            <li><a href="/hakkimizda" class="hover:text-foreground">Hakkımızda</a></li>
            <li><a href="/iletisim" class="hover:text-foreground">İletişim</a></li>
            <li><a href="/siparis-takip" class="hover:text-foreground">Sipariş Takibi</a></li>
          </ul>
        </div>

        <div>
          <div class="font-semibold text-foreground mb-2">Kurumsal & Yasal</div>
          <ul class="space-y-1.5 text-muted-foreground">
            <li><a href="/mesafeli-satis-sozlesmesi" class="hover:text-foreground">Mesafeli Satış Sözleşmesi</a></li>
            <li><a href="/teslimat-ve-iade" class="hover:text-foreground">Teslimat ve İade</a></li>
            <li><a href="/gizlilik-politikasi" class="hover:text-foreground">Gizlilik Politikası</a></li>
            <li><a href="/kvkk" class="hover:text-foreground">KVKK Aydınlatma Metni</a></li>
          </ul>
        </div>

        <div class="space-y-2">
          <div class="font-semibold text-foreground">Güvenli Altyapı</div>
          <p class="text-muted-foreground leading-relaxed">
            Shopier 256-Bit SSL sertifikası ile 3D Secure korumalı ödeme altyapısı.
          </p>
          <div class="pt-2 flex items-center gap-2 text-muted-foreground">
            <span class="px-2 py-1 rounded bg-secondary text-[10px] font-mono">HydroGo Engine</span>
            <span class="px-2 py-1 rounded bg-secondary text-[10px] font-mono">Shopier Pay</span>
          </div>
        </div>
      </div>

      <div class="border-t border-border pt-6 flex flex-col sm:flex-row items-center justify-between gap-4 text-[11px] text-muted-foreground">
        <div>© {new Date().getFullYear()} {storeName}. Tüm hakları saklıdır.</div>
        <div class="flex items-center gap-4">
          <span>Güvenli Ödeme</span>
          <span>Hızlı Teslimat</span>
          <span>Müşteri Desteği</span>
        </div>
      </div>
    </div>
  </footer>

  <!-- Overlays -->
  <CartDrawer oncheckout={() => (checkoutOpen = true)} />
  <FastCheckoutModal bind:open={checkoutOpen} {storePhone} />
  <ProductDetailModal bind:product={selectedProduct} />
</div>
