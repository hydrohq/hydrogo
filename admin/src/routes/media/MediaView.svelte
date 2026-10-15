<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import MediaUploadDropzone from './MediaUploadDropzone.svelte';
  import MediaDetailModal from './MediaDetailModal.svelte';

  interface MediaItem {
    id: string;
    original_filename?: string;
    filename?: string;
    mime_type?: string;
    size_bytes?: number;
    url?: string;
    folder_id?: string;
    visibility?: 'public' | 'private';
    tags?: string[];
    created_at?: string;
  }

  interface Folder {
    id: string;
    name: string;
  }

  let assets = $state<MediaItem[]>([]);
  let folders = $state<Folder[]>([]);
  let loading = $state(true);
  let uploading = $state(false);

  let search = $state('');
  let selectedFolder = $state<string>('');
  let selectedKind = $state<'all' | 'image' | 'document'>('all');

  let selectedAsset = $state<MediaItem | null>(null);
  let detailOpen = $state(false);

  let newFolderOpen = $state(false);
  let newFolderName = $state('');
  let creatingFolder = $state(false);

  async function loadFolders() {
    try {
      const res = await api.get<ApiResponse<Folder[]>>('/api/v1/admin/media/folders');
      if (res && res.data) {
        folders = res.data;
      }
    } catch (e) {
      console.warn('Folders load error:', e);
    }
  }

  async function loadAssets() {
    loading = true;
    try {
      const res = await api.get<ApiResponse<MediaItem[]>>('/api/v1/admin/media', {
        params: {
          q: search || undefined,
          folder: selectedFolder || undefined,
          kind: selectedKind === 'all' ? undefined : selectedKind,
          limit: 100,
        },
      });
      if (res && res.data) {
        assets = Array.isArray(res.data) ? res.data : [];
      }
    } catch (e) {
      console.warn('Assets load error:', e);
      assets = [];
    } finally {
      loading = false;
    }
  }

  async function handleUploadFiles(files: FileList) {
    uploading = true;
    try {
      for (let i = 0; i < files.length; i++) {
        const formData = new FormData();
        formData.append('file', files[i]);
        if (selectedFolder) {
          formData.append('folder_id', selectedFolder);
        }
        await api.upload('/api/v1/admin/media', formData);
      }
      await loadAssets();
    } catch (e: any) {
      alert(`Upload failed: ${e.message}`);
    } finally {
      uploading = false;
    }
  }

  async function handleCreateFolder() {
    if (!newFolderName.trim()) return;
    creatingFolder = true;
    try {
      await api.post('/api/v1/admin/media/folders', { name: newFolderName.trim() });
      newFolderName = '';
      newFolderOpen = false;
      await loadFolders();
    } catch (e: any) {
      alert(`Create folder failed: ${e.message}`);
    } finally {
      creatingFolder = false;
    }
  }

  function openDetail(asset: MediaItem) {
    selectedAsset = asset;
    detailOpen = true;
  }

  function formatBytes(bytes?: number): string {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
  }

  onMount(async () => {
    await Promise.all([loadFolders(), loadAssets()]);
  });
</script>

<MediaDetailModal
  open={detailOpen}
  asset={selectedAsset}
  {folders}
  onclose={() => { detailOpen = false; }}
  onupdated={() => { loadAssets(); }}
/>

<Modal
  open={newFolderOpen}
  title="New Folder"
  description="Create a folder to organise your media files."
  onclose={() => { newFolderOpen = false; }}
>
  <form onsubmit={(e) => { e.preventDefault(); handleCreateFolder(); }}>
    <Input
      label="Folder Name"
      placeholder="e.g. Products, Blog, Team"
      bind:value={newFolderName}
      required
      autofocus
    />
  </form>
  {#snippet footer()}
    <Button variant="ghost" size="sm" onclick={() => { newFolderOpen = false; }}>Cancel</Button>
    <Button variant="primary" size="sm" loading={creatingFolder} onclick={handleCreateFolder}>
      Create Folder
    </Button>
  {/snippet}
</Modal>

<div class="space-y-6 max-w-6xl mx-auto">
  <!-- Upload Dropzone -->
  <MediaUploadDropzone
    {uploading}
    onupload={handleUploadFiles}
  />

  <!-- Controls: Folders, Filter, Search -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 pb-3 border-b border-border">
    <!-- Folder pills -->
    <div class="flex flex-wrap items-center gap-1.5">
      <button
        type="button"
        class="px-3 py-1.5 rounded-lg text-xs font-medium transition-colors {selectedFolder === '' ? 'bg-primary/15 text-primary border border-primary/30 font-semibold' : 'text-muted-foreground hover:text-foreground bg-muted/60 border border-border'}"
        onclick={() => { selectedFolder = ''; loadAssets(); }}
      >
        All Files
      </button>
      {#each folders as f (f.id)}
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg text-xs font-medium transition-colors {selectedFolder === f.id ? 'bg-primary/15 text-primary border border-primary/30 font-semibold' : 'text-muted-foreground hover:text-foreground bg-muted/60 border border-border'}"
          onclick={() => { selectedFolder = f.id; loadAssets(); }}
        >
          📁 {f.name}
        </button>
      {/each}
      <button
        type="button"
        class="px-2.5 py-1.5 rounded-lg text-xs font-medium text-muted-foreground hover:text-foreground border border-dashed border-border hover:border-muted-foreground transition-colors"
        onclick={() => { newFolderOpen = true; }}
      >
        + Folder
      </button>
    </div>

    <!-- Kind & Search -->
    <div class="flex items-center gap-2.5 w-full sm:w-auto">
      <div class="flex items-center rounded-lg bg-muted border border-border p-0.5 text-xs">
        <button
          type="button"
          class="px-2.5 py-1 rounded-md transition-colors {selectedKind === 'all' ? 'bg-background text-foreground shadow-sm font-medium' : 'text-muted-foreground hover:text-foreground'}"
          onclick={() => { selectedKind = 'all'; loadAssets(); }}
        >
          All
        </button>
        <button
          type="button"
          class="px-2.5 py-1 rounded-md transition-colors {selectedKind === 'image' ? 'bg-background text-foreground shadow-sm font-medium' : 'text-muted-foreground hover:text-foreground'}"
          onclick={() => { selectedKind = 'image'; loadAssets(); }}
        >
          Images
        </button>
        <button
          type="button"
          class="px-2.5 py-1 rounded-md transition-colors {selectedKind === 'document' ? 'bg-background text-foreground shadow-sm font-medium' : 'text-muted-foreground hover:text-foreground'}"
          onclick={() => { selectedKind = 'document'; loadAssets(); }}
        >
          Docs
        </button>
      </div>

      <input
        type="search"
        placeholder="Filter assets…"
        bind:value={search}
        oninput={() => loadAssets()}
        class="bg-background border border-input rounded-md px-3 py-1.5 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring w-36 sm:w-44 shadow-sm"
      />
    </div>
  </div>

  <!-- Assets Grid -->
  {#if loading}
    <div class="py-16 text-center text-slate-500 text-sm">
      <span class="animate-spin inline-block mr-2">⏳</span> Loading media assets…
    </div>
  {:else if assets.length === 0}
    <div class="py-16 text-center text-slate-400 text-sm border-2 border-dashed border-slate-800 rounded-2xl">
      No assets found in this view.
    </div>
  {:else}
    <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
      {#each assets as asset (asset.id)}
        <button
          type="button"
          class="group relative aspect-square rounded-xl bg-slate-900/60 border border-slate-800/80 hover:border-sky-500/60 overflow-hidden flex flex-col justify-between p-2.5 transition-all duration-150 cursor-pointer text-left hover:shadow-lg hover:shadow-sky-950/20"
          onclick={() => openDetail(asset)}
        >
          <!-- Asset Preview -->
          <div class="w-full flex-1 flex items-center justify-center overflow-hidden rounded-lg bg-slate-950">
            {#if asset.mime_type?.startsWith('image/')}
              <img
                src={asset.url || `/media/${asset.id}`}
                alt={asset.original_filename}
                class="w-full h-full object-cover transition-transform duration-200 group-hover:scale-105"
                loading="lazy"
              />
            {:else}
              <div class="flex flex-col items-center gap-1 text-slate-500">
                <span class="text-3xl">📄</span>
                <span class="text-[9px] uppercase font-mono tracking-wider">{asset.mime_type?.split('/')[1] || 'DOC'}</span>
              </div>
            {/if}
          </div>

          <!-- Filename & Size -->
          <div class="pt-2">
            <p class="text-xs font-medium text-slate-200 truncate group-hover:text-sky-400 transition-colors">
              {asset.original_filename || asset.filename || asset.id}
            </p>
            <p class="text-[10px] text-slate-500 mt-0.5">
              {formatBytes(asset.size_bytes)}
            </p>
          </div>
        </button>
      {/each}
    </div>
  {/if}
</div>
