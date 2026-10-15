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
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 pb-4 border-b border-border">
    <div>
      <h2 class="text-xl font-bold tracking-tight text-foreground flex items-center gap-2">
        <span>News & Articles</span>
      </h2>
      <p class="text-xs text-muted-foreground mt-1">
        Manage news posts, editorial workflow, scheduling, and categories.
      </p>
    </div>

    <div class="flex items-center gap-2.5">
      <Button
        variant="secondary"
        size="md"
        onclick={() => { taxonomyOpen = true; }}
        class="gap-1.5"
      >
        <svg class="w-4 h-4 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"></path>
          <line x1="7" y1="7" x2="7.01" y2="7"></line>
        </svg>
        <span>Taxonomy</span>
      </Button>
      <Button
        variant="primary"
        size="md"
        onclick={() => router.navigate('/news/new')}
        class="gap-1.5 shadow-sm"
      >
        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="12" y1="5" x2="12" y2="19"></line>
          <line x1="5" y1="12" x2="19" y2="12"></line>
        </svg>
        <span>New Article</span>
      </Button>
    </div>
  </div>

  <!-- Status Tabs & Search -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
    <!-- Status Tabs -->
    <div class="flex items-center rounded-lg bg-muted p-1 text-xs border border-border">
      <button
        type="button"
        class="px-3 py-1.5 rounded-md transition-all font-medium {statusFilter === '' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { statusFilter = ''; loadNews(); }}
      >
        All
      </button>
      <button
        type="button"
        class="px-3 py-1.5 rounded-md transition-all font-medium {statusFilter === 'published' ? 'bg-background text-emerald-400 shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { statusFilter = 'published'; loadNews(); }}
      >
        Published
      </button>
      <button
        type="button"
        class="px-3 py-1.5 rounded-md transition-all font-medium {statusFilter === 'draft' ? 'bg-background text-amber-400 shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { statusFilter = 'draft'; loadNews(); }}
      >
        Drafts
      </button>
      <button
        type="button"
        class="px-3 py-1.5 rounded-md transition-all font-medium {statusFilter === 'scheduled' ? 'bg-background text-primary shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { statusFilter = 'scheduled'; loadNews(); }}
      >
        Scheduled
      </button>
      <button
        type="button"
        class="px-3 py-1.5 rounded-md transition-all font-medium {statusFilter === 'retracted' ? 'bg-background text-rose-400 shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
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
        class="w-full bg-background border border-input rounded-md px-3.5 py-1.5 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-sm"
      />
    </div>
  </div>

  <!-- Articles Table -->
  <div class="rounded-xl border border-border bg-card overflow-hidden shadow-sm">
    {#if loading}
      <div class="py-16 text-center text-muted-foreground text-sm">
        <span class="animate-pulse inline-block mr-2">⏳</span> Loading news articles…
      </div>
    {:else if items.length === 0}
      <div class="py-16 text-center text-muted-foreground text-sm flex flex-col items-center justify-center gap-3">
        <p>No articles found for this filter.</p>
        <Button variant="secondary" size="sm" onclick={() => router.navigate('/news/new')}>
          Write New Article
        </Button>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm text-foreground">
          <thead class="bg-muted/40 text-[11px] uppercase tracking-wider text-muted-foreground border-b border-border">
            <tr>
              <th class="px-5 py-3 font-semibold">Title</th>
              <th class="px-5 py-3 font-semibold">Category</th>
              <th class="px-5 py-3 font-semibold">Author</th>
              <th class="px-5 py-3 font-semibold">Status</th>
              <th class="px-5 py-3 font-semibold">Date</th>
              <th class="px-5 py-3 font-semibold text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border/60">
            {#each items as item (item.id)}
              <tr
                class="hover:bg-muted/30 transition-colors cursor-pointer"
                onclick={() => router.navigate(`/news/${item.id}`)}
              >
                <!-- Title & Slug -->
                <td class="px-5 py-3.5 max-w-sm">
                  <div class="flex items-center gap-2">
                    {#if item.featured}
                      <span class="text-amber-400 text-xs" title="Featured post">★</span>
                    {/if}
                    <span class="font-semibold text-foreground truncate">{item.title}</span>
                  </div>
                  <span class="text-[11px] text-muted-foreground font-mono">/{item.slug}</span>
                </td>

                <!-- Category -->
                <td class="px-5 py-3.5 whitespace-nowrap">
                  {#if item.category}
                    <span class="text-xs bg-muted text-muted-foreground px-2 py-0.5 rounded-full border border-border/80">
                      {item.category}
                    </span>
                  {:else}
                    <span class="text-xs text-muted-foreground/60">—</span>
                  {/if}
                </td>

                <!-- Author -->
                <td class="px-5 py-3.5 whitespace-nowrap text-xs text-muted-foreground">
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
                <td class="px-5 py-3.5 whitespace-nowrap text-xs text-muted-foreground">
                  {new Date(item.created_at).toLocaleDateString()}
                </td>

                <!-- Actions -->
                <td class="px-5 py-3.5 text-right whitespace-nowrap">
                  <div class="inline-flex items-center gap-1.5">
                    <button
                      type="button"
                      class="px-2.5 py-1 text-xs text-muted-foreground hover:text-foreground rounded hover:bg-muted transition-colors cursor-pointer"
                      onclick={(e) => { e.stopPropagation(); router.navigate(`/news/${item.id}`); }}
                    >
                      Edit
                    </button>
                    <button
                      type="button"
                      class="px-2.5 py-1 text-xs text-rose-400 hover:text-rose-300 rounded hover:bg-destructive/15 transition-colors cursor-pointer"
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
