<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';

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

  function removeBlock(index: number) {
    if (!confirm('Remove this block?')) return;
    const updated = blocks.filter((_, i) => i !== index);
    onchange(updated);
    if (editingBlockIndex === index) {
      editingBlockIndex = null;
    }
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
        class="p-3.5 rounded-xl border border-slate-800 bg-slate-950/80 hover:bg-slate-900 hover:border-sky-500/50 transition-all text-left flex flex-col justify-between cursor-pointer group"
        onclick={() => addBlock(item.type)}
      >
        <div>
          <span class="text-xs font-semibold text-slate-100 group-hover:text-sky-400 transition-colors">
            {item.label}
          </span>
          <p class="text-[10px] text-slate-500 mt-0.5 uppercase font-mono">{item.badge}</p>
        </div>
        <span class="text-xs text-sky-400 mt-2 opacity-0 group-hover:opacity-100 transition-opacity">+ Add</span>
      </button>
    {/each}
  </div>
  {#snippet footer()}
    <Button variant="ghost" size="sm" onclick={() => { addModalOpen = false; }}>Cancel</Button>
  {/snippet}
</Modal>

<div class="space-y-4">
  <div class="flex items-center justify-between pb-3 border-b border-slate-800/80">
    <div class="flex items-center gap-2">
      <h3 class="text-xs font-semibold uppercase tracking-wider text-slate-400">Page Blocks</h3>
      <span class="text-xs bg-slate-900 border border-slate-800 text-slate-400 px-2 py-0.5 rounded-full font-mono">
        {blocks.length}
      </span>
    </div>
    <Button variant="primary" size="sm" onclick={() => { addModalOpen = true; }}>
      + Add Block
    </Button>
  </div>

  {#if blocks.length === 0}
    <div class="py-12 text-center text-slate-400 text-sm border-2 border-dashed border-slate-800 rounded-2xl flex flex-col items-center justify-center gap-2">
      <p>No content blocks added yet.</p>
      <Button variant="secondary" size="sm" onclick={() => { addModalOpen = true; }}>
        Insert First Block
      </Button>
    </div>
  {:else}
    <div class="space-y-3">
      {#each blocks as block, index (index)}
        <div class="rounded-xl border border-slate-800 bg-slate-900/80 overflow-hidden shadow-sm">
          <!-- Block Card Header -->
          <div class="px-4 py-3 bg-slate-950/60 border-b border-slate-800/60 flex items-center justify-between">
            <div class="flex items-center gap-2.5">
              <span class="text-xs font-mono text-slate-500">#{index + 1}</span>
              <span class="text-xs font-semibold text-slate-200 capitalize">
                {block.type.replace(/_/g, ' ')}
              </span>
            </div>

            <div class="flex items-center gap-1">
              <button
                type="button"
                disabled={index === 0}
                class="p-1 text-slate-400 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed"
                onclick={() => moveBlock(index, 'up')}
                title="Move Up"
              >
                ▲
              </button>
              <button
                type="button"
                disabled={index === blocks.length - 1}
                class="p-1 text-slate-400 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed"
                onclick={() => moveBlock(index, 'down')}
                title="Move Down"
              >
                ▼
              </button>
              <button
                type="button"
                class="ml-2 p-1 text-xs text-rose-400 hover:text-rose-300 hover:bg-rose-950/40 rounded transition-colors"
                onclick={() => removeBlock(index)}
                title="Remove Block"
              >
                ✕
              </button>
            </div>
          </div>

          <!-- Block Fields -->
          <div class="p-4 space-y-3 bg-slate-900/40">
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
              <label for={`block-content-${index}`} class="text-xs font-semibold uppercase tracking-wider text-slate-300">
                Content / Body
              </label>
              <textarea
                id={`block-content-${index}`}
                rows="3"
                placeholder="Block body text…"
                value={block.props?.content || ''}
                oninput={(e) => updateBlockProps(index, 'content', (e.target as HTMLTextAreaElement).value)}
                class="w-full rounded-lg bg-slate-900 border border-slate-800 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:border-sky-500"
              ></textarea>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
