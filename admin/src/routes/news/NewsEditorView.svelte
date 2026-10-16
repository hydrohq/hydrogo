<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { router } from '$lib/router.svelte';
  import type { ApiResponse } from '$lib/api/types';
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
  let version = $state(1);
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
          tags: Array.isArray(item.tags) ? item.tags : (item.tags ? [item.tags] : []),
          seo_title: item.seo_title || '',
          seo_description: item.seo_description || '',
        };
        version = item.version || 1;
        status = item.status || 'draft';
        tagsInput = typeof item.tags === 'string' ? item.tags : (item.tags || []).join(', ');
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
      const payload: Record<string, any> = {
        title: formData.title,
        slug: formData.slug,
        locale: formData.locale || 'en',
        excerpt: formData.excerpt,
        body: formData.body,
        cover_media_id: formData.cover_media_id,
        category: formData.category,
        author_name: formData.author_name,
        featured: Boolean(formData.featured),
        tags: tagsInput.trim(),
        seo_title: formData.seo_title,
        seo_description: formData.seo_description,
      };
      if (isNew) {
        const res = await api.post<ApiResponse<any>>('/api/v1/admin/news', payload);
        const newId = res?.data?.id || (res?.data?.Item && res.data.Item.id);
        if (newId) {
          router.navigate(`/news/${newId}`);
        }
      } else {
        payload.version = version || 1;
        const res = await api.put<ApiResponse<any>>(`/api/v1/admin/news/${id}`, payload);
        const updated = res?.data?.Item || res?.data;
        if (updated?.version) {
          version = updated.version;
        } else {
          version = (version || 1) + 1;
        }
        successMsg = 'Draft saved successfully.';
      }
      setTimeout(() => { successMsg = null; }, 3000);
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
    errorMsg = null;
    try {
      if (action === 'publish') {
        const res = await api.post<ApiResponse<any>>(`/api/v1/admin/news/${id}/publish`, {
          version: version || 1,
        });
        const updated = res?.data?.Item || res?.data;
        if (updated?.version) version = updated.version;
        else version = (version || 1) + 1;
        status = 'published';
        successMsg = 'Article published!';
      } else if (action === 'retract') {
        const res = await api.post<ApiResponse<any>>(`/api/v1/admin/news/${id}/retract`, {
          version: version || 1,
        });
        const updated = res?.data?.Item || res?.data;
        if (updated?.version) version = updated.version;
        else version = (version || 1) + 1;
        status = 'retracted';
        successMsg = 'Article retracted to draft state.';
      } else if (action === 'schedule') {
        if (!scheduledAt) {
          errorMsg = 'Please specify a scheduled publication date and time.';
          return;
        }
        const res = await api.post<ApiResponse<any>>(`/api/v1/admin/news/${id}/schedule`, {
          version: version || 1,
          scheduled_at: new Date(scheduledAt).toISOString(),
        });
        const updated = res?.data?.Item || res?.data;
        if (updated?.version) version = updated.version;
        else version = (version || 1) + 1;
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

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-2">
    <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <!-- Top Action Bar -->
      <div class="flex items-center justify-between px-4 lg:px-6">
        <div class="flex items-center gap-2.5">
          <button
            type="button"
            onclick={() => router.navigate('/news')}
            class="h-8 px-2.5 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-muted-foreground hover:text-foreground transition-colors cursor-pointer flex items-center gap-1.5"
          >
            <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="m15 18-6-6 6-6" />
            </svg>
            <span>Back</span>
          </button>
          <div class="flex items-center gap-2">
            <span class="text-sm font-semibold text-foreground">
              {isNew ? 'New Article' : 'Edit Article'}
            </span>
            {#if !isNew}
              <Badge variant={status === 'published' ? 'success' : status === 'scheduled' ? 'secondary' : status === 'retracted' ? 'destructive' : 'outline'} size="sm">
                {status}
              </Badge>
            {/if}
            {#if loading}
              <span class="text-xs text-muted-foreground animate-pulse">Loading...</span>
            {/if}
          </div>
        </div>

        <div class="flex items-center gap-2">
          <button
            type="button"
            disabled={saving}
            onclick={handleSaveDraft}
            class="h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs"
          >
            Save Draft
          </button>

          {#if !isNew && status === 'published'}
            <button
              type="button"
              disabled={saving}
              onclick={() => handleWorkflowAction('retract')}
              class="h-8 px-3 rounded-md border border-destructive/20 bg-destructive/10 text-destructive hover:bg-destructive/20 text-xs font-medium transition-colors cursor-pointer shadow-xs"
            >
              Retract
            </button>
          {:else if !isNew}
            <button
              type="button"
              disabled={saving}
              onclick={() => handleWorkflowAction('publish')}
              class="h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium transition-colors cursor-pointer shadow-xs"
            >
              Publish Now
            </button>
          {/if}
        </div>
      </div>

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

      <!-- Main Grid: Editor & Settings -->
      <div class="px-4 lg:px-6 grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Left 2 Cols: Main Content -->
        <div class="lg:col-span-2 space-y-4">
          <div class="p-6 rounded-xl border border-border bg-card shadow-xs space-y-4">
            <Input
              label="Title"
              placeholder="Article headline..."
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
              <label for="news-excerpt" class="text-xs font-medium text-foreground">
                Excerpt / Summary
              </label>
              <textarea
                id="news-excerpt"
                bind:value={formData.excerpt}
                rows="3"
                placeholder="A brief summary for cards and social previews..."
                class="w-full rounded-md border border-input bg-transparent px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
              ></textarea>
            </div>

            <RichTextEditor
              label="Body Content"
              bind:value={formData.body}
            />
          </div>
        </div>

        <!-- Right Col: Sidebar Meta -->
        <div class="space-y-4">
          <!-- Publishing Card -->
          <div class="p-5 rounded-xl border border-border bg-card shadow-xs space-y-4">
            <h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Editorial & Schedule
            </h3>

            {#if !isNew && status !== 'published'}
              <div class="space-y-2">
                <Input
                  type="datetime-local"
                  label="Schedule Release"
                  bind:value={scheduledAt}
                />
                <button
                  type="button"
                  class="w-full h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs"
                  disabled={saving}
                  onclick={() => handleWorkflowAction('schedule')}
                >
                  Set Schedule
                </button>
              </div>
            {/if}

            <label class="flex items-center gap-2 cursor-pointer select-none pt-2 border-t border-border">
              <input
                type="checkbox"
                bind:checked={formData.featured}
                class="size-4 rounded border-input text-primary focus:ring-ring"
              />
              <span class="text-xs font-medium text-foreground">Featured Article (Pinned)</span>
            </label>
          </div>

          <!-- Featured Image -->
          <div class="p-5 rounded-xl border border-border bg-card shadow-xs space-y-3">
            <h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Cover Image
            </h3>
            {#if formData.cover_media_id}
              <div class="w-full aspect-video rounded-lg bg-muted border border-border overflow-hidden relative">
                <img
                  src={`/media/${formData.cover_media_id}`}
                  alt="Cover asset"
                  class="w-full h-full object-cover"
                />
              </div>
            {/if}
            <button
              type="button"
              class="w-full h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs"
              onclick={() => { mediaPickerOpen = true; }}
            >
              {formData.cover_media_id ? 'Change Cover Image' : 'Select Cover Image'}
            </button>
          </div>

          <!-- Organization -->
          <div class="p-5 rounded-xl border border-border bg-card shadow-xs space-y-4">
            <h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
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
          <div class="p-5 rounded-xl border border-border bg-card shadow-xs space-y-3">
            <h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
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
  </div>
</div>
