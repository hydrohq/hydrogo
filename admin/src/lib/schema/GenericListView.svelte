<script lang="ts">
  import { api } from '$lib/api/client';
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
        total = res.meta?.total || items.length;
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
  title={`Delete ${collection?.singular || collection?.label || 'Item'}`}
  description="Are you sure you want to permanently delete this item? This action cannot be undone."
  loading={deleteLoading}
  onconfirm={confirmDelete}
  oncancel={() => { deleteTargetId = null; }}
/>

{#if schema.isLoading}
  <div class="p-12 text-center text-xs text-muted-foreground flex flex-col items-center justify-center gap-2">
    <div class="size-6 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
    <span>Loading {type} schemaâ€¦</span>
  </div>
{:else if !collection}
  <div class="p-12 text-center text-xs text-muted-foreground">
    <p>Content type <strong>{type}</strong> not found in schema.</p>
    <button
      type="button"
      class="mt-4 inline-flex items-center gap-1.5 h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer"
      onclick={() => router.navigate('/dashboard')}
    >
      Back to Dashboard
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
                All
              </button>
              <button
                type="button"
                class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {statusFilter === 'published' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
                onclick={() => { statusFilter = 'published'; }}
              >
                Published
              </button>
              <button
                type="button"
                class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {statusFilter === 'draft' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
                onclick={() => { statusFilter = 'draft'; }}
              >
                Drafts
              </button>
            </div>
          {:else}
            <div class="text-sm font-medium text-muted-foreground">
              {collection.plural || collection.label || type}
            </div>
          {/if}

          <!-- Search & Add CTA -->
          <div class="flex items-center gap-2">
            <div class="relative w-48 sm:w-64">
              <svg class="absolute left-2.5 top-2.5 size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"></circle>
                <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
              </svg>
              <input
                type="search"
                placeholder="Search {(collection.plural || collection.label || type).toLowerCase()}â€¦"
                bind:value={search}
                class="h-8 w-full rounded-md border border-input bg-background pl-8 pr-3 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
              />
            </div>

            <button
              type="button"
              class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs cursor-pointer shrink-0 transition-colors"
              onclick={openCreate}
            >
              <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="12" y1="5" x2="12" y2="19"></line>
                <line x1="5" y1="12" x2="19" y2="12"></line>
              </svg>
              <span>Add {collection.singular || collection.label || 'Item'}</span>
            </button>
          </div>
        </div>

        <!-- Table Container -->
        <div class="px-4 lg:px-6">
          <div class="overflow-hidden rounded-xl border border-border bg-card shadow-xs">
            {#if loading}
              <div class="p-12 text-center text-xs text-muted-foreground">
                Loading {(collection.plural || collection.label || type).toLowerCase()}â€¦
              </div>
            {:else if filteredItems.length === 0}
              <div class="p-12 text-center text-xs text-muted-foreground flex flex-col items-center justify-center gap-3">
                <p>No entries found.</p>
                <button
                  type="button"
                  class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer"
                  onclick={openCreate}
                >
                  Create First {collection.singular || collection.label || 'Item'}
                </button>
              </div>
            {:else}
              <div class="overflow-x-auto">
                <table class="w-full text-left text-sm text-foreground caption-bottom">
                  <thead class="sticky top-0 z-10 bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
                    <tr>
                      {#each listFields as field}
                        <th class="h-10 px-4 font-medium">{field.label || getFieldKey(field)}</th>
                      {/each}
                      {#if collection.workflow?.enabled}
                        <th class="h-10 px-4 font-medium">Status</th>
                      {/if}
                      <th class="h-10 w-24 px-4 font-medium text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-border">
                    {#each filteredItems as item (item.id)}
                      <tr
                        class="hover:bg-muted/50 transition-colors cursor-pointer"
                        onclick={() => openEdit(item.id)}
                      >
                        {#each listFields as field}
                          {@const key = getFieldKey(field)}
                          <td class="px-4 py-3 text-xs text-foreground max-w-xs truncate">
                            {#if field.kind === 'boolean'}
                              <Badge variant={item[key] ? 'success' : 'outline'} size="sm">
                                {item[key] ? 'Yes' : 'No'}
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
                                <span class="font-mono text-muted-foreground">â€”</span>
                              {/if}
                            {:else}
                              <span class="font-medium text-foreground">
                                {String(item[key] ?? 'â€”')}
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
                              title="Edit"
                            >
                              <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9"></path><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"></path></svg>
                            </button>
                            <button
                              type="button"
                              class="size-7 inline-flex items-center justify-center rounded-md text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors cursor-pointer"
                              onclick={(e) => handleDelete(item.id, e)}
                              title="Delete"
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
                <span>Showing {filteredItems.length} of {total || items.length} entries</span>
                <div class="flex items-center gap-2">
                  <button
                    type="button"
                    disabled={page <= 1}
                    onclick={() => { page--; loadItems(); }}
                    class="h-7 px-2.5 rounded-md border border-input bg-background text-xs font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:bg-accent transition-colors cursor-pointer"
                  >
                    Previous
                  </button>
                  <button
                    type="button"
                    disabled={items.length < 50}
                    onclick={() => { page++; loadItems(); }}
                    class="h-7 px-2.5 rounded-md border border-input bg-background text-xs font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:bg-accent transition-colors cursor-pointer"
                  >
                    Next
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
