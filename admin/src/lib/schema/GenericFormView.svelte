<script lang="ts">
  import PageSpinner from '$lib/components/ui/PageSpinner.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import { t } from '$lib/i18n/index.svelte';
  import { statusLabel } from '$lib/i18n/labels';
  import { onDestroy } from 'svelte';
  import { hiddenStatusField, setPublication, usesDrafts } from './publication';
  import { api, ApiClientError } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import { auth } from '$lib/stores/auth.svelte';
  import { router } from '$lib/router.svelte';
  import { confirmAction } from '$lib/confirm';
  import { slugify } from '$lib/commerce/slugify';
  import type { ApiResponse, Field } from '$lib/api/types';
  import Input from '$lib/components/ui/Input.svelte';
  import RichTextEditor from '$lib/components/editor/RichTextEditor.svelte';
  import JsonFieldEditor from '$lib/components/editor/JsonFieldEditor.svelte';
  import MediaPickerModal from '$lib/components/media/MediaPickerModal.svelte';
  import RelationField from './RelationField.svelte';
  import SelectOptionsModal from './SelectOptionsModal.svelte';
  import ContentRevisionsModal from './ContentRevisionsModal.svelte';
  import { isManyRelation, isRelation, relationTarget, relationValueField } from './relations';
  import { createPresence } from '$lib/presence.svelte';
  import DocumentPresence from '$lib/components/presence/DocumentPresence.svelte';

  interface Props {
    type: string;
    id?: string;
    singleton?: boolean;
  }

  let { type, id, singleton = false }: Props = $props();

  const collection = $derived(schema.getCollection(type));
  const endpoint = $derived(collection?.api_path || `/api/v1/admin/content/${type}`);
  const canManageSchema = $derived(
    collection?.source === 'content_type' && auth.hasPermission('admin.content.write')
  );

  let recordId = $state<string | undefined>(undefined);
  const isNew = $derived(!recordId);

  let formData = $state<Record<string, any>>({});
  let fieldErrors = $state<Record<string, string>>({});
  let loading = $state(true);
  let saving = $state(false);
  let errorMessage = $state<string | null>(null);
  let successMessage = $state<string | null>(null);
  let slugTouched = $state(false);
  let mediaPickerOpen = $state(false);
  let activeMediaField = $state<string | null>(null);
  let optionsTarget = $state<{ fieldId: string; fieldLabel: string } | null>(null);
  let revisionsOpen = $state(false);
  let copiedToken = $state<string | null>(null);
  let initialSnapshot = $state('');

  const presence = createPresence(
    () => `/api/v1/admin/content/${type}/${recordId ?? ''}/presence`,
    () => recordId
  );

  const isDirty = $derived.by(() => {
    if (loading || saving || !initialSnapshot) return false;
    return JSON.stringify(formData) !== initialSnapshot;
  });

  $effect(() => {
    router.setDirty(isDirty, t('schema.unsaved'));
  });

  onDestroy(() => {
    router.setDirty(false);
  });

  function getFieldKey(f: Field): string {
    return f.id || f.name || '';
  }

  function isJsonField(field: Field, key: string, val: any): boolean {
    const kind = field?.kind as string;
    if (kind === 'json' || field?.widget === 'json' || kind === 'map' || kind === 'object') return true;
    if (key.includes('json') || key === 'votes_data' || key.endsWith('_data') || key.endsWith('_map')) return true;
    if (field?.label && field.label.toLowerCase().includes('json')) return true;
    if (typeof val === 'object' && val !== null && !Array.isArray(val)) return true;
    if (typeof val === 'string' && val.trim().length > 1) {
      const trimmed = val.trim();
      if ((trimmed.startsWith('{') && trimmed.endsWith('}')) || (trimmed.startsWith('[') && trimmed.endsWith(']'))) {
        try {
          JSON.parse(trimmed);
          return true;
        } catch {}
      }
    }
    return false;
  }

  function initFormData(data: Record<string, any> = {}, forNew = true): Record<string, any> {
    const initialized: Record<string, any> = {};
    const today = new Date().toISOString().slice(0, 10);
    for (const field of collection?.fields ?? []) {
      const key = getFieldKey(field);
      if (data[key] !== undefined && data[key] !== null) {
        initialized[key] = data[key];
      } else if (field.kind === 'boolean' || field.widget === 'checkbox') {
        initialized[key] = false;
      } else if (field.kind === 'number') {
        initialized[key] = '';
      } else if (isManyRelation(field) && relationValueField(field) === 'id') {
        initialized[key] = [];
      } else if (forNew && key === 'published_on' && field.kind === 'date') {
        initialized[key] = today;
      } else {
        initialized[key] = '';
      }
    }
    for (const [k, v] of Object.entries(data)) {
      if (initialized[k] === undefined) initialized[k] = v;
    }
    return initialized;
  }

  function captureSnapshot() {
    initialSnapshot = JSON.stringify(formData);
  }

  let loadedRouteKey = '';
  async function loadData(explicitId: string | undefined, isSingleton: boolean) {
    if (!collection) {
      loading = false;
      return;
    }
    loading = true;
    errorMessage = null;
    fieldErrors = {};
    try {
      if (isSingleton) {
        const res = await api.get<ApiResponse<Record<string, any>[]>>(endpoint, { params: { limit: 1 } });
        const existing = Array.isArray(res?.data) ? res.data[0] : undefined;
        if (existing && existing.id !== undefined) {
          recordId = String(existing.id);
          slugTouched = true;
          formData = initFormData(existing, false);
        } else {
          recordId = undefined;
          slugTouched = false;
          formData = initFormData({}, true);
        }
      } else if (!explicitId || explicitId === 'new') {
        recordId = undefined;
        slugTouched = false;
        formData = initFormData({}, true);
      } else {
        recordId = explicitId;
        slugTouched = true;
        const res = await api.get<ApiResponse<Record<string, any>>>(`${endpoint}/${encodeURIComponent(explicitId)}`);
        formData = initFormData(res?.data ?? {}, false);
      }
      captureSnapshot();
    } catch (err: any) {
      errorMessage = err?.message || t('schema.err.load');
      formData = initFormData({}, !recordId);
      captureSnapshot();
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    const colReady = Boolean(collection);
    const key = `${type}|${id ?? ''}|${singleton}|${colReady}`;
    if (!colReady || key === loadedRouteKey) return;
    loadedRouteKey = key;
    void loadData(id, singleton);
  });

  function formatApiError(err: unknown, fallback: string): string {
    if (err instanceof ApiClientError) {
      if (err.fields) fieldErrors = err.fields;
      const suffix = err.requestId ? ` (${t('schema.requestId')}: ${err.requestId})` : '';
      return `${err.message || fallback}${suffix}`;
    }
    return err instanceof Error ? err.message : fallback;
  }

  async function handleSave(statusToSet?: string) {
    if (!collection) return;
    saving = true;
    errorMessage = null;
    successMessage = null;
    fieldErrors = {};

    try {
      const payload: Record<string, any> = { ...formData };
      if (statusToSet) payload.status = statusToSet;
      if (!isNew && formData.version !== undefined) payload.version = Number(formData.version) || 0;

      for (const f of collection.fields ?? []) {
        const key = getFieldKey(f);
        if (f.read_only) {
          delete payload[key];
          continue;
        }
        if (f.kind === 'number') {
          if (payload[key] === '' || payload[key] === null || payload[key] === undefined) delete payload[key];
          else payload[key] = Number(payload[key]);
        } else if (f.kind === 'boolean' || f.widget === 'checkbox') {
          payload[key] = Boolean(payload[key]);
        } else if (f.kind === 'relation') {
          if (payload[key] === '' || payload[key] === null || payload[key] === undefined) delete payload[key];
        } else if (typeof payload[key] === 'object' && payload[key] !== null) {
          payload[key] = JSON.stringify(payload[key]);
        }
      }

      const creating = isNew;
      const res = creating
        ? await api.post<ApiResponse<Record<string, any>>>(endpoint, payload)
        : await api.put<ApiResponse<Record<string, any>>>(`${endpoint}/${encodeURIComponent(String(recordId))}`, payload);

      const savedRecord = res?.data;
      const savedId = savedRecord?.id ? String(savedRecord.id) : recordId;

      if (savedRecord) {
        formData = initFormData(savedRecord, false);
        recordId = savedId;
        slugTouched = true;
      }

      if (statusToSet && usesDrafts(collection) && savedId) {
        const target = statusToSet === 'published' ? 'published' : 'draft';
        await setPublication(endpoint, savedId, target, {
          status: formData.status,
          version: Number(formData.version) || 0,
        });
        const refreshed = await api.get<ApiResponse<Record<string, any>>>(
          `${endpoint}/${encodeURIComponent(savedId)}`
        );
        if (refreshed?.data) formData = initFormData(refreshed.data, false);
        else formData = { ...formData, status: target };
      }

      captureSnapshot();
      router.setDirty(false);
      successMessage = singleton ? t('schema.singletonSaved') : creating ? t('schema.createdRecord') : t('common.saved');
      setTimeout(() => { successMessage = null; }, 3500);

      if (creating && !singleton && savedId) {
        loadedRouteKey = `${type}|${savedId}|false|true`;
        router.navigate(`/content/${type}/${savedId}`, true);
      }
    } catch (err) {
      errorMessage = formatApiError(err, t('schema.err.saveItem'));
    } finally {
      saving = false;
    }
  }

  const machine = $derived(collection?.state_machine);
  const currentState = $derived(machine ? String(formData[machine.field] || machine.initial) : '');
  const stateSteps = $derived(machine && !isNew ? machine.transitions.filter((s) => s.from === currentState) : []);

  async function changeState(to: string) {
    if (!collection || !recordId) return;
    const ok = await confirmAction({
      title: t('schema.confirm.moveTitle', { state: to }),
      confirmLabel: t('schema.confirm.moveButton'),
      cancelLabel: t('common.cancel'),
      variant: 'primary',
    });
    if (!ok) return;
    saving = true;
    errorMessage = null;
    try {
      const res = await api.post<ApiResponse<Record<string, any>>>(
        `${endpoint}/${encodeURIComponent(recordId)}/state`,
        { to, version: Number(formData.version) || 0 }
      );
      if (res?.data) {
        formData = initFormData(res.data, false);
        captureSnapshot();
      }
    } catch (err) {
      errorMessage = formatApiError(err, t('schema.err.saveItem'));
    } finally {
      saving = false;
    }
  }

  const visibleFields = $derived(
    (collection?.fields ?? []).filter((f) => getFieldKey(f) !== hiddenStatusField(collection))
  );
  const hasSlugField = $derived(visibleFields.some((f) => getFieldKey(f) === 'slug'));
  const hasRichEditor = $derived(
    visibleFields.some((f) => ['richtext', 'textarea'].includes(f.kind) || ['richtext', 'textarea'].includes(f.widget || ''))
  );
  const useSettingsGrid = $derived(
    Boolean(singleton && visibleFields.length >= 6 && !visibleFields.some((f) => f.kind === 'richtext' || f.widget === 'richtext'))
  );

  function isSideField(f: Field): boolean {
    const key = getFieldKey(f);
    if (key === collection?.title_field || key === 'slug') return false;
    if (isJsonField(f, key, formData[key])) return false;
    if (['richtext', 'textarea'].includes(f.kind) || ['richtext', 'textarea'].includes(f.widget || '')) return false;
    return (
      isRelation(f) ||
      f.kind === 'select' ||
      f.widget === 'select' ||
      f.kind === 'date' ||
      f.kind === 'datetime' ||
      f.kind === 'media' ||
      f.widget === 'media' ||
      f.kind === 'number' ||
      f.kind === 'boolean' ||
      f.widget === 'checkbox' ||
      Boolean(f.read_only)
    );
  }

  const primaryFields = $derived(visibleFields.filter((f) => !isSideField(f)));
  const sideFields = $derived(visibleFields.filter((f) => isSideField(f)));

  const resolvedActions = $derived.by(() => {
    if (!collection?.actions?.length) return [];
    const base = (schema.siteUrl || (typeof window !== 'undefined' ? window.location.origin : '')).replace(/\/+$/, '');
    return collection.actions
      .map((act) => {
        let href = act.href || '';
        if (!href) return null;
        if (href.includes('{slug}')) {
          const slugVal = String(formData.slug ?? '').trim();
          if (!slugVal) return null;
          href = href.replaceAll('{slug}', encodeURIComponent(slugVal));
        }
        if (href.includes('{id}')) {
          if (!recordId) return null;
          href = href.replaceAll('{id}', encodeURIComponent(recordId));
        }
        const fullHref = href.startsWith('/') ? `${base}${href}` : href;
        return { label: act.label, href: fullHref, isSvg: href.toLowerCase().endsWith('.svg') };
      })
      .filter((a): a is { label: string; href: string; isSvg: boolean } => a !== null);
  });

  const useTwoColumn = $derived(
    !useSettingsGrid &&
      (hasRichEditor ||
        resolvedActions.length > 0 ||
        Boolean(collection?.revisions && !isNew) ||
        Boolean(collection?.state_machine && !isNew) ||
        (primaryFields.length >= 2 && sideFields.length >= 2))
  );

  const placeholderTokens = $derived.by(() => {
    const desc = collection?.description ?? '';
    const found = new Set<string>();
    for (const match of desc.matchAll(/\{\{\s*([a-zA-Z0-9_]+)\s*\}\}/g)) {
      found.add(`{{${match[1]}}}`);
    }
    return [...found];
  });

  function isVideoUrl(val: unknown): boolean {
    return typeof val === 'string' && /\.(mp4|webm|mov)(\?|$)/i.test(val.trim());
  }

  async function copyToken(token: string) {
    try {
      await navigator.clipboard?.writeText(token);
      copiedToken = token;
      setTimeout(() => {
        if (copiedToken === token) copiedToken = null;
      }, 1500);
    } catch {}
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

  function openOrder(orderId: string) {
    if (!orderId) return;
    router.navigate(`/commerce?order=${encodeURIComponent(orderId)}`);
  }
</script>

<MediaPickerModal
  open={mediaPickerOpen}
  onclose={() => { mediaPickerOpen = false; }}
  onselect={handleMediaSelect}
/>

{#if optionsTarget}
  <SelectOptionsModal
    open={!!optionsTarget}
    collectionId={type}
    fieldId={optionsTarget.fieldId}
    fieldLabel={optionsTarget.fieldLabel}
    onclose={() => { optionsTarget = null; }}
  />
{/if}

{#if recordId}
  <ContentRevisionsModal
    open={revisionsOpen}
    {endpoint}
    {recordId}
    currentVersion={Number(formData.version) || 1}
    onclose={() => { revisionsOpen = false; }}
    onrestored={(restored) => {
      formData = initFormData(restored, false);
      captureSnapshot();
      successMessage = t('schema.revisions.restored', { version: Number(restored.version) || formData.version });
      setTimeout(() => { successMessage = null; }, 3500);
    }}
  />
{/if}

{#snippet renderField(field: Field)}
  {@const key = getFieldKey(field)}
  {@const err = fieldErrors[key]}
  <div class="flex flex-col gap-1.5 {useSettingsGrid && (key === collection?.title_field || field.kind === 'textarea' || field.widget === 'textarea') ? 'md:col-span-2' : ''}">
    {#if field.read_only}
      <span class="text-xs font-medium text-foreground">{field.label || key}</span>
      <div class="flex items-center gap-2">
        <div class="h-9 flex-1 flex items-center rounded-md border border-input bg-muted/40 px-3 text-xs text-muted-foreground font-mono" data-readonly-field={key}>
          {formData[key] === undefined || formData[key] === null || formData[key] === '' ? (isNew ? t('schema.setByServer') : '—') : String(formData[key])}
        </div>
        {#if key === 'order_id' && formData[key] && schema.modules.commerce}
          <button
            type="button"
            class="h-9 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shrink-0"
            onclick={() => openOrder(String(formData[key]))}
          >{t('schema.openOrder')} →</button>
        {/if}
      </div>
    {:else if isRelation(field) && relationTarget(field)}
      <RelationField
        id={key}
        label={field.label || key}
        target={relationTarget(field)}
        many={isManyRelation(field)}
        valueField={relationValueField(field)}
        required={field.required}
        bind:value={formData[key]}
      />
    {:else if isJsonField(field, key, formData[key])}
      <JsonFieldEditor label={field.label || key} bind:value={formData[key]} required={field.required} />
    {:else if field.kind === 'media' || field.widget === 'media'}
      <label for={`media-${key}`} class="text-xs font-medium text-foreground">
        {field.label || key}{#if field.required}<span class="text-destructive"> *</span>{/if}
      </label>
      {#if formData[key]}
        {#if isVideoUrl(formData[key])}
          <video src={String(formData[key])} controls muted class="w-full max-h-44 rounded-lg border border-border bg-black"></video>
        {:else}
          <div class="relative w-full h-36 rounded-lg overflow-hidden border border-border bg-muted/40">
            <img
              src={String(formData[key])}
              alt={t('schema.selectedAsset')}
              class="w-full h-full object-cover"
              onerror={(e) => { (e.currentTarget as HTMLElement).style.display = 'none'; }}
            />
          </div>
        {/if}
      {/if}
      <div class="flex items-center gap-2">
        <Input
          id={`media-${key}`}
          placeholder={t('schema.assetPlaceholder')}
          bind:value={formData[key]}
          error={err}
          class="flex-1 font-mono text-xs"
        />
        <button
          type="button"
          class="h-9 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shrink-0"
          onclick={() => openMediaPicker(key)}
        >{t('schema.browse')}</button>
        {#if formData[key]}
          <button
            type="button"
            class="h-9 w-9 inline-flex items-center justify-center rounded-md border border-input bg-background hover:bg-destructive/10 hover:text-destructive text-xs cursor-pointer shrink-0"
            aria-label={t('schema.clearSelection')}
            onclick={() => { formData[key] = ''; }}
          >×</button>
        {/if}
      </div>
    {:else if field.kind === 'select' || field.widget === 'select'}
      <div class="flex items-center justify-between gap-2">
        <label for={`field-${key}`} class="text-xs font-medium text-foreground">
          {field.label || key}{#if field.required}<span class="text-destructive"> *</span>{/if}
        </label>
        {#if canManageSchema}
          <button
            type="button"
            class="text-[11px] font-medium text-primary hover:underline cursor-pointer"
            onclick={() => { optionsTarget = { fieldId: key, fieldLabel: field.label || key }; }}
          >{t('schema.manageOptions')}</button>
        {/if}
      </div>
      <select
        id={`field-${key}`}
        bind:value={formData[key]}
        required={field.required}
        class="h-9 w-full rounded-md border border-input bg-background px-3 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
      >
        <option value="">{t('schema.selectOption')}</option>
        {#each (field.options || []) as opt}
          {@const optVal = typeof opt === 'string' ? opt : opt.value}
          {@const optLabel = typeof opt === 'string' ? opt : opt.label}
          <option value={optVal}>{optLabel}</option>
        {/each}
      </select>
    {:else if field.kind === 'textarea' || field.widget === 'textarea'}
      <label for={`field-${key}`} class="text-xs font-medium text-foreground">
        {field.label || key}{#if field.required}<span class="text-destructive"> *</span>{/if}
      </label>
      <textarea
        id={`field-${key}`}
        bind:value={formData[key]}
        rows="4"
        required={field.required}
        placeholder={t('schema.enter', { name: field.label || key })}
        class="w-full rounded-md border border-input bg-background px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
      ></textarea>
    {:else if field.kind === 'richtext' || field.widget === 'richtext'}
      {#if placeholderTokens.length > 0}
        <div class="rounded-lg border border-border bg-muted/30 p-2.5 space-y-1.5">
          <div class="text-[11px] font-medium text-muted-foreground">{t('schema.placeholders')}</div>
          <div class="flex flex-wrap gap-1.5">
            {#each placeholderTokens as token (token)}
              <button
                type="button"
                class="rounded border border-border bg-background px-2 py-0.5 font-mono text-[11px] text-foreground hover:border-primary transition-colors cursor-pointer"
                onclick={() => copyToken(token)}
              >{copiedToken === token ? `${token} ✓` : token}</button>
            {/each}
          </div>
        </div>
      {/if}
      <RichTextEditor label={field.label || key} bind:value={formData[key]} error={err} />
    {:else if field.kind === 'boolean' || field.widget === 'checkbox'}
      <label class="flex items-center gap-2 cursor-pointer select-none pt-1">
        <input type="checkbox" bind:checked={formData[key]} class="size-4 rounded border-input text-primary focus:ring-ring" />
        <span class="text-xs font-medium text-foreground">{field.label || key}</span>
      </label>
    {:else if field.kind === 'number'}
      <Input id={`field-${key}`} type="number" label={field.label || key} bind:value={formData[key]} required={field.required} error={err} />
    {:else if field.kind === 'date' || field.kind === 'datetime'}
      <Input id={`field-${key}`} type={field.kind === 'date' ? 'date' : 'datetime-local'} label={field.label || key} bind:value={formData[key]} required={field.required} error={err} />
    {:else}
      <div class="flex items-end gap-2">
        <div class="flex-1">
          <Input
            id={`field-${key}`}
            label={field.label || key}
            placeholder={t('schema.enter', { name: field.label || key })}
            bind:value={formData[key]}
            required={field.required}
            error={err}
            class={key === 'slug' || key === 'order_id' ? 'font-mono text-xs' : ''}
            oninput={(e: Event) => {
              const val = (e.currentTarget as HTMLInputElement).value;
              if (key === collection?.title_field && isNew && hasSlugField && !slugTouched) {
                formData.slug = slugify(val);
              }
              if (key === 'slug') slugTouched = true;
            }}
          />
        </div>
        {#if key === 'order_id' && formData[key] && schema.modules.commerce}
          <button
            type="button"
            class="h-9 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shrink-0"
            onclick={() => openOrder(String(formData[key]))}
          >{t('schema.openOrder')} →</button>
        {/if}
      </div>
    {/if}
    {#if err && (field.kind === 'select' || field.kind === 'textarea' || isRelation(field))}
      <p class="text-xs text-destructive font-medium">{err}</p>
    {/if}
  </div>
{/snippet}

{#if schema.isLoading}
  <div class="p-12 text-center text-xs text-muted-foreground flex flex-col items-center justify-center gap-2">
    <div class="size-6 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
    <span>{t('schema.loadingSchema', { type })}</span>
  </div>
{:else if !collection}
  <div class="p-12 text-center text-xs text-muted-foreground">
    <p>{t('schema.notFoundBefore')} <strong>{type}</strong> {t('schema.notFoundAfter')}</p>
    <button
      type="button"
      class="mt-4 inline-flex items-center gap-1.5 h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground cursor-pointer"
      onclick={() => router.navigate('/dashboard')}
    >{t('schema.backToDashboard')}</button>
  </div>
{:else if loading}
  <PageSpinner />
{:else}
  <div class="flex flex-1 flex-col gap-5">
    <!-- Top Toolbar -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div class="flex items-start sm:items-center gap-2.5 flex-wrap">
        {#if !singleton}
          <button
            type="button"
            onclick={() => router.navigate(`/content/${type}`)}
            class="h-8 px-2.5 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-muted-foreground hover:text-foreground transition-colors cursor-pointer inline-flex items-center gap-1.5"
          >
            <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m15 18-6-6 6-6" /></svg>
            <span>{t('common.back')}</span>
          </button>
        {/if}
        <div>
          <h1 class="text-base font-semibold tracking-tight text-foreground">
            {singleton ? collection.label : (isNew ? t('title.newItem', { name: collection.singular || collection.label || type }) : t('title.editItem', { name: collection.singular || collection.label || type }))}
          </h1>
          {#if singleton && collection.description}
            <p class="text-xs text-muted-foreground mt-0.5 max-w-3xl">{collection.description}</p>
          {/if}
        </div>
        {#if !isNew}
          <DocumentPresence {presence} mode="pill" />
        {/if}
      </div>

      <div class="flex items-center gap-2 shrink-0">
        {#if collection.workflow?.enabled || usesDrafts(collection)}
          <button
            type="button"
            disabled={saving}
            class="h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs"
            onclick={() => handleSave('draft')}
          >{t('news.editor.saveDraft')}</button>
          <button
            type="button"
            disabled={saving}
            class="h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium transition-colors cursor-pointer shadow-xs"
            onclick={() => handleSave('published')}
          >{t('pages.editor.publish')}</button>
        {:else}
          <button
            type="button"
            disabled={saving}
            class="h-8 px-3.5 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium transition-colors cursor-pointer shadow-xs"
            onclick={() => handleSave()}
          >{isNew && !singleton ? t('schema.createItem') : t('media.detail.save')}</button>
        {/if}
      </div>
    </div>

    {#if !isNew}
      <DocumentPresence {presence} mode="banner" />
    {/if}

    {#if errorMessage}
      <div class="p-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs" role="alert">
        {errorMessage}
      </div>
    {/if}
    {#if successMessage}
      <div class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs" role="status">
        {successMessage}
      </div>
    {/if}

    {#if useSettingsGrid}
      <div class="rounded-xl border border-border bg-card p-6 shadow-xs">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          {#each visibleFields as field (getFieldKey(field))}
            {@render renderField(field)}
          {/each}
        </div>
      </div>
    {:else if useTwoColumn}
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        <div class="lg:col-span-2 rounded-xl border border-border bg-card p-6 shadow-xs space-y-5">
          {#each primaryFields as field (getFieldKey(field))}
            {@render renderField(field)}
          {/each}
        </div>

        <div class="lg:col-span-1 space-y-4">
          {#if (collection.workflow?.enabled || usesDrafts(collection) || (collection.revisions && !isNew) || stateSteps.length > 0)}
            <div class="rounded-xl border border-border bg-card p-5 shadow-xs space-y-3">
              <div class="flex items-center justify-between gap-2">
                <span class="text-xs font-medium text-muted-foreground">{t('common.status')}</span>
                <div class="flex items-center gap-2">
                  {#if formData.version}
                    <span class="font-mono text-[11px] text-muted-foreground tabular-nums">v{formData.version}</span>
                  {/if}
                  {#if collection.workflow?.enabled || usesDrafts(collection)}
                    <Badge variant={formData.status === 'published' ? 'success' : 'warning'} size="sm">
                      {statusLabel(formData.status || 'draft')}
                    </Badge>
                  {:else if currentState}
                    <Badge variant="outline" size="sm">{currentState}</Badge>
                  {/if}
                </div>
              </div>
              {#if stateSteps.length > 0}
                <div class="flex flex-wrap gap-2 pt-1">
                  {#each stateSteps as step (step.to)}
                    <button
                      type="button"
                      disabled={saving}
                      class="h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium cursor-pointer"
                      onclick={() => changeState(step.to)}
                    >{t('schema.moveTo', { state: step.to })}</button>
                  {/each}
                </div>
              {/if}
              {#if collection.revisions && !isNew}
                <button
                  type="button"
                  class="w-full h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer"
                  onclick={() => { revisionsOpen = true; }}
                >{t('schema.revisions')}</button>
              {/if}
            </div>
          {/if}

          {#if resolvedActions.length > 0}
            <div class="rounded-xl border border-border bg-card p-5 shadow-xs space-y-3">
              <div class="text-xs font-semibold text-foreground">{t('schema.actions')}</div>
              {#each resolvedActions as act (act.href)}
                {#if act.isSvg}
                  <div class="rounded-lg border border-border bg-background p-3 flex items-center justify-between gap-3">
                    <img
                      src={act.href}
                      alt={act.label}
                      class="size-16 rounded bg-white p-1 border border-border"
                      onerror={(e) => { (e.currentTarget as HTMLElement).style.display = 'none'; }}
                    />
                    <a
                      href={act.href}
                      download
                      class="inline-flex items-center h-8 px-3 rounded-md border border-input bg-card hover:bg-accent text-xs font-medium text-foreground"
                    >{t('schema.downloadSvg')}</a>
                  </div>
                {:else}
                  <a
                    href={act.href}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="w-full inline-flex items-center justify-between h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors"
                  >
                    <span>{act.label}</span>
                    <span>↗</span>
                  </a>
                {/if}
              {/each}
            </div>
          {/if}

          {#if sideFields.length > 0}
            <div class="rounded-xl border border-border bg-card p-5 shadow-xs space-y-4">
              {#each sideFields as field (getFieldKey(field))}
                {@render renderField(field)}
              {/each}
            </div>
          {/if}
        </div>
      </div>
    {:else}
      <div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-5 max-w-3xl">
        {#each visibleFields as field (getFieldKey(field))}
          {@render renderField(field)}
        {/each}
      </div>
    {/if}
  </div>
{/if}
