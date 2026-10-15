<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { router } from '$lib/router.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import BlocksEditor, { type Block } from './BlocksEditor.svelte';
  import PageRevisionsModal from './PageRevisionsModal.svelte';

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
  let revisionsOpen = $state(false);
  let errorMsg = $state<string | null>(null);
  let successMsg = $state<string | null>(null);

  function autoSlug() {
    if (isNew && !formData.slug && formData.title) {
      formData.slug = formData.title
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/(^-|-$)/g, '');
    }
  }

  async function loadPage() {
    if (isNew) return;
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
      }
    } catch (e: any) {
      errorMsg = e.message || 'Failed to load page';
    }
  }

  async function handleSaveDraft() {
    saving = true;
    errorMsg = null;
    successMsg = null;
    try {
      if (isNew) {
        const res = await api.post<ApiResponse<any>>('/api/v1/admin/pages', formData);
        const newId = res?.data?.id;
        if (newId) {
          router.navigate(`/pages/${newId}`);
        }
      } else {
        await api.put(`/api/v1/admin/pages/${id}`, formData);
        successMsg = 'Page draft saved.';
        setTimeout(() => { successMsg = null; }, 3000);
      }
    } catch (e: any) {
      errorMsg = e.message || 'Save failed';
    } finally {
      saving = false;
    }
  }

  async function handleWorkflowAction(action: 'publish' | 'retract') {
    if (isNew) {
      await handleSaveDraft();
    }
    saving = true;
    try {
      if (action === 'publish') {
        await api.post(`/api/v1/admin/pages/${id}/publish`, { version: formData.version });
        formData.status = 'published';
        successMsg = 'Page published live!';
      } else {
        await api.post(`/api/v1/admin/pages/${id}/retract`, { version: formData.version });
        formData.status = 'draft';
        successMsg = 'Page reverted to draft.';
      }
      setTimeout(() => { successMsg = null; }, 3000);
    } catch (e: any) {
      errorMsg = e.message || `Action ${action} failed`;
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

<div class="space-y-6 max-w-6xl mx-auto">
  <!-- Header Bar -->
  <div class="flex items-center justify-between pb-4 border-b border-slate-800">
    <div class="flex items-center gap-3">
      <button
        type="button"
        onclick={() => router.navigate('/pages')}
        class="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-slate-800 transition-colors"
      >
        ← Back
      </button>
      <div>
        <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2">
          <span>{isNew ? 'New Page' : 'Edit Page'}</span>
          <Badge variant={formData.status === 'published' ? 'success' : 'warning'} size="sm">
            {formData.status}
          </Badge>
        </h2>
      </div>
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-2.5">
      {#if !isNew}
        <Button
          variant="outline"
          size="md"
          onclick={() => { revisionsOpen = true; }}
        >
          <span>🕒</span>
          <span>History</span>
        </Button>
      {/if}

      <Button
        variant="secondary"
        size="md"
        loading={saving}
        onclick={handleSaveDraft}
      >
        Save Draft
      </Button>

      {#if !isNew && formData.status === 'published'}
        <Button
          variant="danger"
          size="md"
          loading={saving}
          onclick={() => handleWorkflowAction('retract')}
        >
          Retract
        </Button>
      {:else if !isNew}
        <Button
          variant="primary"
          size="md"
          loading={saving}
          onclick={() => handleWorkflowAction('publish')}
        >
          Publish
        </Button>
      {/if}
    </div>
  </div>

  {#if errorMsg}
    <div class="p-3.5 rounded-xl bg-rose-950/60 border border-rose-800/80 text-rose-300 text-xs">
      ⚠️ {errorMsg}
    </div>
  {/if}
  {#if successMsg}
    <div class="p-3.5 rounded-xl bg-emerald-950/60 border border-emerald-800/80 text-emerald-300 text-xs">
      ✓ {successMsg}
    </div>
  {/if}

  <!-- Page Settings & Blocks Split -->
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Blocks Editor (Left 2 cols) -->
    <div class="lg:col-span-2 space-y-6">
      <div class="p-6 rounded-2xl bg-slate-900/60 border border-slate-800 space-y-5">
        <BlocksEditor
          blocks={formData.blocks}
          onchange={(newBlocks) => { formData.blocks = newBlocks; }}
        />
      </div>
    </div>

    <!-- Metadata Sidebar (Right col) -->
    <div class="space-y-5">
      <div class="p-5 rounded-2xl bg-slate-900/60 border border-slate-800 space-y-4">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-slate-400">
          Page Attributes
        </h3>

        <Input
          label="Page Title"
          placeholder="e.g. About Us"
          bind:value={formData.title}
          oninput={autoSlug}
          required
        />

        <Input
          label="URL Slug"
          placeholder="e.g. about-us"
          bind:value={formData.slug}
          required
        />

        <div>
          <label for="layout-template-select" class="text-xs font-semibold uppercase tracking-wider text-slate-300">
            Layout Template
          </label>
          <select
            id="layout-template-select"
            bind:value={formData.layout_template}
            class="w-full mt-1 rounded-lg bg-slate-900 border border-slate-800 px-3 py-2 text-sm text-slate-100 focus:outline-none focus:border-sky-500"
          >
            <option value="default">Default Page</option>
            <option value="landing">Full-width Landing</option>
            <option value="contained">Contained / Centered</option>
          </select>
        </div>
      </div>
    </div>
  </div>
</div>
