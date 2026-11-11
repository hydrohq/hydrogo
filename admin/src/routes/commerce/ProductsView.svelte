<script lang="ts">
  import { onMount } from 'svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { auth } from '$lib/stores/auth.svelte';
  import { router } from '$lib/router.svelte';
  import { t } from '$lib/i18n/index.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';

  interface Category {
    id: string;
    name: string;
    slug?: string;
    [key: string]: unknown;
  }

  interface Product {
    id: string;
    name: string;
    slug?: string;
    category?: string | { id: string; name?: string } | null;
    cover_image?: string | null;
    base_price?: number | string | null;
    currency?: string | null;
    status?: 'published' | 'draft' | string;
    updated_at?: string | null;
    expanded?: {
      category?: { id: string; name: string; [key: string]: unknown };
      [key: string]: unknown;
    };
    [key: string]: unknown;
  }

  interface Variant {
    id: string;
    product: string | { id: string };
    sku?: string | null;
    [key: string]: unknown;
  }

  interface StockItem {
    variant_id: string;
    available: number;
    reserved: number;
  }

  let products = $state<Product[]>([]);
  let variants = $state<Variant[]>([]);
  let categories = $state<Category[]>([]);
  let stock = $state<Record<string, { available: number; reserved: number }>>({});
  let loading = $state(true);
  let error = $state('');

  let search = $state('');
  let categoryFilter = $state('all');
  let statusFilter = $state('all');

  const canCreate = $derived(
    auth.hasPermission('admin.commerce.write') || auth.hasPermission('admin.content.write')
  );

  function resolveMediaUrl(cover: unknown): string {
    if (!cover || typeof cover !== 'string') return '';
    const s = cover.trim();
    if (!s) return '';
    if (s.startsWith('http://') || s.startsWith('https://') || s.startsWith('/')) {
      return s;
    }
    return `/media/${s}`;
  }

  const categoryMap = $derived.by(() => {
    const map: Record<string, string> = {};
    for (const c of categories) {
      map[c.id] = c.name;
    }
    return map;
  });

  function getCategoryName(p: Product): string {
    if (p.expanded?.category?.name) return String(p.expanded.category.name);
    if (typeof p.category === 'object' && p.category?.name) return String(p.category.name);
    const catId = typeof p.category === 'object' && p.category?.id ? p.category.id : String(p.category ?? '');
    if (catId && categoryMap[catId]) return categoryMap[catId];
    return '—';
  }

  function formatPrice(amount: number | string | null | undefined, currency: string | null | undefined): string {
    if (amount == null || amount === '') return '—';
    const num = typeof amount === 'number' ? amount : Number(amount);
    if (isNaN(num)) return '—';
    const curr = (currency || 'TRY').toUpperCase();
    try {
      return new Intl.NumberFormat('tr-TR', {
        style: 'currency',
        currency: curr,
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      }).format(num);
    } catch {
      return `${num.toFixed(2)} ${curr}`;
    }
  }

  function getVariantProductId(v: Variant): string {
    if (!v.product) return '';
    if (typeof v.product === 'object' && 'id' in v.product) {
      return String((v.product as { id: string }).id);
    }
    return String(v.product);
  }

  const variantsByProduct = $derived.by(() => {
    const map: Record<string, Variant[]> = {};
    for (const v of variants) {
      const pid = getVariantProductId(v);
      if (!map[pid]) {
        map[pid] = [];
      }
      map[pid].push(v);
    }
    return map;
  });

  function getProductStockInfo(productId: string) {
    const vars = variantsByProduct[productId] ?? [];
    const count = vars.length;
    if (count === 0) {
      return { count: 0, total: 0, isZero: true, isLow: false };
    }
    let total = 0;
    let anyLow = false;
    for (const v of vars) {
      const avail = stock[v.id]?.available ?? 0;
      total += avail;
      if (avail <= 3) {
        anyLow = true;
      }
    }
    const isZero = total === 0;
    return {
      count,
      total,
      isZero,
      isLow: !isZero && anyLow,
    };
  }

  const filteredProducts = $derived.by(() => {
    const q = search.trim().toLowerCase();
    return products.filter((p) => {
      if (q && !(p.name ?? '').toLowerCase().includes(q)) {
        return false;
      }
      if (categoryFilter !== 'all') {
        const catId = typeof p.category === 'object' && p.category?.id ? p.category.id : String(p.category ?? '');
        if (catId !== categoryFilter) return false;
      }
      if (statusFilter !== 'all') {
        const st = (p.status ?? 'published').toLowerCase();
        if (st !== statusFilter) return false;
      }
      return true;
    });
  });

  async function loadData() {
    loading = true;
    error = '';
    try {
      // 1. Fetch categories
      try {
        const catRes = await api.get<{ data: Category[] }>('/api/v1/admin/content/categories?limit=100');
        categories = catRes.data ?? [];
      } catch {
        categories = [];
      }

      // 2. Fetch all products (limit 100 per page, with &expand=category)
      const prods: Product[] = [];
      let pOffset = 0;
      while (true) {
        const res = await api.get<{ data: Product[] }>(
          `/api/v1/admin/content/products?limit=100&offset=${pOffset}&expand=category`
        );
        const batch = res.data ?? [];
        prods.push(...batch);
        if (batch.length < 100) break;
        pOffset += 100;
      }
      products = prods;

      // 3. Fetch all variants (limit 100 per page)
      const vars: Variant[] = [];
      let vOffset = 0;
      while (true) {
        const res = await api.get<{ data: Variant[] }>(
          `/api/v1/admin/content/variants?limit=100&offset=${vOffset}`
        );
        const batch = res.data ?? [];
        vars.push(...batch);
        if (batch.length < 100) break;
        vOffset += 100;
      }
      variants = vars;

      // 4. Fetch stock for variants (chunk MAX 500 ids per call)
      const variantIds = vars.map((v) => v.id).filter(Boolean);
      const stockMap: Record<string, { available: number; reserved: number }> = {};
      const chunkSize = 500;
      if (variantIds.length > 0) {
        for (let i = 0; i < variantIds.length; i += chunkSize) {
          const chunk = variantIds.slice(i, i + chunkSize);
          const res = await api.get<{ data: StockItem[] }>(
            `/api/v1/admin/commerce/stock?variant_ids=${chunk.map(encodeURIComponent).join(',')}`
          );
          for (const item of res.data ?? []) {
            stockMap[item.variant_id] = {
              available: Number(item.available) || 0,
              reserved: Number(item.reserved) || 0,
            };
          }
        }
      }
      stock = stockMap;
    } catch (err) {
      error = err instanceof ApiClientError ? `${err.message}${err.requestId ? ` (${err.requestId})` : ''}` : t('products.error');
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    loadData();
  });
</script>

<div class="flex flex-1 flex-col gap-4 w-full">
  <!-- Page Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div>
      <h1 class="text-xl font-semibold tracking-tight text-foreground">{t('products.title')}</h1>
    </div>
    {#if canCreate}
      <Button
        variant="primary"
        size="sm"
        data-testid="new-product-button"
        onclick={() => router.navigate('/commerce/products/new')}
      >
        <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="12" y1="5" x2="12" y2="19"></line>
          <line x1="5" y1="12" x2="19" y2="12"></line>
        </svg>
        <span>{t('products.new')}</span>
      </Button>
    {/if}
  </div>

  <!-- Filters & Search Toolbar -->
  <div class="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
    <div class="relative flex-1 sm:max-w-xs">
      <svg class="absolute left-2.5 top-2.5 size-4 text-muted-foreground pointer-events-none" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="11" cy="11" r="8"></circle>
        <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
      </svg>
      <input
        type="search"
        placeholder={t('products.search')}
        aria-label={t('products.search')}
        bind:value={search}
        class="h-9 w-full rounded-md border border-input bg-background pl-8 pr-3 text-xs text-foreground placeholder:text-muted-foreground shadow-xs focus:outline-none focus:ring-1 focus:ring-ring"
      />
    </div>

    <div class="flex items-center gap-2">
      <!-- Category Filter -->
      <select
        aria-label={t('products.col.category')}
        bind:value={categoryFilter}
        class="h-9 rounded-md border border-input bg-background px-3 py-1 text-xs text-foreground shadow-xs focus:outline-none focus:ring-1 focus:ring-ring"
      >
        <option value="all">{t('products.allCategories')}</option>
        {#each categories as cat (cat.id)}
          <option value={cat.id}>{cat.name}</option>
        {/each}
      </select>

      <!-- Status Filter -->
      <select
        aria-label={t('products.col.status')}
        bind:value={statusFilter}
        class="h-9 rounded-md border border-input bg-background px-3 py-1 text-xs text-foreground shadow-xs focus:outline-none focus:ring-1 focus:ring-ring"
      >
        <option value="all">{t('products.allStatuses')}</option>
        <option value="published">{t('products.status.published')}</option>
        <option value="draft">{t('products.status.draft')}</option>
      </select>
    </div>
  </div>

  <!-- Table Container -->
  <div class="overflow-hidden rounded-xl border border-border bg-card shadow-xs">
    {#if error}
      <div class="p-6 text-center text-xs text-destructive bg-destructive/10 border-b border-border">
        {error}
      </div>
    {/if}

    {#if loading}
      <div class="p-12 text-center text-xs text-muted-foreground">
        {t('products.loading')}
      </div>
    {:else if filteredProducts.length === 0}
      <div class="p-12 text-center text-xs text-muted-foreground">
        {t('products.empty')}
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm text-foreground caption-bottom">
          <thead class="sticky top-0 z-10 bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
            <tr>
              <th class="h-10 px-4 font-medium">{t('products.col.product')}</th>
              <th class="h-10 px-4 font-medium">{t('products.col.category')}</th>
              <th class="h-10 px-4 font-medium">{t('products.col.price')}</th>
              <th class="h-10 px-4 font-medium">{t('products.col.variants')}</th>
              <th class="h-10 px-4 font-medium">{t('products.col.stock')}</th>
              <th class="h-10 px-4 font-medium">{t('products.col.status')}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            {#each filteredProducts as product (product.id)}
              {@const stockInfo = getProductStockInfo(product.id)}
              <tr
                class="cursor-pointer hover:bg-muted/50 transition-colors"
                tabindex="0"
                role="button"
                onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') router.navigate(`/commerce/products/${product.id}`); }}
                onclick={() => router.navigate(`/commerce/products/${product.id}`)}
              >
                <!-- Product Cover & Name -->
                <td class="px-4 py-3">
                  <div class="flex items-center gap-3">
                    <div class="size-10 rounded-md border border-border bg-muted overflow-hidden flex items-center justify-center shrink-0">
                      {#if resolveMediaUrl(product.cover_image)}
                        <img
                          src={resolveMediaUrl(product.cover_image)}
                          alt={product.name || 'product'}
                          class="size-full object-cover"
                          onerror={(e) => { (e.target as HTMLElement).style.display = 'none'; }}
                        />
                      {:else}
                        <svg class="size-5 text-muted-foreground/60" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                          <rect width="18" height="18" x="3" y="3" rx="2" ry="2"></rect>
                          <circle cx="9" cy="9" r="2"></circle>
                          <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"></path>
                        </svg>
                      {/if}
                    </div>
                    <div>
                      <div class="font-medium text-foreground">{product.name || '—'}</div>
                      {#if product.slug}
                        <div class="text-[11px] text-muted-foreground font-mono">{product.slug}</div>
                      {/if}
                    </div>
                  </div>
                </td>

                <!-- Category -->
                <td class="px-4 py-3 text-xs text-muted-foreground">
                  {getCategoryName(product)}
                </td>

                <!-- Price -->
                <td class="px-4 py-3 text-xs font-medium text-foreground whitespace-nowrap">
                  {formatPrice(product.base_price, product.currency)}
                </td>

                <!-- Variant count -->
                <td class="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                  {t('products.variantsCount', { count: stockInfo.count })}
                </td>

                <!-- Total Stock -->
                <td class="px-4 py-3 whitespace-nowrap">
                  {#if stockInfo.isZero}
                    <Badge variant="destructive" size="sm">Stok yok</Badge>
                  {:else if stockInfo.isLow}
                    <Badge variant="warning" size="sm">Düşük</Badge>
                  {:else}
                    <span class="text-xs font-mono text-foreground">{stockInfo.total}</span>
                  {/if}
                </td>

                <!-- Status -->
                <td class="px-4 py-3 whitespace-nowrap">
                  {#if product.status === 'draft'}
                    <Badge variant="outline" size="sm">Taslak</Badge>
                  {:else}
                    <Badge variant="success" size="sm">Yayında</Badge>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>
