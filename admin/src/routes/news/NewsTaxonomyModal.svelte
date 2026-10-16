<script lang="ts">
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';

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
  let formError = $state<string | null>(null);

  let deleteTarget = $state<Term | null>(null);
  let deleteLoading = $state(false);
  let filterKind = $state<'all' | 'category' | 'tag'>('all');

  const filteredTerms = $derived.by(() => {
    if (filterKind === 'all') return terms;
    return terms.filter((t) => t.kind === filterKind);
  });

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
    formError = null;
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
      formError = err?.message || 'Failed to add taxonomy term';
    } finally {
      saving = false;
    }
  }

  async function confirmDeleteTerm() {
    if (!deleteTarget) return;
    deleteLoading = true;
    try {
      await api.delete(`/api/v1/admin/news/taxonomy/${deleteTarget.id}`);
      deleteTarget = null;
      await loadTerms();
    } catch (err: any) {
      formError = err?.message || 'Failed to delete term';
    } finally {
      deleteLoading = false;
    }
  }

  $effect(() => {
    if (open) {
      formError = null;
      loadTerms();
    }
  });
</script>

<ConfirmDialog
  open={!!deleteTarget}
  title="Delete Taxonomy Term"
  description={`Are you sure you want to delete "${deleteTarget?.label}"? Articles associated with this term may lose their categorization.`}
  confirmLabel="Delete Term"
  variant="destructive"
  loading={deleteLoading}
  onconfirm={confirmDeleteTerm}
  oncancel={() => { if (!deleteLoading) deleteTarget = null; }}
/>

<Modal
  {open}
  title="News Taxonomy (Categories & Tags)"
  description="Manage categories and tags used to organize news articles and publications."
  {onclose}
>
  <div class="space-y-5">
    {#if formError}
      <div class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive flex items-center justify-between animate-in fade-in">
        <span>{formError}</span>
        <button
          type="button"
          class="text-xs hover:underline cursor-pointer"
          onclick={() => { formError = null; }}
        >
          Dismiss
        </button>
      </div>
    {/if}

    <!-- Add Term Form -->
    <form onsubmit={handleAddTerm} class="p-4 rounded-xl border border-border bg-muted/40 space-y-3">
      <div class="flex items-center justify-between">
        <h4 class="text-xs font-semibold text-foreground">Create New Term</h4>
        <div class="flex items-center rounded-lg bg-background p-0.5 border border-border text-[11px]">
          <button
            type="button"
            class="px-2.5 py-0.5 rounded font-medium transition-all {newKind === 'category' ? 'bg-primary text-primary-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
            onclick={() => { newKind = 'category'; }}
          >
            Category
          </button>
          <button
            type="button"
            class="px-2.5 py-0.5 rounded font-medium transition-all {newKind === 'tag' ? 'bg-primary text-primary-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
            onclick={() => { newKind = 'tag'; }}
          >
            Tag
          </button>
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
        <Input
          placeholder={newKind === 'category' ? 'Category name (e.g. Technology)' : 'Tag name (e.g. Open Source)'}
          bind:value={newLabel}
          required
        />
        <Input
          placeholder="Slug (optional, auto-generated)"
          bind:value={newSlug}
        />
      </div>

      <div class="flex items-center justify-between pt-1">
        <span class="text-[11px] text-muted-foreground">
          {newKind === 'category' ? 'Categories provide primary hierarchical grouping' : 'Tags provide flexible cross-cutting labels'}
        </span>
        <button
          type="submit"
          disabled={saving || !newLabel.trim()}
          class="h-8 px-3.5 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs transition-colors cursor-pointer disabled:opacity-50 inline-flex items-center gap-1.5"
        >
          {#if saving}
            <div class="size-3 animate-spin rounded-full border-2 border-current border-t-transparent"></div>
          {/if}
          <span>Add {newKind === 'category' ? 'Category' : 'Tag'}</span>
        </button>
      </div>
    </form>

    <!-- Terms List Header & Filters -->
    <div class="space-y-2.5">
      <div class="flex items-center justify-between">
        <h4 class="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          Existing Terms ({filteredTerms.length})
        </h4>
        <div class="flex items-center gap-1 text-[11px]">
          <button
            type="button"
            class="px-2 py-0.5 rounded {filterKind === 'all' ? 'bg-accent text-foreground font-semibold' : 'text-muted-foreground hover:text-foreground'}"
            onclick={() => { filterKind = 'all'; }}
          >
            All
          </button>
          <button
            type="button"
            class="px-2 py-0.5 rounded {filterKind === 'category' ? 'bg-accent text-foreground font-semibold' : 'text-muted-foreground hover:text-foreground'}"
            onclick={() => { filterKind = 'category'; }}
          >
            Categories
          </button>
          <button
            type="button"
            class="px-2 py-0.5 rounded {filterKind === 'tag' ? 'bg-accent text-foreground font-semibold' : 'text-muted-foreground hover:text-foreground'}"
            onclick={() => { filterKind = 'tag'; }}
          >
            Tags
          </button>
        </div>
      </div>

      {#if loading}
        <div class="py-8 flex flex-col items-center justify-center text-muted-foreground text-xs gap-2">
          <div class="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
          <span>Loading termsâ€¦</span>
        </div>
      {:else if filteredTerms.length === 0}
        <div class="p-8 text-center rounded-xl border border-dashed border-border bg-card">
          <p class="text-xs text-muted-foreground">No taxonomy terms found matching the current filter.</p>
        </div>
      {:else}
        <div class="divide-y divide-border rounded-xl border border-border bg-card max-h-64 overflow-y-auto">
          {#each filteredTerms as term (term.id)}
            <div class="p-3 flex items-center justify-between text-xs hover:bg-muted/30 transition-colors">
              <div class="flex items-center gap-2.5 min-w-0">
                <Badge variant={term.kind === 'category' ? 'default' : 'outline'} size="sm">
                  {term.kind || 'term'}
                </Badge>
                <span class="font-medium text-foreground truncate">{term.label}</span>
                <span class="text-muted-foreground font-mono text-[11px]">/{term.slug}</span>
              </div>
              <button
                type="button"
                class="size-7 inline-flex items-center justify-center rounded-md text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors cursor-pointer shrink-0"
                onclick={() => { deleteTarget = term; }}
                title="Delete term"
              >
                <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M3 6h18"></path>
                  <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"></path>
                  <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"></path>
                </svg>
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
