<script lang="ts">
  import { onMount } from 'svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { auth } from '$lib/stores/auth.svelte';
  import { t } from '$lib/i18n/index.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Button from '$lib/components/ui/Button.svelte';
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

  let products = $state<Product[]>([]);
  let variants = $state<Variant[]>([]);
  let stock = $state<Record<string, { available: number; reserved: number }>>({});
  let loading = $state(true);
  let error = $state('');

  let search = $state('');
  let lowStockOnly = $state(false);

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

  const filteredProducts = $derived.by(() => {
    const q = search.trim().toLowerCase();
    return products.filter((product) => {
      if (q && !(product.name ?? '').toLowerCase().includes(q)) {
        return false;
      }
      if (lowStockOnly) {
        const productVars = variantsByProduct[product.id] ?? [];
        if (productVars.length === 0) return false;
        const hasLowStock = productVars.some((v) => {
          const item = stock[v.id];
          const avail = item ? item.available : 0;
          return avail <= 3;
        });
        if (!hasLowStock) return false;
      }
      return true;
    });
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

  async function loadData() {
    loading = true;
    error = '';
    try {
      // 1. Fetch all products (limit 100 per page)
      const prods: Product[] = [];
      let pOffset = 0;
      while (true) {
        const res = await api.get<{ data: Product[] }>(`/api/v1/admin/content/products?limit=100&offset=${pOffset}`);
        const batch = res.data ?? [];
        prods.push(...batch);
        if (batch.length < 100) break;
        pOffset += 100;
      }
      products = prods;

      // 2. Fetch all variants (limit 100 per page)
      const vars: Variant[] = [];
      let vOffset = 0;
      while (true) {
        const res = await api.get<{ data: Variant[] }>(`/api/v1/admin/content/variants?limit=100&offset=${vOffset}`);
        const batch = res.data ?? [];
        vars.push(...batch);
        if (batch.length < 100) break;
        vOffset += 100;
      }
      variants = vars;

      // 3. Fetch stock for variants (chunk MAX 500 ids per call)
      const variantIds = vars.map((v) => v.id).filter(Boolean);
      const stockMap: Record<string, { available: number; reserved: number }> = {};
      const chunkSize = 500;
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
      stock = stockMap;
    } catch (err) {
      error = err instanceof ApiClientError ? `${err.message}${err.requestId ? ` (${err.requestId})` : ''}` : t('stock.error');
    } finally {
      loading = false;
    }
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
    void loadData();
  });
</script>

<section class="w-full space-y-6">
  <header>
    <p class="text-sm text-muted-foreground">{t('nav.store')}</p>
    <h1 class="text-2xl font-semibold tracking-tight">{t('stock.title')}</h1>
  </header>

  <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
    <div class="w-full sm:max-w-xs">
      <Input
        placeholder={t('stock.searchPlaceholder')}
        bind:value={search}
      />
    </div>
    <label class="flex items-center gap-2 text-sm text-foreground cursor-pointer select-none">
      <input
        type="checkbox"
        bind:checked={lowStockOnly}
        class="h-4 w-4 rounded border-border text-primary focus:ring-ring"
      />
      <span>{t('stock.lowStockOnly')}</span>
    </label>
  </div>

  {#if loading}
    <div class="flex items-center justify-center p-12 text-sm text-muted-foreground">
      {t('stock.loading')}
    </div>
  {:else if error}
    <div class="rounded-lg border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive" role="alert">
      <p>{error}</p>
      <Button variant="outline" size="sm" class="mt-3" onclick={loadData}>
        {t('error.retry')}
      </Button>
    </div>
  {:else if filteredProducts.length === 0}
    <div class="rounded-lg border border-border p-12 text-center text-sm text-muted-foreground">
      {t('stock.empty')}
    </div>
  {:else}
    <div class="space-y-6">
      {#each filteredProducts as product (product.id)}
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
  {/if}
</section>
