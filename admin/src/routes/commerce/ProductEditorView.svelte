<script lang="ts">
  import { onMount } from 'svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { router } from '$lib/router.svelte';
  import { schema } from '$lib/stores/schema.svelte';
  import { t } from '$lib/i18n/index.svelte';
  import PageSpinner from '$lib/components/ui/PageSpinner.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import RichTextEditor from '$lib/components/editor/RichTextEditor.svelte';
  import MediaPickerModal from '$lib/components/media/MediaPickerModal.svelte';
  import RelationField from '$lib/schema/RelationField.svelte';
  import VariantsTab from './product/VariantsTab.svelte';
  import { productTabs, type ProductTabContext } from './product/productTabs';
  import MediaTab from './product/MediaTab.svelte';

  import { slugify } from '$lib/commerce/slugify';

  interface Props {
    id: string;
  }

  let { id }: Props = $props();

  const isNew = $derived(id === 'new');
  const hasGenuineLeatherField = $derived(
    Boolean(schema.getCollection('products')?.fields?.some((f) => (f.id || f.name) === 'genuine_leather'))
  );

export interface ProductRecord {
  id: string;
  version?: number;
  name?: string;
  slug?: string;
  category?: string | { id: string; name?: string; slug?: string } | null;
  description?: string;
  cover_image?: string | null;
  base_price?: number | string | null;
  currency?: string;
  compare_at_price?: number | string | null;
  genuine_leather?: boolean;
  is_bestseller?: boolean;
  sku?: string | null;
  seo_title?: string | null;
  seo_description?: string | null;
  status?: 'published' | 'draft' | string;
  [key: string]: unknown;
}

  let product = $state<ProductRecord | null>(null);
  let categories = $state<Array<{ id: string; name?: string; slug?: string }>>([]);
  // A saved product loads on mount: start in the loading state so the form never flashes empty.
  // svelte-ignore state_referenced_locally
  let loading = $state(id !== 'new');
  let saving = $state(false);
  let publishing = $state(false);
  let errorMessage = $state('');
  let successMessage = $state('');
  let fieldErrors = $state<Record<string, string>>({});

  let activeTab = $state<string>('general');
  let mediaPickerOpen = $state(false);
  let showAdvanced = $state(false);
  let slugEditedManually = $state(false);

  let formData = $state({
    name: '',
    slug: '',
    category: '',
    description: '',
    cover_image: '',
    base_price: '' as string | number,
    currency: 'TRY',
    compare_at_price: '' as string | number,
    genuine_leather: false,
    is_bestseller: false,
    sku: '',
    seo_title: '',
    seo_description: '',
    status: 'draft',
  });

  function resolveMediaUrl(cover: unknown): string {
    if (!cover || typeof cover !== 'string') return '';
    const s = cover.trim();
    if (!s) return '';
    if (s.startsWith('http://') || s.startsWith('https://') || s.startsWith('/')) {
      return s;
    }
    return `/media/${s}`;
  }

  function syncTabFromHash() {
    if (typeof window === 'undefined') return;
    const h = window.location.hash.replace(/^#/, '');
    if (h === 'variants' || h === 'media' || h === 'general' || productTabs.some((x) => x.id === h)) {
      activeTab = h;
    }
  }

  function setTab(tab: string) {
    activeTab = tab;
    if (typeof window !== 'undefined') {
      window.location.hash = `#${tab}`;
    }
  }

  function handleNameInput(e: Event) {
    const val = (e.target as HTMLInputElement).value;
    formData.name = val;
    if (fieldErrors.name) {
      delete fieldErrors.name;
    }
    if (!slugEditedManually) {
      formData.slug = slugify(val);
      if (fieldErrors.slug) {
        delete fieldErrors.slug;
      }
    }
  }

  function handleSlugInput(e: Event) {
    slugEditedManually = true;
    formData.slug = (e.target as HTMLInputElement).value;
    if (fieldErrors.slug) {
      delete fieldErrors.slug;
    }
  }

  function productInCategory(slug: string): boolean {
    if (!product || isNew) return false;
    if (typeof product.category === 'object' && product.category !== null) {
      if ((product.category as { slug?: string }).slug === slug) return true;
    }
    let catId = '';
    if (typeof product.category === 'object' && product.category !== null) {
      catId = (product.category as { id: string }).id;
    } else if (typeof product.category === 'string') {
      catId = product.category;
    }
    if (!catId && formData.category) {
      catId = formData.category;
    }
    const cat = categories.find((c) => c.slug === slug);
    return Boolean(cat && catId === cat.id);
  }

  const extensionTabs = $derived.by(() => {
    const ctx: ProductTabContext = { product, isNew, isInCategory: productInCategory };
    return productTabs.filter((x) => !x.visible || x.visible(ctx));
  });

  $effect(() => {
    if (!['general', 'variants', 'media'].includes(activeTab) && !extensionTabs.some((x) => x.id === activeTab)) {
      activeTab = 'general';
    }
  });

  async function loadProduct() {
    if (isNew) {
      product = { id: 'new', status: 'draft' };
      return;
    }

    loading = true;
    errorMessage = '';
    try {
      try {
        const catRes = await api.get<{ data: Array<{ id: string; name?: string; slug?: string }> }>('/api/v1/admin/content/categories');
        categories = catRes.data ?? [];
      } catch {
        categories = [];
      }

      const res = await api.get<{ data: ProductRecord }>(`/api/v1/admin/content/products/${id}`);
      const data = res.data ?? (res as unknown as ProductRecord);
      product = data;

      let catId = '';
      if (typeof data.category === 'object' && data.category !== null) {
        catId = (data.category as { id: string }).id;
      } else if (typeof data.category === 'string') {
        catId = data.category;
      }

      formData = {
        name: data.name ?? '',
        slug: data.slug ?? '',
        category: catId,
        description: data.description ?? '',
        cover_image: data.cover_image ?? '',
        base_price: data.base_price != null ? data.base_price : '',
        currency: data.currency || 'TRY',
        compare_at_price: data.compare_at_price != null ? data.compare_at_price : '',
        genuine_leather: Boolean(data.genuine_leather),
        is_bestseller: Boolean(data.is_bestseller),
        sku: data.sku ?? '',
        seo_title: data.seo_title ?? '',
        seo_description: data.seo_description ?? '',
        status: data.status || 'draft',
      };
      slugEditedManually = true;
    } catch (err: unknown) {
      errorMessage = err instanceof ApiClientError ? err.message : t('products.error');
    } finally {
      loading = false;
    }
  }

  function validate(): boolean {
    const errors: Record<string, string> = {};

    if (!formData.name || !formData.name.trim()) {
      errors.name = t('products.editor.errorRequiredName');
    }

    if (!formData.slug || !formData.slug.trim()) {
      errors.slug = t('products.editor.errorRequiredSlug');
    }

    if (formData.base_price === '' || formData.base_price == null) {
      errors.base_price = t('products.editor.errorPrice');
    } else {
      const priceNum = Number(formData.base_price);
      if (isNaN(priceNum) || priceNum < 0) {
        errors.base_price = t('products.editor.errorPrice');
      }
    }

    if (formData.compare_at_price !== '' && formData.compare_at_price != null) {
      const compareNum = Number(formData.compare_at_price);
      const baseNum = Number(formData.base_price);
      if (!isNaN(compareNum)) {
        if (isNaN(baseNum) || compareNum <= baseNum) {
          errors.compare_at_price = t('products.editor.errorComparePrice');
        }
      }
    }

    fieldErrors = errors;
    return Object.keys(errors).length === 0;
  }

  async function handleSave() {
    errorMessage = '';
    successMessage = '';

    if (!validate()) {
      return;
    }

    saving = true;
    try {
      const payload: Record<string, unknown> = {
        name: formData.name.trim(),
        slug: formData.slug.trim(),
        category: formData.category || null,
        description: formData.description || '',
        cover_image: formData.cover_image || null,
        base_price: Number(formData.base_price),
        currency: formData.currency || 'TRY',
        compare_at_price:
          formData.compare_at_price !== '' && formData.compare_at_price != null && !isNaN(Number(formData.compare_at_price))
            ? Number(formData.compare_at_price)
            : null,
        is_bestseller: Boolean(formData.is_bestseller),
        sku: formData.sku.trim() || null,
        seo_title: formData.seo_title.trim() || null,
        seo_description: formData.seo_description.trim() || null,
      };
      if (hasGenuineLeatherField) {
        payload.genuine_leather = Boolean(formData.genuine_leather);
      }

      if (isNew) {
        const res = await api.post<{ data: ProductRecord }>('/api/v1/admin/content/products', payload);
        const created = res.data ?? (res as unknown as ProductRecord);
        product = created;
        router.navigate(`/commerce/products/${created.id}#variants`);
      } else {
        payload.version = product?.version;
        const res = await api.put<{ data: ProductRecord }>(`/api/v1/admin/content/products/${id}`, payload);
        const updated = res.data ?? (res as unknown as ProductRecord);
        product = updated;
        if (updated.status) {
          formData.status = updated.status;
        }
        successMessage = t('products.editor.saved');
      }
    } catch (err: unknown) {
      if (err instanceof ApiClientError) {
        if (err.fields) {
          fieldErrors = { ...err.fields };
        }
        errorMessage = err.message || t('products.editor.errorGeneral');
      } else if (err instanceof Error) {
        errorMessage = err.message;
      } else {
        errorMessage = t('products.editor.errorGeneral');
      }
    } finally {
      saving = false;
    }
  }

  async function handleToggleWorkflow() {
    if (isNew || !product) return;

    errorMessage = '';
    successMessage = '';
    publishing = true;

    const target = formData.status === 'published' ? 'draft' : 'published';
    try {
      const res = await api.post<{ data: ProductRecord }>(`/api/v1/admin/content/products/${id}/workflow`, {
        target,
        version: product.version,
      });
      const updated = res.data ?? (res as unknown as ProductRecord);
      product = updated;
      formData.status = updated.status || target;
      successMessage = t('products.editor.saved');
    } catch (err: unknown) {
      if (err instanceof ApiClientError) {
        errorMessage = err.message || t('products.editor.errorGeneral');
      } else if (err instanceof Error) {
        errorMessage = err.message;
      } else {
        errorMessage = t('products.editor.errorGeneral');
      }
    } finally {
      publishing = false;
    }
  }

  onMount(() => {
    syncTabFromHash();
    if (typeof window !== 'undefined') {
      window.addEventListener('hashchange', syncTabFromHash);
    }
    void loadProduct();
    return () => {
      if (typeof window !== 'undefined') {
        window.removeEventListener('hashchange', syncTabFromHash);
      }
    };
  });
</script>

<MediaPickerModal
  open={mediaPickerOpen}
  onclose={() => { mediaPickerOpen = false; }}
  onselect={(media) => {
    formData.cover_image = media.id || media.path || media.url || '';
    mediaPickerOpen = false;
  }}
/>

<div class="flex flex-1 flex-col gap-6 w-full">
  <!-- Header bar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-border pb-4">
    <div class="space-y-1">
      <div class="flex items-center gap-2">
        <button
          type="button"
          onclick={() => router.navigate('/commerce/products')}
          class="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground cursor-pointer transition-colors"
        >
          <svg class="size-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="15 18 9 12 15 6"></polyline>
          </svg>
          <span>{t('products.editor.back')}</span>
        </button>
      </div>
      <div class="flex items-center gap-3">
        <h1 class="text-xl font-semibold tracking-tight text-foreground">
          {isNew ? t('products.editor.titleNew') : (formData.name || t('products.editor.titleEdit'))}
        </h1>
        {#if !isNew}
          {#if formData.status === 'published'}
            <Badge variant="success" size="sm">{t('products.status.published')}</Badge>
          {:else}
            <Badge variant="outline" size="sm">{t('products.status.draft')}</Badge>
          {/if}
        {/if}
      </div>
    </div>

    <div class="flex items-center gap-2">
      <!-- Preview link (hidden when new) -->
      {#if !isNew && formData.slug}
        <a
          href={`${typeof window !== 'undefined' ? window.location.origin : ''}/products/${encodeURIComponent(formData.slug)}`}
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex items-center gap-1.5 h-8 px-3 text-xs font-medium rounded-md border border-input bg-background hover:bg-accent hover:text-accent-foreground text-foreground transition-colors shadow-xs"
        >
          <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
            <polyline points="15 3 21 3 21 9"></polyline>
            <line x1="10" y1="14" x2="21" y2="3"></line>
          </svg>
          <span>{t('products.editor.preview')}</span>
        </a>
      {/if}

      <!-- Publish / Unpublish (hidden when new) -->
      {#if !isNew}
        <Button
          variant="outline"
          size="sm"
          loading={publishing}
          onclick={handleToggleWorkflow}
        >
          {formData.status === 'published' ? t('products.editor.unpublish') : t('products.editor.publish')}
        </Button>
      {/if}

      <!-- Save button -->
      <Button
        variant="primary"
        size="sm"
        loading={saving}
        onclick={handleSave}
      >
        {t('products.editor.save')}
      </Button>
    </div>
  </div>

  <!-- Notifications -->
  {#if errorMessage}
    <div class="p-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs">
      {errorMessage}
    </div>
  {/if}
  {#if successMessage}
    <div class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs">
      {successMessage}
    </div>
  {/if}

  {#if loading}
    <PageSpinner />
  {:else}
    <!-- Tabs Header -->
    <div class="border-b border-border" role="tablist">
      <div class="flex gap-4">
        <button
          type="button"
          role="tab"
          id="tab-general"
          aria-selected={activeTab === 'general'}
          aria-controls="tabpanel-general"
          onclick={() => setTab('general')}
          class="pb-3 text-sm font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'general' ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'}"
        >
          {t('products.editor.tabGeneral')}
        </button>
        <button
          type="button"
          role="tab"
          id="tab-variants"
          aria-selected={activeTab === 'variants'}
          aria-controls="tabpanel-variants"
          onclick={() => setTab('variants')}
          class="pb-3 text-sm font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'variants' ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'}"
        >
          {t('products.editor.tabVariants')}
        </button>
        <button
          type="button"
          role="tab"
          id="tab-media"
          aria-selected={activeTab === 'media'}
          aria-controls="tabpanel-media"
          onclick={() => setTab('media')}
          class="pb-3 text-sm font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'media' ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'}"
        >
          Photos & Video
        </button>
        {#each extensionTabs as ext (ext.id)}
          <button
            type="button"
            role="tab"
            id="tab-{ext.id}"
            aria-selected={activeTab === ext.id}
            aria-controls="tabpanel-{ext.id}"
            onclick={() => setTab(ext.id)}
            class="pb-3 text-sm font-medium border-b-2 transition-colors cursor-pointer {activeTab === ext.id ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'}"
          >
            {ext.label}
          </button>
        {/each}
      </div>
    </div>

    <!-- Tab Panels -->
    {#if activeTab === 'general'}
      <div id="tabpanel-general" role="tabpanel" aria-labelledby="tab-general" class="space-y-6">
        <div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-5">
          <!-- Name & Slug -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input
              id="product-name"
              label={t('products.editor.name')}
              placeholder={t('products.editor.namePlaceholder')}
              value={formData.name}
              oninput={handleNameInput}
              error={fieldErrors.name}
              required
            />
            <Input
              id="product-slug"
              label={t('products.editor.slug')}
              placeholder={t('products.editor.slugPlaceholder')}
              value={formData.slug}
              oninput={handleSlugInput}
              error={fieldErrors.slug}
              required
            />
          </div>

          <!-- Category -->
          <RelationField
            id="category"
            label={t('products.editor.category')}
            target="categories"
            bind:value={formData.category}
          />

          <!-- Description -->
          <div class="space-y-1.5">
            <RichTextEditor
              label={t('products.editor.description')}
              bind:value={formData.description}
            />
          </div>

          <!-- Cover Image -->
          <div class="flex flex-col gap-1.5">
            <span class="text-xs font-medium text-foreground">
              {t('products.editor.coverImage')}
            </span>
            <div class="flex items-center gap-2">
              {#if formData.cover_image}
                <div class="relative size-12 rounded-md overflow-hidden border border-border bg-muted shrink-0">
                  <img
                    src={resolveMediaUrl(formData.cover_image)}
                    alt={formData.name || t('products.editor.coverAlt')}
                    class="size-full object-cover"
                  />
                </div>
              {/if}
              <input
                type="text"
                placeholder={t('products.editor.mediaPlaceholder')}
                bind:value={formData.cover_image}
                class="flex-1 h-9 rounded-md border border-input bg-background px-3 font-mono text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
              />
              <button
                type="button"
                class="h-9 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs shrink-0"
                onclick={() => { mediaPickerOpen = true; }}
              >
                {t('products.editor.browse')}
              </button>
              {#if formData.cover_image}
                <button
                  type="button"
                  class="h-9 px-2.5 rounded-md border border-input bg-background hover:bg-destructive/10 hover:text-destructive text-xs font-medium text-muted-foreground transition-colors cursor-pointer shadow-xs shrink-0"
                  onclick={() => { formData.cover_image = ''; }}
                  aria-label={t('products.editor.removeImage')}
                >
                  ×
                </button>
              {/if}
            </div>
          </div>

          <!-- Pricing & Currency -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="flex flex-col gap-1.5">
              <label for="product-price" class="text-xs font-medium text-foreground">
                {t('products.editor.price')} <span class="text-destructive">*</span>
              </label>
              <div class="flex items-center gap-2">
                <input
                  id="product-price"
                  type="number"
                  step="0.01"
                  min="0"
                  placeholder="0.00"
                  bind:value={formData.base_price}
                  oninput={() => {
                    if (fieldErrors.base_price) delete fieldErrors.base_price;
                    if (fieldErrors.compare_at_price) delete fieldErrors.compare_at_price;
                  }}
                  class="flex-1 h-9 rounded-md border {fieldErrors.base_price ? 'border-destructive focus:ring-destructive' : 'border-input focus:ring-ring'} bg-background px-3 text-xs text-foreground focus:outline-none focus:ring-1 shadow-xs"
                />
                <select
                  aria-label={t('products.editor.currency')}
                  bind:value={formData.currency}
                  class="h-9 w-24 rounded-md border border-input bg-background px-2 text-xs font-medium text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
                >
                  <option value="TRY">TRY</option>
                  <option value="USD">USD</option>
                  <option value="EUR">EUR</option>
                </select>
              </div>
              {#if fieldErrors.base_price}
                <p class="text-xs text-destructive">{fieldErrors.base_price}</p>
              {/if}
            </div>

            <!-- Compare-at price -->
            <div class="flex flex-col gap-1.5">
              <label for="product-compare-price" class="text-xs font-medium text-foreground">
                {t('products.editor.comparePrice')}
              </label>
              <input
                id="product-compare-price"
                type="number"
                step="0.01"
                min="0"
                placeholder="0.00"
                bind:value={formData.compare_at_price}
                oninput={() => {
                  if (fieldErrors.compare_at_price) delete fieldErrors.compare_at_price;
                }}
                class="h-9 rounded-md border {fieldErrors.compare_at_price ? 'border-destructive focus:ring-destructive' : 'border-input focus:ring-ring'} bg-background px-3 text-xs text-foreground focus:outline-none focus:ring-1 shadow-xs"
              />
              <p class="text-[11px] text-muted-foreground">
                {t('products.editor.comparePriceHelp')}
              </p>
              {#if fieldErrors.compare_at_price}
                <p class="text-xs text-destructive">{fieldErrors.compare_at_price}</p>
              {/if}
            </div>
          </div>

          <!-- Genuine Leather Toggle (only shown when the products schema defines genuine_leather) -->
          {#if hasGenuineLeatherField}
            <div class="pt-2">
              <label class="flex items-center gap-2 cursor-pointer select-none">
                <input
                  type="checkbox"
                  bind:checked={formData.genuine_leather}
                  class="size-4 rounded border-input text-primary focus:ring-ring"
                />
                <span class="text-xs font-medium text-foreground">{t('products.editor.genuineLeather')}</span>
              </label>
            </div>
          {/if}

          <!-- Advanced Collapsed Section -->
          <div class="border-t border-border pt-4">
            <button
              type="button"
              class="flex items-center gap-2 text-xs font-medium text-muted-foreground hover:text-foreground cursor-pointer transition-colors"
              onclick={() => { showAdvanced = !showAdvanced; }}
              aria-expanded={showAdvanced}
            >
              <svg
                class="size-4 transition-transform duration-200 {showAdvanced ? 'rotate-90' : ''}"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <polyline points="9 18 15 12 9 6"></polyline>
              </svg>
              <span>{t('products.editor.advanced')}</span>
            </button>

            {#if showAdvanced}
              <div class="mt-4 space-y-4 pl-6 border-l-2 border-border/60">
                <Input
                  id="product-sku"
                  label={t('products.editor.sku')}
                  placeholder={t('products.editor.skuPlaceholder')}
                  bind:value={formData.sku}
                  error={fieldErrors.sku}
                />
                <Input
                  id="product-seo-title"
                  label={t('products.editor.seoTitle')}
                  bind:value={formData.seo_title}
                  error={fieldErrors.seo_title}
                />
                <div class="flex flex-col gap-1.5">
                  <label for="product-seo-description" class="text-xs font-medium text-foreground">
                    {t('products.editor.seoDescription')}
                  </label>
                  <textarea
                    id="product-seo-description"
                    bind:value={formData.seo_description}
                    rows="3"
                    class="w-full rounded-md border border-input bg-background px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
                  ></textarea>
                </div>
              </div>
            {/if}
          </div>
        </div>
      </div>
    {:else if activeTab === 'variants'}
      <div id="tabpanel-variants" role="tabpanel" aria-labelledby="tab-variants">
        <VariantsTab {product} />
      </div>
    {:else if activeTab === 'media'}
      <div id="tabpanel-media" role="tabpanel" aria-labelledby="tab-media">
        <MediaTab {product} />
      </div>
    {:else}
      {#each extensionTabs as ext (ext.id)}
        {#if activeTab === ext.id}
          {@const Ext = ext.component}
          <div id="tabpanel-{ext.id}" role="tabpanel" aria-labelledby="tab-{ext.id}">
            <Ext
              {product}
              onsaved={(updated: ProductRecord) => {
                product = updated;
                if (typeof updated.status === 'string') {
                  formData.status = updated.status;
                }
              }}
            />
          </div>
        {/if}
      {/each}
    {/if}
  {/if}
</div>
