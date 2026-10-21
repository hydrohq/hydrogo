<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { importCsvFile, importErrorMessage } from './csvImport';
  import { schema } from '$lib/stores/schema.svelte';
  import { router } from '$lib/router.svelte';
  import type { ContentItem, ApiResponse, Field } from '$lib/api/types';
  import Badge from '$lib/components/ui/Badge.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
  import GenericItemModal from './GenericItemModal.svelte';

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
  let total = $state(0);
  let statusFilter = $state('');

  // Item Modal State (Create / Edit)
  let modalOpen = $state(false);
  let activeItemId = $state<string | null>(null);

  // Delete Dialog State
  let deleteTargetId = $state<string | null>(null);
  let deleteLoading = $state(false);

  const listFields = $derived.by(() => {
    if (!collection?.fields) return [];
    const explicit = collection.fields.filter(f => f.visible_in_list || f.list);
    if (explicit.length > 0) return explicit;
    return collection.fields.slice(0, 4);
  });

  function getFieldKey(f: Field): string {
    return f.id || f.name || '';
  }

  const filteredItems = $derived.by(() => {
    let list = items;
    if (statusFilter && collection?.workflow?.enabled) {
      list = list.filter((i) => i.status === statusFilter);
    }
    if (search.trim()) {
      const q = search.toLowerCase();
      list = list.filter((i) => {
        return Object.values(i).some((v) => String(v).toLowerCase().includes(q));
      });
    }
    return list;
  });

  // Bulk selection. Content types delete the selection in one atomic batch
  // request; other resources have no batch endpoint yet.
  const bulkEnabled = $derived(!!collection && collection.api_path?.startsWith('/api/v1/admin/content/'));
  let selected = $state<Record<string, boolean>>({});
  const selectedIds = $derived(Object.keys(selected).filter((id) => selected[id]));
  let bulkDeleteOpen = $state(false);
  let bulkLoading = $state(false);
  let bulkError = $state('');

  const allShownSelected = $derived(filteredItems.length > 0 && filteredItems.every((item) => selected[item.id]));

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
      // The batch is atomic: nothing was deleted. Show why, with the request id.
      bulkError = err instanceof ApiClientError
        ? `${err.message}${err.requestId ? ` (${t('schema.requestId')}: ${err.requestId})` : ''}`
        : t('schema.bulkFailed');
      bulkDeleteOpen = false;
    } finally {
      bulkLoading = false;
    }
  }

  // CSV transfer (content types only): export downloads the list through the
  // browser session; import sends the file as the request body.
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

  async function loadItems() {
    if (!collection) return;
    loading = true;
    try {
      const res = await api.get<ApiResponse<ContentItem[]>>(collection.api_path || `/api/v1/admin/content/${type}`, {
        params: {
          page,
          per_page: 50,
        },
      });
      if (res && res.data) {
        items = Array.isArray(res.data) ? res.data : [];
        total = res.metadata?.total ?? res.meta?.total ?? items.length;
      }
    } catch (e) {
      console.warn('Failed to load items:', e);
      items = [];
    } finally {
      loading = false;
    }
  }

  function openCreate() {
    activeItemId = 'new';
    modalOpen = true;
  }

  function openEdit(id: string, e?: MouseEvent) {
    if (e) e.stopPropagation();
    activeItemId = id;
    modalOpen = true;
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
      await api.delete(`${collection?.api_path || `/api/v1/admin/content/${type}`}/${deleteTargetId}`);
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
      page = 1;
      statusFilter = '';
      search = '';
      selected = {};
      bulkError = '';
      loadItems();
    }
  });

  $effect(() => {
    if (initialItemId) {
      activeItemId = initialItemId;
      modalOpen = true;
    }
  });
</script>

<!-- Create / Edit Modal -->
<GenericItemModal
  open={modalOpen}
  {type}
  itemId={activeItemId}
  onclose={closeModal}
  onsaved={loadItems}
/>

<!-- Delete Confirm Dialog -->
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
    >
      {t('schema.backToDashboard')}
    </button>
  </div>
{:else}
  <div class="flex flex-1 flex-col">
    <div class="@container/main flex flex-1 flex-col gap-2">
      <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
        <!-- Toolbar (exact shadcn Tabs + Search + CTA layout) -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 px-4 lg:px-6">
          <!-- Segmented Status Tabs if workflow enabled -->
          {#if collection.workflow?.enabled}
            <div class="inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground">
              <button
                type="button"
                class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {statusFilter === '' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
                onclick={() => { statusFilter = ''; }}
              >
                {t('common.all')}
              </button>
              <button
                type="button"
                class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {statusFilter === 'published' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
                onclick={() => { statusFilter = 'published'; }}
              >
                {t('common.published')}
              </button>
              <button
                type="button"
                class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {statusFilter === 'draft' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
                onclick={() => { statusFilter = 'draft'; }}
              >
                {t('news.list.drafts')}
              </button>
            </div>
          {:else}
            <div class="text-sm font-medium text-muted-foreground">
              {collection.plural || collection.label || type}
            </div>
          {/if}

          <!-- Search & Add CTA -->
          <div class="flex flex-wrap items-center gap-2">
            <div class="relative w-full sm:w-64">
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
          <div class="mx-4 lg:mx-6 flex items-center justify-between gap-3 rounded-lg border border-border bg-muted/50 px-4 py-2 text-xs">
            <span>{t('schema.selected', { count: selectedIds.length })}</span>
            <div class="flex items-center gap-2">
              <button type="button" class="h-7 px-2.5 rounded-md border border-input bg-background hover:bg-accent cursor-pointer" onclick={() => { selected = {}; }}>{t('schema.clearSelection')}</button>
              <button type="button" class="h-7 px-2.5 rounded-md bg-destructive text-white hover:bg-destructive/90 cursor-pointer" onclick={() => { bulkDeleteOpen = true; }}>{t('schema.bulkDelete')}</button>
            </div>
          </div>
        {/if}
        {#if importMessage}
          <p class="mx-4 lg:mx-6 text-xs {importFailed ? 'text-destructive' : 'text-muted-foreground'}" role={importFailed ? 'alert' : 'status'}>{importMessage}</p>
        {/if}
        {#if bulkError}
          <p class="mx-4 lg:mx-6 text-xs text-destructive" role="alert">{bulkError}</p>
        {/if}

        <!-- Table Container -->
        <div class="px-4 lg:px-6">
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
                >
                  {t('schema.createFirst', { name: collection.singular || collection.label || t('schema.item') })}
                </button>
              </div>
            {:else}
              <div class="overflow-x-auto">
                <table class="w-full text-left text-sm text-foreground caption-bottom">
                  <thead class="sticky top-0 z-10 bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
                    <tr>
                      {#if bulkEnabled}
                        <th class="h-10 w-10 px-4"><input type="checkbox" checked={allShownSelected} onchange={toggleAll} aria-label={t('schema.selectAll')} /></th>
                      {/if}
                      {#each listFields as field}
                        <th class="h-10 px-4 font-medium">{field.label || getFieldKey(field)}</th>
                      {/each}
                      {#if collection.workflow?.enabled}
                        <th class="h-10 px-4 font-medium">{t('common.status')}</th>
                      {/if}
                      <th class="h-10 w-24 px-4 font-medium text-right">{t('common.actions')}</th>
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
                        {#each listFields as field}
                          {@const key = getFieldKey(field)}
                          <td class="px-4 py-3 text-xs text-foreground max-w-xs truncate">
                            {#if field.kind === 'boolean'}
                              <Badge variant={item[key] ? 'success' : 'outline'} size="sm">
                                {item[key] ? t('common.yes') : t('common.no')}
                              </Badge>
                            {:else if field.kind === 'media'}
                              {#if item[key]}
                                <div class="size-7 rounded border border-border bg-muted overflow-hidden inline-flex items-center justify-center">
                                  <img
                                    src={String(item[key])}
                                    alt="thumb"
                                    class="size-full object-cover"
                                    onerror={(e) => { (e.target as HTMLElement).style.display = 'none'; }}
                                  />
                                </div>
                              {:else}
                                <span class="font-mono text-muted-foreground">—</span>
                              {/if}
                            {:else}
                              <span class="font-medium text-foreground">
                                {String(item[key] ?? '—')}
                              </span>
                            {/if}
                          </td>
                        {/each}

                        {#if collection.workflow?.enabled}
                          <td class="px-4 py-3">
                            <Badge
                              variant={item.status === 'published' ? 'success' : item.status === 'archived' ? 'destructive' : 'outline'}
                              size="sm"
                            >
                              {item.status || 'draft'}
                            </Badge>
                          </td>
                        {/if}

                        <td class="px-4 py-3 text-right">
                          <div class="inline-flex items-center gap-1">
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
                              <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18"></path><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"></path><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"></path></svg>
                            </button>
                          </div>
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>

              <!-- Footer Pagination / Stats -->
              <div class="flex items-center justify-between px-4 py-3 border-t border-border text-xs text-muted-foreground">
                <span>{t('schema.showing', { shown: filteredItems.length, total: total || items.length })}</span>
                <div class="flex items-center gap-2">
                  <button
                    type="button"
                    disabled={page <= 1}
                    onclick={() => { page--; loadItems(); }}
                    class="h-7 px-2.5 rounded-md border border-input bg-background text-xs font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:bg-accent transition-colors cursor-pointer"
                  >
                    {t('common.previous')}
                  </button>
                  <button
                    type="button"
                    disabled={items.length < 50}
                    onclick={() => { page++; loadItems(); }}
                    class="h-7 px-2.5 rounded-md border border-input bg-background text-xs font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:bg-accent transition-colors cursor-pointer"
                  >
                    {t('common.next')}
                  </button>
                </div>
              </div>
            {/if}
          </div>
        </div>
      </div>
    </div>
  </div>
{/if}
