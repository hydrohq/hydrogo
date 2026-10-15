<script lang="ts">
  import { api } from '$lib/api/client';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';

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

  interface Props {
    asset: MediaItem | null;
    folders: Folder[];
    open: boolean;
    onclose: () => void;
    onupdated: () => void;
  }

  let { asset, folders, open, onclose, onupdated }: Props = $props();

  let folderId = $state('');
  let visibility = $state<'public' | 'private'>('public');
  let tagsStr = $state('');
  let saving = $state(false);
  let deleteLoading = $state(false);
  let showDeleteConfirm = $state(false);
  let copied = $state(false);
  let errorMessage = $state<string | null>(null);

  $effect(() => {
    if (asset) {
      folderId = asset.folder_id || '';
      visibility = asset.visibility || 'public';
      tagsStr = (asset.tags || []).join(', ');
      errorMessage = null;
    }
  });

  function formatBytes(bytes?: number): string {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
  }

  function copyUrl() {
    if (!asset) return;
    const fullUrl = `${window.location.origin}${asset.url || `/media/${asset.id}`}`;
    navigator.clipboard.writeText(fullUrl);
    copied = true;
    setTimeout(() => { copied = false; }, 2000);
  }

  async function handleSave() {
    if (!asset) return;
    saving = true;
    errorMessage = null;
    try {
      const tags = tagsStr.split(',').map(t => t.trim()).filter(Boolean);
      await api.patch(`/api/v1/admin/media/${asset.id}`, {
        folder_id: folderId || undefined,
        visibility,
        tags,
      });
      onupdated();
      onclose();
    } catch (e: any) {
      errorMessage = e?.message || 'Failed to save changes';
    } finally {
      saving = false;
    }
  }

  async function confirmDelete() {
    if (!asset) return;
    deleteLoading = true;
    errorMessage = null;
    try {
      await api.delete(`/api/v1/admin/trash/media/${asset.id}`);
      showDeleteConfirm = false;
      onupdated();
      onclose();
    } catch (e: any) {
      errorMessage = e?.message || 'Failed to delete asset';
    } finally {
      deleteLoading = false;
    }
  }
</script>

<ConfirmDialog
  open={showDeleteConfirm}
  title="Delete Media Asset"
  description={`Permanently delete "${asset?.original_filename || asset?.id}"? This file will be removed from storage and any pages referencing its URL will show broken links.`}
  confirmLabel="Delete Asset"
  variant="destructive"
  loading={deleteLoading}
  onconfirm={confirmDelete}
  oncancel={() => { if (!deleteLoading) showDeleteConfirm = false; }}
/>

<Modal
  {open}
  title={asset?.original_filename || 'Asset Details'}
  description={`ID: ${asset?.id || ''}`}
  {onclose}
>
  {#if asset}
    <div class="space-y-4">
      {#if errorMessage}
        <div class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive flex items-center justify-between">
          <span>{errorMessage}</span>
          <button
            type="button"
            class="text-xs hover:underline cursor-pointer"
            onclick={() => { errorMessage = null; }}
          >
            Dismiss
          </button>
        </div>
      {/if}

      <!-- Media Preview Frame -->
      <div class="w-full aspect-video rounded-xl bg-muted/60 border border-border overflow-hidden flex items-center justify-center relative group">
        {#if asset.mime_type?.startsWith('image/')}
          <img
            src={asset.url || `/media/${asset.id}`}
            alt={asset.original_filename}
            class="max-w-full max-h-full object-contain"
          />
        {:else if asset.mime_type?.startsWith('video/')}
          <video
            src={asset.url || `/media/${asset.id}`}
            controls
            class="max-w-full max-h-full"
          >
            <track kind="captions" />
          </video>
        {:else}
          <div class="flex flex-col items-center gap-2 text-muted-foreground p-6">
            <svg class="size-12 text-muted-foreground/60" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path>
              <polyline points="14 2 14 8 20 8"></polyline>
            </svg>
            <span class="text-xs font-mono font-medium">{asset.mime_type || 'Unknown Type'}</span>
          </div>
        {/if}

        <a
          href={asset.url || `/media/${asset.id}`}
          target="_blank"
          rel="noopener noreferrer"
          class="absolute top-2 right-2 size-8 rounded-lg bg-background/80 backdrop-blur-xs border border-border inline-flex items-center justify-center text-foreground opacity-0 group-hover:opacity-100 transition-opacity hover:bg-background"
          title="Open original"
        >
          <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
            <polyline points="15 3 21 3 21 9"></polyline>
            <line x1="10" y1="14" x2="21" y2="3"></line>
          </svg>
        </a>
      </div>

      <!-- Quick info & Link Copy -->
      <div class="flex items-center justify-between p-3 rounded-lg bg-muted/40 border border-border text-xs">
        <div class="space-x-2 text-muted-foreground">
          <span class="font-medium text-foreground">{formatBytes(asset.size_bytes)}</span>
          <span>•</span>
          <span class="font-mono text-[11px]">{asset.mime_type}</span>
          {#if asset.created_at}
            <span>•</span>
            <span>{new Date(asset.created_at).toLocaleDateString()}</span>
          {/if}
        </div>
        <button
          type="button"
          class="h-7 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs inline-flex items-center gap-1.5"
          onclick={copyUrl}
        >
          {#if copied}
            <span class="text-emerald-500 font-semibold">✓ Copied URL</span>
          {:else}
            <span>Copy Direct URL</span>
          {/if}
        </button>
      </div>

      <!-- Settings / Metadata Form -->
      <div class="space-y-3 pt-1">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <label for="asset-folder-select" class="text-xs font-medium text-foreground block mb-1">
              Folder
            </label>
            <select
              id="asset-folder-select"
              bind:value={folderId}
              class="w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
            >
              <option value="">No folder (Root)</option>
              {#each folders as f (f.id)}
                <option value={f.id}>{f.name}</option>
              {/each}
            </select>
          </div>

          <div>
            <label for="asset-visibility-select" class="text-xs font-medium text-foreground block mb-1">
              Visibility
            </label>
            <select
              id="asset-visibility-select"
              bind:value={visibility}
              class="w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
            >
              <option value="public">Public (Direct URL accessible)</option>
              <option value="private">Private (Restricted token needed)</option>
            </select>
          </div>
        </div>

        <Input
          label="Tags (comma-separated)"
          placeholder="e.g. hero, marketing, logo, avatar"
          bind:value={tagsStr}
        />
      </div>
    </div>
  {/if}

  {#snippet footer()}
    <Button
      variant="danger"
      size="sm"
      onclick={() => { showDeleteConfirm = true; }}
    >
      Delete Asset
    </Button>
    <div class="flex-1"></div>
    <Button variant="ghost" size="sm" onclick={onclose}>
      Cancel
    </Button>
    <Button variant="primary" size="sm" loading={saving} onclick={handleSave}>
      Save Changes
    </Button>
  {/snippet}
</Modal>
