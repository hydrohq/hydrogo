<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { onMount, onDestroy } from 'svelte';
  import { api } from '$lib/api/client';
  import { router } from '$lib/router.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import RichTextEditor from '$lib/components/editor/RichTextEditor.svelte';
  import BlocksEditor, { type Block } from './BlocksEditor.svelte';
  import PageRevisionsModal from './PageRevisionsModal.svelte';
  import { createPresence } from '$lib/presence.svelte';
  import DocumentPresence from '$lib/components/presence/DocumentPresence.svelte';

  interface Props {
    id: string;
  }

  let { id }: Props = $props();
  const isNew = $derived(!id || id === 'new');

  interface PageData {
    id?: string;
    title: string;
    slug: string;
    locale: string;
    body?: string;
    blocks: Block[];
    layout_template?: string;
    status: 'draft' | 'published';
    version?: number;
  }

  let formData = $state<PageData>({
    title: '',
    slug: '',
    locale: 'en',
    body: '',
    blocks: [],
    layout_template: 'default',
    status: 'draft',
  });

  let saving = $state(false);
  let activeEditorTab = $state<'content' | 'blocks'>('content');
  let revisionsOpen = $state(false);
  let errorMsg = $state<string | null>(null);
  let successMsg = $state<string | null>(null);
  let initialSnapshot = $state('');

  const presence = createPresence(
    () => `/api/v1/admin/pages/${id}/presence`,
    () => (isNew ? null : id)
  );

  function captureSnapshot() {
    initialSnapshot = JSON.stringify(formData);
  }

  const isDirty = $derived.by(() => {
    if (saving || !initialSnapshot) return false;
    return JSON.stringify(formData) !== initialSnapshot;
  });

  $effect(() => {
    router.setDirty(
      isDirty,
      t('pages.editor.unsaved')
    );
  });

  onDestroy(() => {
    router.setDirty(false);
  });

  function autoSlug() {
    if (isNew && !formData.slug && formData.title) {
      formData.slug = formData.title
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/(^-|-$)/g, '');
    }
  }

  async function loadPage() {
    if (isNew) {
      captureSnapshot();
      return;
    }
    try {
      const res = await api.get<ApiResponse<PageData>>(`/api/v1/admin/pages/${id}`);
      if (res && res.data) {
        formData = {
          title: res.data.title || '',
          slug: res.data.slug || '',
          locale: res.data.locale || 'en',
          body: res.data.body || '',
          blocks: res.data.blocks || [],
          layout_template: res.data.layout_template || 'default',
          status: res.data.status || 'draft',
          version: res.data.version || 1,
        };
        captureSnapshot();
      }
    } catch (e: any) {
      errorMsg = e.message || t('pages.editor.loadFailed');
    }
  }

  async function handleSaveDraft() {
    saving = true;
    errorMsg = null;
    successMsg = null;
    try {
      const payload: Record<string, any> = {
        title: formData.title,
        slug: formData.slug,
        locale: formData.locale,
        body: formData.body,
        blocks: formData.blocks,
        layout_template: formData.layout_template,
      };
      if (isNew) {
        const res = await api.post<ApiResponse<PageData>>('/api/v1/admin/pages', payload);
        if (res && res.data && res.data.id) {
          captureSnapshot();
          router.setDirty(false);
          router.navigate(`/pages/${res.data.id}`, true);
        }
      } else {
        payload.version = formData.version || 1;
        const res = await api.put<ApiResponse<PageData>>(`/api/v1/admin/pages/${id}`, payload);
        if (res?.data?.version) {
          formData.version = res.data.version;
        }
        formData.status = 'draft';
        captureSnapshot();
        router.setDirty(false);
        successMsg = t('pages.editor.draftSaved');
        setTimeout(() => { successMsg = null; }, 3000);
      }
    } catch (e: any) {
      errorMsg = e.message || t('news.editor.saveFailed');
    } finally {
      saving = false;
    }
  }

  async function handleWorkflowAction(action: 'publish' | 'retract') {
    if (isNew) {
      await handleSaveDraft();
    }
    saving = true;
    errorMsg = null;
    try {
      if (action === 'publish') {
        const res = await api.post<ApiResponse<PageData>>(`/api/v1/admin/pages/${id}/publish`, {
          version: formData.version || 1,
        });
        if (res?.data?.version) {
          formData.version = res.data.version;
        } else {
          formData.version = (formData.version || 1) + 1;
        }
        formData.status = 'published';
        captureSnapshot();
        router.setDirty(false);
        successMsg = t('pages.editor.published');
      } else {
        // Saving a page draft in the backend resets the draft state
        await handleSaveDraft();
        successMsg = t('pages.editor.setDraft');
      }
      setTimeout(() => { successMsg = null; }, 3000);
    } catch (e: any) {
      errorMsg = e.message || t('news.editor.actionFailed', { action });
    } finally {
      saving = false;
    }
  }

  onMount(() => {
    loadPage();
  });
</script>

<PageRevisionsModal
  pageId={id}
  open={revisionsOpen}
  onclose={() => { revisionsOpen = false; }}
  onrestored={() => { loadPage(); }}
/>

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-2">
    <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <!-- Top Action Bar -->
      <div class="flex items-center justify-between px-4 lg:px-6">
        <div class="flex items-center gap-2.5">
          <button
            type="button"
            onclick={() => router.navigate('/pages')}
            class="h-8 px-2.5 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-muted-foreground hover:text-foreground transition-colors cursor-pointer flex items-center gap-1.5"
          >
            <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="m15 18-6-6 6-6" />
            </svg>
            <span>{t('common.back')}</span>
          </button>
          <div class="flex items-center gap-2">
            <span class="text-sm font-semibold text-foreground">
              {isNew ? t('pages.editor.new') : t('pages.editor.edit')}
            </span>
            <Badge variant={formData.status === 'published' ? 'success' : 'outline'} size="sm">
              {formData.status}
            </Badge>
            {#if !isNew}
              <DocumentPresence {presence} mode="pill" />
            {/if}
          </div>
        </div>

        <div class="flex items-center gap-2">
          {#if !isNew}
            <button
              type="button"
              onclick={() => { revisionsOpen = true; }}
              class="h-8 px-2.5 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs"
            >
              {t('pages.editor.history')}
            </button>
          {/if}

          <button
            type="button"
            disabled={saving}
            onclick={handleSaveDraft}
            class="h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs"
          >
            {t('news.editor.saveDraft')}
          </button>

          {#if !isNew && formData.status === 'published'}
            <button
              type="button"
              disabled={saving}
              onclick={() => handleWorkflowAction('retract')}
              class="h-8 px-3 rounded-md border border-destructive/20 bg-destructive/10 text-destructive hover:bg-destructive/20 text-xs font-medium transition-colors cursor-pointer shadow-xs"
            >
              {t('news.editor.retract')}
            </button>
          {:else if !isNew}
            <button
              type="button"
              disabled={saving}
              onclick={() => handleWorkflowAction('publish')}
              class="h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium transition-colors cursor-pointer shadow-xs"
            >
              {t('pages.editor.publish')}
            </button>
          {/if}
        </div>
      </div>

      {#if !isNew}
        <DocumentPresence {presence} mode="banner" />
      {/if}

      {#if errorMsg}
        <div class="mx-4 lg:mx-6 p-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs">
          {errorMsg}
        </div>
      {/if}
      {#if successMsg}
        <div class="mx-4 lg:mx-6 p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs">
          {successMsg}
        </div>
      {/if}

      <!-- Page Settings & Blocks Split -->
      <div class="px-4 lg:px-6 grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Main Content & Blocks Editor (Left 2 cols) -->
        <div class="lg:col-span-2 space-y-4">
          <div class="p-6 rounded-xl border border-border bg-card shadow-xs space-y-4">
            <!-- Tabs Header -->
            <div class="flex items-center justify-between border-b border-border pb-3">
              <div class="flex items-center gap-1.5 bg-muted/60 p-1 rounded-lg">
                <button
                  type="button"
                  onclick={() => { activeEditorTab = 'content'; }}
                  class="flex items-center gap-2 px-3 py-1.5 text-xs font-medium rounded-md transition-all cursor-pointer {activeEditorTab === 'content' ? 'bg-background text-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
                >
                  <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                    <polyline points="14 2 14 8 20 8"></polyline>
                    <line x1="16" y1="13" x2="8" y2="13"></line>
                    <line x1="16" y1="17" x2="8" y2="17"></line>
                    <polyline points="10 9 9 9 8 9"></polyline>
                  </svg>
                  <span>{t('pages.editor.tabContent')}</span>
                </button>
                <button
                  type="button"
                  onclick={() => { activeEditorTab = 'blocks'; }}
                  class="flex items-center gap-2 px-3 py-1.5 text-xs font-medium rounded-md transition-all cursor-pointer {activeEditorTab === 'blocks' ? 'bg-background text-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
                >
                  <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <rect x="3" y="3" width="7" height="7"></rect>
                    <rect x="14" y="3" width="7" height="7"></rect>
                    <rect x="14" y="14" width="7" height="7"></rect>
                    <rect x="3" y="14" width="7" height="7"></rect>
                  </svg>
                  <span>{t('pages.editor.tabBlocks')}</span>
                  {#if formData.blocks && formData.blocks.length > 0}
                    <span class="ml-1 px-1.5 py-0.2 rounded-full text-[10px] bg-primary/10 text-primary font-semibold">
                      {formData.blocks.length}
                    </span>
                  {/if}
                </button>
              </div>

              <span class="text-[11px] text-muted-foreground hidden sm:inline-block">
                {activeEditorTab === 'content' ? t('pages.editor.modeContent') : t('pages.editor.modeBlocks')}
              </span>
            </div>

            {#if activeEditorTab === 'content'}
              <RichTextEditor
                label={t('pages.editor.contentLabel')}
                bind:value={formData.body}
                placeholder={t('pages.editor.contentPlaceholder')}
              />
            {:else}
              <BlocksEditor
                blocks={formData.blocks}
                onchange={(newBlocks) => { formData.blocks = newBlocks; }}
              />
            {/if}
          </div>
        </div>

        <!-- Metadata Sidebar (Right col) -->
        <div class="space-y-4">
          <div class="p-5 rounded-xl border border-border bg-card shadow-xs space-y-4">
            <h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              {t('pages.editor.attributes')}
            </h3>

            <Input
              label={t('pages.editor.pageTitle')}
              placeholder={t('pages.editor.titlePlaceholder')}
              bind:value={formData.title}
              oninput={autoSlug}
              required
            />

            <Input
              label={t('pages.editor.slug')}
              placeholder={t('pages.editor.slugPlaceholder')}
              bind:value={formData.slug}
              required
            />

            <div class="flex flex-col gap-1.5">
              <label for="layout-template-select" class="text-xs font-medium text-foreground">
                {t('pages.editor.layout')}
              </label>
              <select
                id="layout-template-select"
                bind:value={formData.layout_template}
                class="w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
              >
                <option value="default">{t('pages.editor.layoutDefault')}</option>
                <option value="landing">{t('pages.editor.layoutLanding')}</option>
                <option value="contained">{t('pages.editor.layoutContained')}</option>
              </select>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>
