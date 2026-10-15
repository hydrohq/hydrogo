<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { router } from '$lib/router.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';

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

  async function loadPages() {
    loading = true;
    try {
      const res = await api.get<ApiResponse<PageRow[]>>('/api/v1/admin/pages', {
        params: {
          search: search || undefined,
          status: statusFilter || undefined,
          limit: 100,
        },
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

  async function handleDelete(id: string, title: string, e: MouseEvent) {
    e.stopPropagation();
    if (!confirm(`Delete page "${title}"?`)) return;
    try {
      await api.delete(`/api/v1/admin/pages/${id}`);
      await loadPages();
    } catch (err: any) {
      alert(`Delete failed: ${err.message}`);
    }
  }

  onMount(() => {
    loadPages();
  });
</script>

<div class="space-y-6 max-w-6xl mx-auto">
  <!-- Header -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 pb-4 border-b border-border">
    <div>
      <h2 class="text-xl font-bold tracking-tight text-foreground flex items-center gap-2">
        <span>Pages</span>
      </h2>
      <p class="text-xs text-muted-foreground mt-1">
        Design landing pages and structured content layouts with dynamic blocks.
      </p>
    </div>

    <Button
      variant="primary"
      size="md"
      onclick={() => router.navigate('/pages/new')}
      class="gap-1.5 shadow-sm"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="12" y1="5" x2="12" y2="19"></line>
        <line x1="5" y1="12" x2="19" y2="12"></line>
      </svg>
      <span>New Page</span>
    </Button>
  </div>

  <!-- Filters & Search -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
    <div class="flex items-center rounded-lg bg-muted p-1 text-xs border border-border">
      <button
        type="button"
        class="px-3 py-1.5 rounded-md transition-all font-medium {statusFilter === '' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { statusFilter = ''; loadPages(); }}
      >
        All
      </button>
      <button
        type="button"
        class="px-3 py-1.5 rounded-md transition-all font-medium {statusFilter === 'published' ? 'bg-background text-emerald-400 shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { statusFilter = 'published'; loadPages(); }}
      >
        Published
      </button>
      <button
        type="button"
        class="px-3 py-1.5 rounded-md transition-all font-medium {statusFilter === 'draft' ? 'bg-background text-amber-400 shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { statusFilter = 'draft'; loadPages(); }}
      >
        Drafts
      </button>
    </div>

    <input
      type="search"
      placeholder="Search pages…"
      bind:value={search}
      oninput={() => loadPages()}
      class="bg-background border border-input rounded-md px-3.5 py-1.5 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring w-full sm:w-64 shadow-sm"
    />
  </div>

  <!-- Pages Table -->
  <div class="rounded-xl border border-border bg-card overflow-hidden shadow-sm">
    {#if loading}
      <div class="py-16 text-center text-muted-foreground text-sm">
        <span class="animate-pulse inline-block mr-2">⏳</span> Loading pages…
      </div>
    {:else if pages.length === 0}
      <div class="py-16 text-center text-muted-foreground text-sm flex flex-col items-center justify-center gap-3">
        <p>No pages found.</p>
        <Button variant="secondary" size="sm" onclick={() => router.navigate('/pages/new')}>
          Create First Page
        </Button>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm text-foreground">
          <thead class="bg-muted/40 text-[11px] uppercase tracking-wider text-muted-foreground border-b border-border">
            <tr>
              <th class="px-5 py-3 font-semibold">Title</th>
              <th class="px-5 py-3 font-semibold">Path / Slug</th>
              <th class="px-5 py-3 font-semibold">Template</th>
              <th class="px-5 py-3 font-semibold">Status</th>
              <th class="px-5 py-3 font-semibold">Last Updated</th>
              <th class="px-5 py-3 font-semibold text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border/60">
            {#each pages as page (page.id)}
              <tr
                class="hover:bg-muted/30 transition-colors cursor-pointer"
                onclick={() => router.navigate(`/pages/${page.id}`)}
              >
                <td class="px-5 py-3.5 font-semibold text-foreground">
                  {page.title}
                </td>
                <td class="px-5 py-3.5 font-mono text-xs text-muted-foreground">
                  /{page.slug}
                </td>
                <td class="px-5 py-3.5 text-xs text-muted-foreground">
                  {page.layout_template || 'default'}
                </td>
                <td class="px-5 py-3.5">
                  <Badge variant={page.status === 'published' ? 'success' : 'warning'} size="sm">
                    {page.status}
                  </Badge>
                </td>
                <td class="px-5 py-3.5 text-xs text-muted-foreground">
                  {new Date(page.updated_at).toLocaleDateString()}
                </td>
                <td class="px-5 py-3.5 text-right whitespace-nowrap">
                  <div class="inline-flex items-center gap-1.5">
                    <button
                      type="button"
                      class="px-2.5 py-1 text-xs text-muted-foreground hover:text-foreground rounded hover:bg-muted transition-colors cursor-pointer"
                      onclick={(e) => { e.stopPropagation(); router.navigate(`/pages/${page.id}`); }}
                    >
                      Edit
                    </button>
                    <button
                      type="button"
                      class="px-2.5 py-1 text-xs text-rose-400 hover:text-rose-300 rounded hover:bg-destructive/15 transition-colors cursor-pointer"
                      onclick={(e) => handleDelete(page.id, page.title, e)}
                    >
                      Delete
                    </button>
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>
