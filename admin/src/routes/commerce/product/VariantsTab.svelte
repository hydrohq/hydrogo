<script lang="ts">
  import { api, ApiClientError } from '$lib/api/client';
  import { auth } from '$lib/stores/auth.svelte';
  import { t } from '$lib/i18n/index.svelte';
  import PageSpinner from '$lib/components/ui/PageSpinner.svelte';
  import { confirmAction } from '$lib/confirm';
  import Button from '$lib/components/ui/Button.svelte';
  import { sortSizes, type MatrixVariant } from '$lib/commerce/variantMatrix';
  import { generateVariantSku, getMissingCombos } from '$lib/commerce/variantSku';

  interface Product {
    id: string;
    slug?: string;
    name?: string;
    [key: string]: unknown;
  }

  interface Props {
    product: Product | null;
  }

  let { product }: Props = $props();

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

  const isNew = $derived(!product || product.id === 'new' || !product.id);
  const canWrite = $derived(auth.hasPermission('admin.commerce.write'));

  let variants = $state<Variant[]>([]);
  let stock = $state<Record<string, { available: number; reserved: number }>>({});
  let colors = $state<string[]>([]);
  let sizes = $state<string[]>([]);
  let chipsInitialized = $state(false);

  // Starts true for a saved product: the first render must not flash the empty state before data arrives.
  // svelte-ignore state_referenced_locally
  let loading = $state(!!product?.id && product.id !== 'new');
  let error = $state('');
  let refuseMessage = $state('');

  let colorInput = $state('');
  let sizeInput = $state('');

  let initialStock = $state(0);
  let creatingMissing = $state(false);
  let creationProgress = $state({ current: 0, total: 0 });
  let creationErrors = $state<string[]>([]);

  let cellStatus = $state<Record<string, 'idle' | 'saving' | 'saved' | 'error'>>({});
  let cellError = $state<Record<string, string>>({});

  const matrixCells = $derived.by(() => {
    const map: Record<string, Variant> = {};
    for (const v of variants) {
      const c = (v.color ?? '').trim();
      const s = (v.size ?? '').trim();
      map[`${c}:${s}`] = v;
    }
    return map;
  });

  const missingCombos = $derived(getMissingCombos(colors, sizes, variants));

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
    if (isNew || !product?.id) return;
    const prodId = product.id;
    loading = true;
    error = '';
    refuseMessage = '';
    try {
      // 1. Fetch all variants for this product (page until short page)
      const loadedVariants: Variant[] = [];
      let offset = 0;
      while (true) {
        const res = await api.get<{ data: Variant[] }>(
          `/api/v1/admin/content/variants?limit=100&offset=${offset}&filter[product]=${encodeURIComponent(prodId)}`
        );
        const batch = res.data ?? [];
        loadedVariants.push(...batch);
        if (batch.length < 100) break;
        offset += 100;
      }
      variants = loadedVariants;

      // 2. Fetch stock for variants
      const variantIds = loadedVariants.map((v) => v.id).filter(Boolean);
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

      // 3. Initialize or sync distinct chips
      const distinctColors = Array.from(new Set(loadedVariants.map((v) => (v.color ?? '').trim()).filter(Boolean)));
      const distinctSizes = sortSizes(Array.from(new Set(loadedVariants.map((v) => (v.size ?? '').trim()).filter(Boolean))));

      if (!chipsInitialized) {
        colors = distinctColors;
        sizes = distinctSizes;
        chipsInitialized = true;
      } else {
        const newColors = [...colors];
        for (const c of distinctColors) {
          if (!newColors.includes(c)) newColors.push(c);
        }
        colors = newColors;

        const newSizes = [...sizes];
        for (const s of distinctSizes) {
          if (!newSizes.includes(s)) newSizes.push(s);
        }
        sizes = sortSizes(newSizes);
      }
    } catch (err) {
      error = err instanceof ApiClientError ? `${err.message}${err.requestId ? ` (${err.requestId})` : ''}` : t('products.variants.errorLoad');
    } finally {
      loading = false;
    }
  }

  function handleAddColor(raw: string) {
    if (!raw.trim()) return;
    const parts = raw.split(',').map((s) => s.trim()).filter(Boolean);
    const next = [...colors];
    for (const p of parts) {
      if (!next.includes(p)) {
        next.push(p);
      }
    }
    colors = next;
    colorInput = '';
  }

  function handleAddSize(raw: string) {
    if (!raw.trim()) return;
    const parts = raw.split(',').map((s) => s.trim()).filter(Boolean);
    const next = [...sizes];
    for (const p of parts) {
      if (!next.includes(p)) {
        next.push(p);
      }
    }
    sizes = sortSizes(next);
    sizeInput = '';
  }

  async function handleRemoveColorChip(color: string) {
    if (!canWrite) return;
    refuseMessage = '';
    const matching = variants.filter((v) => (v.color ?? '').trim() === color.trim());
    if (matching.length === 0) {
      colors = colors.filter((c) => c !== color);
      return;
    }

    const totalReserved = matching.reduce((sum, v) => sum + (stock[v.id]?.reserved ?? 0), 0);
    if (totalReserved > 0) {
      refuseMessage = t('products.variants.reservedRefuse', { count: totalReserved });
      return;
    }

    const totalStock = matching.reduce((sum, v) => sum + (stock[v.id]?.available ?? 0), 0);
    const confirmed = await confirmAction({
      title: t('products.variants.confirmDeleteTitle'),
      description: t('products.variants.confirmDeleteDesc', {
        count: matching.length,
        stock: totalStock,
      }),
      confirmLabel: t('products.variants.confirmDeleteButton'),
      cancelLabel: t('products.variants.cancelButton'),
      variant: 'destructive',
    });

    if (!confirmed) return;

    try {
      for (const v of matching) {
        await api.delete(`/api/v1/admin/content/variants/${v.id}`);
      }
      colors = colors.filter((c) => c !== color);
      await loadData();
    } catch (err: unknown) {
      refuseMessage = err instanceof ApiClientError ? err.message : t('products.variants.errorLoad');
    }
  }

  async function handleRemoveSizeChip(size: string) {
    if (!canWrite) return;
    refuseMessage = '';
    const matching = variants.filter((v) => (v.size ?? '').trim() === size.trim());
    if (matching.length === 0) {
      sizes = sizes.filter((s) => s !== size);
      return;
    }

    const totalReserved = matching.reduce((sum, v) => sum + (stock[v.id]?.reserved ?? 0), 0);
    if (totalReserved > 0) {
      refuseMessage = t('products.variants.reservedRefuse', { count: totalReserved });
      return;
    }

    const totalStock = matching.reduce((sum, v) => sum + (stock[v.id]?.available ?? 0), 0);
    const confirmed = await confirmAction({
      title: t('products.variants.confirmDeleteTitle'),
      description: t('products.variants.confirmDeleteDesc', {
        count: matching.length,
        stock: totalStock,
      }),
      confirmLabel: t('products.variants.confirmDeleteButton'),
      cancelLabel: t('products.variants.cancelButton'),
      variant: 'destructive',
    });

    if (!confirmed) return;

    try {
      for (const v of matching) {
        await api.delete(`/api/v1/admin/content/variants/${v.id}`);
      }
      sizes = sizes.filter((s) => s !== size);
      await loadData();
    } catch (err: unknown) {
      refuseMessage = err instanceof ApiClientError ? err.message : t('products.variants.errorLoad');
    }
  }

  async function handleSaveStock(variantId: string, inputVal: string) {
    if (!canWrite) return;
    const current = stock[variantId]?.available ?? 0;
    const parsed = parseInt(inputVal, 10);
    if (Number.isNaN(parsed) || parsed < 0) return;
    if (parsed === current) return;

    const prev = current;
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
    } catch (err: unknown) {
      if (stock[variantId]) {
        stock[variantId].available = prev;
      }
      cellStatus[variantId] = 'error';
      cellError[variantId] = err instanceof ApiClientError ? err.message : t('stock.saveError');
    }
  }

  async function handleCreateMissing() {
    if (isNew || !product?.id || missingCombos.length === 0 || creatingMissing || !canWrite) return;
    const prodId = product.id;
    const prodSlug = product.slug;
    const prodName = product.name;
    creatingMissing = true;
    creationProgress = { current: 0, total: missingCombos.length };
    creationErrors = [];

    const stockVal = Math.max(0, parseInt(String(initialStock), 10) || 0);

    for (let i = 0; i < missingCombos.length; i++) {
      const combo = missingCombos[i];
      creationProgress = { current: i + 1, total: missingCombos.length };

      const baseSku = generateVariantSku(prodSlug || prodName || 'PRD', combo.color, combo.size);
      let variantId: string | null = null;

      try {
        const res = await api.post<{ data: { id: string } }>('/api/v1/admin/content/variants', {
          product: prodId,
          sku: baseSku,
          color: combo.color,
          size: combo.size,
        });
        variantId = res.data?.id;
      } catch (err: unknown) {
        const anyErr = err as { status?: number; statusCode?: number; message?: string };
        const is422 = (err instanceof ApiClientError && err.status === 422) || anyErr?.status === 422 || anyErr?.statusCode === 422;
        if (is422) {
          try {
            const retryRes = await api.post<{ data: { id: string } }>('/api/v1/admin/content/variants', {
              product: prodId,
              sku: `${baseSku}-2`,
              color: combo.color,
              size: combo.size,
            });
            variantId = retryRes.data?.id;
          } catch (retryErr: unknown) {
            const rMsg = (retryErr as Error)?.message || 'Error';
            creationErrors = [...creationErrors, `${combo.color} / ${combo.size}: ${rMsg}`];
            continue;
          }
        } else {
          const msg = (err as Error)?.message || 'Error';
          creationErrors = [...creationErrors, `${combo.color} / ${combo.size}: ${msg}`];
          continue;
        }
      }

      if (variantId) {
        try {
          await api.post('/api/v1/admin/commerce/stock', {
            variant_id: variantId,
            available: stockVal,
          });
        } catch (stockErr: unknown) {
          const sMsg = (stockErr as Error)?.message || 'Stock error';
          creationErrors = [...creationErrors, `${combo.color} / ${combo.size} (${t('products.col.stock')}): ${sMsg}`];
        }
      }
    }

    await loadData();
    creatingMissing = false;
  }

  $effect(() => {
    if (product?.id && product.id !== 'new') {
      void loadData();
    }
  });
</script>

<div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-6" data-testid="variants-tab">
  {#if isNew}
    <div class="flex flex-col items-center justify-center text-center p-8 space-y-3">
      <div class="size-10 rounded-full bg-muted flex items-center justify-center text-muted-foreground">
        <svg class="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect width="18" height="18" x="3" y="3" rx="2" ry="2"></rect>
          <line x1="3" y1="9" x2="21" y2="9"></line>
          <line x1="9" y1="21" x2="9" y2="9"></line>
        </svg>
      </div>
      <div class="space-y-1">
        <h3 class="text-sm font-medium text-foreground">{t('products.editor.saveFirstNotice')}</h3>
        <p class="text-xs text-muted-foreground max-w-sm">
          {t('products.variants.saveFirstHelp')}
        </p>
      </div>
    </div>
  {:else if loading && variants.length === 0}
    <PageSpinner compact />
  {:else if error}
    <div class="rounded-lg border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive" role="alert">
      <p>{error}</p>
      <Button variant="outline" size="sm" class="mt-3" onclick={loadData}>
        {t('error.retry')}
      </Button>
    </div>
  {:else}
    <!-- Refuse error banner -->
    {#if refuseMessage}
      <div class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive flex items-center justify-between" role="alert">
        <span>{refuseMessage}</span>
        <button
          type="button"
          aria-label={t('products.variants.removeChip')}
          class="text-destructive/80 hover:text-destructive cursor-pointer p-0.5"
          onclick={() => { refuseMessage = ''; }}
        >
          <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>
    {/if}

    <!-- 1. Chip Editors -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <!-- Colors Chip Editor -->
      <div class="space-y-1.5">
        <label for="color-input" class="text-xs font-medium text-foreground">
          {t('products.variants.colors')}
        </label>
        <div class="flex flex-wrap items-center gap-1.5 p-2 rounded-lg border border-input bg-background min-h-10">
          {#each colors as color}
            <span class="inline-flex items-center gap-1 rounded-md bg-secondary px-2.5 py-1 text-xs font-medium text-secondary-foreground">
              <span>{color}</span>
              {#if canWrite}
                <button
                  type="button"
                  aria-label={`${t('products.variants.removeChip')} ${color}`}
                  class="text-muted-foreground hover:text-foreground cursor-pointer rounded p-0.5"
                  onclick={() => handleRemoveColorChip(color)}
                >
                  <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <line x1="18" y1="6" x2="6" y2="18"></line>
                    <line x1="6" y1="6" x2="18" y2="18"></line>
                  </svg>
                </button>
              {/if}
            </span>
          {/each}
          {#if canWrite}
            <input
              id="color-input"
              type="text"
              placeholder={t('products.variants.addColorPlaceholder')}
              bind:value={colorInput}
              class="flex-1 min-w-28 bg-transparent text-xs text-foreground placeholder:text-muted-foreground focus:outline-none px-1 py-0.5"
              onkeydown={(e) => {
                if (e.key === 'Enter' || e.key === ',') {
                  e.preventDefault();
                  handleAddColor(colorInput);
                }
              }}
              onblur={() => {
                if (colorInput.trim()) {
                  handleAddColor(colorInput);
                }
              }}
            />
          {/if}
        </div>
      </div>

      <!-- Sizes Chip Editor -->
      <div class="space-y-1.5">
        <label for="size-input" class="text-xs font-medium text-foreground">
          {t('products.variants.sizes')}
        </label>
        <div class="flex flex-wrap items-center gap-1.5 p-2 rounded-lg border border-input bg-background min-h-10">
          {#each sizes as size}
            <span class="inline-flex items-center gap-1 rounded-md bg-secondary px-2.5 py-1 text-xs font-medium text-secondary-foreground">
              <span>{size}</span>
              {#if canWrite}
                <button
                  type="button"
                  aria-label={`${t('products.variants.removeChip')} ${size}`}
                  class="text-muted-foreground hover:text-foreground cursor-pointer rounded p-0.5"
                  onclick={() => handleRemoveSizeChip(size)}
                >
                  <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <line x1="18" y1="6" x2="6" y2="18"></line>
                    <line x1="6" y1="6" x2="18" y2="18"></line>
                  </svg>
                </button>
              {/if}
            </span>
          {/each}
          {#if canWrite}
            <input
              id="size-input"
              type="text"
              placeholder={t('products.variants.addSizePlaceholder')}
              bind:value={sizeInput}
              class="flex-1 min-w-28 bg-transparent text-xs text-foreground placeholder:text-muted-foreground focus:outline-none px-1 py-0.5"
              onkeydown={(e) => {
                if (e.key === 'Enter' || e.key === ',') {
                  e.preventDefault();
                  handleAddSize(sizeInput);
                }
              }}
              onblur={() => {
                if (sizeInput.trim()) {
                  handleAddSize(sizeInput);
                }
              }}
            />
          {/if}
        </div>
      </div>
    </div>

    <!-- 3. Create Missing Variants Control Panel -->
    <div class="flex flex-wrap items-center justify-between gap-4 border-t border-b border-border/60 py-3">
      <div class="flex items-center gap-3">
        <label for="initial-stock-input" class="text-xs font-medium text-foreground whitespace-nowrap">
          {t('products.variants.initialStock')}
        </label>
        <input
          id="initial-stock-input"
          type="number"
          min="0"
          bind:value={initialStock}
          disabled={!canWrite || creatingMissing}
          class="h-8 w-20 rounded-md border border-input bg-background px-2 text-center font-mono text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-50"
        />
      </div>

      <div class="flex items-center gap-3">
        {#if creatingMissing}
          <span class="text-xs text-muted-foreground font-medium">
            {t('products.variants.creating', { current: creationProgress.current, total: creationProgress.total })}
          </span>
        {/if}
        <Button
          variant="primary"
          size="sm"
          disabled={!canWrite || missingCombos.length === 0 || creatingMissing}
          loading={creatingMissing}
          onclick={handleCreateMissing}
        >
          {t('products.variants.createMissing', { count: missingCombos.length })}
        </Button>
      </div>
    </div>

    <!-- Per-item creation errors -->
    {#if creationErrors.length > 0}
      <div class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive space-y-1" role="alert">
        <p class="font-medium">{t('products.variants.creationErrors')}</p>
        <ul class="list-disc pl-4 space-y-0.5">
          {#each creationErrors as itemErr}
            <li>{itemErr}</li>
          {/each}
        </ul>
      </div>
    {/if}

    <!-- 5. Empty State when product has no variants -->
    {#if variants.length === 0}
      <div class="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
        {t('products.variants.emptyNotice')}
      </div>
    {/if}

    <!-- 2. The MATRIX -->
    {#if colors.length > 0 && sizes.length > 0}
      <div class="overflow-x-auto rounded-lg border border-border">
        <table class="w-full border-collapse text-left text-sm">
          <thead>
            <tr class="border-b border-border bg-muted/40 text-xs text-muted-foreground">
              <th class="whitespace-nowrap px-4 py-2.5 font-medium">{t('stock.colorAndSize')}</th>
              {#each sizes as size}
                <th class="whitespace-nowrap px-3 py-2.5 text-center font-medium">{size || '-'}</th>
              {/each}
            </tr>
          </thead>
          <tbody class="divide-y divide-border/60">
            {#each colors as color}
              <tr class="hover:bg-muted/30 transition-colors">
                <td class="whitespace-nowrap px-4 py-3 font-medium text-xs text-foreground bg-muted/10">
                  {color || '-'}
                </td>
                {#each sizes as size}
                  {@const variant = matrixCells[`${color}:${size}`]}
                  <td class="px-3 py-2.5 text-center">
                    {#if variant}
                      {@const variantStock = stock[variant.id] ?? { available: 0, reserved: 0 }}
                      <div class="flex flex-col items-center gap-0.5">
                        <div class="relative flex items-center justify-center">
                          <input
                            type="number"
                            min="0"
                            aria-label={`${product?.name ?? ''} ${color || ''} ${size || ''}`.trim()}
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
                            {t('stock.reserved', { count: variantStock.reserved })}
                          </span>
                        {/if}
                        {#if cellStatus[variant.id] === 'error'}
                          <span class="text-[10px] text-destructive leading-tight" role="alert">
                            {cellError[variant.id]}
                          </span>
                        {/if}
                      </div>
                    {:else}
                      <div class="mx-auto flex h-8 w-16 items-center justify-center rounded border border-dashed border-border/60 text-muted-foreground font-mono text-xs select-none">
                        -
                      </div>
                    {/if}
                  </td>
                {/each}
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  {/if}
</div>
