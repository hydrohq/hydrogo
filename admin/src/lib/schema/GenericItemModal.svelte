<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import RichTextEditor from '$lib/components/editor/RichTextEditor.svelte';
  import JsonFieldEditor from '$lib/components/editor/JsonFieldEditor.svelte';
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

  function isJsonField(field: any, key: string, val: any): boolean {
    if (field?.kind === 'json' || field?.widget === 'json' || field?.kind === 'map' || field?.kind === 'object') return true;
    if (key.includes('json') || key === 'votes_data' || key.endsWith('_data') || key.endsWith('_map')) return true;
    if (field?.label && (field.label.toLowerCase().includes('(json)') || field.label.toLowerCase().includes('json'))) return true;
    if (typeof val === 'object' && val !== null) return true;
    if (typeof val === 'string' && val.trim().length > 1) {
      const t = val.trim();
      if ((t.startsWith('{') && t.endsWith('}')) || (t.startsWith('[') && t.endsWith(']'))) {
        try {
          JSON.parse(t);
          return true;
        } catch {}
      }
    }
    return false;
  }

  function initFormData(data: Record<string, any> = {}) {
    const initialized: Record<string, any> = {};
    if (collection?.fields) {
      for (const field of collection.fields) {
        const key = getFieldKey(field);
        if (data[key] !== undefined && data[key] !== null) {
          initialized[key] = data[key];
        } else if (field.kind === 'boolean') {
          initialized[key] = false;
        } else if (field.kind === 'number') {
          initialized[key] = 0;
        } else {
          initialized[key] = '';
        }
      }
    }
    for (const [k, v] of Object.entries(data)) {
      if (initialized[k] === undefined) {
        initialized[k] = v;
      }
    }
    return initialized;
  }

  async function loadItem() {
    if (isNew || !collection || !itemId) {
      formData = initFormData({});
      return;
    }
    loading = true;
    errorMessage = null;
    try {
      const endpoint = collection.api_path || `/api/v1/admin/content/${type}`;
      const res = await api.get<ApiResponse<Record<string, any>>>(`${endpoint}/${itemId}`);
      if (res && res.data) {
        formData = initFormData(res.data);
      } else {
        formData = initFormData({});
      }
    } catch (err: any) {
      errorMessage = err?.message || t('schema.err.loadItem');
      formData = initFormData({});
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

  // State machine: the steps allowed from the record's current state.
  const machine = $derived(collection?.state_machine);
  const currentState = $derived(machine ? String(formData[machine.field] || machine.initial) : '');
  const steps = $derived(machine && !isNew ? machine.transitions.filter((step) => step.from === currentState) : []);

  async function changeState(to: string) {
    if (!collection || !itemId) return;
    saving = true;
    errorMessage = null;
    try {
      const endpoint = collection.api_path || `/api/v1/admin/content/${type}`;
      await api.post(`${endpoint}/${itemId}/state`, { to, version: Number(formData.version) || 0 });
      await loadItem();
      onsaved();
    } catch (err) {
      errorMessage = err instanceof ApiClientError
        ? `${err.message}${err.fields?._rule ? `: ${err.fields._rule}` : ''}${err.fields?._state ? `: ${err.fields._state}` : ''}${err.requestId ? ` (${t('schema.requestId')}: ${err.requestId})` : ''}`
        : t('schema.err.saveItem');
    } finally {
      saving = false;
    }
  }

  async function handleSave(statusToSet?: string) {
    if (!collection) return;
    saving = true;
    errorMessage = null;

    try {
      const payload: Record<string, any> = { ...formData };
      if (statusToSet) {
        payload.status = statusToSet;
      }

      if (collection.fields) {
        for (const f of collection.fields) {
          const key = getFieldKey(f);
          if (f.kind === 'number' && payload[key] !== undefined && payload[key] !== '') {
            payload[key] = Number(payload[key]);
          } else if (f.kind === 'boolean') {
            payload[key] = Boolean(payload[key]);
          } else if (typeof payload[key] === 'object' && payload[key] !== null) {
            payload[key] = JSON.stringify(payload[key]);
          }
        }
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
      errorMessage = err?.message || t('schema.err.saveItem');
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
    ? t('schema.create', { name: collection?.singular || collection?.label || type })
    : t('schema.edit', { name: collection?.singular || collection?.label || type })}
  description={collection?.description || t('schema.configure', { name: collection?.label || type })}
  size="2xl"
  {onclose}
>
  {#if loading}
    <div class="py-16 flex flex-col items-center justify-center text-muted-foreground text-xs gap-2">
      <div class="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
      <span>{t('schema.loadingRecord')}</span>
    </div>
  {:else if !collection}
    <div class="p-8 text-center text-xs text-muted-foreground">
      {t('schema.schemaNotFound')}
    </div>
  {:else}
    <form onsubmit={(e) => { e.preventDefault(); handleSave(); }} class="space-y-4">
      {#if errorMessage}
        <div class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive flex items-center justify-between">
          <span>{errorMessage}</span>
          <button type="button" class="hover:underline cursor-pointer" onclick={() => { errorMessage = null; }}>{t('common.dismiss')}</button>
        </div>
      {/if}

      {#each collection.fields as field (getFieldKey(field))}
        {@const key = getFieldKey(field)}
        <div class="flex flex-col gap-1.5">
          {#if field.read_only}
            <!-- Set by the server (sequence numbers, computed values, states): shown, never edited. -->
            <span class="text-xs font-medium text-foreground">{field.label || key}</span>
            <div class="h-8 flex items-center rounded-md border border-input bg-muted/40 px-3 text-xs text-muted-foreground" data-readonly-field={key}>
              {formData[key] === undefined || formData[key] === null || formData[key] === '' ? (isNew ? t('schema.setByServer') : '—') : String(formData[key])}
            </div>
          {:else if isJsonField(field, key, formData[key])}
            <JsonFieldEditor
              label={field.label || key}
              bind:value={formData[key]}
              required={field.required}
            />
          {:else if field.kind === 'text' && field.widget !== 'textarea'}
            <Input
              label={field.label || key}
              placeholder={t('schema.enter', { name: field.label || key })}
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
              <option value="">{t('schema.selectOption')}</option>
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
              placeholder={t('schema.enter', { name: field.label || key })}
              class="w-full rounded-md border border-input bg-transparent px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
            ></textarea>
          {:else if field.kind === 'richtext' || field.widget === 'richtext'}
            <RichTextEditor
              label={field.label || key}
              bind:value={formData[key]}
              placeholder={t('schema.write', { name: field.label || key })}
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
                  placeholder={t('schema.urlPlaceholder')}
                  bind:value={formData[key]}
                  class="h-8 flex-1 rounded-md border border-input bg-background px-3 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
                />
                <button
                  type="button"
                  class="h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs whitespace-nowrap"
                  onclick={() => openMediaPicker(key)}
                >
                  {t('schema.selectMedia')}
                </button>
              </div>
            </div>
          {:else}
            <Input
              label={field.label || key}
              placeholder={t('schema.enter', { name: field.label || key })}
              bind:value={formData[key]}
            />
          {/if}
        </div>
      {/each}
    </form>
  {/if}

  {#snippet footer()}
    <Button variant="ghost" size="sm" onclick={onclose}>
      {t('common.cancel')}
    </Button>
    {#each steps as step (step.to)}
      <Button variant="secondary" size="sm" disabled={saving || loading} onclick={() => changeState(step.to)}>
        {t('schema.moveTo', { state: step.to })}
      </Button>
    {/each}
    {#if collection?.workflow?.enabled || collection?.drafts}
      <Button
        variant="secondary"
        size="sm"
        disabled={saving || loading}
        onclick={() => handleSave('draft')}
      >
        {t('news.editor.saveDraft')}
      </Button>
      <Button
        variant="primary"
        size="sm"
        loading={saving}
        disabled={loading}
        onclick={() => handleSave('published')}
      >
        {t('pages.editor.publish')}
      </Button>
    {:else}
      <Button
        variant="primary"
        size="sm"
        loading={saving}
        disabled={loading}
        onclick={() => handleSave()}
      >
        {isNew ? t('schema.createItem') : t('media.detail.save')}
      </Button>
    {/if}
  {/snippet}
</Modal>
