<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { api } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import { router } from '$lib/router.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Input from '$lib/components/ui/Input.svelte';
  import RichTextEditor from '$lib/components/editor/RichTextEditor.svelte';
  import MediaPickerModal from '$lib/components/media/MediaPickerModal.svelte';
  import { createPresence } from '$lib/presence.svelte';
  import DocumentPresence from '$lib/components/presence/DocumentPresence.svelte';

  interface Props {
    type: string;
    id?: string;
  }

  let { type, id }: Props = $props();

  const isNew = $derived(!id || id === 'new');
  const collection = $derived(schema.getCollection(type));

  let formData = $state<Record<string, any>>({});
  let loading = $state(false);
  let saving = $state(false);
  let errorMessage = $state<string | null>(null);
  let successMessage = $state<string | null>(null);
  let mediaPickerOpen = $state(false);
  let activeMediaField = $state<string | null>(null);
  let initialSnapshot = $state('');

  const presence = createPresence(
    () => `/api/v1/admin/content/${type}/${id}/presence`,
    () => (isNew ? null : id)
  );

  function captureSnapshot() {
    initialSnapshot = JSON.stringify(formData);
  }

  const isDirty = $derived.by(() => {
    if (loading || saving || !initialSnapshot) return false;
    return JSON.stringify(formData) !== initialSnapshot;
  });

  $effect(() => {
    router.setDirty(
      isDirty,
      'İçerikte kaydedilmemiş değişiklikleriniz var. Kaydetmeden çıkmak istediğinizden emin misiniz?'
    );
  });

  onDestroy(() => {
    router.setDirty(false);
  });

  async function loadData() {
    if (isNew || !collection) {
      captureSnapshot();
      return;
    }
    loading = true;
    try {
      const res = await api.get<ApiResponse<Record<string, any>>>(
        `${collection.api_path || `/api/v1/admin/content/${type}`}/${id}`
      );
      if (res && res.data) {
        formData = res.data;
        captureSnapshot();
      }
    } catch (err: any) {
      errorMessage = err.message || 'Failed to load entry';
    } finally {
      loading = false;
    }
  }

  async function handleSave(statusToSet?: string) {
    if (!collection) return;
    saving = true;
    errorMessage = null;
    successMessage = null;

    try {
      const payload = { ...formData };
      if (statusToSet) {
        payload.status = statusToSet;
      }

      let res: ApiResponse<any>;
      const endpoint = collection.api_path || `/api/v1/admin/content/${type}`;

      if (isNew) {
        res = await api.post(endpoint, payload);
      } else {
        res = await api.put(`${endpoint}/${id}`, payload);
      }

      captureSnapshot();
      router.setDirty(false);
      successMessage = 'Saved successfully!';
      setTimeout(() => { successMessage = null; }, 3000);

      if (isNew && res && res.data && res.data.id) {
        router.navigate(`/content/${type}/${res.data.id}`, true);
      }
    } catch (err: any) {
      errorMessage = err.message || 'Save failed';
    } finally {
      saving = false;
    }
  }

  function getFieldKey(f: any): string {
    return f.id || f.name || '';
  }

  function openMediaPicker(fieldName: string) {
    activeMediaField = fieldName;
    mediaPickerOpen = true;
  }

  function handleMediaSelect(media: any) {
    if (activeMediaField) {
      formData[activeMediaField] = media.path || media.id;
    }
  }

  onMount(() => {
    loadData();
  });
</script>

<MediaPickerModal
  open={mediaPickerOpen}
  onclose={() => { mediaPickerOpen = false; }}
  onselect={handleMediaSelect}
/>

{#if schema.isLoading}
  <div class="p-12 text-center text-xs text-muted-foreground flex flex-col items-center justify-center gap-2">
    <div class="size-6 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
    <span>Loading {type} schema...</span>
  </div>
{:else if !collection}
  <div class="p-12 text-center text-xs text-muted-foreground">
    <p>Content type <strong>{type}</strong> not found in schema.</p>
    <button
      type="button"
      class="mt-4 inline-flex items-center gap-1.5 h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer"
      onclick={() => router.navigate('/dashboard')}
    >
      Back to Dashboard
    </button>
  </div>
{:else}
  <div class="flex flex-1 flex-col">
    <div class="@container/main flex flex-1 flex-col gap-2">
      <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
        <!-- Top Toolbar -->
        <div class="flex items-center justify-between px-4 lg:px-6">
          <div class="flex items-center gap-2">
            <button
              type="button"
              onclick={() => router.navigate(`/content/${type}`)}
              class="h-8 px-2.5 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-muted-foreground hover:text-foreground transition-colors cursor-pointer flex items-center gap-1.5"
            >
              <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="m15 18-6-6 6-6" />
              </svg>
              <span>Back</span>
            </button>
            <span class="text-sm font-semibold text-foreground">
              {isNew ? `New ${collection.singular || collection.label || type}` : `Edit ${collection.singular || collection.label || type}`}
            </span>
            {#if !isNew}
              <DocumentPresence {presence} mode="pill" />
            {/if}
            {#if loading}
              <span class="text-xs text-muted-foreground animate-pulse">Loading...</span>
            {/if}
          </div>

          <div class="flex items-center gap-2">
            {#if collection.workflow?.enabled}
              <button
                type="button"
                disabled={saving}
                class="h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs"
                onclick={() => handleSave('draft')}
              >
                Save Draft
              </button>
              <button
                type="button"
                disabled={saving}
                class="h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium transition-colors cursor-pointer shadow-xs"
                onclick={() => handleSave('published')}
              >
                Publish
              </button>
            {:else}
              <button
                type="button"
                disabled={saving}
                class="h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium transition-colors cursor-pointer shadow-xs"
                onclick={() => handleSave()}
              >
                Save Changes
              </button>
            {/if}
          </div>
        </div>

        {#if !isNew}
          <DocumentPresence {presence} mode="banner" />
        {/if}

        {#if errorMessage}
          <div class="mx-4 lg:mx-6 p-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs">
            {errorMessage}
          </div>
        {/if}
        {#if successMessage}
          <div class="mx-4 lg:mx-6 p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs">
            {successMessage}
          </div>
        {/if}

        <!-- Form Card Container -->
        <div class="px-4 lg:px-6">
          <div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-5 max-w-3xl">
            {#each collection.fields as field (getFieldKey(field))}
              {@const key = getFieldKey(field)}
              <div class="flex flex-col gap-1.5">
                {#if field.kind === 'text' && field.widget !== 'textarea'}
                  <Input
                    label={field.label || key}
                    placeholder={`Enter ${field.label || key}...`}
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
                {:else if field.kind === 'textarea' || field.widget === 'textarea'}
                  <label for={`field-${key}`} class="text-xs font-medium text-foreground">
                    {field.label || key}
                  </label>
                  <textarea
                    id={`field-${key}`}
                    bind:value={formData[key]}
                    rows="4"
                    placeholder={`Enter ${field.label || key}...`}
                    class="w-full rounded-md border border-input bg-transparent px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
                  ></textarea>
                {:else if field.kind === 'richtext' || field.widget === 'richtext'}
                  <RichTextEditor
                    label={field.label || key}
                    bind:value={formData[key]}
                  />
                {:else if field.kind === 'boolean' || field.widget === 'checkbox'}
                  <label class="flex items-center gap-2 cursor-pointer select-none">
                    <input
                      type="checkbox"
                      bind:checked={formData[key]}
                      class="size-4 rounded border-input text-primary focus:ring-ring"
                    />
                    <span class="text-xs font-medium text-foreground">{field.label || key}</span>
                  </label>
                {:else if field.kind === 'select' || field.widget === 'select'}
                  <label for={`field-${key}`} class="text-xs font-medium text-foreground">
                    {field.label || key}
                  </label>
                  <select
                    id={`field-${key}`}
                    bind:value={formData[key]}
                    class="w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
                  >
                    <option value="">Select option...</option>
                    {#each (field.options || []) as opt}
                      {@const optVal = typeof opt === 'string' ? opt : opt.value}
                      {@const optLabel = typeof opt === 'string' ? opt : opt.label}
                      <option value={optVal}>{optLabel}</option>
                    {/each}
                  </select>
                {:else if field.kind === 'date' || field.kind === 'datetime'}
                  <Input
                    type={field.kind === 'date' ? 'date' : 'datetime-local'}
                    label={field.label || key}
                    bind:value={formData[key]}
                  />
                {:else if field.kind === 'media' || field.widget === 'media'}
                  <span class="text-xs font-medium text-foreground">
                    {field.label || key}
                  </span>
                  <div class="flex items-center gap-2">
                    {#if formData[key]}
                      <div class="relative size-12 rounded-md overflow-hidden border border-border bg-muted shrink-0">
                        <img src={formData[key]} alt="Selected asset" class="w-full h-full object-cover" />
                      </div>
                    {/if}
                    <Input
                      placeholder="Asset URL or path..."
                      bind:value={formData[key]}
                      class="flex-1 font-mono text-xs"
                    />
                    <button
                      type="button"
                      class="h-8 px-2.5 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs shrink-0"
                      onclick={() => openMediaPicker(key)}
                    >
                      Browse
                    </button>
                  </div>
                {:else}
                  <Input
                    label={field.label || key}
                    bind:value={formData[key]}
                  />
                {/if}
              </div>
            {/each}
          </div>
        </div>
      </div>
    </div>
  </div>
{/if}
