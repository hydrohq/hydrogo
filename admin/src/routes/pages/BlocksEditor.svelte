<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';

  export interface Block {
    id?: string;
    type: string;
    props?: Record<string, any>;
    items?: any[];
  }

  interface BlockCatalogType {
    type: string;
    label: string;
    badge: string;
    payload_mode: string;
  }

  interface Props {
    blocks: Block[];
    onchange: (newBlocks: Block[]) => void;
  }

  let { blocks = [], onchange }: Props = $props();

  let catalog = $state<BlockCatalogType[]>([]);
  let addModalOpen = $state(false);
  let editingBlockIndex = $state<number | null>(null);
  let deleteBlockIndex = $state<number | null>(null);

  async function loadCatalog() {
    try {
      const res = await api.get<ApiResponse<{ types: BlockCatalogType[] }>>('/api/v1/admin/pages/blocks');
      if (res && res.data && res.data.types) {
        catalog = res.data.types;
      }
    } catch (e) {
      console.warn('Block catalog load error:', e);
    }
  }

  function addBlock(type: string) {
    const newBlock: Block = {
      type,
      props: {
        title: '',
        subtitle: '',
        content: '',
      },
      items: [],
    };
    const updated = [...blocks, newBlock];
    onchange(updated);
    addModalOpen = false;
    editingBlockIndex = updated.length - 1;
  }

  function confirmRemoveBlock() {
    if (deleteBlockIndex === null) return;
    const idx = deleteBlockIndex;
    const updated = blocks.filter((_, i) => i !== idx);
    onchange(updated);
    if (editingBlockIndex === idx) {
      editingBlockIndex = null;
    } else if (editingBlockIndex !== null && editingBlockIndex > idx) {
      editingBlockIndex -= 1;
    }
    deleteBlockIndex = null;
  }

  function moveBlock(index: number, direction: 'up' | 'down') {
    const target = direction === 'up' ? index - 1 : index + 1;
    if (target < 0 || target >= blocks.length) return;
    const updated = [...blocks];
    const temp = updated[index];
    updated[index] = updated[target];
    updated[target] = temp;
    onchange(updated);
    if (editingBlockIndex === index) editingBlockIndex = target;
  }

  function updateBlockProps(index: number, key: string, value: any) {
    const updated = [...blocks];
    updated[index] = {
      ...updated[index],
      props: {
        ...(updated[index].props || {}),
        [key]: value,
      },
    };
    onchange(updated);
  }

  onMount(() => {
    loadCatalog();
  });
</script>

<ConfirmDialog
  open={deleteBlockIndex !== null}
  title="Remove Content Block"
  description={`Are you sure you want to remove block #${(deleteBlockIndex ?? 0) + 1}? Content configured in this block will be removed from the draft.`}
  confirmLabel="Remove Block"
  variant="destructive"
  onconfirm={confirmRemoveBlock}
  oncancel={() => { deleteBlockIndex = null; }}
/>

<!-- Add Block Modal -->
<Modal
  open={addModalOpen}
  title="Add Content Block"
  description="Select a block type to insert into your page layout."
  onclose={() => { addModalOpen = false; }}
>
  <div class="grid grid-cols-2 sm:grid-cols-3 gap-3 max-h-80 overflow-y-auto pr-1">
    {#each catalog as item (item.type)}
      <button
        type="button"
        class="p-3.5 rounded-lg border border-border bg-card hover:bg-accent hover:border-foreground/20 transition-all text-left flex flex-col justify-between cursor-pointer group shadow-xs"
        onclick={() => addBlock(item.type)}
      >
        <div>
          <span class="text-xs font-semibold text-foreground group-hover:text-primary transition-colors">
            {item.label}
          </span>
          <p class="text-[10px] text-muted-foreground mt-0.5 uppercase font-mono">{item.badge}</p>
        </div>
        <span class="text-xs text-primary mt-2 opacity-0 group-hover:opacity-100 transition-opacity">+ Add</span>
      </button>
    {/each}
  </div>
  {#snippet footer()}
    <Button variant="ghost" size="sm" onclick={() => { addModalOpen = false; }}>Cancel</Button>
  {/snippet}
</Modal>

<div class="space-y-4">
  <div class="flex items-center justify-between pb-3 border-b border-border">
    <div class="flex items-center gap-2">
      <h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Page Blocks</h3>
      <span class="text-xs bg-muted border border-border text-foreground px-2 py-0.5 rounded-md font-mono">
        {blocks.length}
      </span>
    </div>
    <button
      type="button"
      class="h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs transition-colors cursor-pointer"
      onclick={() => { addModalOpen = true; }}
    >
      + Add Block
    </button>
  </div>

  {#if blocks.length === 0}
    <div class="py-12 text-center text-muted-foreground text-xs border border-dashed border-border rounded-xl flex flex-col items-center justify-center gap-2">
      <p>No content blocks added yet.</p>
      <button
        type="button"
        class="h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs"
        onclick={() => { addModalOpen = true; }}
      >
        Insert First Block
      </button>
    </div>
  {:else}
    <div class="space-y-3">
      {#each blocks as block, index (index)}
        <div class="rounded-xl border border-border bg-card overflow-hidden shadow-xs">
          <!-- Block Card Header -->
          <div class="px-4 py-2.5 bg-muted/40 border-b border-border flex items-center justify-between">
            <div class="flex items-center gap-2.5">
              <span class="text-xs font-mono text-muted-foreground">#{index + 1}</span>
              <span class="text-xs font-medium text-foreground capitalize">
                {block.type.replace(/_/g, ' ')}
              </span>
            </div>

            <div class="flex items-center gap-1">
              <button
                type="button"
                disabled={index === 0}
                class="size-6 inline-flex items-center justify-center rounded text-muted-foreground hover:text-foreground disabled:opacity-30 disabled:cursor-not-allowed hover:bg-accent transition-colors"
                onclick={() => moveBlock(index, 'up')}
                title="Move Up"
              >
                ▲
              </button>
              <button
                type="button"
                disabled={index === blocks.length - 1}
                class="size-6 inline-flex items-center justify-center rounded text-muted-foreground hover:text-foreground disabled:opacity-30 disabled:cursor-not-allowed hover:bg-accent transition-colors"
                onclick={() => moveBlock(index, 'down')}
                title="Move Down"
              >
                ▼
              </button>
              <button
                type="button"
                class="ml-1.5 size-6 inline-flex items-center justify-center rounded text-destructive hover:bg-destructive/10 transition-colors cursor-pointer"
                onclick={() => { deleteBlockIndex = index; }}
                title="Remove Block"
              >
                ✕
              </button>
            </div>
          </div>

          <!-- Block Fields -->
          <div class="p-4 space-y-3">
            <Input
              label="Block Title"
              placeholder="Headline for this section…"
              value={block.props?.title || ''}
              oninput={(e) => updateBlockProps(index, 'title', (e.target as HTMLInputElement).value)}
            />

            <Input
              label="Subtitle / Eyebrow"
              placeholder="Optional subtitle…"
              value={block.props?.subtitle || ''}
              oninput={(e) => updateBlockProps(index, 'subtitle', (e.target as HTMLInputElement).value)}
            />

            <div class="flex flex-col gap-1.5">
              <label for={`block-content-${index}`} class="text-xs font-medium text-foreground">
                Content / Body
              </label>
              <textarea
                id={`block-content-${index}`}
                rows="3"
                placeholder="Block body text…"
                value={block.props?.content || ''}
                oninput={(e) => updateBlockProps(index, 'content', (e.target as HTMLTextAreaElement).value)}
                class="w-full rounded-md border border-input bg-transparent px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
              ></textarea>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
