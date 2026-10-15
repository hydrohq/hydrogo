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
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 pb-4 border-b border-slate-800">
    <div>
      <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2">
        <span>📑</span>
        <span>Pages</span>
      </h2>
      <p class="text-xs text-slate-400 mt-1">
        Design landing pages and structured content layouts with dynamic blocks.
      </p>
    </div>

    <Button
      variant="primary"
      size="md"
      onclick={() => router.navigate('/pages/new')}
    >
      <span>+</span>
      <span>New Page</span>
    </Button>
  </div>

  <!-- Filters -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
    <div class="flex items-center rounded-xl bg-slate-900 border border-slate-800 p-1 text-xs">
      <button
        type="button"
        class="px-3 py-1.5 rounded-lg transition-colors {statusFilter === '' ? 'bg-slate-800 text-white font-semibold' : 'text-slate-400 hover:text-slate-200'}"
        onclick={() => { statusFilter = ''; loadPages(); }}
      >
        All
      </button>
      <button
        type="button"
        class="px-3 py-1.5 rounded-lg transition-colors {statusFilter === 'published' ? 'bg-slate-800 text-emerald-400 font-semibold' : 'text-slate-400 hover:text-slate-200'}"
        onclick={() => { statusFilter = 'published'; loadPages(); }}
      >
        Published
      </button>
      <button
        type="button"
        class="px-3 py-1.5 rounded-lg transition-colors {statusFilter === 'draft' ? 'bg-slate-800 text-amber-400 font-semibold' : 'text-slate-400 hover:text-slate-200'}"
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
      class="bg-slate-900 border border-slate-800 rounded-lg px-3.5 py-1.5 text-xs text-slate-100 placeholder:text-slate-500 focus:outline-none focus:border-sky-500 w-full sm:w-64"
    />
  </div>

  <!-- Pages Table -->
  <div class="rounded-xl border border-slate-800 bg-slate-900/60 overflow-hidden shadow-sm">
    {#if loading}
      <div class="py-16 text-center text-slate-500 text-sm">
        <span class="animate-spin inline-block mr-2">⏳</span> Loading pages…
      </div>
    {:else if pages.length === 0}
      <div class="py-16 text-center text-slate-400 text-sm flex flex-col items-center justify-center gap-3">
        <p>No pages found.</p>
        <Button variant="secondary" size="sm" onclick={() => router.navigate('/pages/new')}>
          Create First Page
        </Button>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm text-slate-300">
          <thead class="bg-slate-950/80 text-[11px] uppercase tracking-wider text-slate-400 border-b border-slate-800">
            <tr>
              <th class="px-5 py-3 font-semibold">Title</th>
              <th class="px-5 py-3 font-semibold">Path / Slug</th>
              <th class="px-5 py-3 font-semibold">Template</th>
              <th class="px-5 py-3 font-semibold">Status</th>
              <th class="px-5 py-3 font-semibold">Last Updated</th>
              <th class="px-5 py-3 font-semibold text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/60">
            {#each pages as page (page.id)}
              <tr
                class="hover:bg-slate-800/40 transition-colors cursor-pointer"
                onclick={() => router.navigate(`/pages/${page.id}`)}
              >
                <td class="px-5 py-3.5 font-semibold text-slate-100">
                  {page.title}
                </td>
                <td class="px-5 py-3.5 font-mono text-xs text-slate-400">
                  /{page.slug}
                </td>
                <td class="px-5 py-3.5 text-xs text-slate-400">
                  {page.layout_template || 'default'}
                </td>
                <td class="px-5 py-3.5">
                  <Badge variant={page.status === 'published' ? 'success' : 'warning'} size="sm">
                    {page.status}
                  </Badge>
                </td>
                <td class="px-5 py-3.5 text-xs text-slate-500">
                  {new Date(page.updated_at).toLocaleDateString()}
                </td>
                <td class="px-5 py-3.5 text-right whitespace-nowrap">
                  <div class="inline-flex items-center gap-1.5">
                    <button
                      type="button"
                      class="px-2.5 py-1 text-xs text-slate-300 hover:text-white rounded hover:bg-slate-800 transition-colors"
                      onclick={(e) => { e.stopPropagation(); router.navigate(`/pages/${page.id}`); }}
                    >
                      Edit
                    </button>
                    <button
                      type="button"
                      class="px-2.5 py-1 text-xs text-rose-400 hover:text-rose-300 rounded hover:bg-rose-950/40 transition-colors"
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
