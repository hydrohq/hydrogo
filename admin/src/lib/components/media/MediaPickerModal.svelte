<script lang="ts">
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';

  export interface MediaAsset {
    id: string;
    original_filename?: string;
    filename?: string;
    name?: string;
    mime_type?: string;
    content_type?: string;
    url?: string;
    path?: string;
    size_bytes?: number;
    created_at?: string;
  }

  interface Props {
    open: boolean;
    onselect: (media: any) => void;
    onclose: () => void;
  }

  let {
    open = $bindable(false),
    onselect,
    onclose,
  }: Props = $props();

  let mediaItems = $state<MediaAsset[]>([]);
  let loading = $state(false);
  let uploading = $state(false);
  let fileInput: HTMLInputElement | null = $state(null);
  let selectedId = $state<string | null>(null);
  let search = $state('');

  function getUrl(item: MediaAsset): string {
    return item.url || item.path || `/media/${item.id}`;
  }

  function getName(item: MediaAsset): string {
    return item.original_filename || item.filename || item.name || item.id;
  }

  function isImage(item: MediaAsset): boolean {
    const type = item.mime_type || item.content_type || '';
    return type.startsWith('image/');
  }

  const filteredItems = $derived.by(() => {
    if (!search.trim()) return mediaItems;
    const q = search.toLowerCase();
    return mediaItems.filter((i) => getName(i).toLowerCase().includes(q));
  });

  const selectedItem = $derived(mediaItems.find((m) => m.id === selectedId));

  async function loadMedia() {
    loading = true;
    try {
      const res = await api.get<ApiResponse<MediaAsset[]>>('/api/v1/admin/media');
      if (res && res.data) {
        mediaItems = Array.isArray(res.data) ? res.data : [];
      }
    } catch (e) {
      console.warn('Failed to load media:', e);
    } finally {
      loading = false;
    }
  }

  async function handleFileUpload(e: Event) {
    const target = e.target as HTMLInputElement;
    if (!target.files || target.files.length === 0) return;

    uploading = true;
    try {
      const formData = new FormData();
      formData.append('file', target.files[0]);
      const res = await api.upload<ApiResponse<MediaAsset>>('/api/v1/admin/media', formData);
      if (res && res.data) {
        await loadMedia();
        selectedId = res.data.id;
      }
    } catch (e) {
      console.error('Upload failed:', e);
    } finally {
      uploading = false;
      if (target) target.value = '';
    }
  }

  function confirmSelection() {
    if (selectedItem) {
      onselect(selectedItem);
      onclose();
    }
  }

  $effect(() => {
    if (open) {
      loadMedia();
    }
  });
</script>

<Modal
  {open}
  title="Media Library"
  description="Select an asset from the media library or upload new files."
  {onclose}
>
  <div class="flex flex-col gap-3.5">
    <!-- Top toolbar: upload & search -->
    <div class="flex items-center justify-between gap-3 pb-3 border-b border-border">
      <input
        type="file"
        class="hidden"
        bind:this={fileInput}
        onchange={handleFileUpload}
        accept="image/*,video/*,application/pdf"
      />

      <div class="relative flex-1 max-w-xs">
        <svg class="size-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"></circle>
          <path d="m21 21-4.3-4.3"></path>
        </svg>
        <input
          type="text"
          placeholder="Filter assetsâ€¦"
          bind:value={search}
          class="h-8 w-full pl-8 pr-3 text-xs rounded-md border border-input bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
        />
      </div>

      <button
        type="button"
        disabled={uploading}
        class="h-8 px-3.5 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs transition-colors cursor-pointer disabled:opacity-50 inline-flex items-center gap-1.5"
        onclick={() => fileInput?.click()}
      >
        {#if uploading}
          <div class="size-3 animate-spin rounded-full border-2 border-current border-t-transparent"></div>
          <span>Uploadingâ€¦</span>
        {:else}
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
            <polyline points="17 8 12 3 7 8"></polyline>
            <line x1="12" y1="3" x2="12" y2="15"></line>
          </svg>
          <span>Upload Asset</span>
        {/if}
      </button>
    </div>

    {#if loading}
      <div class="py-16 flex flex-col items-center justify-center text-muted-foreground text-xs gap-2">
        <div class="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
        <span>Loading library assetsâ€¦</span>
      </div>
    {:else if filteredItems.length === 0}
      <div class="py-14 text-center text-muted-foreground text-xs border border-dashed border-border rounded-xl">
        {#if search}
          <p>No assets matched "{search}".</p>
        {:else}
          <p>No media assets found in library. Click "Upload Asset" above.</p>
        {/if}
      </div>
    {:else}
      <div class="grid grid-cols-3 sm:grid-cols-4 gap-2.5 max-h-[360px] overflow-y-auto pr-1">
        {#each filteredItems as item (item.id)}
          {@const isImg = isImage(item)}
          {@const name = getName(item)}
          {@const isSelected = selectedId === item.id}
          <button
            type="button"
            class="group relative aspect-square rounded-xl overflow-hidden border transition-all bg-muted/30 flex flex-col items-center justify-center text-center p-1 cursor-pointer shadow-xs {isSelected ? 'border-primary ring-2 ring-primary/30' : 'border-border hover:border-foreground/30'}"
            onclick={() => { selectedId = item.id; }}
          >
            {#if isImg}
              <img
                src={getUrl(item)}
                alt={name}
                class="w-full h-full object-cover rounded-lg"
              />
            {:else}
              <div class="flex flex-col items-center justify-center p-2 text-muted-foreground gap-1">
                <svg class="size-6 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path>
                  <polyline points="14 2 14 8 20 8"></polyline>
                </svg>
                <span class="text-[10px] text-muted-foreground truncate max-w-[80px] font-mono">{name}</span>
              </div>
            {/if}

            {#if isSelected}
              <div class="absolute top-1.5 right-1.5 size-5 rounded-full bg-primary text-primary-foreground flex items-center justify-center shadow-xs">
                <svg class="size-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                  <polyline points="20 6 9 17 4 12"></polyline>
                </svg>
              </div>
            {/if}

            <div class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/85 via-black/40 to-transparent p-1.5 opacity-0 group-hover:opacity-100 transition-opacity">
              <p class="text-[10px] text-white truncate text-left">{name}</p>
            </div>
          </button>
        {/each}
      </div>
    {/if}

    <!-- Selection Bar -->
    {#if selectedItem}
      <div class="flex items-center justify-between p-2 rounded-lg bg-muted/40 border border-border text-xs">
        <span class="truncate font-medium text-foreground max-w-[280px]">
          Selected: {getName(selectedItem)}
        </span>
        <button
          type="button"
          class="text-xs text-muted-foreground hover:text-foreground cursor-pointer"
          onclick={() => { selectedId = null; }}
        >
          Deselect
        </button>
      </div>
    {/if}
  </div>

  {#snippet footer()}
    <Button variant="ghost" size="sm" onclick={onclose}>Cancel</Button>
    <Button
      variant="primary"
      size="sm"
      disabled={!selectedId}
      onclick={confirmSelection}
    >
      Choose Asset
    </Button>
  {/snippet}
</Modal>
