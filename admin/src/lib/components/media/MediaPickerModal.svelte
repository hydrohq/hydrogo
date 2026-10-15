<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse, MediaItem } from '$lib/api/types';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';

  interface Props {
    open: boolean;
    onselect: (media: MediaItem) => void;
    onclose: () => void;
  }

  let {
    open = $bindable(false),
    onselect,
    onclose,
  }: Props = $props();

  let mediaItems = $state<MediaItem[]>([]);
  let loading = $state(false);
  let uploading = $state(false);
  let fileInput: HTMLInputElement | null = $state(null);
  let selectedId = $state<string | null>(null);

  async function loadMedia() {
    loading = true;
    try {
      const res = await api.get<ApiResponse<MediaItem[]>>('/api/v1/admin/media');
      if (res && res.data) {
        mediaItems = res.data;
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
      const res = await api.upload<ApiResponse<MediaItem>>('/api/v1/admin/media', formData);
      if (res && res.data) {
        await loadMedia();
        selectedId = res.data.id;
      }
    } catch (e) {
      console.error('Upload failed:', e);
    } finally {
      uploading = false;
    }
  }

  function confirmSelection() {
    const item = mediaItems.find(m => m.id === selectedId);
    if (item) {
      onselect(item);
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
  description="Select an asset from the media library or upload a new one."
  {onclose}
>
  <div class="flex flex-col gap-4">
    <div class="flex items-center justify-between pb-3 border-b border-slate-800">
      <input
        type="file"
        class="hidden"
        bind:this={fileInput}
        onchange={handleFileUpload}
        accept="image/*,video/*,application/pdf"
      />
      <Button
        variant="primary"
        size="sm"
        loading={uploading}
        onclick={() => fileInput?.click()}
      >
        Upload File
      </Button>
      <span class="text-xs text-slate-400">{mediaItems.length} assets</span>
    </div>

    {#if loading}
      <div class="py-12 flex flex-col items-center justify-center text-slate-500 text-sm gap-2">
        <span class="animate-spin text-sky-500 text-xl">⏳</span>
        Loading assets…
      </div>
    {:else if mediaItems.length === 0}
      <div class="py-12 text-center text-slate-400 text-sm border-2 border-dashed border-slate-800 rounded-xl">
        No media assets found. Upload one to get started.
      </div>
    {:else}
      <div class="grid grid-cols-3 sm:grid-cols-4 gap-3 max-h-[360px] overflow-y-auto pr-1">
        {#each mediaItems as item (item.id)}
          <button
            type="button"
            class="group relative aspect-square rounded-xl overflow-hidden border-2 transition-all duration-150 bg-slate-950 flex flex-col items-center justify-center text-center p-2 cursor-pointer {selectedId === item.id ? 'border-sky-500 ring-2 ring-sky-500/30' : 'border-slate-800 hover:border-slate-700'}"
            onclick={() => selectedId = item.id}
          >
            {#if item.content_type?.startsWith('image/')}
              <img
                src={item.path}
                alt={item.name}
                class="w-full h-full object-cover rounded-lg"
              />
            {:else}
              <div class="text-2xl mb-1">📄</div>
              <span class="text-[10px] text-slate-300 truncate w-full px-1">{item.name}</span>
            {/if}
            <div class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/80 via-black/40 to-transparent p-1.5 opacity-0 group-hover:opacity-100 transition-opacity">
              <p class="text-[10px] text-slate-200 truncate">{item.name}</p>
            </div>
          </button>
        {/each}
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
