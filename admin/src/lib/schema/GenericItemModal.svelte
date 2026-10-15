<script lang="ts">
  import { api } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import RichTextEditor from '$lib/components/editor/RichTextEditor.svelte';
  import MediaPickerModal from '$lib/components/media/MediaPickerModal.svelte';

  interface Props {
    open: boolean;
    type: string;
    itemId: string | null;
    onclose: () => void;
    onsaved: () => void;
  }

  let { open = false, type, itemId = null, onclose, onsaved }: Props = $props();

  const collection = $derived(schema.getCollection(type));
  const isNew = $derived(!itemId || itemId === 'new');

  let formData = $state<Record<string, any>>({});
  let loading = $state(false);
  let saving = $state(false);
  let errorMessage = $state<string | null>(null);
  let mediaPickerOpen = $state(false);
  let activeMediaField = $state<string | null>(null);

  function getFieldKey(f: any): string {
    return f.id || f.name || '';
  }

  async function loadItem() {
    if (isNew || !collection || !itemId) {
      formData = {};
      return;
    }
    loading = true;
    errorMessage = null;
    try {
      const endpoint = collection.api_path || `/api/v1/admin/content/${type}`;
      const res = await api.get<ApiResponse<Record<string, any>>>(`${endpoint}/${itemId}`);
      if (res && res.data) {
        formData = { ...res.data };
      }
    } catch (err: any) {
      errorMessage = err?.message || 'Failed to load item';
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    if (open) {
      errorMessage = null;
      loadItem();
    }
  });

  async function handleSave(statusToSet?: string) {
    if (!collection) return;
    saving = true;
    errorMessage = null;

    try {
      const payload = { ...formData };
      if (statusToSet) {
        payload.status = statusToSet;
      }

      const endpoint = collection.api_path || `/api/v1/admin/content/${type}`;

      if (isNew) {
        await api.post(endpoint, payload);
      } else {
        await api.put(`${endpoint}/${itemId}`, payload);
      }

      onsaved();
      onclose();
    } catch (err: any) {
      errorMessage = err?.message || 'Failed to save item';
    } finally {
      saving = false;
    }
  }

  function openMediaPicker(fieldName: string) {
    activeMediaField = fieldName;
    mediaPickerOpen = true;
  }

  function handleMediaSelect(media: any) {
    if (activeMediaField) {
      formData[activeMediaField] = media.url || media.path || `/media/${media.id}`;
    }
    mediaPickerOpen = false;
  }
</script>

<MediaPickerModal
  open={mediaPickerOpen}
  onclose={() => { mediaPickerOpen = false; }}
  onselect={handleMediaSelect}
/>

<Modal
  {open}
  title={isNew
    ? `Create ${collection?.singular || collection?.label || type}`
    : `Edit ${collection?.singular || collection?.label || type}`}
  description={collection?.description || `Configure fields for this ${collection?.label || type} record.`}
  size="2xl"
  {onclose}
>
  {#if loading}
    <div class="py-16 flex flex-col items-center justify-center text-muted-foreground text-xs gap-2">
      <div class="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
      <span>Loading record details…</span>
    </div>
  {:else if !collection}
    <div class="p-8 text-center text-xs text-muted-foreground">
      Content type schema not found.
    </div>
  {:else}
    <form onsubmit={(e) => { e.preventDefault(); handleSave(); }} class="space-y-4">
      {#if errorMessage}
        <div class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive flex items-center justify-between">
          <span>{errorMessage}</span>
          <button type="button" class="hover:underline cursor-pointer" onclick={() => { errorMessage = null; }}>Dismiss</button>
        </div>
      {/if}

      {#each collection.fields as field (getFieldKey(field))}
        {@const key = getFieldKey(field)}
        <div class="flex flex-col gap-1.5">
          {#if field.kind === 'text' && field.widget !== 'textarea'}
            <Input
              label={field.label || key}
              placeholder={`Enter ${field.label || key}…`}
              bind:value={formData[key]}
              required={field.required}
            />
          {:else if field.kind === 'number'}
            <Input
              type="number"
              label={field.label || key}
              bind:value={formData[key]}
              required={field.required}
            />
          {:else if field.kind === 'date'}
            <Input
              type="date"
              label={field.label || key}
              bind:value={formData[key]}
              required={field.required}
            />
          {:else if field.kind === 'select'}
            <label for={`select-${key}`} class="text-xs font-medium text-foreground">
              {field.label || key}
            </label>
            <select
              id={`select-${key}`}
              bind:value={formData[key]}
              class="w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
            >
              <option value="">Select option…</option>
              {#if field.options}
                {#each field.options as opt}
                  <option value={typeof opt === 'string' ? opt : opt.value}>
                    {typeof opt === 'string' ? opt : opt.label || opt.value}
                  </option>
                {/each}
              {/if}
            </select>
          {:else if field.kind === 'boolean'}
            <label class="flex items-center gap-2 cursor-pointer pt-1">
              <input
                type="checkbox"
                bind:checked={formData[key]}
                class="rounded border-input text-primary focus:ring-ring size-4 cursor-pointer"
              />
              <span class="text-xs font-medium text-foreground">{field.label || key}</span>
            </label>
          {:else if field.kind === 'textarea' || field.widget === 'textarea'}
            <label for={`textarea-${key}`} class="text-xs font-medium text-foreground">
              {field.label || key}
            </label>
            <textarea
              id={`textarea-${key}`}
              bind:value={formData[key]}
              rows="3"
              placeholder={`Enter ${field.label || key}…`}
              class="w-full rounded-md border border-input bg-transparent px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
            ></textarea>
          {:else if field.kind === 'richtext' || field.widget === 'richtext'}
            <RichTextEditor
              label={field.label || key}
              bind:value={formData[key]}
              placeholder={`Write content for ${field.label || key}…`}
            />
          {:else if field.kind === 'media'}
            <label for={`media-input-${key}`} class="text-xs font-medium text-foreground">
              {field.label || key}
            </label>
            <div class="flex items-center gap-3">
              {#if formData[key]}
                <div class="size-12 rounded-lg border border-border bg-muted overflow-hidden flex items-center justify-center shrink-0">
                  <img
                    src={formData[key]}
                    alt="Preview"
                    class="size-full object-cover"
                    onerror={(e) => { (e.target as HTMLElement).style.display = 'none'; }}
                  />
                </div>
              {/if}
              <div class="flex-1 flex items-center gap-2">
                <input
                  id={`media-input-${key}`}
                  type="text"
                  placeholder="URL or select from library…"
                  bind:value={formData[key]}
                  class="h-8 flex-1 rounded-md border border-input bg-background px-3 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
                />
                <button
                  type="button"
                  class="h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs whitespace-nowrap"
                  onclick={() => openMediaPicker(key)}
                >
                  Select Media
                </button>
              </div>
            </div>
          {:else}
            <Input
              label={field.label || key}
              placeholder={`Enter ${field.label || key}…`}
              bind:value={formData[key]}
            />
          {/if}
        </div>
      {/each}
    </form>
  {/if}

  {#snippet footer()}
    <Button variant="ghost" size="sm" onclick={onclose}>
      Cancel
    </Button>
    {#if collection?.workflow?.enabled || collection?.drafts}
      <Button
        variant="secondary"
        size="sm"
        disabled={saving || loading}
        onclick={() => handleSave('draft')}
      >
        Save Draft
      </Button>
      <Button
        variant="primary"
        size="sm"
        loading={saving}
        disabled={loading}
        onclick={() => handleSave('published')}
      >
        Publish
      </Button>
    {:else}
      <Button
        variant="primary"
        size="sm"
        loading={saving}
        disabled={loading}
        onclick={() => handleSave()}
      >
        {isNew ? 'Create Item' : 'Save Changes'}
      </Button>
    {/if}
  {/snippet}
</Modal>
