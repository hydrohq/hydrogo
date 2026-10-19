<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import { router } from '$lib/router.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';

  interface ContentTypeField {
    id: string;
    label: string;
    kind: string;
    required?: boolean;
    list?: boolean;
    searchable?: boolean;
    filterable?: boolean;
    options?: string[];
    to?: string;
  }

  interface ContentTypeDefinition {
    id: string;
    label: string;
    description?: string;
    title_field?: string;
    public?: boolean;
    drafts?: boolean;
    revisions?: boolean;
    localization?: boolean;
    fields: ContentTypeField[];
  }

  let contentTypes = $state<ContentTypeDefinition[]>([]);
  let loading = $state(true);
  let statusMsg = $state<string | null>(null);
  let statusErr = $state<string | null>(null);

  // New Object Modal
  let createModalOpen = $state(false);
  let creating = $state(false);
  let newLabel = $state('');
  let newId = $state('');
  let newDescription = $state('');
  let newTitleField = $state('');
  let newPublic = $state(true);
  let newDrafts = $state(true);
  let newRevisions = $state(true);
  let newFields = $state<ContentTypeField[]>([
    { id: 'title', label: 'Title', kind: 'text', required: true, list: true, searchable: true },
  ]);

  // Delete Target
  let deleteTarget = $state<ContentTypeDefinition | null>(null);
  let deleting = $state(false);

  function notify(msg: string, isErr = false) {
    if (isErr) {
      statusErr = msg;
      statusMsg = null;
    } else {
      statusMsg = msg;
      statusErr = null;
    }
    setTimeout(() => {
      statusMsg = null;
      statusErr = null;
    }, 3500);
  }

  function slugify(text: string): string {
    return text
      .toLowerCase()
      .trim()
      .replace(/[^a-z0-9_]+/g, '_')
      .replace(/^_+|_+$/g, '');
  }

  function handleLabelChange(val: string) {
    newLabel = val;
    if (!newId || newId === slugify(newLabel.slice(0, -1))) {
      newId = slugify(val);
    }
  }

  function addField() {
    newFields = [
      ...newFields,
      { id: '', label: '', kind: 'text', required: false, list: true, searchable: false, filterable: false },
    ];
  }

  function removeField(index: number) {
    if (newFields.length <= 1) return;
    newFields = newFields.filter((_, i) => i !== index);
  }

  async function loadContentTypes() {
    loading = true;
    try {
      const res = await api.get<ApiResponse<ContentTypeDefinition[]>>('/api/v1/admin/content-types');
      if (res?.data && Array.isArray(res.data)) {
        contentTypes = res.data;
      }
    } catch (e: any) {
      console.warn('Failed to load content types:', e);
      notify(e.message || t('ct.err.load'), true);
    } finally {
      loading = false;
    }
  }

  async function handleCreateObject() {
    const id = slugify(newId);
    if (!id || !newLabel.trim()) {
      notify(t('ct.err.required'), true);
      return;
    }

    const cleanedFields = newFields
      .filter((f) => f.label.trim() !== '')
      .map((f) => {
        const fieldId = slugify(f.id || f.label);
        const res: ContentTypeField = {
          id: fieldId,
          label: f.label.trim(),
          kind: f.kind,
          required: !!f.required,
          list: !!f.list,
          searchable: !!f.searchable,
          filterable: !!f.filterable,
        };
        if (f.kind === 'select' && typeof f.options === 'string') {
          res.options = (f.options as string)
            .split(',')
            .map((s) => s.trim())
            .filter(Boolean);
        }
        if (f.kind === 'relation' && f.to) {
          res.to = f.to.trim();
        }
        return res;
      });

    if (cleanedFields.length === 0) {
      notify(t('ct.err.field'), true);
      return;
    }

    creating = true;
    try {
      const payload: ContentTypeDefinition = {
        id,
        label: newLabel.trim(),
        description: newDescription.trim() || undefined,
        title_field: newTitleField.trim() || cleanedFields[0].id,
        public: newPublic,
        drafts: newDrafts,
        revisions: newRevisions || newDrafts,
        fields: cleanedFields,
      };

      await api.post('/api/v1/admin/content-types', payload);
      createModalOpen = false;
      notify(t('ct.created', { label: payload.label }));

      // Refresh schema so sidebar and routing immediately discover the new object
      await schema.loadSchema(true);
      await loadContentTypes();

      // Navigate to the newly generated object panel
      router.navigate(`/content/${id}`);
    } catch (e: any) {
      notify(e.message || t('ct.err.create'), true);
    } finally {
      creating = false;
    }
  }

  async function handleDeleteObject() {
    if (!deleteTarget) return;
    deleting = true;
    try {
      await api.delete(`/api/v1/admin/content-types/${deleteTarget.id}`);
      notify(t('ct.deleted', { label: deleteTarget.label }));
      deleteTarget = null;
      await schema.loadSchema(true);
      await loadContentTypes();
    } catch (e: any) {
      notify(e.message || t('ct.err.delete'), true);
    } finally {
      deleting = false;
    }
  }

  onMount(() => {
    loadContentTypes();
  });
</script>

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-2">
    <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <!-- Header -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 px-4 lg:px-6">
        <div>
          <h1 class="text-xl font-semibold tracking-tight text-foreground">{t('ct.title')}</h1>
          <p class="text-xs text-muted-foreground mt-0.5">
            {t('ct.subtitle')}
          </p>
        </div>

        <Button onclick={() => { createModalOpen = true; }} class="gap-1.5 self-start sm:self-auto cursor-pointer shadow-xs">
          <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          <span>{t('ct.new')}</span>
        </Button>
      </div>

      <!-- Notifications -->
      {#if statusMsg}
        <div class="mx-4 lg:mx-6 p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs">
          {statusMsg}
        </div>
      {/if}
      {#if statusErr}
        <div class="mx-4 lg:mx-6 p-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs">
          {statusErr}
        </div>
      {/if}

      <!-- Grid of Object Types -->
      <div class="px-4 lg:px-6">
        {#if loading}
          <div class="p-12 text-center text-xs text-muted-foreground flex flex-col items-center justify-center gap-2">
            <div class="size-6 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
            <span>{t('ct.loading')}</span>
          </div>
        {:else if contentTypes.length === 0}
          <div class="rounded-xl border border-dashed border-border p-12 text-center text-xs text-muted-foreground">
            <p>{t('ct.empty')}</p>
            <Button variant="outline" class="mt-3 text-xs" onclick={() => { createModalOpen = true; }}>
              {t('ct.createFirst')}
            </Button>
          </div>
        {:else}
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {#each contentTypes as ct (ct.id || ct.label)}
              <div class="flex flex-col justify-between rounded-xl border border-border bg-card p-5 shadow-xs hover:border-primary/50 transition-colors">
                <div>
                  <div class="flex items-start justify-between gap-2">
                    <div class="flex items-center gap-2">
                      <div class="size-8 rounded-lg bg-primary/10 text-primary flex items-center justify-center shrink-0">
                        <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
                          <polyline points="2 17 12 22 22 17"></polyline>
                          <polyline points="2 12 12 17 22 12"></polyline>
                        </svg>
                      </div>
                      <div>
                        <h2 class="text-sm font-semibold text-foreground">{ct.label}</h2>
                        <span class="font-mono text-[11px] text-muted-foreground">/api/v1/content/{ct.id}</span>
                      </div>
                    </div>

                    <div class="flex items-center gap-1">
                      <button
                        type="button"
                        class="size-7 rounded hover:bg-muted text-muted-foreground hover:text-destructive flex items-center justify-center transition-colors cursor-pointer"
                        title={t('ct.deleteDefinition')}
                        onclick={() => { deleteTarget = ct; }}
                      >
                        <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <polyline points="3 6 5 6 21 6"></polyline>
                          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                        </svg>
                      </button>
                    </div>
                  </div>

                  {#if ct.description}
                    <p class="text-xs text-muted-foreground mt-2 line-clamp-2">{ct.description}</p>
                  {/if}

                  <div class="flex flex-wrap gap-1.5 mt-3">
                    <Badge variant="outline" class="text-[10px]">{ct.fields?.length || 0} fields</Badge>
                    {#if ct.public}
                      <Badge variant="success" class="text-[10px]">{t('ct.badge.api')}</Badge>
                    {/if}
                    {#if ct.drafts}
                      <Badge variant="secondary" class="text-[10px]">{t('ct.badge.drafts')}</Badge>
                    {/if}
                    {#if ct.revisions}
                      <Badge variant="outline" class="text-[10px]">{t('ct.badge.revisions')}</Badge>
                    {/if}
                  </div>
                </div>

                <div class="mt-4 pt-3 border-t border-border flex items-center justify-between">
                  <span class="text-[11px] text-muted-foreground">{t('ct.panelReady')}</span>
                  <button
                    type="button"
                    class="text-xs font-medium text-primary hover:underline flex items-center gap-1 cursor-pointer"
                    onclick={() => router.navigate(`/content/${ct.id}`)}
                  >
                    <span>{t('ct.openPanel')}</span>
                    <svg class="size-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="9 18 15 12 9 6"></polyline>
                    </svg>
                  </button>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>

<!-- Create New Object Modal -->
<Modal open={createModalOpen} onclose={() => { createModalOpen = false; }} title={t('ct.modalTitle')}>
  <div class="space-y-4 max-h-[75vh] overflow-y-auto px-1">
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <Input
        label={t('ct.label')}
        placeholder={t('ct.labelPlaceholder')}
        value={newLabel}
        oninput={(e: any) => handleLabelChange(e.target.value)}
        required
      />
      <Input
        label={t('ct.id')}
        placeholder={t('ct.idPlaceholder')}
        bind:value={newId}
        required
      />
    </div>

    <div>
      <label for="new-desc" class="text-xs font-medium text-foreground">{t('ct.description')}</label>
      <input
        id="new-desc"
        bind:value={newDescription}
        placeholder={t('ct.descriptionPlaceholder')}
        class="mt-1 w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
      />
    </div>

    <!-- Toggles -->
    <div class="rounded-lg border border-border p-3 space-y-2 bg-muted/20">
      <div class="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">{t('ct.features')}</div>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-2 pt-1">
        <label class="flex items-center gap-2 cursor-pointer text-xs select-none">
          <input type="checkbox" bind:checked={newPublic} class="size-4 rounded border-input text-primary" />
          <span>{t('ct.featApi')}</span>
        </label>
        <label class="flex items-center gap-2 cursor-pointer text-xs select-none">
          <input type="checkbox" bind:checked={newDrafts} class="size-4 rounded border-input text-primary" />
          <span>{t('ct.featDrafts')}</span>
        </label>
        <label class="flex items-center gap-2 cursor-pointer text-xs select-none">
          <input type="checkbox" bind:checked={newRevisions} class="size-4 rounded border-input text-primary" />
          <span>{t('ct.featRevisions')}</span>
        </label>
      </div>
    </div>

    <!-- Field Definitions -->
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <span class="text-xs font-semibold text-foreground">{t('ct.fields', { count: newFields.length })}</span>
        <button
          type="button"
          class="text-xs text-primary hover:underline flex items-center gap-1 cursor-pointer font-medium"
          onclick={addField}
        >
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
          <span>{t('ct.addField')}</span>
        </button>
      </div>

      <div class="space-y-2.5">
        {#each newFields as f, idx (idx)}
          <div class="rounded-lg border border-border bg-card p-3 shadow-xs space-y-2 relative group">
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-2">
              <input
                bind:value={f.label}
                placeholder={t('ct.fieldLabel')}
                class="rounded-md border border-input bg-background px-2.5 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              />
              <input
                bind:value={f.id}
                placeholder={t('ct.fieldId')}
                class="rounded-md border border-input bg-background px-2.5 py-1 text-xs font-mono text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              />
              <select
                bind:value={f.kind}
                class="rounded-md border border-input bg-background px-2.5 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              >
                <option value="text">{t('ct.kind.text')}</option>
                <option value="textarea">{t('ct.kind.textarea')}</option>
                <option value="richtext">{t('ct.kind.richtext')}</option>
                <option value="number">{t('ct.kind.number')}</option>
                <option value="boolean">{t('ct.kind.boolean')}</option>
                <option value="select">{t('ct.kind.select')}</option>
                <option value="date">{t('ct.kind.date')}</option>
                <option value="media">{t('ct.kind.media')}</option>
                <option value="relation">{t('ct.kind.relation')}</option>
              </select>
            </div>

            {#if f.kind === 'select'}
              <input
                bind:value={f.options}
                placeholder={t('ct.options')}
                class="w-full rounded-md border border-input bg-background px-2.5 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              />
            {:else if f.kind === 'relation'}
              <input
                bind:value={f.to}
                placeholder={t('ct.target')}
                class="w-full rounded-md border border-input bg-background px-2.5 py-1 text-xs font-mono text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              />
            {/if}

            <div class="flex items-center justify-between pt-1">
              <div class="flex flex-wrap items-center gap-3 text-[11px] text-muted-foreground select-none">
                <label class="flex items-center gap-1.5 cursor-pointer">
                  <input type="checkbox" bind:checked={f.required} class="size-3.5 rounded text-primary" />
                  <span>{t('ct.required')}</span>
                </label>
                <label class="flex items-center gap-1.5 cursor-pointer">
                  <input type="checkbox" bind:checked={f.list} class="size-3.5 rounded text-primary" />
                  <span>{t('ct.showInTable')}</span>
                </label>
                <label class="flex items-center gap-1.5 cursor-pointer">
                  <input type="checkbox" bind:checked={f.searchable} class="size-3.5 rounded text-primary" />
                  <span>{t('ct.searchable')}</span>
                </label>
                <label class="flex items-center gap-1.5 cursor-pointer">
                  <input type="checkbox" bind:checked={f.filterable} class="size-3.5 rounded text-primary" />
                  <span>{t('ct.filterable')}</span>
                </label>
              </div>

              {#if newFields.length > 1}
                <button
                  type="button"
                  onclick={() => removeField(idx)}
                  class="text-xs text-muted-foreground hover:text-destructive cursor-pointer"
                  title={t('ct.removeField')}
                >
                  <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                </button>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    </div>
  </div>

  <div class="mt-4 pt-3 border-t border-border flex justify-end gap-2">
    <Button variant="outline" onclick={() => { createModalOpen = false; }} disabled={creating}>
      {t('common.cancel')}
    </Button>
    <Button onclick={handleCreateObject} disabled={creating || !newLabel || !newId}>
      {creating ? t('ct.generating') : t('ct.modalTitle')}
    </Button>
  </div>
</Modal>

<!-- Delete Confirm Dialog -->
<ConfirmDialog
  open={!!deleteTarget}
  title={t('ct.deleteTitle', { label: deleteTarget?.label ?? '' })}
  description={t('ct.deleteDescription')}
  confirmLabel={deleting ? t('ct.deleting') : t('ct.deleteConfirm')}
  variant="destructive"
  loading={deleting}
  onconfirm={handleDeleteObject}
  oncancel={() => { deleteTarget = null; }}
/>
