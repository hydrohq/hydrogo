<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
  import MediaUrlInput from '$lib/components/media/MediaUrlInput.svelte';

  export interface Block {
    id?: string;
    type: string;
    payload?: Record<string, any>;
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
  let deleteBlockIndex = $state<number | null>(null);

  function blockData(block: Block): Record<string, any> {
    return {
      ...(block.props || {}),
      ...(block.payload || {}),
      ...(Array.isArray(block.items) && block.items.length > 0 && !block.payload?.items ? { items: block.items } : {}),
    };
  }

  function defaultPayloadFor(type: string): Record<string, any> {
    switch (type) {
      case 'hero':
      case 'cta':
        return { title: '', subtitle: '', button_label: '', button_href: '' };
      case 'carousel':
        return { slides: [{ src: '', alt: '', caption: '' }] };
      case 'testimonials':
        return { items: [{ name: '', quote: '' }] };
      case 'faq':
        return { items: [{ question: '', answer: '' }] };
      case 'pricing_table':
        return { items: [{ name: '', price_cents: 0, currency: 'USD', features: [] }] };
      case 'stats':
        return { stats: [{ number_value: 0, suffix: '', label: '' }] };
      case 'trust_bar':
        return { items: [{ title: '', subtitle: '' }] };
      case 'feature_list':
        return { items: [{ title: '', text: '' }] };
      case 'columns':
        return { count: 2, columns: [{ title: '', text: '', button_url: '' }, { title: '', text: '', button_url: '' }] };
      case 'contact_cta':
        return { phone: '', whatsapp: '' };
      case 'callout':
        return { title: '', text: '', tone: 'info' };
      case 'action_card':
        return { title: '', text: '', button_text: '', button_url: '' };
      case 'figure':
        return { src: '', alt: '', caption: '' };
      default:
        return { title: '', subtitle: '', content: '' };
    }
  }

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
    const initial = defaultPayloadFor(type);
    const newBlock: Block = {
      type,
      payload: initial,
      props: { ...initial },
      items: Array.isArray(initial.items) ? initial.items : [],
    };
    onchange([...blocks, newBlock]);
    addModalOpen = false;
  }

  function confirmRemoveBlock() {
    if (deleteBlockIndex === null) return;
    const idx = deleteBlockIndex;
    onchange(blocks.filter((_, i) => i !== idx));
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
  }

  function patchBlock(index: number, patch: Record<string, any>) {
    const updated = [...blocks];
    const current = blockData(updated[index]);
    const nextPayload = { ...current, ...patch };
    updated[index] = {
      ...updated[index],
      payload: nextPayload,
      props: nextPayload,
      ...(Array.isArray(nextPayload.items) ? { items: nextPayload.items } : {}),
    };
    onchange(updated);
  }

  function arrayField(block: Block, key: string): any[] {
    const data = blockData(block);
    return Array.isArray(data[key]) ? data[key] : [];
  }

  function addArrayItem(blockIndex: number, key: string, template: Record<string, any>) {
    const list = [...arrayField(blocks[blockIndex], key), { ...template }];
    patchBlock(blockIndex, { [key]: list });
  }

  function updateArrayItem(blockIndex: number, key: string, itemIndex: number, itemPatch: Record<string, any>) {
    const list = arrayField(blocks[blockIndex], key).map((item, i) =>
      i === itemIndex ? { ...item, ...itemPatch } : item
    );
    patchBlock(blockIndex, { [key]: list });
  }

  function removeArrayItem(blockIndex: number, key: string, itemIndex: number) {
    const list = arrayField(blocks[blockIndex], key).filter((_, i) => i !== itemIndex);
    patchBlock(blockIndex, { [key]: list });
  }

  onMount(() => {
    loadCatalog();
  });
</script>

<ConfirmDialog
  open={deleteBlockIndex !== null}
  title={t('pages.blocks.removeTitle')}
  description={t('pages.blocks.removeDescription', { number: (deleteBlockIndex ?? 0) + 1 })}
  confirmLabel={t('pages.blocks.removeConfirm')}
  variant="destructive"
  onconfirm={confirmRemoveBlock}
  oncancel={() => { deleteBlockIndex = null; }}
/>

<!-- Add Block Modal -->
<Modal
  open={addModalOpen}
  title={t('pages.blocks.addTitle')}
  description={t('pages.blocks.addDescription')}
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
        <span class="text-xs text-primary mt-2 opacity-0 group-hover:opacity-100 transition-opacity">{t('pages.blocks.addShort')}</span>
      </button>
    {/each}
  </div>
  {#snippet footer()}
    <Button variant="ghost" size="sm" onclick={() => { addModalOpen = false; }}>{t('common.cancel')}</Button>
  {/snippet}
</Modal>

<div class="space-y-4">
  <div class="flex items-center justify-between pb-3 border-b border-border">
    <div class="flex items-center gap-2">
      <h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">{t('pages.blocks.title')}</h3>
      <span class="text-xs bg-muted border border-border text-foreground px-2 py-0.5 rounded-md font-mono">
        {blocks.length}
      </span>
    </div>
    <button
      type="button"
      class="h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs transition-colors cursor-pointer"
      onclick={() => { addModalOpen = true; }}
    >
      {t('pages.blocks.add')}
    </button>
  </div>

  {#if blocks.length === 0}
    <div class="py-12 text-center text-muted-foreground text-xs border border-dashed border-border rounded-xl flex flex-col items-center justify-center gap-2">
      <p>{t('pages.blocks.empty')}</p>
      <button
        type="button"
        class="h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs"
        onclick={() => { addModalOpen = true; }}
      >
        {t('pages.blocks.insertFirst')}
      </button>
    </div>
  {:else}
    <div class="space-y-3">
      {#each blocks as block, index (index)}
        {@const data = blockData(block)}
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
                class="size-7 inline-flex items-center justify-center rounded-md border border-input bg-background text-muted-foreground hover:text-foreground disabled:opacity-30 disabled:cursor-not-allowed hover:bg-accent transition-colors cursor-pointer"
                onclick={() => moveBlock(index, 'up')}
                title={t('pages.blocks.moveUp')}
                aria-label={t('pages.blocks.moveUpAria')}
              >
                <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="m18 15-6-6-6 6" />
                </svg>
              </button>
              <button
                type="button"
                disabled={index === blocks.length - 1}
                class="size-7 inline-flex items-center justify-center rounded-md border border-input bg-background text-muted-foreground hover:text-foreground disabled:opacity-30 disabled:cursor-not-allowed hover:bg-accent transition-colors cursor-pointer"
                onclick={() => moveBlock(index, 'down')}
                title={t('pages.blocks.moveDown')}
                aria-label={t('pages.blocks.moveDownAria')}
              >
                <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="m6 9 6 6 6-6" />
                </svg>
              </button>
              <button
                type="button"
                class="ml-1 size-7 inline-flex items-center justify-center rounded-md border border-destructive/30 bg-destructive/5 text-destructive hover:bg-destructive hover:text-destructive-foreground transition-colors cursor-pointer"
                onclick={() => { deleteBlockIndex = index; }}
                title={t('pages.blocks.remove')}
                aria-label={t('pages.blocks.remove')}
              >
                <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M18 6 6 18" />
                  <path d="m6 6 12 12" />
                </svg>
              </button>
            </div>
          </div>

          <!-- Type-Aware Block Fields -->
          <div class="p-4 space-y-3">
            {#if block.type === 'hero' || block.type === 'cta' || block.type === 'action_card'}
              <Input
                label={t('pages.blocks.blockTitle')}
                placeholder={t('pages.blocks.blockTitlePlaceholder')}
                value={data.title || ''}
                oninput={(e) => patchBlock(index, { title: (e.target as HTMLInputElement).value })}
              />
              <Input
                label={t('pages.blocks.subtitle')}
                placeholder={t('pages.blocks.subtitlePlaceholder')}
                value={data.subtitle ?? data.text ?? ''}
                oninput={(e) => {
                  const val = (e.target as HTMLInputElement).value;
                  patchBlock(index, block.type === 'action_card' ? { text: val } : { subtitle: val });
                }}
              />
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <Input
                  label={t('pages.blocks.buttonLabel')}
                  placeholder={t('pages.blocks.buttonLabelPlaceholder')}
                  value={data.button_label ?? data.button_text ?? ''}
                  oninput={(e) => {
                    const val = (e.target as HTMLInputElement).value;
                    patchBlock(index, block.type === 'action_card' ? { button_text: val } : { button_label: val });
                  }}
                />
                <Input
                  label={t('pages.blocks.buttonHref')}
                  placeholder="/products"
                  value={data.button_href ?? data.button_url ?? ''}
                  oninput={(e) => {
                    const val = (e.target as HTMLInputElement).value;
                    patchBlock(index, block.type === 'action_card' ? { button_url: val } : { button_href: val });
                  }}
                />
              </div>
              {#if block.type === 'hero'}
                <div>
                  <label for={`block-media-${index}`} class="block text-xs font-medium text-foreground mb-1">{t('pages.blocks.mediaUrl')}</label>
                  <MediaUrlInput
                    id={`block-media-${index}`}
                    bind:value={() => data.image_url || '', (val) => patchBlock(index, { image_url: val })}
                    placeholder={t('pages.blocks.imageUrlPlaceholder')}
                  />
                </div>
              {/if}

            {:else if block.type === 'carousel'}
              {@const slides = arrayField(block, 'slides')}
              <div class="flex items-center justify-between">
                <span class="text-xs font-medium text-foreground">{t('pages.blocks.slides')} ({slides.length})</span>
                <Button type="button" size="sm" variant="outline" onclick={() => addArrayItem(index, 'slides', { src: '', alt: '', caption: '' })}>
                  {t('pages.blocks.addSlide')}
                </Button>
              </div>
              <div class="space-y-2.5">
                {#each slides as slide, sIdx}
                  <div class="p-3 rounded-lg border border-border bg-muted/20 space-y-2">
                    <div class="flex items-center justify-between">
                      <span class="text-[11px] font-mono text-muted-foreground">#{sIdx + 1}</span>
                      <button type="button" class="text-[11px] text-destructive hover:underline cursor-pointer" onclick={() => removeArrayItem(index, 'slides', sIdx)}>
                        {t('pages.blocks.removeItem')}
                      </button>
                    </div>
                    <MediaUrlInput
                      bind:value={() => slide.src || '', (val) => updateArrayItem(index, 'slides', sIdx, { src: val })}
                      placeholder="/media/slide.jpg"
                    />
                    <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
                      <Input
                        label={t('pages.blocks.slideAlt')}
                        value={slide.alt || ''}
                        oninput={(e) => updateArrayItem(index, 'slides', sIdx, { alt: (e.target as HTMLInputElement).value })}
                      />
                      <Input
                        label={t('pages.blocks.slideCaption')}
                        value={slide.caption || ''}
                        oninput={(e) => updateArrayItem(index, 'slides', sIdx, { caption: (e.target as HTMLInputElement).value })}
                      />
                    </div>
                  </div>
                {/each}
              </div>

            {:else if block.type === 'testimonials'}
              {@const items = arrayField(block, 'items')}
              <div class="flex items-center justify-between">
                <span class="text-xs font-medium text-foreground">{t('pages.blocks.testimonials')} ({items.length})</span>
                <Button type="button" size="sm" variant="outline" onclick={() => addArrayItem(index, 'items', { name: '', quote: '' })}>
                  {t('pages.blocks.addTestimonial')}
                </Button>
              </div>
              <div class="space-y-2.5">
                {#each items as item, iIdx}
                  <div class="p-3 rounded-lg border border-border bg-muted/20 space-y-2">
                    <div class="flex items-center justify-between">
                      <span class="text-[11px] font-mono text-muted-foreground">#{iIdx + 1}</span>
                      <button type="button" class="text-[11px] text-destructive hover:underline cursor-pointer" onclick={() => removeArrayItem(index, 'items', iIdx)}>
                        {t('pages.blocks.removeItem')}
                      </button>
                    </div>
                    <Input
                      label={t('pages.blocks.authorName')}
                      value={item.name || ''}
                      oninput={(e) => updateArrayItem(index, 'items', iIdx, { name: (e.target as HTMLInputElement).value })}
                    />
                    <div class="flex flex-col gap-1">
                      <span class="text-xs font-medium text-foreground">{t('pages.blocks.quote')}</span>
                      <textarea
                        rows="2"
                        value={item.quote || ''}
                        oninput={(e) => updateArrayItem(index, 'items', iIdx, { quote: (e.target as HTMLTextAreaElement).value })}
                        class="w-full rounded-md border border-input bg-transparent px-3 py-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                      ></textarea>
                    </div>
                  </div>
                {/each}
              </div>

            {:else if block.type === 'faq'}
              {@const items = arrayField(block, 'items')}
              <div class="flex items-center justify-between">
                <span class="text-xs font-medium text-foreground">{t('pages.blocks.faqItems')} ({items.length})</span>
                <Button type="button" size="sm" variant="outline" onclick={() => addArrayItem(index, 'items', { question: '', answer: '' })}>
                  {t('pages.blocks.addFaq')}
                </Button>
              </div>
              <div class="space-y-2.5">
                {#each items as item, iIdx}
                  <div class="p-3 rounded-lg border border-border bg-muted/20 space-y-2">
                    <div class="flex items-center justify-between">
                      <span class="text-[11px] font-mono text-muted-foreground">#{iIdx + 1}</span>
                      <button type="button" class="text-[11px] text-destructive hover:underline cursor-pointer" onclick={() => removeArrayItem(index, 'items', iIdx)}>
                        {t('pages.blocks.removeItem')}
                      </button>
                    </div>
                    <Input
                      label={t('pages.blocks.question')}
                      value={item.question || ''}
                      oninput={(e) => updateArrayItem(index, 'items', iIdx, { question: (e.target as HTMLInputElement).value })}
                    />
                    <Input
                      label={t('pages.blocks.answer')}
                      value={item.answer || ''}
                      oninput={(e) => updateArrayItem(index, 'items', iIdx, { answer: (e.target as HTMLInputElement).value })}
                    />
                  </div>
                {/each}
              </div>

            {:else if block.type === 'pricing_table'}
              {@const items = arrayField(block, 'items')}
              <div class="flex items-center justify-between">
                <span class="text-xs font-medium text-foreground">{t('pages.blocks.plans')} ({items.length})</span>
                <Button type="button" size="sm" variant="outline" onclick={() => addArrayItem(index, 'items', { name: '', price_cents: 0, currency: 'USD', features: [] })}>
                  {t('pages.blocks.addPlan')}
                </Button>
              </div>
              <div class="space-y-2.5">
                {#each items as item, iIdx}
                  <div class="p-3 rounded-lg border border-border bg-muted/20 space-y-2">
                    <div class="flex items-center justify-between">
                      <span class="text-[11px] font-mono text-muted-foreground">#{iIdx + 1}</span>
                      <button type="button" class="text-[11px] text-destructive hover:underline cursor-pointer" onclick={() => removeArrayItem(index, 'items', iIdx)}>
                        {t('pages.blocks.removeItem')}
                      </button>
                    </div>
                    <div class="grid grid-cols-1 sm:grid-cols-3 gap-2">
                      <Input
                        label={t('pages.blocks.planName')}
                        value={item.name || ''}
                        oninput={(e) => updateArrayItem(index, 'items', iIdx, { name: (e.target as HTMLInputElement).value })}
                      />
                      <Input
                        type="number"
                        label={t('pages.blocks.priceCents')}
                        value={String(item.price_cents ?? 0)}
                        oninput={(e) => updateArrayItem(index, 'items', iIdx, { price_cents: Number((e.target as HTMLInputElement).value) || 0 })}
                      />
                      <Input
                        label={t('pages.blocks.currency')}
                        value={item.currency || 'USD'}
                        oninput={(e) => updateArrayItem(index, 'items', iIdx, { currency: (e.target as HTMLInputElement).value.toUpperCase().slice(0, 3) })}
                      />
                    </div>
                    <Input
                      label={t('pages.blocks.featuresList')}
                      value={Array.isArray(item.features) ? item.features.join(', ') : ''}
                      oninput={(e) => updateArrayItem(index, 'items', iIdx, {
                        features: (e.target as HTMLInputElement).value.split(',').map((s) => s.trim()).filter(Boolean),
                      })}
                    />
                  </div>
                {/each}
              </div>

            {:else if block.type === 'stats'}
              {@const stats = arrayField(block, 'stats')}
              <div class="flex items-center justify-between">
                <span class="text-xs font-medium text-foreground">{t('pages.blocks.stats')} ({stats.length})</span>
                <Button type="button" size="sm" variant="outline" onclick={() => addArrayItem(index, 'stats', { number_value: 0, suffix: '', label: '' })}>
                  {t('pages.blocks.addStat')}
                </Button>
              </div>
              <div class="space-y-2">
                {#each stats as stat, sIdx}
                  <div class="grid grid-cols-1 sm:grid-cols-4 gap-2 items-end p-2.5 rounded-lg border border-border bg-muted/20">
                    <Input
                      type="number"
                      label={t('pages.blocks.statValue')}
                      value={String(stat.number_value ?? 0)}
                      oninput={(e) => updateArrayItem(index, 'stats', sIdx, { number_value: Number((e.target as HTMLInputElement).value) || 0 })}
                    />
                    <Input
                      label={t('pages.blocks.statSuffix')}
                      value={stat.suffix || ''}
                      oninput={(e) => updateArrayItem(index, 'stats', sIdx, { suffix: (e.target as HTMLInputElement).value })}
                    />
                    <Input
                      label={t('pages.blocks.statLabel')}
                      value={stat.label || ''}
                      oninput={(e) => updateArrayItem(index, 'stats', sIdx, { label: (e.target as HTMLInputElement).value })}
                    />
                    <button type="button" class="h-9 text-xs text-destructive hover:underline cursor-pointer" onclick={() => removeArrayItem(index, 'stats', sIdx)}>
                      {t('pages.blocks.removeItem')}
                    </button>
                  </div>
                {/each}
              </div>

            {:else if block.type === 'trust_bar' || block.type === 'feature_list'}
              {@const items = arrayField(block, 'items')}
              {@const textKey = block.type === 'trust_bar' ? 'subtitle' : 'text'}
              <div class="flex items-center justify-between">
                <span class="text-xs font-medium text-foreground">{t('pages.blocks.items')} ({items.length})</span>
                <Button type="button" size="sm" variant="outline" onclick={() => addArrayItem(index, 'items', { title: '', [textKey]: '' })}>
                  {t('pages.blocks.addItem')}
                </Button>
              </div>
              <div class="space-y-2">
                {#each items as item, iIdx}
                  <div class="p-3 rounded-lg border border-border bg-muted/20 space-y-2">
                    <div class="flex items-center justify-between">
                      <span class="text-[11px] font-mono text-muted-foreground">#{iIdx + 1}</span>
                      <button type="button" class="text-[11px] text-destructive hover:underline cursor-pointer" onclick={() => removeArrayItem(index, 'items', iIdx)}>
                        {t('pages.blocks.removeItem')}
                      </button>
                    </div>
                    <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
                      <Input
                        label={t('pages.blocks.itemTitle')}
                        value={item.title || ''}
                        oninput={(e) => updateArrayItem(index, 'items', iIdx, { title: (e.target as HTMLInputElement).value })}
                      />
                      <Input
                        label={t('pages.blocks.itemText')}
                        value={item[textKey] || ''}
                        oninput={(e) => updateArrayItem(index, 'items', iIdx, { [textKey]: (e.target as HTMLInputElement).value })}
                      />
                    </div>
                  </div>
                {/each}
              </div>

            {:else if block.type === 'columns'}
              {@const cols = arrayField(block, 'columns')}
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
                {#each cols as col, cIdx}
                  <div class="p-3 rounded-lg border border-border bg-muted/20 space-y-2">
                    <span class="text-[11px] font-mono text-muted-foreground">#{cIdx + 1}</span>
                    <Input
                      label={t('pages.blocks.itemTitle')}
                      value={col.title || ''}
                      oninput={(e) => updateArrayItem(index, 'columns', cIdx, { title: (e.target as HTMLInputElement).value })}
                    />
                    <Input
                      label={t('pages.blocks.itemText')}
                      value={col.text || ''}
                      oninput={(e) => updateArrayItem(index, 'columns', cIdx, { text: (e.target as HTMLInputElement).value })}
                    />
                  </div>
                {/each}
              </div>

            {:else if block.type === 'contact_cta'}
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <Input
                  label={t('pages.blocks.phone')}
                  placeholder="+90 212 000 00 00"
                  value={data.phone || ''}
                  oninput={(e) => patchBlock(index, { phone: (e.target as HTMLInputElement).value })}
                />
                <Input
                  label={t('pages.blocks.whatsapp')}
                  placeholder="+90 555 000 00 00"
                  value={data.whatsapp || ''}
                  oninput={(e) => patchBlock(index, { whatsapp: (e.target as HTMLInputElement).value })}
                />
              </div>

            {:else if block.type === 'callout'}
              <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <div class="sm:col-span-2">
                  <Input
                    label={t('pages.blocks.blockTitle')}
                    value={data.title || ''}
                    oninput={(e) => patchBlock(index, { title: (e.target as HTMLInputElement).value })}
                  />
                </div>
                <label class="space-y-1 text-xs font-medium text-foreground">
                  <span>{t('pages.blocks.tone')}</span>
                  <select
                    value={data.tone || 'info'}
                    onchange={(e) => patchBlock(index, { tone: (e.currentTarget as HTMLSelectElement).value })}
                    class="w-full h-9 rounded-md border border-input bg-background px-3 text-xs text-foreground"
                  >
                    <option value="info">{t('pages.blocks.toneInfo')}</option>
                    <option value="success">{t('pages.blocks.toneSuccess')}</option>
                    <option value="warning">{t('pages.blocks.toneWarning')}</option>
                    <option value="tip">{t('pages.blocks.toneTip')}</option>
                  </select>
                </label>
              </div>
              <Input
                label={t('pages.blocks.itemText')}
                value={data.text || ''}
                oninput={(e) => patchBlock(index, { text: (e.target as HTMLInputElement).value })}
              />

            {:else if block.type === 'figure'}
              <MediaUrlInput
                bind:value={() => data.src || '', (val) => patchBlock(index, { src: val })}
                placeholder="/media/image.jpg"
              />
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
                <Input
                  label={t('pages.blocks.slideAlt')}
                  value={data.alt || ''}
                  oninput={(e) => patchBlock(index, { alt: (e.target as HTMLInputElement).value })}
                />
                <Input
                  label={t('pages.blocks.slideCaption')}
                  value={data.caption || ''}
                  oninput={(e) => patchBlock(index, { caption: (e.target as HTMLInputElement).value })}
                />
              </div>

            {:else}
              <Input
                label={t('pages.blocks.blockTitle')}
                placeholder={t('pages.blocks.blockTitlePlaceholder')}
                value={data.title || ''}
                oninput={(e) => patchBlock(index, { title: (e.target as HTMLInputElement).value })}
              />
              <Input
                label={t('pages.blocks.subtitle')}
                placeholder={t('pages.blocks.subtitlePlaceholder')}
                value={data.subtitle || ''}
                oninput={(e) => patchBlock(index, { subtitle: (e.target as HTMLInputElement).value })}
              />
              <div class="flex flex-col gap-1.5">
                <label for={`block-content-${index}`} class="text-xs font-medium text-foreground">
                  {t('pages.blocks.content')}
                </label>
                <textarea
                  id={`block-content-${index}`}
                  rows="3"
                  placeholder={t('pages.blocks.contentPlaceholder')}
                  value={data.content || ''}
                  oninput={(e) => patchBlock(index, { content: (e.target as HTMLTextAreaElement).value })}
                  class="w-full rounded-md border border-input bg-transparent px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
                ></textarea>
              </div>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
