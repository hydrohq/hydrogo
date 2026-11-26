<script lang="ts">
  import { fetchVariantsForProducts } from '$lib/commerce/variantLoader';
  import { onDestroy, onMount } from 'svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { auth } from '$lib/stores/auth.svelte';
  import { router } from '$lib/router.svelte';
  import { t } from '$lib/i18n/index.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import PageSpinner from '$lib/components/ui/PageSpinner.svelte';

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

  interface ProductsResponse {
    data: Product[];
    metadata?: {
      limit?: number;
      offset?: number;
      total?: number;
      has_more?: boolean;
      next_cursor?: string;
    };
  }

  let products = $state<Product[]>([]);
  let variants = $state<Variant[]>([]);
  let categories = $state<Category[]>([]);
  let stock = $state<Record<string, { available: number; reserved: number }>>({});

  let initialLoading = $state(true);
  let pageChanging = $state(false);
  let error = $state('');

  let page = $state(0);
  let hasMore = $state(false);
  let total = $state<number | undefined>(undefined);

  let searchInput = $state('');
  let search = $state('');
  let categoryFilter = $state('all');
  let statusFilter = $state('all');

  let categoriesLoaded = false;
  let requestIdCounter = 0;
  let debounceTimer: ReturnType<typeof setTimeout> | undefined;

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
    let totalStock = 0;
    let anyLow = false;
    for (const v of vars) {
      const avail = stock[v.id]?.available ?? 0;
      totalStock += avail;
      if (avail <= 3) {
        anyLow = true;
      }
    }
    const isZero = totalStock === 0;
    return {
      count,
      total: totalStock,
      isZero,
      isLow: !isZero && anyLow,
    };
  }

  function buildProductsUrl(pageNum: number, searchVal: string, catVal: string, statusVal: string): string {
    let url = `/api/v1/admin/content/products?limit=20&offset=${pageNum * 20}&sort=-updated_at`;
    const q = searchVal.trim();
    if (q) {
      url += `&search=${encodeURIComponent(q)}`;
    }
    if (catVal && catVal !== 'all') {
      url += `&filter[category]=${encodeURIComponent(catVal)}`;
    }
    if (statusVal && statusVal !== 'all') {
      url += `&status=${encodeURIComponent(statusVal)}`;
    }
    return url;
  }

  async function fetchVariantsAndStock(prods: Product[]): Promise<{
    vars: Variant[];
    stockMap: Record<string, { available: number; reserved: number }>;
  }> {
    const productIds = prods.map((p) => p.id).filter(Boolean);
    if (productIds.length === 0) {
      return { vars: [], stockMap: {} };
    }

    const vars = await fetchVariantsForProducts<Variant>(productIds);

    const variantIds = vars.map((v) => v.id).filter(Boolean);
    const stockMap: Record<string, { available: number; reserved: number }> = {};
    const chunkSize = 500;
    if (variantIds.length > 0) {
      const chunkPromises = [];
      for (let i = 0; i < variantIds.length; i += chunkSize) {
        const chunk = variantIds.slice(i, i + chunkSize);
        chunkPromises.push(
          api.get<{ data: StockItem[] }>(
            `/api/v1/admin/commerce/stock?variant_ids=${chunk.map(encodeURIComponent).join(',')}`
          )
        );
      }
      const stockResults = await Promise.all(chunkPromises);
      for (const res of stockResults) {
        for (const item of res.data ?? []) {
          stockMap[item.variant_id] = {
            available: Number(item.available) || 0,
            reserved: Number(item.reserved) || 0,
          };
        }
      }
    }

    return { vars, stockMap };
  }

  async function loadInitial() {
    initialLoading = true;
    error = '';
    const reqId = ++requestIdCounter;

    try {
      const [catRes, prodRes] = await Promise.all([
        api.get<{ data: Category[] }>('/api/v1/admin/content/categories?limit=100').catch(() => ({ data: [] })),
        api.get<ProductsResponse>(buildProductsUrl(0, search, categoryFilter, statusFilter)),
      ]);

      if (reqId !== requestIdCounter) return;

      categories = catRes.data ?? [];
      categoriesLoaded = true;

      const prods = prodRes.data ?? [];
      const metaHasMore = Boolean(prodRes.metadata?.has_more);
      const metaTotal = typeof prodRes.metadata?.total === 'number' && prodRes.metadata.total > 0
        ? prodRes.metadata.total
        : undefined;

      const { vars, stockMap } = await fetchVariantsAndStock(prods);

      if (reqId !== requestIdCounter) return;

      products = prods;
      variants = vars;
      stock = stockMap;
      hasMore = metaHasMore;
      total = metaTotal;
    } catch (err) {
      if (reqId !== requestIdCounter) return;
      error = err instanceof ApiClientError ? `${err.message}${err.requestId ? ` (${err.requestId})` : ''}` : t('products.error');
    } finally {
      if (reqId === requestIdCounter) {
        initialLoading = false;
      }
    }
  }

  async function loadProducts() {
    if (!categoriesLoaded) {
      return loadInitial();
    }

    pageChanging = true;
    error = '';
    const reqId = ++requestIdCounter;

    try {
      const prodRes = await api.get<ProductsResponse>(
        buildProductsUrl(page, search, categoryFilter, statusFilter)
      );

      if (reqId !== requestIdCounter) return;

      const prods = prodRes.data ?? [];
      const metaHasMore = Boolean(prodRes.metadata?.has_more);
      const metaTotal = typeof prodRes.metadata?.total === 'number' && prodRes.metadata.total > 0
        ? prodRes.metadata.total
        : undefined;

      const { vars, stockMap } = await fetchVariantsAndStock(prods);

      if (reqId !== requestIdCounter) return;

      products = prods;
      variants = vars;
      stock = stockMap;
      hasMore = metaHasMore;
      total = metaTotal;
    } catch (err) {
      if (reqId !== requestIdCounter) return;
      error = err instanceof ApiClientError ? `${err.message}${err.requestId ? ` (${err.requestId})` : ''}` : t('products.error');
    } finally {
      if (reqId === requestIdCounter) {
        pageChanging = false;
      }
    }
  }

  function handleSearchInput(e: Event) {
    const target = e.target as HTMLInputElement;
    searchInput = target.value;
    if (debounceTimer) clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => {
      const val = searchInput.trim();
      if (search !== val) {
        search = val;
        page = 0;
        loadProducts();
      }
    }, 300);
  }

  function handleCategoryChange(e: Event) {
    const val = (e.target as HTMLSelectElement).value;
    if (categoryFilter !== val) {
      categoryFilter = val;
      page = 0;
      loadProducts();
    }
  }

  function handleStatusChange(e: Event) {
    const val = (e.target as HTMLSelectElement).value;
    if (statusFilter !== val) {
      statusFilter = val;
      page = 0;
      loadProducts();
    }
  }

  function goToPage(newPage: number) {
    if (newPage < 0 || newPage === page || pageChanging) return;
    page = newPage;
    loadProducts();
  }

  onMount(() => {
    loadInitial();
  });

  onDestroy(() => {
    if (debounceTimer) clearTimeout(debounceTimer);
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
        value={searchInput}
        oninput={handleSearchInput}
        class="h-9 w-full rounded-md border border-input bg-background pl-8 pr-3 text-xs text-foreground placeholder:text-muted-foreground shadow-xs focus:outline-none focus:ring-1 focus:ring-ring"
      />
    </div>

    <div class="flex items-center gap-2">
      <!-- Category Filter -->
      <select
        aria-label={t('products.col.category')}
        value={categoryFilter}
        onchange={handleCategoryChange}
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
        value={statusFilter}
        onchange={handleStatusChange}
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

    {#if initialLoading}
      <PageSpinner />
    {:else if products.length === 0}
      <div class="p-12 text-center text-xs text-muted-foreground">
        {t('products.empty')}
      </div>
    {:else}
      <div class="overflow-x-auto relative transition-opacity duration-150 {pageChanging ? 'opacity-50 pointer-events-none' : ''}">
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
            {#each products as product (product.id)}
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
                    <Badge variant="destructive" size="sm">{t('commerce.products.outOfStock')}</Badge>
                  {:else if stockInfo.isLow}
                    <Badge variant="warning" size="sm">{t('commerce.products.lowStock')}</Badge>
                  {:else}
                    <span class="text-xs font-mono text-foreground">{stockInfo.total}</span>
                  {/if}
                </td>

                <!-- Status -->
                <td class="px-4 py-3 whitespace-nowrap">
                  {#if product.status === 'draft'}
                    <Badge variant="outline" size="sm">{t('common.draft')}</Badge>
                  {:else}
                    <Badge variant="success" size="sm">{t('common.published')}</Badge>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <!-- Pagination bar -->
      <div class="flex items-center justify-between px-4 py-3 border-t border-border text-xs text-muted-foreground">
        <div>
          {#if typeof total === 'number' && total > 0}
            <span>{t('products.pagination.rangeOfTotal', { start: page * 20 + 1, end: page * 20 + products.length, total })}</span>
          {:else}
            <span>{t('products.pagination.page', { page: page + 1 })}</span>
          {/if}
        </div>
        <div class="flex items-center gap-2">
          {#if pageChanging}
            <div class="inline-flex items-center gap-1.5 text-xs text-muted-foreground" role="status">
              <div class="size-3.5 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
              <span class="sr-only">{t('products.loading')}</span>
            </div>
          {/if}
          <Button
            variant="outline"
            size="sm"
            disabled={page === 0 || pageChanging}
            onclick={() => goToPage(page - 1)}
          >
            {t('products.pagination.previous')}
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={!hasMore || pageChanging}
            onclick={() => goToPage(page + 1)}
          >
            {t('products.pagination.next')}
          </Button>
        </div>
      </div>
    {/if}
  </div>
</div>
