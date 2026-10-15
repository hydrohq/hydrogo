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
      console.error('Upload failed:', e);
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
      console.error('Create folder failed:', e);
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

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-2">
    <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <!-- Upload Dropzone Container -->
      <div class="px-4 lg:px-6">
        <MediaUploadDropzone
          {uploading}
          onupload={handleUploadFiles}
        />
      </div>

      <!-- Toolbar: Folders / Kind tabs + Search & New Folder -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 px-4 lg:px-6">
        <!-- Kind Tabs -->
        <div class="inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground">
          <button
            type="button"
            class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {selectedKind === 'all' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
            onclick={() => { selectedKind = 'all'; loadAssets(); }}
          >
            All Media
          </button>
          <button
            type="button"
            class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {selectedKind === 'image' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
            onclick={() => { selectedKind = 'image'; loadAssets(); }}
          >
            Images
          </button>
          <button
            type="button"
            class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {selectedKind === 'document' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
            onclick={() => { selectedKind = 'document'; loadAssets(); }}
          >
            Documents
          </button>
        </div>

        <!-- Search & Folder Actions -->
        <div class="flex items-center gap-2">
          <div class="relative w-44 sm:w-56">
            <svg class="absolute left-2.5 top-2.5 size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"></circle>
              <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
            </svg>
            <input
              type="search"
              placeholder="Search files…"
              bind:value={search}
              oninput={() => loadAssets()}
              class="h-8 w-full rounded-md border border-input bg-background pl-8 pr-3 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
            />
          </div>

          <button
            type="button"
            class="inline-flex items-center gap-1.5 h-8 px-2.5 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shrink-0 shadow-xs"
            onclick={() => { newFolderOpen = true; }}
          >
            <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2Z"></path>
              <line x1="12" y1="10" x2="12" y2="16"></line>
              <line x1="9" y1="13" x2="15" y2="13"></line>
            </svg>
            <span>+ Folder</span>
          </button>
        </div>
      </div>

      <!-- Assets Grid Container -->
      <div class="px-4 lg:px-6">
        <div class="rounded-xl border border-border bg-card p-4 shadow-xs">
          {#if loading}
            <div class="py-16 text-center text-xs text-muted-foreground">
              Loading media library…
            </div>
          {:else if assets.length === 0}
            <div class="py-16 text-center text-xs text-muted-foreground flex flex-col items-center justify-center gap-2">
              <svg class="size-8 text-muted-foreground/60" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <rect width="18" height="18" x="3" y="3" rx="2" ry="2"></rect>
                <circle cx="9" cy="9" r="2"></circle>
                <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"></path>
              </svg>
              <p>No media assets found in this view.</p>
            </div>
          {:else}
            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
              {#each assets as asset (asset.id)}
                <button
                  type="button"
                  class="group relative aspect-square rounded-lg bg-background border border-border hover:border-foreground/40 overflow-hidden flex flex-col justify-between p-2 transition-all cursor-pointer text-left shadow-xs hover:shadow-sm"
                  onclick={() => openDetail(asset)}
                >
                  <!-- Asset Preview -->
                  <div class="w-full flex-1 flex items-center justify-center overflow-hidden rounded-md bg-muted/40">
                    {#if asset.mime_type?.startsWith('image/')}
                      <img
                        src={asset.url || `/media/${asset.id}`}
                        alt={asset.original_filename}
                        class="w-full h-full object-cover transition-transform duration-200 group-hover:scale-105"
                        loading="lazy"
                      />
                    {:else}
                      <div class="flex flex-col items-center gap-1 text-muted-foreground">
                        <svg class="size-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                          <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path>
                          <polyline points="14 2 14 8 20 8"></polyline>
                        </svg>
                        <span class="text-[9px] uppercase font-mono tracking-wider">{asset.mime_type?.split('/')[1] || 'DOC'}</span>
                      </div>
                    {/if}
                  </div>

                  <!-- Filename & Size -->
                  <div class="pt-1.5 min-w-0">
                    <p class="text-xs font-medium text-foreground truncate group-hover:text-primary transition-colors">
                      {asset.original_filename || asset.filename || asset.id}
                    </p>
                    <p class="text-[10px] text-muted-foreground">
                      {formatBytes(asset.size_bytes)}
                    </p>
                  </div>
                </button>
              {/each}
            </div>

            <!-- Footer Stats -->
            <div class="mt-4 pt-3 border-t border-border flex items-center justify-between text-xs text-muted-foreground">
              <span>{assets.length} assets total</span>
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>
