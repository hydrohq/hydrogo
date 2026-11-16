<script lang="ts">
  import { untrack } from 'svelte';
  import { t } from '$lib/i18n/index.svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { importCsvFile, importErrorMessage } from './csvImport';
  import { schema } from '$lib/stores/schema.svelte';
  import { auth } from '$lib/stores/auth.svelte';
  import { router } from '$lib/router.svelte';
  import type { ContentItem, ApiResponse, Field } from '$lib/api/types';
  import Badge from '$lib/components/ui/Badge.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
  import GenericItemModal from './GenericItemModal.svelte';
  import SelectOptionsModal from './SelectOptionsModal.svelte';
  import {
    expandParam,
    isRelation,
    recordTitle,
    relationCell,
    relationTarget,
    relationValueField,
  } from './relations';
  import { hiddenStatusField, setPublication, usesDrafts } from './publication';
  import { confirmAction } from '$lib/confirm';
  import { statusLabel } from '$lib/i18n/labels';

  interface Props {
    type: string;
    initialItemId?: string | null;
  }

  let { type, initialItemId = null }: Props = $props();

  const collection = $derived(schema.getCollection(type));
  let items = $state<ContentItem[]>([]);
  let loading = $state(true);
  let search = $state('');
  let page = $state(1);
  let hasMore = $state(false);
  const PAGE_SIZE = 25;
  let statusFilter = $state('');
  let fieldFilters = $state<Record<string, string>>({});
  let sortField = $state('');
  let slugLookup = $state<Record<string, Record<string, string>>>({});
  let optionsModalOpen = $state(false);

  // Modal state is retained only when initialItemId is explicitly passed by an embedder.
  let modalOpen = $state(false);
  let activeItemId = $state<string | null>(null);

  let deleteTargetId = $state<string | null>(null);
  let deleteLoading = $state(false);

  function getFieldKey(f: Field): string {
    return f.id || f.name || '';
  }

  const listFields = $derived.by(() => {
    if (!collection?.fields) return [];
    const hidden = hiddenStatusField(collection);
    const fields = collection.fields.filter((f) => getFieldKey(f) !== hidden);
    const explicit = fields.filter((f) => f.visible_in_list || f.list);
    if (explicit.length > 0) return explicit;
    return fields.slice(0, 4);
  });

  const filterableFields = $derived.by(() => {
    if (!collection?.fields) return [];
    const hidden = hiddenStatusField(collection);
    return collection.fields.filter(
      (f) =>
        f.filterable &&
        getFieldKey(f) !== hidden &&
        (f.kind === 'select' || f.kind === 'boolean')
    );
  });

  const manageSelectField = $derived.by(() => {
    if (collection?.source !== 'content_type' || !auth.hasPermission('admin.content.write')) {
      return null;
    }
    const hidden = hiddenStatusField(collection);
    const selects = (collection.fields ?? []).filter(
      (f) => f.kind === 'select' && getFieldKey(f) !== hidden
    );
    return selects.find((f) => getFieldKey(f) === 'category') ?? selects[0] ?? null;
  });

  const hasServerSearch = $derived(
    collection?.source !== 'content_type' ||
      (collection?.fields ?? []).some((f) => f.searchable)
  );

  const filteredItems = $derived.by(() => {
    const q = search.trim().toLowerCase();
    if (!q || hasServerSearch) return items;
    return items.filter((item) => JSON.stringify(item).toLowerCase().includes(q));
  });

  const bulkEnabled = $derived(
    !!collection && collection.api_path?.startsWith('/api/v1/admin/content/')
  );
  let selected = $state<Record<string, boolean>>({});
  const selectedIds = $derived(Object.keys(selected).filter((id) => selected[id]));
  let bulkDeleteOpen = $state(false);
  let bulkLoading = $state(false);
  let bulkError = $state('');

  const allShownSelected = $derived(
    filteredItems.length > 0 && filteredItems.every((item) => selected[item.id])
  );

  function toggleAll() {
    const next = !allShownSelected;
    const map: Record<string, boolean> = { ...selected };
    for (const item of filteredItems) map[item.id] = next;
    selected = map;
  }

  async function confirmBulkDelete() {
    if (!collection) return;
    bulkLoading = true;
    bulkError = '';
    try {
      await api.post('/api/v1/admin/content/_batch', {
        operations: selectedIds.map((id) => ({ op: 'delete', type, id })),
      });
      bulkDeleteOpen = false;
      selected = {};
      await loadItems();
    } catch (err) {
      bulkError =
        err instanceof ApiClientError
          ? `${err.message}${err.requestId ? ` (${t('schema.requestId')}: ${err.requestId})` : ''}`
          : t('schema.bulkFailed');
      bulkDeleteOpen = false;
    } finally {
      bulkLoading = false;
    }
  }

  const csvBase = $derived(collection?.api_path || `/api/v1/admin/content/${type}`);
  let importInput = $state<HTMLInputElement | null>(null);
  let importMessage = $state('');
  let importFailed = $state(false);

  async function importCsv(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file) return;
    importMessage = '';
    importFailed = false;
    try {
      importMessage = t('schema.importDone', { count: await importCsvFile(csvBase, file) });
      await loadItems();
    } catch (err) {
      importFailed = true;
      importMessage = importErrorMessage(err, t('schema.requestId'), t('schema.importFailed'));
    }
  }

  let loadedKey = '';
  const queryKey = () =>
    `${type}|${statusFilter}|${hasServerSearch ? search.trim() : ''}|${sortField}|${JSON.stringify(fieldFilters)}`;

  const hasStatus = $derived(Boolean(collection?.workflow?.enabled || usesDrafts(collection)));
  let actingId = $state<string | null>(null);

  async function togglePublication(item: ContentItem, e: MouseEvent) {
    e.stopPropagation();
    if (!collection) return;
    const target = (item.status || 'draft') === 'published' ? 'draft' : 'published';
    const publishing = target === 'published';
    if (
      !(await confirmAction({
        title: t(publishing ? 'schema.confirm.publishTitle' : 'schema.confirm.unpublishTitle'),
        description: t(publishing ? 'schema.confirm.publishDesc' : 'schema.confirm.unpublishDesc'),
        confirmLabel: t(publishing ? 'schema.confirm.publishButton' : 'schema.confirm.unpublishButton'),
        cancelLabel: t('common.cancel'),
        variant: publishing ? 'primary' : 'destructive',
      }))
    ) {
      return;
    }
    actingId = String(item.id);
    try {
      await setPublication(
        collection.api_path || `/api/v1/admin/content/${type}`,
        String(item.id),
        target,
        { status: item.status, version: Number(item.version) || 0 }
      );
      await loadItems();
    } catch (err) {
      console.warn('Failed to change publication:', err);
    } finally {
      actingId = null;
    }
  }

  async function loadSlugLookups(fields: Field[]) {
    for (const f of fields) {
      if (f.widget !== 'relation') continue;
      const target = relationTarget(f);
      if (!target || slugLookup[target]) continue;
      const targetCol = schema.getCollection(target);
      const targetEndpoint = targetCol?.api_path || `/api/v1/admin/content/${target}`;
      const valField = relationValueField(f);
      try {
        const res = await api.get<ApiResponse<Record<string, unknown>[]>>(targetEndpoint, {
          params: { limit: 100 },
        });
        const nextMap: Record<string, string> = {};
        for (const rec of res?.data ?? []) {
          const title = recordTitle(rec, targetCol?.title_field);
          const key = rec[valField] !== undefined ? String(rec[valField]) : String(rec.id ?? '');
          if (key) nextMap[key] = title;
          if (rec.id) nextMap[String(rec.id)] = title;
        }
        slugLookup = { ...slugLookup, [target]: nextMap };
      } catch {
        slugLookup = { ...slugLookup, [target]: {} };
      }
    }
  }

  async function loadItems() {
    if (!collection) return;
    loading = true;
    loadedKey = queryKey();
    void loadSlugLookups(listFields);
    try {
      const params: Record<string, string | number> = {
        limit: PAGE_SIZE,
        offset: (page - 1) * PAGE_SIZE,
      };
      if (hasServerSearch && search.trim()) params.search = search.trim();
      if (statusFilter && hasStatus) params.status = statusFilter;
      if (sortField) params.sort = sortField;
      for (const [k, v] of Object.entries(fieldFilters)) {
        if (v !== '') params[`filter[${k}]`] = v;
      }
      const expand = expandParam(listFields);
      if (expand) params.expand = expand;
      const res = await api.get<ApiResponse<ContentItem[]>>(
        collection.api_path || `/api/v1/admin/content/${type}`,
        { params }
      );
      if (res && res.data) {
        items = Array.isArray(res.data) ? res.data : [];
        hasMore = Boolean((res.metadata as { has_more?: boolean } | undefined)?.has_more);
      }
    } catch (e) {
      console.warn('Failed to load items:', e);
      items = [];
    } finally {
      loading = false;
    }
  }

  function canSortField(f: Field): boolean {
    return (
      collection?.source === 'content_type' &&
      !f.private &&
      f.kind !== 'relation' &&
      f.kind !== 'media' &&
      f.widget !== 'media'
    );
  }

  function toggleSort(key: string) {
    if (!key) return;
    if (sortField === key) sortField = `-${key}`;
    else if (sortField === `-${key}`) sortField = '';
    else sortField = key;
  }

  function isVideoUrl(val: unknown): boolean {
    return typeof val === 'string' && /\.(mp4|webm|mov)(\?|$)/i.test(val.trim());
  }

  function openCreate() {
    if (initialItemId) {
      activeItemId = 'new';
      modalOpen = true;
      return;
    }
    router.navigate(`/content/${type}/new`);
  }

  function openEdit(id: string, e?: MouseEvent) {
    if (e) e.stopPropagation();
    if (initialItemId) {
      activeItemId = id;
      modalOpen = true;
      return;
    }
    router.navigate(`/content/${type}/${id}`);
  }

  function closeModal() {
    modalOpen = false;
    activeItemId = null;
    if (router.path !== `/content/${type}`) {
      router.navigate(`/content/${type}`);
    }
  }

  function handleDelete(id: string, e: MouseEvent) {
    e.stopPropagation();
    deleteTargetId = id;
  }

  async function confirmDelete() {
    if (!deleteTargetId) return;
    deleteLoading = true;
    try {
      await api.delete(
        `${collection?.api_path || `/api/v1/admin/content/${type}`}/${deleteTargetId}`
      );
      deleteTargetId = null;
      await loadItems();
    } catch (err: any) {
      console.error('Delete failed:', err);
    } finally {
      deleteLoading = false;
    }
  }

  $effect(() => {
    if (type) {
      untrack(() => {
        page = 1;
        statusFilter = '';
        fieldFilters = {};
        sortField = '';
        search = '';
        selected = {};
        slugLookup = {};
        bulkError = '';
        void loadItems();
      });
    }
  });

  $effect(() => {
    const key = queryKey();
    if (key === loadedKey) return;
    const delay = hasServerSearch && search.trim() ? 300 : 0;
    const timer = setTimeout(() => {
      page = 1;
      void loadItems();
    }, delay);
    return () => clearTimeout(timer);
  });

  $effect(() => {
    if (initialItemId) {
      activeItemId = initialItemId;
      modalOpen = true;
    }
  });
</script>

{#if initialItemId}
  <GenericItemModal
    open={modalOpen}
    {type}
    itemId={activeItemId}
    onclose={closeModal}
    onsaved={loadItems}
  />
{/if}

{#if manageSelectField}
  <SelectOptionsModal
    open={optionsModalOpen}
    collectionId={type}
    fieldId={getFieldKey(manageSelectField)}
    fieldLabel={manageSelectField.label || getFieldKey(manageSelectField)}
    onclose={() => { optionsModalOpen = false; }}
    onupdated={() => { void loadItems(); }}
  />
{/if}

<ConfirmDialog
  open={deleteTargetId !== null}
  title={t('schema.deleteTitle', { name: collection?.singular || collection?.label || t('schema.item') })}
  description={t('schema.deleteDescription')}
  loading={deleteLoading}
  onconfirm={confirmDelete}
  oncancel={() => { deleteTargetId = null; }}
/>

<ConfirmDialog
  open={bulkDeleteOpen}
  title={t('schema.bulkDeleteTitle', { count: selectedIds.length })}
  description={t('schema.bulkDeleteDescription')}
  loading={bulkLoading}
  onconfirm={confirmBulkDelete}
  oncancel={() => { bulkDeleteOpen = false; }}
/>

{#if schema.isLoading}
  <div class="p-12 text-center text-xs text-muted-foreground flex flex-col items-center justify-center gap-2">
    <div class="size-6 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
    <span>{t('schema.loadingSchema', { type })}</span>
  </div>
{:else if !collection}
  <div class="p-12 text-center text-xs text-muted-foreground">
    <p>{t('schema.notFoundBefore')} <strong>{type}</strong> {t('schema.notFoundAfter')}</p>
    <button
      type="button"
      class="mt-4 inline-flex items-center gap-1.5 h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer"
      onclick={() => router.navigate('/dashboard')}
    >{t('schema.backToDashboard')}</button>
  </div>
{:else}
  <div class="flex flex-1 flex-col gap-4 md:gap-6">
    <!-- Toolbar -->
    <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-3">
      <div class="flex flex-wrap items-center gap-2">
        {#if hasStatus}
          <div class="inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground">
            <button
              type="button"
              class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium transition-all cursor-pointer {statusFilter === '' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
              onclick={() => { statusFilter = ''; }}
            >{t('common.all')}</button>
            <button
              type="button"
              class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium transition-all cursor-pointer {statusFilter === 'published' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
              onclick={() => { statusFilter = 'published'; }}
            >{t('common.published')}</button>
            <button
              type="button"
              class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium transition-all cursor-pointer {statusFilter === 'draft' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
              onclick={() => { statusFilter = 'draft'; }}
            >{t('news.list.drafts')}</button>
          </div>
        {:else}
          <div class="text-sm font-medium text-muted-foreground">
            {collection.plural || collection.label || type}
          </div>
        {/if}

        {#each filterableFields as ff (getFieldKey(ff))}
          {@const fk = getFieldKey(ff)}
          <select
            aria-label={ff.label || fk}
            value={fieldFilters[fk] ?? ''}
            onchange={(e) => {
              const val = (e.currentTarget as HTMLSelectElement).value;
              const next = { ...fieldFilters };
              if (val === '') delete next[fk];
              else next[fk] = val;
              fieldFilters = next;
            }}
            class="h-8 rounded-md border border-input bg-background px-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
          >
            <option value="">{t('schema.filterAll', { label: ff.label || fk })}</option>
            {#if ff.kind === 'boolean'}
              <option value="true">{t('common.yes')}</option>
              <option value="false">{t('common.no')}</option>
            {:else}
              {#each (ff.options || []) as opt}
                {@const optVal = typeof opt === 'string' ? opt : String(opt.value)}
                {@const optLabel = typeof opt === 'string' ? opt : opt.label}
                <option value={optVal}>{optLabel}</option>
              {/each}
            {/if}
          </select>
        {/each}
      </div>

      <!-- Search & Action Buttons -->
      <div class="flex flex-wrap items-center gap-2">
        <div class="relative w-full sm:w-56">
          <svg class="absolute left-2.5 top-2.5 size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"></circle>
            <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
          </svg>
          <input
            type="search"
            placeholder={t('schema.search', { name: (collection.plural || collection.label || type).toLowerCase() })}
            bind:value={search}
            class="h-8 w-full rounded-md border border-input bg-background pl-8 pr-3 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
          />
        </div>

        {#if manageSelectField}
          <button
            type="button"
            class="inline-flex items-center h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium shadow-xs cursor-pointer shrink-0"
            onclick={() => { optionsModalOpen = true; }}
          >{getFieldKey(manageSelectField) === 'category' ? t('schema.manageCategories') : t('schema.manageOptions')}</button>
        {/if}

        {#if bulkEnabled}
          <a
            href="{csvBase}/_export"
            download
            class="inline-flex items-center h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium shadow-xs shrink-0"
          >{t('schema.exportCsv')}</a>
          <button
            type="button"
            class="inline-flex items-center h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium shadow-xs cursor-pointer shrink-0"
            onclick={() => importInput?.click()}
          >{t('schema.importCsv')}</button>
          <input bind:this={importInput} type="file" accept=".csv,text/csv" class="hidden" onchange={importCsv} />
        {/if}

        <button
          type="button"
          class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs cursor-pointer shrink-0 transition-colors"
          onclick={openCreate}
        >
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          <span>{t('schema.add', { name: collection.singular || collection.label || t('schema.item') })}</span>
        </button>
      </div>
    </div>

    {#if bulkEnabled && selectedIds.length > 0}
      <div class="flex items-center justify-between gap-3 rounded-lg border border-border bg-muted/50 px-4 py-2 text-xs">
        <span>{t('schema.selected', { count: selectedIds.length })}</span>
        <div class="flex items-center gap-2">
          <button type="button" class="h-7 px-2.5 rounded-md border border-input bg-background hover:bg-accent cursor-pointer" onclick={() => { selected = {}; }}>{t('schema.clearSelection')}</button>
          <button type="button" class="h-7 px-2.5 rounded-md bg-destructive text-white hover:bg-destructive/90 cursor-pointer" onclick={() => { bulkDeleteOpen = true; }}>{t('schema.bulkDelete')}</button>
        </div>
      </div>
    {/if}
    {#if importMessage}
      <p class="text-xs {importFailed ? 'text-destructive' : 'text-muted-foreground'}" role={importFailed ? 'alert' : 'status'}>{importMessage}</p>
    {/if}
    {#if bulkError}
      <p class="text-xs text-destructive" role="alert">{bulkError}</p>
    {/if}

    <!-- Table Container -->
    <div class="overflow-hidden rounded-xl border border-border bg-card shadow-xs">
      {#if loading}
        <div class="p-12 text-center text-xs text-muted-foreground">
          {t('schema.loadingItems', { name: (collection.plural || collection.label || type).toLowerCase() })}
        </div>
      {:else if filteredItems.length === 0}
        <div class="p-12 text-center text-xs text-muted-foreground flex flex-col items-center justify-center gap-3">
          <p>{t('schema.empty')}</p>
          <button
            type="button"
            class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer"
            onclick={openCreate}
          >{t('schema.createFirst', { name: collection.singular || collection.label || t('schema.item') })}</button>
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-left text-sm text-foreground caption-bottom">
            <thead class="sticky top-0 z-10 bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
              <tr>
                {#if bulkEnabled}
                  <th class="h-10 w-10 px-4"><input type="checkbox" checked={allShownSelected} onchange={toggleAll} aria-label={t('schema.selectAll')} /></th>
                {/if}
                {#each listFields as field (getFieldKey(field))}
                  {@const key = getFieldKey(field)}
                  <th class="h-10 px-4 font-medium">
                    {#if canSortField(field)}
                      <button
                        type="button"
                        class="inline-flex items-center gap-1 hover:text-foreground cursor-pointer"
                        onclick={() => toggleSort(key)}
                      >
                        <span>{field.label || key}</span>
                        {#if sortField === key}
                          <span>↑</span>
                        {:else if sortField === `-${key}`}
                          <span>↓</span>
                        {/if}
                      </button>
                    {:else}
                      {field.label || key}
                    {/if}
                  </th>
                {/each}
                {#if hasStatus}
                  <th class="h-10 px-4 font-medium">{t('common.status')}</th>
                {/if}
                <th class="h-10 w-48 px-4 font-medium text-right">{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              {#each filteredItems as item (item.id)}
                <tr
                  class="hover:bg-muted/50 transition-colors cursor-pointer"
                  onclick={() => openEdit(item.id)}
                >
                  {#if bulkEnabled}
                    <td class="w-10 px-4 py-3" onclick={(e) => e.stopPropagation()}>
                      <input type="checkbox" bind:checked={selected[item.id]} aria-label={t('schema.selectRow')} />
                    </td>
                  {/if}
                  {#each listFields as field (getFieldKey(field))}
                    {@const key = getFieldKey(field)}
                    <td class="px-4 py-3 text-xs text-foreground max-w-xs truncate">
                      {#if field.kind === 'boolean'}
                        <Badge variant={item[key] ? 'success' : 'outline'} size="sm">
                          {item[key] ? t('common.yes') : t('common.no')}
                        </Badge>
                      {:else if field.kind === 'media' || field.widget === 'media'}
                        {#if isVideoUrl(item[key])}
                          <span class="inline-flex items-center px-1.5 py-0.5 rounded border border-border bg-muted/50 font-mono text-[10px]">MP4</span>
                        {:else if item[key]}
                          <div class="size-8 rounded border border-border bg-muted overflow-hidden inline-flex items-center justify-center">
                            <img
                              src={String(item[key])}
                              alt="thumb"
                              class="size-full object-cover"
                              onerror={(e) => { (e.currentTarget as HTMLElement).style.display = 'none'; }}
                            />
                          </div>
                        {:else}
                          <span class="font-mono text-muted-foreground">—</span>
                        {/if}
                      {:else if isRelation(field)}
                        <span class="font-medium text-foreground">
                          {relationCell(
                            item,
                            field,
                            (name) => schema.getCollection(name)?.title_field,
                            (target, k) => slugLookup[target]?.[k]
                          )}
                        </span>
                      {:else if key === 'order_id' && item[key] && schema.modules.commerce}
                        <span class="inline-flex items-center gap-2">
                          <span class="font-mono">{String(item[key])}</span>
                          <button
                            type="button"
                            class="text-[11px] font-medium text-primary hover:underline cursor-pointer"
                            onclick={(e) => {
                              e.stopPropagation();
                              router.navigate(`/commerce?order=${encodeURIComponent(String(item[key]))}`);
                            }}
                          >{t('schema.openOrder')} →</button>
                        </span>
                      {:else}
                        <span class="font-medium text-foreground">
                          {item[key] === undefined || item[key] === null || item[key] === '' ? '—' : String(item[key])}
                        </span>
                      {/if}
                    </td>
                  {/each}

                  {#if hasStatus}
                    <td class="px-4 py-3">
                      <Badge
                        variant={item.status === 'published' ? 'success' : item.status === 'archived' ? 'destructive' : 'warning'}
                        size="sm"
                      >{statusLabel(item.status || 'draft')}</Badge>
                    </td>
                  {/if}

                  <td class="px-4 py-3 text-right whitespace-nowrap">
                    <div class="inline-flex items-center gap-1">
                      {#if usesDrafts(collection)}
                        <button
                          type="button"
                          disabled={actingId === String(item.id)}
                          class="h-7 px-2.5 whitespace-nowrap rounded-md border border-input bg-background text-xs font-medium hover:bg-accent transition-colors cursor-pointer disabled:opacity-50"
                          onclick={(e) => togglePublication(item, e)}
                        >{item.status === 'published' ? t('schema.unpublish') : t('pages.editor.publish')}</button>
                      {/if}
                      <button
                        type="button"
                        class="size-7 inline-flex items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground transition-colors cursor-pointer"
                        onclick={(e) => openEdit(item.id, e)}
                        title={t('common.edit')}
                      >
                        <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9"></path><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"></path></svg>
                      </button>
                      <button
                        type="button"
                        class="size-7 inline-flex items-center justify-center rounded-md text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors cursor-pointer"
                        onclick={(e) => handleDelete(item.id, e)}
                        title={t('common.delete')}
                      >
                        <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18"></path><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6m3 0V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"></path></svg>
                      </button>
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        <div class="flex items-center justify-between px-4 py-3 border-t border-border text-xs text-muted-foreground">
          <span class="tabular-nums">{filteredItems.length === 0 ? t('schema.showing', { shown: 0, total: 0 }) : t('schema.showingRange', { from: (page - 1) * PAGE_SIZE + 1, to: (page - 1) * PAGE_SIZE + filteredItems.length })}</span>
          <div class="flex items-center gap-2">
            <button
              type="button"
              disabled={page <= 1}
              onclick={() => { page--; loadItems(); }}
              class="h-7 px-2.5 rounded-md border border-input bg-background text-xs font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:bg-accent transition-colors cursor-pointer"
            >{t('common.previous')}</button>
            <button
              type="button"
              disabled={!hasMore}
              onclick={() => { page++; loadItems(); }}
              class="h-7 px-2.5 rounded-md border border-input bg-background text-xs font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:bg-accent transition-colors cursor-pointer"
            >{t('common.next')}</button>
          </div>
        </div>
      {/if}
    </div>
  </div>
{/if}
