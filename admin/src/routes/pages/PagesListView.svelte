<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { router } from '$lib/router.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Badge from '$lib/components/ui/Badge.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';

  interface PageRow {
    id: string;
    slug: string;
    title: string;
    locale: string;
    status: 'draft' | 'published';
    layout_template?: string;
    version: number;
    updated_at: string;
  }

  let pages = $state<PageRow[]>([]);
  let loading = $state(true);
  let search = $state('');
  let statusFilter = $state('');

  const filteredPages = $derived.by(() => {
    let list = pages;
    if (statusFilter) {
      list = list.filter((p) => p.status === statusFilter);
    }
    if (search.trim()) {
      const q = search.toLowerCase();
      list = list.filter((p) => p.title.toLowerCase().includes(q) || p.slug.toLowerCase().includes(q));
    }
    return list;
  });

  async function loadPages() {
    loading = true;
    try {
      const res = await api.get<ApiResponse<PageRow[]>>('/api/v1/admin/pages', {
        params: { limit: 100 },
      });
      if (res && res.data) {
        pages = Array.isArray(res.data) ? res.data : [];
      }
    } catch (e) {
      console.warn('Pages load error:', e);
      pages = [];
    } finally {
      loading = false;
    }
  }

  let deletePageTarget = $state<{ id: string; title: string } | null>(null);
  let deleteLoading = $state(false);

  function handleDelete(id: string, title: string, e: MouseEvent) {
    e.stopPropagation();
    deletePageTarget = { id, title };
  }

  async function confirmDeletePage() {
    if (!deletePageTarget) return;
    deleteLoading = true;
    try {
      await api.delete(`/api/v1/admin/pages/${deletePageTarget.id}`);
      deletePageTarget = null;
      await loadPages();
    } catch (err: any) {
      console.error('Delete failed:', err);
    } finally {
      deleteLoading = false;
    }
  }

  onMount(() => {
    loadPages();
  });
</script>

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-2">
    <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <!-- Toolbar (exact shadcn Tabs + Search + CTA layout) -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 px-4 lg:px-6">
        <!-- Segmented Status Tabs -->
        <div class="inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground">
          <button
            type="button"
            class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {statusFilter === '' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
            onclick={() => { statusFilter = ''; }}
          >
            All Pages
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

        <!-- Search & New Page CTA -->
        <div class="flex items-center gap-2">
          <div class="relative w-48 sm:w-64">
            <svg class="absolute left-2.5 top-2.5 size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"></circle>
              <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
            </svg>
            <input
              type="search"
              placeholder="Filter pages…"
              bind:value={search}
              class="h-8 w-full rounded-md border border-input bg-background pl-8 pr-3 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
            />
          </div>

          <button
            type="button"
            class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs cursor-pointer shrink-0 transition-colors"
            onclick={() => router.navigate('/pages/new')}
          >
            <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="12" y1="5" x2="12" y2="19"></line>
              <line x1="5" y1="12" x2="19" y2="12"></line>
            </svg>
            <span>Add Page</span>
          </button>
        </div>
      </div>

      <!-- Table Container -->
      <div class="px-4 lg:px-6">
        <div class="overflow-hidden rounded-xl border border-border bg-card shadow-xs">
          {#if loading}
            <div class="p-12 text-center text-xs text-muted-foreground">
              Loading pages…
            </div>
          {:else if filteredPages.length === 0}
            <div class="p-12 text-center text-xs text-muted-foreground flex flex-col items-center justify-center gap-3">
              <p>No pages match the criteria.</p>
              <button
                type="button"
                class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer"
                onclick={() => router.navigate('/pages/new')}
              >
                Create First Page
              </button>
            </div>
          {:else}
            <div class="overflow-x-auto">
              <table class="w-full text-left text-sm text-foreground caption-bottom">
                <thead class="sticky top-0 z-10 bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
                  <tr>
                    <th class="h-10 px-4 font-medium">Title</th>
                    <th class="h-10 px-4 font-medium">Path / Slug</th>
                    <th class="h-10 px-4 font-medium">Template</th>
                    <th class="h-10 px-4 font-medium">Status</th>
                    <th class="h-10 px-4 font-medium">Updated</th>
                    <th class="h-10 w-24 px-4 font-medium text-right">Actions</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-border">
                  {#each filteredPages as page (page.id)}
                    <tr
                      class="hover:bg-muted/50 transition-colors cursor-pointer"
                      onclick={() => router.navigate(`/pages/${page.id}`)}
                    >
                      <td class="px-4 py-3 font-medium text-foreground">
                        {page.title}
                      </td>
                      <td class="px-4 py-3 font-mono text-xs text-muted-foreground">
                        /{page.slug}
                      </td>
                      <td class="px-4 py-3 text-xs text-muted-foreground">
                        {page.layout_template || 'default'}
                      </td>
                      <td class="px-4 py-3">
                        <Badge variant={page.status === 'published' ? 'success' : 'outline'} size="sm">
                          {page.status}
                        </Badge>
                      </td>
                      <td class="px-4 py-3 text-xs text-muted-foreground">
                        {new Date(page.updated_at).toLocaleDateString()}
                      </td>
                      <td class="px-4 py-3 text-right">
                        <div class="inline-flex items-center gap-1">
                          <button
                            type="button"
                            class="size-7 inline-flex items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
                            onclick={(e) => { e.stopPropagation(); router.navigate(`/pages/${page.id}`); }}
                            title="Edit"
                          >
                            <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9"></path><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"></path></svg>
                          </button>
                          <button
                            type="button"
                            class="size-7 inline-flex items-center justify-center rounded-md text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors"
                            onclick={(e) => handleDelete(page.id, page.title, e)}
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
              <span>Showing {filteredPages.length} of {pages.length} pages</span>
              <div class="flex items-center gap-2">
                <button
                  type="button"
                  disabled
                  class="h-7 px-2.5 rounded-md border border-input bg-background text-xs font-medium opacity-50 cursor-not-allowed"
                >
                  Previous
                </button>
                <button
                  type="button"
                  disabled
                  class="h-7 px-2.5 rounded-md border border-input bg-background text-xs font-medium opacity-50 cursor-not-allowed"
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

<ConfirmDialog
  open={deletePageTarget !== null}
  title={`Delete Page "${deletePageTarget?.title || ''}"`}
  description="Are you sure you want to permanently delete this page? This action cannot be undone."
  loading={deleteLoading}
  onconfirm={confirmDeletePage}
  oncancel={() => { deletePageTarget = null; }}
/>
