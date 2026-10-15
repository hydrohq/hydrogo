<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { router } from '$lib/router.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import RichTextEditor from '$lib/components/editor/RichTextEditor.svelte';
  import MediaPickerModal from '$lib/components/media/MediaPickerModal.svelte';

  interface Props {
    id: string;
  }

  let { id }: Props = $props();
  const isNew = $derived(!id || id === 'new');

  interface NewsPayload {
    id?: string;
    title: string;
    slug: string;
    locale?: string;
    excerpt?: string;
    body: string;
    cover_media_id?: string;
    category?: string;
    author_name?: string;
    featured?: boolean;
    tags?: string[];
    seo_title?: string;
    seo_description?: string;
    status?: 'draft' | 'scheduled' | 'published' | 'retracted';
    scheduled_at?: string;
  }

  let formData = $state<NewsPayload>({
    title: '',
    slug: '',
    body: '',
    excerpt: '',
    category: '',
    author_name: '',
    featured: false,
    tags: [],
    seo_title: '',
    seo_description: '',
  });

  let tagsInput = $state('');
  let status = $state<'draft' | 'scheduled' | 'published' | 'retracted'>('draft');
  let scheduledAt = $state('');
  let loading = $state(false);
  let saving = $state(false);
  let mediaPickerOpen = $state(false);
  let successMsg = $state<string | null>(null);
  let errorMsg = $state<string | null>(null);

  function autoSlug() {
    if (isNew && !formData.slug && formData.title) {
      formData.slug = formData.title
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/(^-|-$)/g, '');
    }
  }

  async function loadItem() {
    if (isNew) return;
    loading = true;
    try {
      const res = await api.get<ApiResponse<any>>(`/api/v1/admin/news/${id}`);
      if (res && res.data) {
        const item = res.data.id ? res.data : (res.data.Item || res.data);
        formData = {
          title: item.title || '',
          slug: item.slug || '',
          locale: item.locale || 'en',
          excerpt: item.excerpt || '',
          body: item.body || '',
          cover_media_id: item.cover_media_id || '',
          category: item.category || '',
          author_name: item.author_name || '',
          featured: Boolean(item.featured),
          tags: item.tags || [],
          seo_title: item.seo_title || '',
          seo_description: item.seo_description || '',
        };
        status = item.status || 'draft';
        tagsInput = (item.tags || []).join(', ');
        if (item.scheduled_at) {
          scheduledAt = new Date(item.scheduled_at).toISOString().slice(0, 16);
        }
      }
    } catch (e: any) {
      errorMsg = e.message || 'Failed to load article';
    } finally {
      loading = false;
    }
  }

  async function handleSaveDraft() {
    saving = true;
    errorMsg = null;
    successMsg = null;
    try {
      const payload = {
        ...formData,
        tags: tagsInput.split(',').map(t => t.trim()).filter(Boolean),
      };
      if (isNew) {
        const res = await api.post<ApiResponse<any>>('/api/v1/admin/news', payload);
        const newId = res?.data?.id || (res?.data?.Item && res.data.Item.id);
        if (newId) {
          router.navigate(`/news/${newId}`);
        }
      } else {
        await api.put(`/api/v1/admin/news/${id}`, payload);
        successMsg = 'Draft saved successfully.';
      }
    } catch (e: any) {
      errorMsg = e.message || 'Save failed';
    } finally {
      saving = false;
    }
  }

  async function handleWorkflowAction(action: 'publish' | 'retract' | 'schedule') {
    if (isNew) {
      await handleSaveDraft();
    }
    saving = true;
    try {
      if (action === 'publish') {
        await api.post(`/api/v1/admin/news/${id}/publish`);
        status = 'published';
        successMsg = 'Article published!';
      } else if (action === 'retract') {
        await api.post(`/api/v1/admin/news/${id}/retract`);
        status = 'retracted';
        successMsg = 'Article retracted to draft state.';
      } else if (action === 'schedule') {
        if (!scheduledAt) {
          alert('Please specify a scheduled publication date and time.');
          return;
        }
        await api.post(`/api/v1/admin/news/${id}/schedule`, { scheduled_at: new Date(scheduledAt).toISOString() });
        status = 'scheduled';
        successMsg = 'Article scheduled for release.';
      }
      setTimeout(() => { successMsg = null; }, 3000);
    } catch (e: any) {
      errorMsg = e.message || `Action ${action} failed`;
    } finally {
      saving = false;
    }
  }

  onMount(() => {
    loadItem();
  });
</script>

<MediaPickerModal
  open={mediaPickerOpen}
  onclose={() => { mediaPickerOpen = false; }}
  onselect={(media) => { formData.cover_media_id = media.id || media.path; }}
/>

<div class="space-y-6 max-w-6xl mx-auto">
  <!-- Top Bar -->
  <div class="flex items-center justify-between pb-4 border-b border-slate-800">
    <div class="flex items-center gap-3">
      <button
        type="button"
        onclick={() => router.navigate('/news')}
        class="text-slate-400 hover:text-white p-1 rounded-lg hover:bg-slate-800 transition-colors"
      >
        ← Back
      </button>
      <div>
        <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2">
          <span>{isNew ? 'New Article' : 'Edit Article'}</span>
          {#if !isNew}
            <Badge variant={status === 'published' ? 'success' : status === 'scheduled' ? 'info' : status === 'retracted' ? 'danger' : 'warning'} size="sm">
              {status}
            </Badge>
          {/if}
        </h2>
      </div>
    </div>

    <!-- Workflow Actions -->
    <div class="flex items-center gap-2.5">
      <Button
        variant="secondary"
        size="md"
        loading={saving}
        onclick={handleSaveDraft}
      >
        Save Draft
      </Button>

      {#if !isNew && status === 'published'}
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
          Publish Now
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

  <!-- Main Grid: Editor & Settings -->
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Left 2 Cols: Main Content -->
    <div class="lg:col-span-2 space-y-5">
      <div class="p-6 rounded-2xl bg-slate-900/60 border border-slate-800 space-y-5">
        <Input
          label="Title"
          placeholder="Article headline…"
          bind:value={formData.title}
          oninput={autoSlug}
          required
        />

        <Input
          label="Slug"
          placeholder="url-friendly-slug"
          bind:value={formData.slug}
          required
        />

        <div class="flex flex-col gap-1.5">
          <label for="news-excerpt" class="text-xs font-semibold uppercase tracking-wider text-slate-300">
            Excerpt / Summary
          </label>
          <textarea
            id="news-excerpt"
            bind:value={formData.excerpt}
            rows="3"
            placeholder="A brief summary for cards and social previews…"
            class="w-full rounded-lg bg-slate-900 border border-slate-800 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:border-sky-500"
          ></textarea>
        </div>

        <RichTextEditor
          label="Body Content"
          bind:value={formData.body}
        />
      </div>
    </div>

    <!-- Right Col: Sidebar Meta -->
    <div class="space-y-5">
      <!-- Publishing Card -->
      <div class="p-5 rounded-2xl bg-slate-900/60 border border-slate-800 space-y-4">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-slate-400">
          Editorial & Schedule
        </h3>

        {#if !isNew && status !== 'published'}
          <div class="space-y-2">
            <Input
              type="datetime-local"
              label="Schedule Release"
              bind:value={scheduledAt}
            />
            <Button
              variant="outline"
              size="sm"
              class="w-full"
              loading={saving}
              onclick={() => handleWorkflowAction('schedule')}
            >
              Set Schedule
            </Button>
          </div>
        {/if}

        <label class="flex items-center gap-3 cursor-pointer select-none pt-2 border-t border-slate-800/80">
          <input
            type="checkbox"
            bind:checked={formData.featured}
            class="w-4 h-4 rounded bg-slate-900 border-slate-700 text-sky-600 focus:ring-sky-500"
          />
          <span class="text-xs font-semibold text-slate-200">Featured Article (Pinned)</span>
        </label>
      </div>

      <!-- Featured Image -->
      <div class="p-5 rounded-2xl bg-slate-900/60 border border-slate-800 space-y-3">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-slate-400">
          Cover Image
        </h3>
        {#if formData.cover_media_id}
          <div class="w-full aspect-video rounded-xl bg-slate-950 border border-slate-800 overflow-hidden relative">
            <img
              src={`/media/${formData.cover_media_id}`}
              alt="Cover asset"
              class="w-full h-full object-cover"
            />
          </div>
        {/if}
        <Button
          variant="outline"
          size="sm"
          class="w-full"
          onclick={() => { mediaPickerOpen = true; }}
        >
          {formData.cover_media_id ? 'Change Cover Image' : 'Select Cover Image'}
        </Button>
      </div>

      <!-- Organization -->
      <div class="p-5 rounded-2xl bg-slate-900/60 border border-slate-800 space-y-4">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-slate-400">
          Organization
        </h3>

        <Input
          label="Category"
          placeholder="e.g. Engineering, Company"
          bind:value={formData.category}
        />

        <Input
          label="Author Name"
          placeholder="e.g. Jane Doe"
          bind:value={formData.author_name}
        />

        <Input
          label="Tags"
          placeholder="Comma-separated: tech, release"
          bind:value={tagsInput}
        />
      </div>

      <!-- SEO -->
      <div class="p-5 rounded-2xl bg-slate-900/60 border border-slate-800 space-y-3">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-slate-400">
          SEO & Meta
        </h3>
        <Input
          label="SEO Title"
          placeholder="Custom title tag"
          bind:value={formData.seo_title}
        />
        <Input
          label="SEO Description"
          placeholder="Meta description"
          bind:value={formData.seo_description}
        />
      </div>
    </div>
  </div>
</div>
