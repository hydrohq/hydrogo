<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import { router } from '$lib/router.svelte';
  import type { Collection, ContentItem, ApiResponse } from '$lib/api/types';
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
  <div class="p-12 text-center text-slate-400">
    <p>Content type <strong>{type}</strong> not found in schema.</p>
    <Button class="mt-4" variant="secondary" onclick={() => router.navigate('/dashboard')}>
      Back to Dashboard
    </Button>
  </div>
{:else}
  <div class="space-y-6 max-w-6xl mx-auto">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 pb-4 border-b border-slate-800">
      <div>
        <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2">
          <span>{collection.icon || '📄'}</span>
          <span>{collection.label || collection.plural}</span>
        </h2>
        <p class="text-xs text-slate-400 mt-1">
          {collection.description || `Manage all ${collection.plural.toLowerCase()} entries.`}
        </p>
      </div>

      <Button
        variant="primary"
        size="md"
        onclick={() => router.navigate(`/content/${type}/new`)}
      >
        <span>+</span>
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
          class="w-full bg-slate-900 border border-slate-800 rounded-lg px-3.5 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:border-sky-500 focus:ring-2 focus:ring-sky-500/20"
        />
      </div>
      <span class="text-xs text-slate-400">{total} items total</span>
    </div>

    <!-- Table -->
    <div class="rounded-xl border border-slate-800 bg-slate-900/60 overflow-hidden shadow-sm">
      {#if loading}
        <div class="py-16 text-center text-slate-500 text-sm">
          <span class="animate-spin inline-block mr-2">⏳</span>
          Loading {collection.plural.toLowerCase()}…
        </div>
      {:else if items.length === 0}
        <div class="py-16 text-center text-slate-400 text-sm flex flex-col items-center justify-center gap-3">
          <p>No {collection.plural.toLowerCase()} found.</p>
          <Button variant="secondary" size="sm" onclick={() => router.navigate(`/content/${type}/new`)}>
            Create First {collection.singular || 'Item'}
          </Button>
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-left text-sm text-slate-300">
            <thead class="bg-slate-950/80 text-[11px] uppercase tracking-wider text-slate-400 border-b border-slate-800">
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
            <tbody class="divide-y divide-slate-800/60">
              {#each items as item (item.id)}
                <tr
                  class="hover:bg-slate-800/40 transition-colors cursor-pointer"
                  onclick={() => router.navigate(`/content/${type}/${item.id}`)}
                >
                  {#each listFields as field}
                    <td class="px-5 py-3.5 max-w-xs truncate">
                      {#if field.kind === 'boolean'}
                        <Badge variant={item[field.name] ? 'success' : 'default'} size="sm">
                          {item[field.name] ? 'Yes' : 'No'}
                        </Badge>
                      {:else if field.kind === 'media'}
                        <span class="text-xs font-mono text-sky-400 truncate">
                          {String(item[field.name] || '—')}
                        </span>
                      {:else}
                        <span class="text-slate-200">
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
                        class="px-2.5 py-1 text-xs text-slate-300 hover:text-white rounded hover:bg-slate-800 transition-colors"
                        onclick={e => { e.stopPropagation(); router.navigate(`/content/${type}/${item.id}`); }}
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        class="px-2.5 py-1 text-xs text-rose-400 hover:text-rose-300 rounded hover:bg-rose-950/40 transition-colors"
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
