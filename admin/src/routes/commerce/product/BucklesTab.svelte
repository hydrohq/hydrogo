<script lang="ts">
  import { onMount } from 'svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { t } from '$lib/i18n/index.svelte';
  import PageSpinner from '$lib/components/ui/PageSpinner.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import type { ProductRecord } from '../ProductEditorView.svelte';

  interface Props {
    product: ProductRecord | null;
    onsaved?: (updatedRecord: ProductRecord) => void;
  }

  let { product, onsaved }: Props = $props();

  interface Category {
    id: string;
    name?: string;
    slug?: string;
    [key: string]: unknown;
  }

  interface BuckleProduct {
    id: string;
    name?: string;
    slug?: string;
    category?: string | { id: string; name?: string; slug?: string } | null;
    cover_image?: string | null;
    base_price?: number | string | null;
    currency?: string | null;
    status?: string;
    [key: string]: unknown;
  }

  const isNew = $derived(!product || product.id === 'new' || !product.id);

  let categories = $state<Category[]>([]);
  let allProducts = $state<BuckleProduct[]>([]);
  // Starts true for a saved product: the first render must not flash the empty state before data arrives.
  // svelte-ignore state_referenced_locally
  let loading = $state(!!product?.id && product.id !== 'new');
  let saving = $state(false);
  let errorMessage = $state('');
  let successMessage = $state('');

  function resolveMediaUrl(cover: unknown): string {
    if (!cover || typeof cover !== 'string') return '';
    const s = cover.trim();
    if (!s) return '';
    if (s.startsWith('http://') || s.startsWith('https://') || s.startsWith('/')) {
      return s;
    }
    return `/media/${s}`;
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

  const tokalarCategory = $derived(categories.find((c) => c.slug === 'tokalar'));
  const tokalarCategoryId = $derived(tokalarCategory?.id ?? null);

  function getProductCategoryId(p: ProductRecord | BuckleProduct | null | undefined): string | null {
    if (!p || !p.category) return null;
    if (typeof p.category === 'object' && p.category !== null) {
      return (p.category as { id: string }).id || null;
    }
    return String(p.category);
  }

  function isProductInTokalarCategory(p: ProductRecord | BuckleProduct | null | undefined): boolean {
    if (!p) return false;
    if (typeof p.category === 'object' && p.category !== null) {
      if ((p.category as { slug?: string }).slug === 'tokalar') return true;
    }
    const catId = getProductCategoryId(p);
    return Boolean(tokalarCategoryId && catId === tokalarCategoryId);
  }

  const isCurrentProductBuckle = $derived(isProductInTokalarCategory(product));

  const candidateBuckles = $derived.by(() => {
    if (!product?.id) return [];
    const curId = product.id;
    const others = allProducts.filter((p) => p.id !== curId);
    const eligible = others.filter(
      (p) => !p.status || p.status === 'published' || p.status === 'draft'
    );

    if (tokalarCategoryId) {
      return eligible.filter((p) => isProductInTokalarCategory(p));
    }
    return eligible;
  });

  const showFallbackHint = $derived(!loading && !tokalarCategoryId);

  function extractBuckleIds(p: ProductRecord | null): string[] {
    if (!p || !Array.isArray(p.compatible_buckles)) return [];
    return p.compatible_buckles
      .map((item: unknown) =>
        typeof item === 'object' && item !== null && 'id' in item
          ? String((item as { id: unknown }).id)
          : String(item ?? '')
      )
      .filter(Boolean);
  }

  let selectedIds = $state<string[]>([]);
  let initialSelectedIds = $state<string[]>([]);
  let lastProductKey = $state<string>('');

  $effect(() => {
    const key = `${product?.id ?? ''}:${product?.version ?? ''}`;
    if (key !== lastProductKey) {
      lastProductKey = key;
      const ids = extractBuckleIds(product);
      selectedIds = [...ids];
      initialSelectedIds = [...ids];
    }
  });

  const isDirty = $derived.by(() => {
    if (selectedIds.length !== initialSelectedIds.length) return true;
    const initSet = new Set(initialSelectedIds);
    return selectedIds.some((id) => !initSet.has(id));
  });

  function toggleSelection(buckleId: string) {
    errorMessage = '';
    successMessage = '';
    if (selectedIds.includes(buckleId)) {
      selectedIds = selectedIds.filter((id) => id !== buckleId);
    } else {
      selectedIds = [...selectedIds, buckleId];
    }
  }

  async function loadData() {
    if (isNew || !product?.id) return;
    loading = true;
    errorMessage = '';
    try {
      const catRes = await api.get<{ data: Category[] }>('/api/v1/admin/content/categories');
      categories = catRes.data ?? [];

      const prods: BuckleProduct[] = [];
      let offset = 0;
      while (true) {
        const pRes = await api.get<{ data: BuckleProduct[] }>(
          `/api/v1/admin/content/products?limit=100&offset=${offset}`
        );
        const batch = pRes.data ?? [];
        prods.push(...batch);
        if (batch.length < 100) break;
        offset += 100;
      }
      allProducts = prods;
    } catch (err: unknown) {
      if (err instanceof ApiClientError) {
        errorMessage = err.message || t('products.buckles.errorLoad');
      } else if (err instanceof Error) {
        errorMessage = err.message;
      } else {
        errorMessage = t('products.buckles.errorLoad');
      }
    } finally {
      loading = false;
    }
  }

  async function handleSave() {
    if (!product?.id || isNew || !isDirty || saving) return;
    saving = true;
    errorMessage = '';
    successMessage = '';

    try {
      const payload: Record<string, unknown> = {};
      for (const [key, value] of Object.entries(product)) {
        if (['id', 'created_at', 'updated_at', 'status', 'version'].includes(key)) {
          continue;
        }
        payload[key] = value;
      }
      payload.compatible_buckles = selectedIds;
      if (product.version !== undefined) {
        payload.version = product.version;
      }

      const res = await api.put<{ data: ProductRecord }>(
        `/api/v1/admin/content/products/${product.id}`,
        payload
      );
      const updated = res.data ?? (res as unknown as ProductRecord);
      initialSelectedIds = [...selectedIds];
      successMessage = t('products.buckles.saved');
      onsaved?.(updated);
    } catch (err: unknown) {
      if (err instanceof ApiClientError) {
        errorMessage = err.message || t('products.buckles.errorSave');
      } else if (err instanceof Error) {
        errorMessage = err.message;
      } else {
        errorMessage = t('products.buckles.errorSave');
      }
    } finally {
      saving = false;
    }
  }

  let loadedForId = '';
  $effect(() => {
    const curId = product?.id;
    if (curId && curId !== 'new' && curId !== loadedForId) {
      loadedForId = curId;
      void loadData();
    }
  });

  onMount(() => {
    if (product?.id && product.id !== 'new' && !loadedForId) {
      loadedForId = product.id;
      void loadData();
    }
  });
</script>

<div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-6" data-testid="buckles-tab">
  {#if isNew}
    <div class="flex flex-col items-center justify-center text-center p-8 space-y-3">
      <div class="size-10 rounded-full bg-muted flex items-center justify-center text-muted-foreground">
        <svg class="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10"></circle>
          <circle cx="12" cy="12" r="4"></circle>
        </svg>
      </div>
      <div class="space-y-1">
        <h3 class="text-sm font-medium text-foreground">{t('products.editor.saveFirstNotice')}</h3>
        <p class="text-xs text-muted-foreground max-w-sm">
          {t('products.buckles.saveFirstHelp')}
        </p>
      </div>
    </div>
  {:else if isCurrentProductBuckle}
    <div class="flex flex-col items-center justify-center text-center p-8 space-y-3" data-testid="buckle-product-note">
      <div class="size-10 rounded-full bg-muted flex items-center justify-center text-muted-foreground">
        <svg class="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="4.93" y1="4.93" x2="19.07" y2="19.07"></line>
        </svg>
      </div>
      <div class="space-y-1">
        <h3 class="text-sm font-medium text-foreground">{t('products.buckles.isBuckleTitle')}</h3>
        <p class="text-xs text-muted-foreground max-w-sm">
          {t('products.buckles.isBuckleNotice')}
        </p>
      </div>
    </div>
  {:else}
    <!-- Controls Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-border pb-4">
      <div class="space-y-1">
        <div class="flex items-center gap-2">
          <h2 class="text-sm font-semibold tracking-tight text-foreground">
            {t('products.buckles.title')}
          </h2>
          <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-muted text-muted-foreground">
            {t('products.buckles.selectedCount', { count: selectedIds.length })}
          </span>
        </div>
        <p class="text-xs text-muted-foreground">
          {t('products.buckles.description')}
        </p>
      </div>

      <div class="flex items-center gap-2">
        <Button
          variant="primary"
          size="sm"
          disabled={!isDirty || saving}
          loading={saving}
          onclick={handleSave}
        >
          {saving ? t('products.buckles.saving') : t('products.buckles.save')}
        </Button>
      </div>
    </div>

    <!-- Notifications -->
    {#if errorMessage}
      <div class="p-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs" role="alert">
        {errorMessage}
      </div>
    {/if}
    {#if successMessage}
      <div class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs">
        {successMessage}
      </div>
    {/if}

    {#if showFallbackHint}
      <div class="p-3 rounded-lg bg-muted/60 border border-border text-xs text-muted-foreground flex items-center gap-2">
        <svg class="size-4 shrink-0 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="12" y1="16" x2="12" y2="12"></line>
          <line x1="12" y1="8" x2="12.01" y2="8"></line>
        </svg>
        <span>{t('products.buckles.fallbackHint')}</span>
      </div>
    {/if}

    {#if loading && candidateBuckles.length === 0}
      <PageSpinner compact />
    {:else if candidateBuckles.length === 0}
      <div class="p-12 text-center text-xs text-muted-foreground">
        {t('products.buckles.emptyNotice')}
      </div>
    {:else}
      <!-- Cards Grid -->
      <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3.5">
        {#each candidateBuckles as buckle (buckle.id)}
          {@const isSelected = selectedIds.includes(buckle.id)}
          <button
            type="button"
            aria-pressed={isSelected}
            onclick={() => toggleSelection(buckle.id)}
            class="relative flex flex-col text-left rounded-xl border p-2.5 transition-all cursor-pointer focus:outline-none focus:ring-2 focus:ring-ring select-none {isSelected ? 'border-primary bg-primary/5 ring-1 ring-primary shadow-xs' : 'border-border bg-card hover:border-muted-foreground/40 shadow-xs'}"
          >
            <!-- Checkmark Indicator -->
            <div
              class="absolute top-2 right-2 size-5 rounded-full flex items-center justify-center transition-colors z-10 {isSelected ? 'bg-primary text-primary-foreground shadow-xs' : 'border border-border bg-background/90 text-transparent'}"
              aria-hidden="true"
            >
              <svg class="size-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="20 6 9 17 4 12"></polyline>
              </svg>
            </div>

            <!-- Cover Thumbnail -->
            <div class="aspect-square w-full rounded-lg overflow-hidden bg-muted mb-2 flex items-center justify-center">
              {#if buckle.cover_image}
                <img
                  src={resolveMediaUrl(buckle.cover_image)}
                  alt={buckle.name || ''}
                  class="size-full object-cover"
                />
              {:else}
                <svg class="size-7 text-muted-foreground/40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                  <rect width="18" height="18" x="3" y="3" rx="2" ry="2"></rect>
                  <circle cx="9" cy="9" r="2"></circle>
                  <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"></path>
                </svg>
              {/if}
            </div>

            <!-- Info -->
            <div class="space-y-0.5 w-full min-w-0 pr-4">
              <div class="text-xs font-medium text-foreground truncate" title={buckle.name}>
                {buckle.name || '—'}
              </div>
              <div class="text-xs text-muted-foreground font-medium">
                {formatPrice(buckle.base_price, buckle.currency)}
              </div>
            </div>
          </button>
        {/each}
      </div>
    {/if}
  {/if}
</div>
