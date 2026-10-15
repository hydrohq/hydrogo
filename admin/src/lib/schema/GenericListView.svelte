<script lang="ts">
  import { api } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import { router } from '$lib/router.svelte';
  import type { ContentItem, ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';

  interface Props {
    type: string;
  }

  let { type }: Props = $props();

  const collection = $derived(schema.getCollection(type));
  let items = $state<ContentItem[]>([]);
  let loading = $state(true);
  let search = $state('');
  let page = $state(1);
  let total = $state(0);

  const listFields = $derived.by(() => {
    if (!collection) return [];
    const explicit = collection.fields.filter(f => f.list);
    if (explicit.length > 0) return explicit;
    return collection.fields.slice(0, 4);
  });

  async function loadItems() {
    if (!collection) return;
    loading = true;
    try {
      const res = await api.get<ApiResponse<ContentItem[]>>(collection.api_path || `/api/v1/admin/content/${type}`, {
        params: {
          q: search || undefined,
          page,
          per_page: 20,
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

  async function handleDelete(id: string, e: MouseEvent) {
    e.stopPropagation();
    if (!confirm('Are you sure you want to delete this item?')) return;
    try {
      await api.delete(`${collection?.api_path || `/api/v1/admin/content/${type}`}/${id}`);
      await loadItems();
    } catch (err: any) {
      alert(`Delete failed: ${err.message}`);
    }
  }

  $effect(() => {
    if (type) {
      page = 1;
      loadItems();
    }
  });
</script>

{#if !collection}
  <div class="p-12 text-center text-muted-foreground">
    <p>Content type <strong>{type}</strong> not found in schema.</p>
    <Button class="mt-4" variant="secondary" onclick={() => router.navigate('/dashboard')}>
      Back to Dashboard
    </Button>
  </div>
{:else}
  <div class="space-y-6 max-w-6xl mx-auto">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 pb-4 border-b border-border">
      <div>
        <h2 class="text-xl font-bold tracking-tight text-foreground flex items-center gap-2">
          <span>{collection.icon || '📦'}</span>
          <span>{collection.label || collection.plural}</span>
        </h2>
        <p class="text-xs text-muted-foreground mt-1">
          {collection.description || `Manage all ${collection.plural.toLowerCase()} entries.`}
        </p>
      </div>

      <Button
        variant="primary"
        size="md"
        onclick={() => router.navigate(`/content/${type}/new`)}
        class="gap-1.5"
      >
        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="12" y1="5" x2="12" y2="19"></line>
          <line x1="5" y1="12" x2="19" y2="12"></line>
        </svg>
        <span>New {collection.singular || 'Item'}</span>
      </Button>
    </div>

    <!-- Search & Filters -->
    <div class="flex items-center gap-3">
      <div class="relative flex-1 max-w-md">
        <input
          type="search"
          placeholder="Search items…"
          bind:value={search}
          oninput={() => { page = 1; loadItems(); }}
          class="w-full bg-background border border-input rounded-md px-3.5 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-sm"
        />
      </div>
      <span class="text-xs text-muted-foreground font-mono">{total} items total</span>
    </div>

    <!-- Table -->
    <div class="rounded-xl border border-border bg-card overflow-hidden shadow-sm">
      {#if loading}
        <div class="py-16 text-center text-muted-foreground text-sm">
          <span class="animate-pulse inline-block mr-2">⏳</span>
          Loading {collection.plural.toLowerCase()}…
        </div>
      {:else if items.length === 0}
        <div class="py-16 text-center text-muted-foreground text-sm flex flex-col items-center justify-center gap-3">
          <p>No {collection.plural.toLowerCase()} found.</p>
          <Button variant="secondary" size="sm" onclick={() => router.navigate(`/content/${type}/new`)}>
            Create First {collection.singular || 'Item'}
          </Button>
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-left text-sm text-foreground">
            <thead class="bg-muted/40 text-[11px] uppercase tracking-wider text-muted-foreground border-b border-border">
              <tr>
                {#each listFields as field}
                  <th class="px-5 py-3 font-semibold">{field.label || field.name}</th>
                {/each}
                {#if collection.workflow?.enabled}
                  <th class="px-5 py-3 font-semibold">Status</th>
                {/if}
                <th class="px-5 py-3 font-semibold text-right">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border/60">
              {#each items as item (item.id)}
                <tr
                  class="hover:bg-muted/30 transition-colors cursor-pointer"
                  onclick={() => router.navigate(`/content/${type}/${item.id}`)}
                >
                  {#each listFields as field}
                    <td class="px-5 py-3.5 max-w-xs truncate">
                      {#if field.kind === 'boolean'}
                        <Badge variant={item[field.name] ? 'success' : 'default'} size="sm">
                          {item[field.name] ? 'Yes' : 'No'}
                        </Badge>
                      {:else if field.kind === 'media'}
                        <span class="text-xs font-mono text-primary truncate">
                          {String(item[field.name] || '—')}
                        </span>
                      {:else}
                        <span class="text-foreground">
                          {String(item[field.name] ?? '—')}
                        </span>
                      {/if}
                    </td>
                  {/each}

                  {#if collection.workflow?.enabled}
                    <td class="px-5 py-3.5">
                      <Badge
                        variant={item.status === 'published' ? 'success' : item.status === 'archived' ? 'danger' : 'warning'}
                        size="sm"
                      >
                        {item.status || 'draft'}
                      </Badge>
                    </td>
                  {/if}

                  <td class="px-5 py-3.5 text-right whitespace-nowrap">
                    <div class="inline-flex items-center gap-1.5">
                      <button
                        type="button"
                        class="px-2.5 py-1 text-xs text-muted-foreground hover:text-foreground rounded hover:bg-muted transition-colors cursor-pointer"
                        onclick={e => { e.stopPropagation(); router.navigate(`/content/${type}/${item.id}`); }}
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        class="px-2.5 py-1 text-xs text-rose-400 hover:text-rose-300 rounded hover:bg-destructive/15 transition-colors cursor-pointer"
                        onclick={e => { e.stopPropagation(); handleDelete(item.id, e); }}
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
{/if}
