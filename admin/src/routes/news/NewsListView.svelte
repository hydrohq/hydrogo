<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { router } from '$lib/router.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import NewsTaxonomyModal from './NewsTaxonomyModal.svelte';

  interface NewsRow {
    id: string;
    slug: string;
    title: string;
    category?: string;
    author_name?: string;
    featured?: boolean;
    status: 'draft' | 'scheduled' | 'published' | 'retracted';
    created_at: string;
    updated_at: string;
    views?: number;
  }

  let items = $state<NewsRow[]>([]);
  let loading = $state(true);
  let statusFilter = $state('');
  let search = $state('');
  let taxonomyOpen = $state(false);

  async function loadNews() {
    loading = true;
    try {
      const res = await api.get<ApiResponse<NewsRow[]>>('/api/v1/admin/news', {
        params: {
          status: statusFilter || undefined,
          search: search || undefined,
          limit: 100,
        },
      });
      if (res && res.data) {
        items = Array.isArray(res.data) ? res.data : [];
      }
    } catch (e) {
      console.warn('News load error:', e);
      items = [];
    } finally {
      loading = false;
    }
  }

  async function handleDelete(id: string, title: string, e: MouseEvent) {
    e.stopPropagation();
    if (!confirm(`Delete article "${title}"?`)) return;
    try {
      await api.delete(`/api/v1/admin/news/${id}`);
      await loadNews();
    } catch (err: any) {
      alert(`Delete failed: ${err.message}`);
    }
  }

  onMount(() => {
    loadNews();
  });
</script>

<NewsTaxonomyModal
  open={taxonomyOpen}
  onclose={() => { taxonomyOpen = false; }}
/>

<div class="space-y-6 max-w-6xl mx-auto">
  <!-- Header -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 pb-4 border-b border-slate-800">
    <div>
      <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2">
        <span>📰</span>
        <span>News & Articles</span>
      </h2>
      <p class="text-xs text-slate-400 mt-1">
        Manage news posts, editorial workflow, scheduling, and categories.
      </p>
    </div>

    <div class="flex items-center gap-2.5">
      <Button
        variant="secondary"
        size="md"
        onclick={() => { taxonomyOpen = true; }}
      >
        <span>🏷️</span>
        <span>Taxonomy</span>
      </Button>
      <Button
        variant="primary"
        size="md"
        onclick={() => router.navigate('/news/new')}
      >
        <span>+</span>
        <span>New Article</span>
      </Button>
    </div>
  </div>

  <!-- Status Tabs & Search -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
    <!-- Status Tabs -->
    <div class="flex items-center rounded-xl bg-slate-900 border border-slate-800 p-1 text-xs">
      <button
        type="button"
        class="px-3 py-1.5 rounded-lg transition-colors {statusFilter === '' ? 'bg-slate-800 text-white font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
        onclick={() => { statusFilter = ''; loadNews(); }}
      >
        All
      </button>
      <button
        type="button"
        class="px-3 py-1.5 rounded-lg transition-colors {statusFilter === 'published' ? 'bg-slate-800 text-emerald-400 font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
        onclick={() => { statusFilter = 'published'; loadNews(); }}
      >
        Published
      </button>
      <button
        type="button"
        class="px-3 py-1.5 rounded-lg transition-colors {statusFilter === 'draft' ? 'bg-slate-800 text-amber-400 font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
        onclick={() => { statusFilter = 'draft'; loadNews(); }}
      >
        Drafts
      </button>
      <button
        type="button"
        class="px-3 py-1.5 rounded-lg transition-colors {statusFilter === 'scheduled' ? 'bg-slate-800 text-sky-400 font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
        onclick={() => { statusFilter = 'scheduled'; loadNews(); }}
      >
        Scheduled
      </button>
      <button
        type="button"
        class="px-3 py-1.5 rounded-lg transition-colors {statusFilter === 'retracted' ? 'bg-slate-800 text-rose-400 font-semibold shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
        onclick={() => { statusFilter = 'retracted'; loadNews(); }}
      >
        Retracted
      </button>
    </div>

    <!-- Search Input -->
    <div class="relative w-full sm:w-64">
      <input
        type="search"
        placeholder="Search articles…"
        bind:value={search}
        oninput={() => loadNews()}
        class="w-full bg-slate-900 border border-slate-800 rounded-lg px-3.5 py-1.5 text-xs text-slate-100 placeholder:text-slate-500 focus:outline-none focus:border-sky-500"
      />
    </div>
  </div>

  <!-- Articles Table -->
  <div class="rounded-xl border border-slate-800 bg-slate-900/60 overflow-hidden shadow-sm">
    {#if loading}
      <div class="py-16 text-center text-slate-500 text-sm">
        <span class="animate-spin inline-block mr-2">⏳</span> Loading news articles…
      </div>
    {:else if items.length === 0}
      <div class="py-16 text-center text-slate-400 text-sm flex flex-col items-center justify-center gap-3">
        <p>No articles found for this filter.</p>
        <Button variant="secondary" size="sm" onclick={() => router.navigate('/news/new')}>
          Write New Article
        </Button>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm text-slate-300">
          <thead class="bg-slate-950/80 text-[11px] uppercase tracking-wider text-slate-400 border-b border-slate-800">
            <tr>
              <th class="px-5 py-3 font-semibold">Title</th>
              <th class="px-5 py-3 font-semibold">Category</th>
              <th class="px-5 py-3 font-semibold">Author</th>
              <th class="px-5 py-3 font-semibold">Status</th>
              <th class="px-5 py-3 font-semibold">Date</th>
              <th class="px-5 py-3 font-semibold text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/60">
            {#each items as item (item.id)}
              <tr
                class="hover:bg-slate-800/40 transition-colors cursor-pointer"
                onclick={() => router.navigate(`/news/${item.id}`)}
              >
                <!-- Title & Slug -->
                <td class="px-5 py-3.5 max-w-sm">
                  <div class="flex items-center gap-2">
                    {#if item.featured}
                      <span class="text-amber-400 text-xs" title="Featured post">★</span>
                    {/if}
                    <span class="font-semibold text-slate-100 truncate">{item.title}</span>
                  </div>
                  <span class="text-[11px] text-slate-500 font-mono">/{item.slug}</span>
                </td>

                <!-- Category -->
                <td class="px-5 py-3.5 whitespace-nowrap">
                  {#if item.category}
                    <span class="text-xs bg-slate-800 text-slate-300 px-2 py-0.5 rounded-full border border-slate-700/80">
                      {item.category}
                    </span>
                  {:else}
                    <span class="text-xs text-slate-600">—</span>
                  {/if}
                </td>

                <!-- Author -->
                <td class="px-5 py-3.5 whitespace-nowrap text-xs text-slate-400">
                  {item.author_name || 'Staff'}
                </td>

                <!-- Status -->
                <td class="px-5 py-3.5 whitespace-nowrap">
                  <Badge
                    variant={item.status === 'published' ? 'success' : item.status === 'scheduled' ? 'info' : item.status === 'retracted' ? 'danger' : 'warning'}
                    size="sm"
                  >
                    {item.status}
                  </Badge>
                </td>

                <!-- Date -->
                <td class="px-5 py-3.5 whitespace-nowrap text-xs text-slate-500">
                  {new Date(item.created_at).toLocaleDateString()}
                </td>

                <!-- Actions -->
                <td class="px-5 py-3.5 text-right whitespace-nowrap">
                  <div class="inline-flex items-center gap-1.5">
                    <button
                      type="button"
                      class="px-2.5 py-1 text-xs text-slate-300 hover:text-white rounded hover:bg-slate-800 transition-colors"
                      onclick={(e) => { e.stopPropagation(); router.navigate(`/news/${item.id}`); }}
                    >
                      Edit
                    </button>
                    <button
                      type="button"
                      class="px-2.5 py-1 text-xs text-rose-400 hover:text-rose-300 rounded hover:bg-rose-950/40 transition-colors"
                      onclick={(e) => handleDelete(item.id, item.title, e)}
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
