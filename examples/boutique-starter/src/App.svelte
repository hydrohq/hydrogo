<script lang="ts">
  import { onMount } from 'svelte';
  import { resolveThemeTokens, type StoreSettings } from './lib/theme';
  import { buildWhatsAppOrderUrl } from './lib/whatsapp';

  interface Category {
    id: string;
    name: string;
    slug: string;
  }

  interface Product {
    id: string;
    title: string;
    slug: string;
    sku?: string;
    category: string;
    price: number;
    compare_at_price?: number;
    cover_image?: string;
    sizes?: string;
    colors?: string;
    stock_quantity: number;
    badge?: string;
    fabric_care?: string;
    description?: string;
    payment_link?: string;
    featured?: boolean;
  }

  let settings = $state<StoreSettings>({
    store_name: 'İpek & Keten Atelier',
    slogan: 'Zamansız Zarafet & Seçkin Tasarımlar',
    whatsapp_number: '+905550000000',
    instagram_handle: '@ipekketenatelier',
    announcement_bar: '1500 TL ve Üzeri Siparişlerde Ücretsiz Kargo · WhatsApp ile Hızlı Sipariş',
    free_shipping_threshold: 1500,
    theme_preset: 'minimal_cream',
    hero_title: 'Yeni Sezon İpek & Keten Koleksiyonu',
    hero_cta_text: 'Koleksiyonu Keşfet'
  });

  let categories = $state<Category[]>([
    { id: 'cat-1', name: 'Yeni Sezon', slug: 'yeni-sezon' },
    { id: 'cat-2', name: 'Üst Giyim & Gömlek', slug: 'ust-giyim' },
    { id: 'cat-3', name: 'Elbise & Takım', slug: 'elbise-takim' }
  ]);

  let products = $state<Product[]>([
    {
      id: 'prod-1',
      title: 'Saf İpek Kruvaze Gömlek',
      slug: 'saf-ipek-kruvaze-gomlek',
      sku: 'BTK-101',
      category: 'cat-1',
      price: 1850,
      compare_at_price: 2200,
      sizes: 'S, M, L',
      colors: 'Ekru, Siyah, Çağla Yeşili',
      stock_quantity: 14,
      badge: 'Yeni Sezon',
      fabric_care: '%100 Saf İpek dokuma. Elde soğuk suda yıkama önerilir.',
      description: 'Doğal dokusu ve dökümlü kesimiyle gün boyu zarafet sunan özel tasarım ipek gömlek.'
    },
    {
      id: 'prod-2',
      title: 'Yüksek Bel Keten Palazzo Pantolon',
      slug: 'yuksek-bel-keten-palazzo-pantolon',
      sku: 'BTK-102',
      category: 'cat-2',
      price: 1490,
      sizes: 'S, M, L, XL',
      colors: 'Bej, Taş Rengi, Lacivert',
      stock_quantity: 20,
      badge: 'Özel Tasarım',
      fabric_care: '%100 Doğal Keten. 30 derecede hassas yıkama.',
      description: 'Nefes alan keten kumaşı ve rahat kalıbıyla gardırobunuzun vazgeçilmez parçası.'
    },
    {
      id: 'prod-3',
      title: 'Saten Dokulu Midi Boy Abiye Elbise',
      slug: 'saten-dokulu-midi-boy-abiye-elbise',
      sku: 'BTK-103',
      category: 'cat-3',
      price: 2450,
      compare_at_price: 2950,
      sizes: 'S, M, L',
      colors: 'Şampanya, Zümrüt, Siyah',
      stock_quantity: 3,
      badge: 'Son 3 Ürün',
      fabric_care: 'Yalnızca kuru temizleme.',
      description: 'Özel davetler için tasarlanmış, zarif sırt detaylı midi boy saten elbise.'
    }
  ]);

  let selectedCategory = $state('all');
  let activeProduct = $state<Product | null>(null);
  let selectedSize = $state('M');
  let selectedColor = $state('');
  let customerName = $state('');
  let shippingAddress = $state('');

  const tokens = $derived(resolveThemeTokens(settings));

  const filteredProducts = $derived(
    selectedCategory === 'all'
      ? products
      : products.filter((p) => p.category === selectedCategory)
  );

  onMount(async () => {
    try {
      const [settingsRes, catsRes, prodsRes] = await Promise.all([
        fetch('/api/v1/content/store_settings'),
        fetch('/api/v1/content/boutique_categories'),
        fetch('/api/v1/content/boutique_products')
      ]);
      if (settingsRes.ok) {
        const body = await settingsRes.json();
        if (Array.isArray(body.data) && body.data.length > 0) {
          settings = { ...settings, ...(body.data[0].data ?? body.data[0]) };
        }
      }
      if (catsRes.ok) {
        const body = await catsRes.json();
        if (Array.isArray(body.data) && body.data.length > 0) {
          categories = body.data.map((r: any) => ({ id: r.id, ...(r.data ?? r) }));
        }
      }
      if (prodsRes.ok) {
        const body = await prodsRes.json();
        if (Array.isArray(body.data) && body.data.length > 0) {
          products = body.data.map((r: any) => ({ id: r.id, ...(r.data ?? r) }));
        }
      }
    } catch {
      // Fallback to embedded luxury preview catalog when running standalone
    }
  });

  function openProduct(prod: Product) {
    activeProduct = prod;
    const sizeList = (prod.sizes ?? 'Standart').split(',').map((s) => s.trim()).filter(Boolean);
    const colorList = (prod.colors ?? 'Standart').split(',').map((c) => c.trim()).filter(Boolean);
    selectedSize = sizeList[0] ?? 'Standart';
    selectedColor = colorList[0] ?? 'Standart';
  }
</script>

<div
  style="
    --bg-primary: {tokens.bgPrimary};
    --text-primary: {tokens.textPrimary};
    --accent: {tokens.accent};
    --font-heading: {tokens.fontHeading};
    --radius: {tokens.radius};
    background-color: var(--bg-primary);
    color: var(--text-primary);
    min-height: 100vh;
    font-family: 'Plus Jakarta Sans', system-ui, sans-serif;
  "
>
  <!-- Top Announcement Bar -->
  {#if settings.announcement_bar}
    <div
      style="background-color: var(--accent); color: #ffffff;"
      class="announcement-bar"
    >
      {settings.announcement_bar}
    </div>
  {/if}

  <!-- Sticky Luxury Header -->
  <header class="site-header">
    <div class="header-inner">
      <div>
        <h1 class="brand-title">{settings.store_name}</h1>
        {#if settings.slogan}
          <p class="brand-slogan">{settings.slogan}</p>
        {/if}
      </div>
      <a
        href="https://wa.me/{settings.whatsapp_number.replace(/[^0-9]/g, '')}"
        target="_blank"
        rel="noopener noreferrer"
        class="whatsapp-pill"
      >
        WhatsApp Destek
      </a>
    </div>

    <!-- Category Strip -->
    <nav class="category-strip" aria-label="Kategoriler">
      <button
        type="button"
        class:active={selectedCategory === 'all'}
        onclick={() => (selectedCategory = 'all')}
      >
        Tüm Koleksiyon
      </button>
      {#each categories as cat (cat.id)}
        <button
          type="button"
          class:active={selectedCategory === cat.id}
          onclick={() => (selectedCategory = cat.id)}
        >
          {cat.name}
        </button>
      {/each}
    </nav>
  </header>

  <!-- Hero Section -->
  <section class="hero-banner">
    <div class="hero-content">
      <span class="hero-eyebrow">YENİ SEZON SEÇKİSİ</span>
      <h2 class="hero-heading">{settings.hero_title || settings.store_name}</h2>
      <p class="hero-sub">{settings.slogan || 'Özel tasarım koleksiyonumuzu inceleyin.'}</p>
    </div>
  </section>

  <!-- Product Grid (2-col mobile, 3/4-col desktop) -->
  <main class="catalog-container">
    <div class="product-grid">
      {#each filteredProducts as prod (prod.id)}
        <article class="product-card" onclick={() => openProduct(prod)}>
          <div class="product-visual">
            {#if prod.badge}
              <span class="product-badge">{prod.badge}</span>
            {/if}
            <div class="product-placeholder">
              <span>{prod.title.slice(0, 1)}</span>
            </div>
          </div>
          <div class="product-meta">
            <h3 class="product-title">{prod.title}</h3>
            <div class="price-row">
              {#if prod.compare_at_price && prod.compare_at_price > prod.price}
                <span class="price-old">{prod.compare_at_price.toLocaleString('tr-TR')} TL</span>
              {/if}
              <span class="price-current">{prod.price.toLocaleString('tr-TR')} TL</span>
            </div>
            <button type="button" class="quick-order-btn" onclick={() => openProduct(prod)}>
              İncele & Sipariş Ver
            </button>
          </div>
        </article>
      {/each}
    </div>
  </main>

  <!-- Product Detail & Quick WhatsApp Order Drawer -->
  {#if activeProduct}
    {@const sizes = (activeProduct.sizes ?? 'Standart').split(',').map((s) => s.trim()).filter(Boolean)}
    {@const colors = (activeProduct.colors ?? 'Standart').split(',').map((c) => c.trim()).filter(Boolean)}
    <div class="drawer-backdrop" role="dialog" aria-modal="true">
      <div class="drawer-sheet">
        <div class="drawer-header">
          <h3 class="drawer-title">{activeProduct.title}</h3>
          <button type="button" class="close-btn" onclick={() => (activeProduct = null)}>✕</button>
        </div>

        <div class="drawer-body">
          <div class="price-row large">
            {#if activeProduct.compare_at_price && activeProduct.compare_at_price > activeProduct.price}
              <span class="price-old">{activeProduct.compare_at_price.toLocaleString('tr-TR')} TL</span>
            {/if}
            <span class="price-current">{activeProduct.price.toLocaleString('tr-TR')} TL</span>
          </div>

          <div class="selector-block">
            <span class="selector-label">Beden Seçimi</span>
            <div class="chip-row">
              {#each sizes as sz}
                <button
                  type="button"
                  class="option-chip"
                  class:selected={selectedSize === sz}
                  onclick={() => (selectedSize = sz)}
                >
                  {sz}
                </button>
              {/each}
            </div>
          </div>

          <div class="selector-block">
            <span class="selector-label">Renk Seçimi</span>
            <div class="chip-row">
              {#each colors as clr}
                <button
                  type="button"
                  class="option-chip"
                  class:selected={selectedColor === clr}
                  onclick={() => (selectedColor = clr)}
                >
                  {clr}
                </button>
              {/each}
            </div>
          </div>

          {#if activeProduct.fabric_care}
            <div class="care-box">
              <strong>Kumaş & Bakım:</strong> {activeProduct.fabric_care}
            </div>
          {/if}

          <div class="customer-fields">
            <input
              type="text"
              bind:value={customerName}
              placeholder="Adınız Soyadınız (İsteğe bağlı)"
            />
            <input
              type="text"
              bind:value={shippingAddress}
              placeholder="Teslimat İl / İlçe / Adres (İsteğe bağlı)"
            />
          </div>

          <div class="cta-stack">
            <a
              href={buildWhatsAppOrderUrl(
                settings.whatsapp_number,
                settings.store_name,
                {
                  title: activeProduct.title,
                  sku: activeProduct.sku,
                  size: selectedSize,
                  color: selectedColor,
                  price: activeProduct.price,
                  quantity: 1
                },
                customerName,
                shippingAddress
              )}
              target="_blank"
              rel="noopener noreferrer"
              class="wa-order-cta"
            >
              WhatsApp ile Hemen Sipariş Ver
            </a>

            {#if activeProduct.payment_link}
              <a
                href={activeProduct.payment_link}
                target="_blank"
                rel="noopener noreferrer"
                class="card-order-cta"
              >
                Kredi Kartı ile Satın Al
              </a>
            {/if}
          </div>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .announcement-bar {
    text-align: center;
    font-size: 12px;
    padding: 8px 16px;
    letter-spacing: 0.04em;
    font-weight: 500;
  }
  .site-header {
    position: sticky;
    top: 0;
    z-index: 20;
    background: var(--bg-primary);
    border-bottom: 1px solid rgba(128, 128, 128, 0.18);
    padding: 14px 20px 10px;
  }
  .header-inner {
    max-width: 1120px;
    margin: 0 auto;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .brand-title {
    font-family: var(--font-heading);
    font-size: 26px;
    margin: 0;
    letter-spacing: 0.02em;
  }
  .brand-slogan {
    margin: 2px 0 0;
    font-size: 11px;
    opacity: 0.7;
  }
  .whatsapp-pill {
    text-decoration: none;
    font-size: 12px;
    font-weight: 600;
    padding: 7px 14px;
    border-radius: var(--radius);
    border: 1px solid var(--accent);
    color: var(--text-primary);
  }
  .category-strip {
    max-width: 1120px;
    margin: 12px auto 0;
    display: flex;
    gap: 8px;
    overflow-x: auto;
    padding-bottom: 4px;
  }
  .category-strip button {
    background: transparent;
    border: 1px solid rgba(128, 128, 128, 0.25);
    color: var(--text-primary);
    padding: 6px 14px;
    border-radius: var(--radius);
    font-size: 12px;
    cursor: pointer;
    white-space: nowrap;
  }
  .category-strip button.active {
    background: var(--accent);
    color: #fff;
    border-color: var(--accent);
  }
  .hero-banner {
    max-width: 1120px;
    margin: 24px auto;
    padding: 40px 24px;
    border-radius: var(--radius);
    border: 1px solid rgba(128, 128, 128, 0.18);
    text-align: center;
  }
  .hero-eyebrow {
    font-size: 11px;
    letter-spacing: 0.18em;
    opacity: 0.75;
  }
  .hero-heading {
    font-family: var(--font-heading);
    font-size: clamp(28px, 5vw, 44px);
    margin: 10px 0 8px;
  }
  .hero-sub {
    font-size: 14px;
    opacity: 0.8;
    margin: 0;
  }
  .catalog-container {
    max-width: 1120px;
    margin: 0 auto;
    padding: 0 20px 64px;
  }
  .product-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
  }
  @media (min-width: 768px) {
    .product-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 24px;
    }
  }
  .product-card {
    border: 1px solid rgba(128, 128, 128, 0.18);
    border-radius: var(--radius);
    overflow: hidden;
    display: flex;
    flex-direction: column;
    cursor: pointer;
  }
  .product-visual {
    position: relative;
    aspect-ratio: 3 / 4;
    background: rgba(128, 128, 128, 0.08);
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .product-placeholder {
    font-family: var(--font-heading);
    font-size: 48px;
    opacity: 0.35;
  }
  .product-badge {
    position: absolute;
    top: 10px;
    left: 10px;
    background: var(--accent);
    color: #fff;
    font-size: 10px;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: var(--radius);
  }
  .product-meta {
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .product-title {
    font-size: 14px;
    font-weight: 600;
    margin: 0;
  }
  .price-row {
    display: flex;
    align-items: baseline;
    gap: 8px;
    font-size: 14px;
  }
  .price-row.large {
    font-size: 20px;
    margin-bottom: 16px;
  }
  .price-old {
    text-decoration: line-through;
    opacity: 0.55;
    font-size: 12px;
  }
  .price-current {
    font-weight: 700;
  }
  .quick-order-btn {
    margin-top: 4px;
    padding: 8px;
    border-radius: var(--radius);
    border: 1px solid var(--accent);
    background: transparent;
    color: var(--text-primary);
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }
  .drawer-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.55);
    display: flex;
    justify-content: flex-end;
    z-index: 50;
  }
  .drawer-sheet {
    background: var(--bg-primary);
    color: var(--text-primary);
    width: 100%;
    max-width: 420px;
    height: 100%;
    padding: 20px;
    box-sizing: border-box;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }
  .drawer-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid rgba(128, 128, 128, 0.2);
    padding-bottom: 12px;
    margin-bottom: 16px;
  }
  .drawer-title {
    font-family: var(--font-heading);
    font-size: 22px;
    margin: 0;
  }
  .close-btn {
    background: transparent;
    border: none;
    color: var(--text-primary);
    font-size: 18px;
    cursor: pointer;
  }
  .selector-block {
    margin-bottom: 14px;
  }
  .selector-label {
    display: block;
    font-size: 12px;
    font-weight: 600;
    margin-bottom: 6px;
  }
  .chip-row {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .option-chip {
    padding: 6px 14px;
    border-radius: var(--radius);
    border: 1px solid rgba(128, 128, 128, 0.35);
    background: transparent;
    color: var(--text-primary);
    cursor: pointer;
    font-size: 12px;
  }
  .option-chip.selected {
    background: var(--accent);
    color: #fff;
    border-color: var(--accent);
  }
  .care-box {
    font-size: 12px;
    opacity: 0.85;
    padding: 10px;
    border-radius: var(--radius);
    border: 1px dashed rgba(128, 128, 128, 0.3);
    margin-bottom: 14px;
  }
  .customer-fields {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 16px;
  }
  .customer-fields input {
    padding: 9px 12px;
    border-radius: var(--radius);
    border: 1px solid rgba(128, 128, 128, 0.35);
    background: transparent;
    color: var(--text-primary);
    font-size: 12px;
  }
  .cta-stack {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .wa-order-cta,
  .card-order-cta {
    display: block;
    text-align: center;
    text-decoration: none;
    padding: 12px 16px;
    border-radius: var(--radius);
    font-weight: 600;
    font-size: 13px;
  }
  .wa-order-cta {
    background: #16a34a;
    color: #ffffff;
  }
  .card-order-cta {
    background: var(--accent);
    color: #ffffff;
  }
</style>
