<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import { router } from '$lib/router.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
  import ContentTypeSchemaModal, { type ContentTypeDefinition } from './ContentTypeSchemaModal.svelte';

  let { mode = 'list' }: { mode?: string } = $props();

  let contentTypes = $state<ContentTypeDefinition[]>([]);
  let loading = $state(true);
  let statusMsg = $state<string | null>(null);
  let statusErr = $state<string | null>(null);

  let modalOpen = $state(false);
  let editingTarget = $state<ContentTypeDefinition | null>(null);

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

  function apiAccessBadge(ct: ContentTypeDefinition): {
    label: string;
    variant: 'success' | 'warning' | 'secondary' | 'outline';
  } {
    if (!ct.public) {
      return { label: t('ct.badge.adminOnly'), variant: 'outline' };
    }
    const rules = ct.access_rules ?? {};
    if (rules.read === 'false' && rules.list === 'false') {
      return { label: t('ct.badge.writeOnly'), variant: 'warning' };
    }
    if ((rules.read ?? '').includes('auth') || (rules.list ?? '').includes('auth')) {
      return { label: t('ct.badge.memberAuth'), variant: 'secondary' };
    }
    return { label: t('ct.badge.api'), variant: 'success' };
  }

  const groupedTypes = $derived.by(() => {
    const groups: Array<{ name: string; items: ContentTypeDefinition[] }> = [];
    const byName = new Map<string, ContentTypeDefinition[]>();
    for (const ct of contentTypes) {
      const groupName = ct.group?.trim() || t('nav.documents');
      let list = byName.get(groupName);
      if (!list) {
        list = [];
        byName.set(groupName, list);
        groups.push({ name: groupName, items: list });
      }
      list.push(ct);
    }
    return groups;
  });

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

  function openCreate() {
    editingTarget = null;
    modalOpen = true;
  }

  function openEdit(ct: ContentTypeDefinition) {
    editingTarget = ct;
    modalOpen = true;
  }

  async function handleSaved(msg: string, createdId?: string) {
    notify(msg);
    await loadContentTypes();
    if (createdId) {
      router.navigate(`/content/${createdId}`);
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
    void loadContentTypes();
    if (mode === 'new') {
      openCreate();
    }
  });
</script>

<ContentTypeSchemaModal
  open={modalOpen}
  editing={editingTarget}
  onclose={() => { modalOpen = false; editingTarget = null; }}
  onsaved={handleSaved}
/>

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

{#snippet cardGrid(items: ContentTypeDefinition[])}
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
    {#each items as ct (ct.id || ct.label)}
      {@const accessBadge = apiAccessBadge(ct)}
      <div class="flex flex-col justify-between rounded-xl border border-border bg-card p-5 shadow-xs hover:border-foreground/25 transition-colors">
        <div>
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <h2 class="text-sm font-semibold text-foreground truncate">{ct.label}</h2>
              <span class="font-mono text-[11px] text-muted-foreground">/api/v1/content/{ct.id}</span>
            </div>

            <button
              type="button"
              class="size-7 rounded hover:bg-muted text-muted-foreground hover:text-destructive flex items-center justify-center transition-colors cursor-pointer shrink-0"
              title={t('ct.deleteDefinition')}
              aria-label={t('ct.deleteDefinition')}
              onclick={() => { deleteTarget = ct; }}
            >
              <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="3 6 5 6 21 6"></polyline>
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
              </svg>
            </button>
          </div>

          {#if ct.description}
            <p class="text-xs text-muted-foreground mt-2 line-clamp-2">{ct.description}</p>
          {/if}

          <div class="flex flex-wrap gap-1.5 mt-3">
            <Badge variant="outline" class="text-[10px] tabular-nums">{t('ct.fieldsCount', { count: ct.fields?.length || 0 })}</Badge>
            <Badge variant={accessBadge.variant} class="text-[10px]">{accessBadge.label}</Badge>
            {#if ct.singleton}
              <Badge variant="secondary" class="text-[10px]">{t('ct.badge.singleton')}</Badge>
            {/if}
            {#if ct.drafts}
              <Badge variant="secondary" class="text-[10px]">{t('ct.badge.drafts')}</Badge>
            {/if}
            {#if ct.revisions}
              <Badge variant="outline" class="text-[10px]">{t('ct.badge.revisions')}</Badge>
            {/if}
          </div>
        </div>

        <div class="mt-4 pt-3 border-t border-border flex items-center justify-between gap-2">
          <button
            type="button"
            class="text-xs font-medium text-muted-foreground hover:text-foreground cursor-pointer"
            onclick={() => openEdit(ct)}
          >{t('ct.edit')}</button>
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
{/snippet}

<div class="flex flex-1 flex-col gap-4 md:gap-6">
  <!-- Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-semibold tracking-tight text-foreground">{t('ct.title')}</h1>
      <p class="text-xs text-muted-foreground mt-0.5">{t('ct.subtitle')}</p>
    </div>

    <Button onclick={openCreate} class="gap-1.5 self-start sm:self-auto cursor-pointer shadow-xs whitespace-nowrap shrink-0">
      <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="12" y1="5" x2="12" y2="19"></line>
        <line x1="5" y1="12" x2="19" y2="12"></line>
      </svg>
      <span>{t('ct.new')}</span>
    </Button>
  </div>

  {#if statusMsg}
    <div class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs" role="status">
      {statusMsg}
    </div>
  {/if}
  {#if statusErr}
    <div class="p-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs" role="alert">
      {statusErr}
    </div>
  {/if}

  {#if loading}
    <div class="p-12 text-center text-xs text-muted-foreground flex flex-col items-center justify-center gap-2">
      <div class="size-6 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
      <span>{t('ct.loading')}</span>
    </div>
  {:else if contentTypes.length === 0}
    <div class="rounded-xl border border-dashed border-border p-12 text-center text-xs text-muted-foreground">
      <p>{t('ct.empty')}</p>
      <Button variant="outline" class="mt-3 text-xs" onclick={openCreate}>
        {t('ct.createFirst')}
      </Button>
    </div>
  {:else if groupedTypes.length > 1}
    <div class="space-y-6">
      {#each groupedTypes as grp (grp.name)}
        <section class="space-y-3">
          <div class="flex items-center justify-between border-b border-border pb-2">
            <h2 class="text-xs font-semibold text-muted-foreground">{grp.name}</h2>
            <span class="font-mono text-[11px] text-muted-foreground tabular-nums">{grp.items.length}</span>
          </div>
          {@render cardGrid(grp.items)}
        </section>
      {/each}
    </div>
  {:else}
    {@render cardGrid(contentTypes)}
  {/if}
</div>
