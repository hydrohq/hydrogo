<script lang="ts">
  import { fetchVariantsForProducts } from '$lib/commerce/variantLoader';
  import { onDestroy, onMount } from 'svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { auth } from '$lib/stores/auth.svelte';
  import { t } from '$lib/i18n/index.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import PageSpinner from '$lib/components/ui/PageSpinner.svelte';
  import { buildMatrix, type MatrixVariant } from '$lib/commerce/variantMatrix';

  interface Product {
    id: string;
    name: string;
    slug?: string;
    status?: string;
    [key: string]: unknown;
  }

  interface Variant extends MatrixVariant {
    id: string;
    product: string | { id: string };
    sku?: string | null;
    size?: string | null;
    color?: string | null;
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
  let stock = $state<Record<string, { available: number; reserved: number }>>({});

  let initialLoading = $state(true);
  let pageChanging = $state(false);
  let error = $state('');

  let page = $state(0);
  let hasMore = $state(false);
  let total = $state<number | undefined>(undefined);

  let searchInput = $state('');
  let search = $state('');

  let requestIdCounter = 0;
  let debounceTimer: ReturnType<typeof setTimeout> | undefined;

  let cellStatus = $state<Record<string, 'idle' | 'saving' | 'saved' | 'error'>>({});
  let cellError = $state<Record<string, string>>({});

  const canWrite = $derived(auth.hasPermission('admin.commerce.write'));

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

  function getProductTotalAvailable(productId: string): number {
    const productVars = variantsByProduct[productId] ?? [];
    return productVars.reduce((acc, v) => acc + (stock[v.id]?.available ?? 0), 0);
  }

  function stockCellClass(available: number): string {
    if (available === 0) {
      return 'bg-red-50 text-red-900 border-red-300 dark:bg-red-950/40 dark:text-red-300 dark:border-red-800';
    }
    if (available >= 1 && available <= 3) {
      return 'bg-amber-50 text-amber-900 border-amber-300 dark:bg-amber-950/40 dark:text-amber-300 dark:border-amber-800';
    }
    return 'bg-background text-foreground border-input';
  }

  function buildProductsUrl(pageNum: number, searchVal: string): string {
    let url = `/api/v1/admin/content/products?limit=10&offset=${pageNum * 10}&sort=name`;
    const q = searchVal.trim();
    if (q) {
      url += `&search=${encodeURIComponent(q)}`;
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

  async function loadData(isInitial = false) {
    if (isInitial) {
      initialLoading = true;
    } else {
      pageChanging = true;
    }
    error = '';
    const reqId = ++requestIdCounter;

    try {
      const prodRes = await api.get<ProductsResponse>(buildProductsUrl(page, search));
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
      error = err instanceof ApiClientError ? `${err.message}${err.requestId ? ` (${err.requestId})` : ''}` : t('stock.error');
    } finally {
      if (reqId === requestIdCounter) {
        if (isInitial) {
          initialLoading = false;
        } else {
          pageChanging = false;
        }
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
        loadData(false);
      }
    }, 300);
  }

  function goToPage(newPage: number) {
    if (newPage < 0 || newPage === page || pageChanging) return;
    page = newPage;
    loadData(false);
  }

  async function handleSaveStock(variantId: string, inputVal: string) {
    if (!canWrite) return;
    const current = stock[variantId]?.available ?? 0;
    const parsed = parseInt(inputVal, 10);
    if (Number.isNaN(parsed) || parsed < 0) {
      return;
    }
    if (parsed === current) {
      return;
    }

    const prev = current;
    // Optimistic update
    if (!stock[variantId]) {
      stock[variantId] = { available: parsed, reserved: 0 };
    } else {
      stock[variantId].available = parsed;
    }
    cellStatus[variantId] = 'saving';
    cellError[variantId] = '';

    try {
      await api.post('/api/v1/admin/commerce/stock', {
        variant_id: variantId,
        available: parsed,
      });
      cellStatus[variantId] = 'saved';
      setTimeout(() => {
        if (cellStatus[variantId] === 'saved') {
          cellStatus[variantId] = 'idle';
        }
      }, 2000);
    } catch (err) {
      // Revert on error
      if (stock[variantId]) {
        stock[variantId].available = prev;
      }
      cellStatus[variantId] = 'error';
      cellError[variantId] = err instanceof ApiClientError ? err.message : t('stock.saveError');
    }
  }

  onMount(() => {
    void loadData(true);
  });

  onDestroy(() => {
    if (debounceTimer) clearTimeout(debounceTimer);
  });
</script>

<section class="w-full space-y-6">
  <header>
    <p class="text-sm text-muted-foreground">{t('nav.store')}</p>
    <h1 class="text-2xl font-semibold tracking-tight">{t('stock.title')}</h1>
  </header>

  <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
    <div class="relative w-full sm:max-w-xs">
      <svg class="absolute left-2.5 top-2.5 size-4 text-muted-foreground pointer-events-none" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="11" cy="11" r="8"></circle>
        <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
      </svg>
      <input
        type="search"
        placeholder={t('stock.searchPlaceholder')}
        aria-label={t('stock.searchPlaceholder')}
        value={searchInput}
        oninput={handleSearchInput}
        class="h-9 w-full rounded-md border border-input bg-background pl-8 pr-3 text-xs text-foreground placeholder:text-muted-foreground shadow-xs focus:outline-none focus:ring-1 focus:ring-ring"
      />
    </div>
  </div>

  {#if initialLoading}
    <PageSpinner />
  {:else if error}
    <div class="rounded-lg border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive" role="alert">
      <p>{error}</p>
      <Button variant="outline" size="sm" class="mt-3" onclick={() => loadData(true)}>
        {t('error.retry')}
      </Button>
    </div>
  {:else if products.length === 0}
    <div class="rounded-lg border border-border p-12 text-center text-sm text-muted-foreground">
      {t('stock.empty')}
    </div>
  {:else}
    <div class="space-y-6 transition-opacity duration-150 {pageChanging ? 'opacity-50 pointer-events-none' : ''}">
      {#each products as product (product.id)}
        {@const vars = variantsByProduct[product.id] ?? []}
        {@const matrix = buildMatrix(vars)}
        {@const totalAvailable = getProductTotalAvailable(product.id)}

        <div class="rounded-xl border border-border bg-card p-5 text-card-foreground shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-2 border-b border-border/60 pb-3">
            <div>
              <h2 class="text-base font-semibold tracking-tight text-foreground">{product.name}</h2>
              {#if product.slug}
                <p class="font-mono text-xs text-muted-foreground">{product.slug}</p>
              {/if}
            </div>
            <div class="flex items-center gap-2">
              <span class="rounded-md bg-secondary px-2.5 py-1 text-xs font-medium text-secondary-foreground">
                {t('stock.totalAvailable', { count: totalAvailable })}
              </span>
            </div>
          </div>

          {#if vars.length === 0}
            <div class="py-6 text-center text-sm text-muted-foreground">
              {t('stock.noVariants')}
            </div>
          {:else}
            <div class="mt-4 overflow-x-auto">
              <table class="w-full border-collapse text-left text-sm">
                <thead>
                  <tr class="border-b border-border text-xs text-muted-foreground">
                    <th class="whitespace-nowrap px-3 py-2 font-medium">{t('stock.colorAndSize')}</th>
                    {#each matrix.sizes as size}
                      <th class="whitespace-nowrap px-3 py-2 text-center font-medium">{size || '-'}</th>
                    {/each}
                  </tr>
                </thead>
                <tbody>
                  {#each matrix.colors as color}
                    <tr class="border-b border-border/50 last:border-0 hover:bg-muted/30">
                      <td class="whitespace-nowrap px-3 py-2.5 font-medium text-xs text-foreground">
                        {color || '-'}
                      </td>
                      {#each matrix.sizes as size}
                        {@const variant = matrix.cells[`${color}:${size}`]}
                        <td class="px-3 py-2.5 text-center">
                          {#if variant}
                            {@const variantStock = stock[variant.id] ?? { available: 0, reserved: 0 }}
                            <div class="flex flex-col items-center gap-0.5">
                              <div class="relative flex items-center justify-center">
                                <input
                                  type="number"
                                  min="0"
                                  aria-label={`${product.name} ${color || ''} ${size || ''}`}
                                  value={variantStock.available}
                                  disabled={!canWrite}
                                  class="h-8 w-16 rounded border px-1.5 py-1 text-center font-mono text-xs transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 {stockCellClass(variantStock.available)}"
                                  onkeydown={(e) => {
                                    if (e.key === 'Enter') {
                                      handleSaveStock(variant.id, (e.currentTarget as HTMLInputElement).value);
                                      (e.currentTarget as HTMLInputElement).blur();
                                    }
                                  }}
                                  onblur={(e) => {
                                    handleSaveStock(variant.id, (e.currentTarget as HTMLInputElement).value);
                                  }}
                                  onchange={(e) => {
                                    handleSaveStock(variant.id, (e.currentTarget as HTMLInputElement).value);
                                  }}
                                />
                                {#if cellStatus[variant.id] === 'saved'}
                                  <span class="absolute -top-1.5 -right-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-emerald-600 text-[10px] text-white shadow-xs" title={t('stock.saved')}>
                                    <svg class="h-2.5 w-2.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                                      <polyline points="20 6 9 17 4 12" />
                                    </svg>
                                  </span>
                                {/if}
                              </div>
                              {#if variantStock.reserved > 0}
                                <span class="text-[10px] text-muted-foreground whitespace-nowrap leading-tight">
                                  {variantStock.reserved} rezerve
                                </span>
                              {/if}
                              {#if cellStatus[variant.id] === 'error'}
                                <span class="text-[10px] text-destructive leading-tight" role="alert">
                                  {cellError[variant.id]}
                                </span>
                              {/if}
                            </div>
                          {:else}
                            <span class="text-muted-foreground font-mono">-</span>
                          {/if}
                        </td>
                      {/each}
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </div>
      {/each}
    </div>

    <!-- Pagination bar -->
    <div class="flex items-center justify-between rounded-xl border border-border bg-card px-4 py-3 text-xs text-muted-foreground shadow-xs">
      <div>
        {#if typeof total === 'number' && total > 0}
          <span>{t('stock.pagination.rangeOfTotal', { start: page * 10 + 1, end: page * 10 + products.length, total })}</span>
        {:else}
          <span>{t('stock.pagination.page', { page: page + 1 })}</span>
        {/if}
      </div>
      <div class="flex items-center gap-2">
        {#if pageChanging}
          <div class="inline-flex items-center gap-1.5 text-xs text-muted-foreground" role="status">
            <div class="size-3.5 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
            <span class="sr-only">{t('stock.loading')}</span>
          </div>
        {/if}
        <Button
          variant="outline"
          size="sm"
          disabled={page === 0 || pageChanging}
          onclick={() => goToPage(page - 1)}
        >
          {t('stock.pagination.previous')}
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={!hasMore || pageChanging}
          onclick={() => goToPage(page + 1)}
        >
          {t('stock.pagination.next')}
        </Button>
      </div>
    </div>
  {/if}
</section>
