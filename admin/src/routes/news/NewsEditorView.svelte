<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { onMount, onDestroy } from 'svelte';
  import { api } from '$lib/api/client';
  import { router } from '$lib/router.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import RichTextEditor from '$lib/components/editor/RichTextEditor.svelte';
  import MediaPickerModal from '$lib/components/media/MediaPickerModal.svelte';
  import { createPresence } from '$lib/presence.svelte';
  import DocumentPresence from '$lib/components/presence/DocumentPresence.svelte';

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
  let initialSnapshot = $state('');

  const presence = createPresence(
    () => `/api/v1/admin/news/${id}/presence`,
    () => (isNew ? null : id)
  );

  function captureSnapshot() {
    initialSnapshot = JSON.stringify({
      formData,
      tagsInput,
    });
  }

  const isDirty = $derived.by(() => {
    if (loading || saving || !initialSnapshot) return false;
    const current = JSON.stringify({
      formData,
      tagsInput,
    });
    return current !== initialSnapshot;
  });

  $effect(() => {
    router.setDirty(
      isDirty,
      t('news.editor.unsaved')
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

  async function loadItem() {
    if (isNew) {
      captureSnapshot();
      return;
    }
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
        captureSnapshot();
      }
    } catch (e: any) {
      errorMsg = e.message || t('news.editor.loadFailed');
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
          captureSnapshot();
          router.setDirty(false);
          router.navigate(`/news/${newId}`, true);
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
        captureSnapshot();
        router.setDirty(false);
        successMsg = t('news.editor.draftSaved');
      }
      setTimeout(() => { successMsg = null; }, 3000);
    } catch (e: any) {
      errorMsg = e.message || t('news.editor.saveFailed');
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
        captureSnapshot();
        router.setDirty(false);
        successMsg = t('news.editor.published');
      } else if (action === 'retract') {
        const res = await api.post<ApiResponse<any>>(`/api/v1/admin/news/${id}/retract`, {
          version: version || 1,
        });
        const updated = res?.data?.Item || res?.data;
        if (updated?.version) version = updated.version;
        else version = (version || 1) + 1;
        status = 'retracted';
        captureSnapshot();
        router.setDirty(false);
        successMsg = t('news.editor.retracted');
      } else if (action === 'schedule') {
        if (!scheduledAt) {
          errorMsg = t('news.editor.needSchedule');
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
        captureSnapshot();
        router.setDirty(false);
        successMsg = t('news.editor.scheduled');
      }
      setTimeout(() => { successMsg = null; }, 3000);
    } catch (e: any) {
      errorMsg = e.message || t('news.editor.actionFailed', { action });
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
            <span>{t('common.back')}</span>
          </button>
          <div class="flex items-center gap-2">
            <span class="text-sm font-semibold text-foreground">
              {isNew ? t('news.editor.new') : t('news.editor.edit')}
            </span>
            {#if !isNew}
              <Badge variant={status === 'published' ? 'success' : status === 'scheduled' ? 'secondary' : status === 'retracted' ? 'destructive' : 'outline'} size="sm">
                {status}
              </Badge>
              <DocumentPresence {presence} mode="pill" />
            {/if}
            {#if loading}
              <span class="text-xs text-muted-foreground animate-pulse">{t('common.loading')}</span>
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
            {t('news.editor.saveDraft')}
          </button>

          {#if !isNew && status === 'published'}
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
              {t('news.editor.publishNow')}
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

      <!-- Main Grid: Editor & Settings -->
      <div class="px-4 lg:px-6 grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Left 2 Cols: Main Content -->
        <div class="lg:col-span-2 space-y-4">
          <div class="p-6 rounded-xl border border-border bg-card shadow-xs space-y-4">
            <Input
              label={t('common.title')}
              placeholder={t('news.editor.titlePlaceholder')}
              bind:value={formData.title}
              oninput={autoSlug}
              required
            />

            <Input
              label={t('common.slug')}
              placeholder={t('news.editor.slugPlaceholder')}
              bind:value={formData.slug}
              required
            />

            <div class="flex flex-col gap-1.5">
              <label for="news-excerpt" class="text-xs font-medium text-foreground">
                {t('news.editor.excerpt')}
              </label>
              <textarea
                id="news-excerpt"
                bind:value={formData.excerpt}
                rows="3"
                placeholder={t('news.editor.excerptPlaceholder')}
                class="w-full rounded-md border border-input bg-transparent px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
              ></textarea>
            </div>

            <RichTextEditor
              label={t('news.editor.body')}
              bind:value={formData.body}
            />
          </div>
        </div>

        <!-- Right Col: Sidebar Meta -->
        <div class="space-y-4">
          <!-- Publishing Card -->
          <div class="p-5 rounded-xl border border-border bg-card shadow-xs space-y-4">
            <h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              {t('news.editor.editorial')}
            </h3>

            {#if !isNew && status !== 'published'}
              <div class="space-y-2">
                <Input
                  type="datetime-local"
                  label={t('news.editor.scheduleRelease')}
                  bind:value={scheduledAt}
                />
                <button
                  type="button"
                  class="w-full h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs"
                  disabled={saving}
                  onclick={() => handleWorkflowAction('schedule')}
                >
                  {t('news.editor.setSchedule')}
                </button>
              </div>
            {/if}

            <label class="flex items-center gap-2 cursor-pointer select-none pt-2 border-t border-border">
              <input
                type="checkbox"
                bind:checked={formData.featured}
                class="size-4 rounded border-input text-primary focus:ring-ring"
              />
              <span class="text-xs font-medium text-foreground">{t('news.editor.featured')}</span>
            </label>
          </div>

          <!-- Featured Image -->
          <div class="p-5 rounded-xl border border-border bg-card shadow-xs space-y-3">
            <h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              {t('news.editor.cover')}
            </h3>
            {#if formData.cover_media_id}
              <div class="w-full aspect-video rounded-lg bg-muted border border-border overflow-hidden relative">
                <img
                  src={`/media/${formData.cover_media_id}`}
                  alt={t('news.editor.coverAlt')}
                  class="w-full h-full object-cover"
                />
              </div>
            {/if}
            <button
              type="button"
              class="w-full h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs"
              onclick={() => { mediaPickerOpen = true; }}
            >
              {formData.cover_media_id ? t('news.editor.changeCover') : t('news.editor.selectCover')}
            </button>
          </div>

          <!-- Organization -->
          <div class="p-5 rounded-xl border border-border bg-card shadow-xs space-y-4">
            <h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              {t('news.editor.organization')}
            </h3>

            <Input
              label={t('common.category')}
              placeholder={t('news.editor.categoryPlaceholder')}
              bind:value={formData.category}
            />

            <Input
              label={t('news.editor.authorName')}
              placeholder={t('news.editor.authorPlaceholder')}
              bind:value={formData.author_name}
            />

            <Input
              label={t('common.tags')}
              placeholder={t('news.editor.tagsPlaceholder')}
              bind:value={tagsInput}
            />
          </div>

          <!-- SEO -->
          <div class="p-5 rounded-xl border border-border bg-card shadow-xs space-y-3">
            <h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              {t('news.editor.seo')}
            </h3>
            <Input
              label={t('news.editor.seoTitle')}
              placeholder={t('news.editor.seoTitlePlaceholder')}
              bind:value={formData.seo_title}
            />
            <Input
              label={t('news.editor.seoDescription')}
              placeholder={t('news.editor.seoDescriptionPlaceholder')}
              bind:value={formData.seo_description}
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</div>
