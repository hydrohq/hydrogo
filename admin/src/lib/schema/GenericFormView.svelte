<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import { router } from '$lib/router.svelte';
  import type { ApiResponse, Field } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import RichTextEditor from '$lib/components/editor/RichTextEditor.svelte';
  import MediaPickerModal from '$lib/components/media/MediaPickerModal.svelte';

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
  let activeLocale = $state('en');
  let mediaPickerOpen = $state(false);
  let activeMediaField = $state<string | null>(null);

  async function loadData() {
    if (isNew || !collection) return;
    loading = true;
    try {
      const res = await api.get<ApiResponse<Record<string, any>>>(
        `${collection.api_path || `/api/v1/admin/content/${type}`}/${id}`
      );
      if (res && res.data) {
        formData = res.data;
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

      successMessage = 'Saved successfully!';
      setTimeout(() => { successMessage = null; }, 3000);

      if (isNew && res && res.data && res.data.id) {
        router.navigate(`/content/${type}/${res.data.id}`);
      }
    } catch (err: any) {
      errorMessage = err.message || 'Save failed';
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
      formData[activeMediaField] = media.path || media.id;
    }
  }

  onMount(() => {
    activeLocale = schema.defaultLocale;
    loadData();
  });
</script>

<MediaPickerModal
  open={mediaPickerOpen}
  onclose={() => { mediaPickerOpen = false; }}
  onselect={handleMediaSelect}
/>

{#if !collection}
  <div class="p-12 text-center text-slate-400">
    <p>Content type <strong>{type}</strong> not found.</p>
  </div>
{:else}
  <div class="max-w-4xl mx-auto space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between pb-4 border-b border-slate-800">
      <div class="flex items-center gap-3">
        <button
          type="button"
          onclick={() => router.navigate(`/content/${type}`)}
          class="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-slate-800 transition-colors"
        >
          ← Back
        </button>
        <div>
          <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2">
            <span>{collection.icon || '📄'}</span>
            <span>{isNew ? `New ${collection.singular || collection.label}` : `Edit ${collection.singular || collection.label}`}</span>
          </h2>
          <p class="text-xs text-slate-400 mt-0.5">
            {collection.label} collection entry
          </p>
        </div>
      </div>

      <div class="flex items-center gap-2.5">
        {#if collection.workflow?.enabled}
          <Button
            variant="secondary"
            size="md"
            loading={saving}
            onclick={() => handleSave('draft')}
          >
            Save Draft
          </Button>
          <Button
            variant="primary"
            size="md"
            loading={saving}
            onclick={() => handleSave('published')}
          >
            Publish
          </Button>
        {:else}
          <Button
            variant="primary"
            size="md"
            loading={saving}
            onclick={() => handleSave()}
          >
            Save Changes
          </Button>
        {/if}
      </div>
    </div>

    <!-- Feedback Alerts -->
    {#if errorMessage}
      <div class="p-3.5 rounded-xl bg-rose-950/60 border border-rose-800/80 text-rose-300 text-xs">
        ⚠️ {errorMessage}
      </div>
    {/if}
    {#if successMessage}
      <div class="p-3.5 rounded-xl bg-emerald-950/60 border border-emerald-800/80 text-emerald-300 text-xs">
        ✓ {successMessage}
      </div>
    {/if}

    <!-- Form Body -->
    <div class="rounded-2xl border border-slate-800 bg-slate-900/60 p-6 space-y-6">
      {#each collection.fields as field (field.name)}
        <div class="flex flex-col gap-1.5">
          {#if field.kind === 'text'}
            <Input
              label={field.label || field.name}
              placeholder={`Enter ${field.label || field.name}…`}
              bind:value={formData[field.name]}
              required={field.required}
            />
          {:else if field.kind === 'number'}
            <Input
              type="number"
              label={field.label || field.name}
              bind:value={formData[field.name]}
              required={field.required}
            />
          {:else if field.kind === 'textarea'}
            <label for={`field-${field.name}`} class="text-xs font-semibold uppercase tracking-wider text-slate-300">
              {field.label || field.name}
            </label>
            <textarea
              id={`field-${field.name}`}
              bind:value={formData[field.name]}
              rows="4"
              placeholder={`Enter ${field.label || field.name}…`}
              class="w-full rounded-lg bg-slate-900/90 border border-slate-800 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:border-sky-500 focus:ring-2 focus:ring-sky-500/20"
            ></textarea>
          {:else if field.kind === 'richtext'}
            <RichTextEditor
              label={field.label || field.name}
              bind:value={formData[field.name]}
            />
          {:else if field.kind === 'boolean'}
            <label class="flex items-center gap-3 cursor-pointer select-none">
              <input
                type="checkbox"
                bind:checked={formData[field.name]}
                class="w-4 h-4 rounded bg-slate-900 border-slate-700 text-sky-600 focus:ring-sky-500"
              />
              <span class="text-sm font-medium text-slate-200">{field.label || field.name}</span>
            </label>
          {:else if field.kind === 'select'}
            <label for={`field-${field.name}`} class="text-xs font-semibold uppercase tracking-wider text-slate-300">
              {field.label || field.name}
            </label>
            <select
              id={`field-${field.name}`}
              bind:value={formData[field.name]}
              class="w-full rounded-lg bg-slate-900/90 border border-slate-800 px-3 py-2 text-sm text-slate-100 focus:outline-none focus:border-sky-500 focus:ring-2 focus:ring-sky-500/20"
            >
              <option value="">Select option…</option>
              {#each (field.options || []) as opt}
                <option value={opt.value}>{opt.label}</option>
              {/each}
            </select>
          {:else if field.kind === 'date' || field.kind === 'datetime'}
            <Input
              type={field.kind === 'date' ? 'date' : 'datetime-local'}
              label={field.label || field.name}
              bind:value={formData[field.name]}
            />
          {:else if field.kind === 'media'}
            <span class="text-xs font-semibold uppercase tracking-wider text-slate-300">
              {field.label || field.name}
            </span>
            <div class="flex items-center gap-3">
              {#if formData[field.name]}
                <div class="relative w-16 h-16 rounded-lg overflow-hidden border border-slate-700 bg-slate-950 flex-shrink-0">
                  <img src={formData[field.name]} alt="Selected asset" class="w-full h-full object-cover" />
                </div>
              {/if}
              <Input
                placeholder="Asset URL or path…"
                bind:value={formData[field.name]}
                class="flex-1 font-mono text-xs"
              />
              <Button
                variant="secondary"
                size="md"
                type="button"
                onclick={() => openMediaPicker(field.name)}
              >
                Browse Media
              </Button>
            </div>
          {:else}
            <Input
              label={field.label || field.name}
              bind:value={formData[field.name]}
            />
          {/if}
        </div>
      {/each}
    </div>
  </div>
{/if}
