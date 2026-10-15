<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';

  interface Term {
    id: string;
    slug: string;
    label: string;
    kind?: 'category' | 'tag' | string;
  }

  interface Props {
    open: boolean;
    onclose: () => void;
  }

  let { open, onclose }: Props = $props();

  let terms = $state<Term[]>([]);
  let loading = $state(false);
  let newLabel = $state('');
  let newSlug = $state('');
  let newKind = $state<'category' | 'tag'>('category');
  let saving = $state(false);

  async function loadTerms() {
    loading = true;
    try {
      const res = await api.get<ApiResponse<Term[]>>('/api/v1/admin/news/taxonomy');
      if (res && res.data) {
        terms = Array.isArray(res.data) ? res.data : [];
      }
    } catch (e) {
      console.warn('Taxonomy load error:', e);
    } finally {
      loading = false;
    }
  }

  async function handleAddTerm(e: SubmitEvent) {
    e.preventDefault();
    if (!newLabel.trim()) return;
    saving = true;
    try {
      const slug = newSlug.trim() || newLabel.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
      await api.post('/api/v1/admin/news/taxonomy', {
        label: newLabel.trim(),
        slug,
        kind: newKind,
      });
      newLabel = '';
      newSlug = '';
      await loadTerms();
    } catch (err: any) {
      alert(`Add term failed: ${err.message}`);
    } finally {
      saving = false;
    }
  }

  async function handleDeleteTerm(id: string) {
    if (!confirm('Delete this taxonomy term?')) return;
    try {
      await api.delete(`/api/v1/admin/news/taxonomy/${id}`);
      await loadTerms();
    } catch (err: any) {
      alert(`Delete failed: ${err.message}`);
    }
  }

  $effect(() => {
    if (open) {
      loadTerms();
    }
  });
</script>

<Modal
  {open}
  title="News Taxonomy (Categories & Tags)"
  description="Manage taxonomy terms used to organize news articles."
  {onclose}
>
  <div class="space-y-6">
    <!-- Add Term Form -->
    <form onsubmit={handleAddTerm} class="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-3">
      <h4 class="text-xs font-semibold text-slate-300 uppercase tracking-wider">Add New Term</h4>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
        <Input
          placeholder="Term label (e.g. Technology)"
          bind:value={newLabel}
          required
        />
        <Input
          placeholder="Slug (optional)"
          bind:value={newSlug}
        />
        <select
          bind:value={newKind}
          class="rounded-lg bg-slate-900 border border-slate-800 px-3 py-2 text-xs text-slate-100 focus:outline-none focus:border-sky-500"
        >
          <option value="category">Category</option>
          <option value="tag">Tag</option>
        </select>
      </div>
      <div class="flex justify-end">
        <Button type="submit" variant="primary" size="sm" loading={saving}>
          Add Term
        </Button>
      </div>
    </form>

    <!-- Terms List -->
    <div>
      <h4 class="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">Existing Terms</h4>
      {#if loading}
        <p class="text-xs text-slate-500 py-4 text-center">Loading terms…</p>
      {:else if terms.length === 0}
        <p class="text-xs text-slate-500 py-4 text-center">No categories or tags created yet.</p>
      {:else}
        <div class="divide-y divide-slate-800/80 rounded-xl border border-slate-800 bg-slate-950/40 max-h-60 overflow-y-auto">
          {#each terms as term (term.id)}
            <div class="p-3 flex items-center justify-between text-xs">
              <div class="flex items-center gap-2">
                <Badge variant={term.kind === 'category' ? 'info' : 'default'} size="sm">
                  {term.kind || 'term'}
                </Badge>
                <span class="font-medium text-slate-200">{term.label}</span>
                <span class="text-slate-500 font-mono">({term.slug})</span>
              </div>
              <button
                type="button"
                class="text-slate-500 hover:text-rose-400 p-1 rounded"
                onclick={() => handleDeleteTerm(term.id)}
              >
                ✕
              </button>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={onclose}>Done</Button>
  {/snippet}
</Modal>
