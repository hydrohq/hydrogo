<script lang="ts">
  import { api } from '$lib/api/client';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';

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
  let copied = $state(false);

  $effect(() => {
    if (asset) {
      folderId = asset.folder_id || '';
      visibility = asset.visibility || 'public';
      tagsStr = (asset.tags || []).join(', ');
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
    try {
      const tags = tagsStr.split(',').map(t => t.trim()).filter(Boolean);
      await api.post(`/api/v1/admin/media/${asset.id}`, {
        folder_id: folderId || undefined,
        visibility,
        tags,
      }, { method: 'PATCH' });
      onupdated();
      onclose();
    } catch (e: any) {
      alert(`Save failed: ${e.message}`);
    } finally {
      saving = false;
    }
  }

  async function handleDelete() {
    if (!asset || !confirm(`Delete "${asset.original_filename || asset.id}" permanently?`)) return;
    try {
      await api.delete(`/api/v1/admin/media/${asset.id}`);
      onupdated();
      onclose();
    } catch (e: any) {
      alert(`Delete failed: ${e.message}`);
    }
  }
</script>

<Modal
  {open}
  title={asset?.original_filename || 'Asset Details'}
  description={`ID: ${asset?.id || ''}`}
  {onclose}
>
  {#if asset}
    <div class="space-y-5">
      <!-- Preview -->
      <div class="w-full aspect-video rounded-xl bg-slate-950 border border-slate-800 overflow-hidden flex items-center justify-center relative">
        {#if asset.mime_type?.startsWith('image/')}
          <img
            src={asset.url || `/media/${asset.id}`}
            alt={asset.original_filename}
            class="max-w-full max-h-full object-contain"
          />
        {:else}
          <div class="flex flex-col items-center gap-2 text-slate-500">
            <span class="text-4xl">📄</span>
            <span class="text-xs font-mono">{asset.mime_type}</span>
          </div>
        {/if}
      </div>

      <!-- Quick info & Link Copy -->
      <div class="flex items-center justify-between p-3 rounded-lg bg-slate-950/80 border border-slate-800 text-xs">
        <div class="space-x-3 text-slate-400">
          <span>{formatBytes(asset.size_bytes)}</span>
          <span>•</span>
          <span>{asset.mime_type}</span>
        </div>
        <Button variant="outline" size="sm" onclick={copyUrl}>
          {copied ? '✓ Copied Link' : 'Copy Public URL'}
        </Button>
      </div>

      <!-- Settings / Metadata -->
      <div class="space-y-3">
        <div>
          <label for="asset-folder-select" class="text-xs font-semibold uppercase tracking-wider text-slate-300">
            Folder
          </label>
          <select
            id="asset-folder-select"
            bind:value={folderId}
            class="w-full mt-1 rounded-lg bg-slate-900 border border-slate-800 px-3 py-2 text-sm text-slate-100 focus:outline-none focus:border-sky-500"
          >
            <option value="">No folder (Root)</option>
            {#each folders as f (f.id)}
              <option value={f.id}>{f.name}</option>
            {/each}
          </select>
        </div>

        <div>
          <label for="asset-visibility-select" class="text-xs font-semibold uppercase tracking-wider text-slate-300">
            Visibility
          </label>
          <select
            id="asset-visibility-select"
            bind:value={visibility}
            class="w-full mt-1 rounded-lg bg-slate-900 border border-slate-800 px-3 py-2 text-sm text-slate-100 focus:outline-none focus:border-sky-500"
          >
            <option value="public">Public (Direct URL accessible)</option>
            <option value="private">Private (Restricted)</option>
          </select>
        </div>

        <Input
          label="Tags (comma-separated)"
          placeholder="e.g. hero, marketing, logo"
          bind:value={tagsStr}
        />
      </div>
    </div>
  {/if}

  {#snippet footer()}
    <Button variant="danger" size="sm" onclick={handleDelete}>
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
