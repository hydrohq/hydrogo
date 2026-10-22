<script lang="ts">
  import { schema } from '$lib/stores/schema.svelte';
  import { t } from '$lib/i18n/index.svelte';
  import { statusLabel } from '$lib/i18n/labels';
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { router } from '$lib/router.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Badge from '$lib/components/ui/Badge.svelte';

  interface DashboardTableItem {
    id: string;
    title: string;
    slug?: string;
    type: 'page' | 'news';
    status: string;
    author: string;
    date: string;
    targetPath?: string;
  }

  let items = $state<DashboardTableItem[]>([]);
  let loading = $state(true);
  let activeTab = $state<'all' | 'pages' | 'news'>('all');
  let selectedIds = $state<string[]>([]);
  let searchQuery = $state('');
  let statusFilter = $state('all');
  let pageIndex = $state(0);
  let pageSize = $state(10);
  let showColumnsMenu = $state(false);

  // Column visibility
  let visibleColumns = $state({
    title: true,
    type: true,
    status: true,
    author: true,
    date: true,
    actions: true,
  });

  async function loadAllContent() {
    loading = true;
    const combined: DashboardTableItem[] = [];

    const on = (m: string) => !schema.schema || schema.modules[m] === true;

    try {
      if (!on('pages')) throw new Error('skip');
      // 1. Fetch Pages
      const pagesRes = await api.get<ApiResponse<any[]>>('/api/v1/admin/pages', {
        params: { limit: 50 },
      });
      if (pagesRes && pagesRes.data && Array.isArray(pagesRes.data)) {
        for (const p of pagesRes.data) {
          combined.push({
            id: `page-${p.id}`,
            title: p.title || p.slug || t('dashboard.table.untitledPage'),
            slug: p.slug,
            type: 'page',
            status: p.status || 'published',
            author: p.author_name || t('dashboard.table.adminAuthor'),
            date: p.updated_at || p.created_at || new Date().toISOString(),
            targetPath: `/pages/${p.id}`,
          });
        }
      }
    } catch (e) {
      if (on('pages')) console.warn('Failed to load pages for dashboard:', e);
    }

    try {
      if (!on('news')) throw new Error('skip');
      // 2. Fetch News
      const newsRes = await api.get<ApiResponse<any[]>>('/api/v1/admin/news', {
        params: { limit: 50 },
      });
      if (newsRes && newsRes.data && Array.isArray(newsRes.data)) {
        for (const n of newsRes.data) {
          combined.push({
            id: `news-${n.id}`,
            title: n.title || n.slug || t('dashboard.table.untitledNews'),
            slug: n.slug,
            type: 'news',
            status: n.status || 'published',
            author: n.author_name || t('dashboard.table.editorialTeam'),
            date: n.published_at || n.updated_at || n.created_at || new Date().toISOString(),
            targetPath: `/news/${n.id}`,
          });
        }
      }
    } catch (e) {
      if (on('news')) console.warn('Failed to load news for dashboard:', e);
    }

    // Sort by most recent first
    combined.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime());
    items = combined;
    loading = false;
  }

  const tabCounts = $derived.by(() => {
    return {
      all: items.length,
      pages: items.filter((i) => i.type === 'page').length,
      news: items.filter((i) => i.type === 'news').length,
    };
  });

  const filteredItems = $derived.by(() => {
    let list = items;

    // Filter by tab
    if (activeTab === 'pages') {
      list = list.filter((i) => i.type === 'page');
    } else if (activeTab === 'news') {
      list = list.filter((i) => i.type === 'news');
    }

    // Filter by status
    if (statusFilter !== 'all') {
      list = list.filter((i) => i.status.toLowerCase() === statusFilter.toLowerCase());
    }

    // Filter by search query
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      list = list.filter(
        (i) =>
          i.title.toLowerCase().includes(q) ||
          (i.slug && i.slug.toLowerCase().includes(q)) ||
          i.author.toLowerCase().includes(q)
      );
    }

    return list;
  });

  const totalPages = $derived(Math.max(1, Math.ceil(filteredItems.length / pageSize)));
  const paginatedItems = $derived(
    filteredItems.slice(pageIndex * pageSize, (pageIndex + 1) * pageSize)
  );

  const allSelected = $derived(
    paginatedItems.length > 0 && paginatedItems.every((i) => selectedIds.includes(i.id))
  );

  function toggleSelectAll() {
    if (allSelected) {
      selectedIds = selectedIds.filter((id) => !paginatedItems.some((i) => i.id === id));
    } else {
      const newIds = new Set([...selectedIds, ...paginatedItems.map((i) => i.id)]);
      selectedIds = Array.from(newIds);
    }
  }

  function toggleSelectRow(id: string) {
    if (selectedIds.includes(id)) {
      selectedIds = selectedIds.filter((i) => i !== id);
    } else {
      selectedIds = [...selectedIds, id];
    }
  }

  function handleRowClick(item: DashboardTableItem) {
    if (item.targetPath) {
      router.navigate(item.targetPath);
    }
  }

  onMount(() => {
    loadAllContent();
  });
</script>

<div class="flex flex-col gap-4 px-4 lg:px-6">
  <!-- Tabs and Action Toolbar Header (Exact shadcn specification) -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <!-- TabsList -->
    <div class="flex items-center rounded-lg bg-muted p-1 text-xs border border-border overflow-x-auto">
      <button
        type="button"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-md font-medium transition-all whitespace-nowrap cursor-pointer {activeTab === 'all' ? 'bg-background text-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { activeTab = 'all'; pageIndex = 0; }}
      >
        <span>{t('dashboard.table.all')}</span>
        <span class="flex h-4 min-w-4 px-1 items-center justify-center rounded-full bg-muted-foreground/20 text-[10px] font-mono">
          {tabCounts.all}
        </span>
      </button>

      <button
        type="button"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-md font-medium transition-all whitespace-nowrap cursor-pointer {activeTab === 'pages' ? 'bg-background text-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { activeTab = 'pages'; pageIndex = 0; }}
      >
        <span>{t('dashboard.table.pages')}</span>
        <span class="flex h-4 min-w-4 px-1 items-center justify-center rounded-full bg-muted-foreground/20 text-[10px] font-mono">
          {tabCounts.pages}
        </span>
      </button>

      <button
        type="button"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-md font-medium transition-all whitespace-nowrap cursor-pointer {activeTab === 'news' ? 'bg-background text-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { activeTab = 'news'; pageIndex = 0; }}
      >
        <span>{t('dashboard.table.news')}</span>
        <span class="flex h-4 min-w-4 px-1 items-center justify-center rounded-full bg-muted-foreground/20 text-[10px] font-mono">
          {tabCounts.news}
        </span>
      </button>
    </div>

    <!-- Right Controls: Status Filter, Search, Columns Menu -->
    <div class="flex items-center gap-2">
      <!-- Search Input -->
      <div class="relative w-full sm:w-56">
        <svg class="size-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"></circle>
          <path d="m21 21-4.3-4.3"></path>
        </svg>
        <input
          type="text"
          placeholder={t('dashboard.table.filter')}
          bind:value={searchQuery}
          class="h-8 w-full pl-8 pr-3 text-xs rounded-md border border-input bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
        />
      </div>

      <!-- Status Filter -->
      <select
        bind:value={statusFilter}
        class="h-8 rounded-md border border-input bg-background px-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs cursor-pointer"
      >
        <option value="all">{t('dashboard.table.allStatus')}</option>
        <option value="published">{t('common.published')}</option>
        <option value="draft">{t('common.draft')}</option>
        <option value="recorded">{t('dashboard.table.recorded')}</option>
      </select>

      <!-- Customize Columns Dropdown -->
      <div class="relative">
        <button
          type="button"
          class="h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors flex items-center gap-1.5 cursor-pointer shadow-xs"
          onclick={() => { showColumnsMenu = !showColumnsMenu; }}
        >
          <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="4" y1="21" x2="4" y2="14"></line>
            <line x1="4" y1="10" x2="4" y2="3"></line>
            <line x1="12" y1="21" x2="12" y2="12"></line>
            <line x1="12" y1="8" x2="12" y2="3"></line>
            <line x1="20" y1="21" x2="20" y2="16"></line>
            <line x1="20" y1="12" x2="20" y2="3"></line>
            <line x1="1" y1="14" x2="7" y2="14"></line>
            <line x1="9" y1="8" x2="15" y2="8"></line>
            <line x1="17" y1="16" x2="23" y2="16"></line>
          </svg>
          <span>{t('dashboard.table.columns')}</span>
        </button>

        {#if showColumnsMenu}
          <div
            class="absolute right-0 top-full mt-1.5 w-44 rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-md z-30 animate-in fade-in zoom-in-95 duration-100"
          >
            {#each [
              { key: 'type', label: t('dashboard.table.colType') },
              { key: 'status', label: t('common.status') },
              { key: 'author', label: t('dashboard.table.colAuthor') },
              { key: 'date', label: t('common.date') },
              { key: 'actions', label: t('dashboard.table.colActions') }
            ] as col}
              <label class="flex items-center gap-2 px-2 py-1.5 text-xs rounded hover:bg-accent cursor-pointer">
                <input
                  type="checkbox"
                  class="rounded border-input text-primary focus:ring-ring size-3.5 cursor-pointer"
                  bind:checked={visibleColumns[col.key as keyof typeof visibleColumns]}
                />
                <span>{col.label}</span>
              </label>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>

  <!-- Table Container (Exact shadcn rounded border table) -->
  <div class="rounded-xl border border-border bg-card shadow-xs overflow-hidden">
    {#if loading}
      <div class="py-20 flex flex-col items-center justify-center text-muted-foreground text-xs gap-2">
        <div class="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
        <span>{t('dashboard.table.loading')}</span>
      </div>
    {:else if paginatedItems.length === 0}
      <div class="py-16 text-center text-xs text-muted-foreground">
        {#if searchQuery}
          <p>{t('dashboard.table.noMatch', { query: searchQuery })}</p>
        {:else}
          <p>{t('dashboard.table.empty')}</p>
        {/if}
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm text-foreground caption-bottom">
          <thead class="sticky top-0 z-10 bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
            <tr>
              <th class="h-10 w-10 px-4 text-center align-middle">
                <input
                  type="checkbox"
                  class="rounded border-input text-primary focus:ring-ring size-3.5 cursor-pointer align-middle"
                  checked={allSelected}
                  onchange={toggleSelectAll}
                  aria-label={t('dashboard.table.selectAll')}
                />
              </th>
              {#if visibleColumns.title}
                <th class="h-10 px-4 font-medium align-middle">{t('dashboard.table.itemTitle')}</th>
              {/if}
              {#if visibleColumns.type}
                <th class="h-10 px-4 font-medium align-middle">{t('common.type')}</th>
              {/if}
              {#if visibleColumns.status}
                <th class="h-10 px-4 font-medium align-middle">{t('common.status')}</th>
              {/if}
              {#if visibleColumns.author}
                <th class="h-10 px-4 font-medium align-middle">{t('dashboard.table.colAuthor')}</th>
              {/if}
              {#if visibleColumns.date}
                <th class="h-10 px-4 font-medium align-middle">{t('common.date')}</th>
              {/if}
              {#if visibleColumns.actions}
                <th class="h-10 w-24 px-4 font-medium text-right align-middle">{t('dashboard.table.action')}</th>
              {/if}
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            {#each paginatedItems as item (item.id)}
              <tr
                class="hover:bg-muted/50 transition-colors cursor-pointer {selectedIds.includes(item.id) ? 'bg-muted/40' : ''}"
                onclick={() => handleRowClick(item)}
              >
                <td
                  class="px-4 py-3 text-center align-middle"
                  onclick={(e) => { e.stopPropagation(); toggleSelectRow(item.id); }}
                >
                  <input
                    type="checkbox"
                    class="rounded border-input text-primary focus:ring-ring size-3.5 cursor-pointer align-middle"
                    checked={selectedIds.includes(item.id)}
                    onchange={() => toggleSelectRow(item.id)}
                    aria-label={t('dashboard.table.selectRow', { title: item.title })}
                  />
                </td>

                {#if visibleColumns.title}
                  <td class="px-4 py-3 align-middle font-medium text-xs text-foreground">
                    <div class="flex flex-col">
                      <span class="truncate max-w-xs sm:max-w-md">{item.title}</span>
                      {#if item.slug}
                        <span class="text-[11px] text-muted-foreground font-mono truncate max-w-xs">{item.slug}</span>
                      {/if}
                    </div>
                  </td>
                {/if}

                {#if visibleColumns.type}
                  <td class="px-4 py-3 align-middle">
                    {#if item.type === 'page'}
                      <Badge variant="outline" size="sm" class="capitalize">{t('common.page')}</Badge>
                    {:else}
                      <Badge variant="default" size="sm" class="capitalize">{t('common.news')}</Badge>
                    {/if}
                  </td>
                {/if}

                {#if visibleColumns.status}
                  <td class="px-4 py-3 align-middle">
                    <Badge
                      variant={item.status === 'published' ? 'success' : item.status === 'draft' ? 'warning' : 'outline'}
                      size="sm"
                      class="capitalize"
                    >
                      {statusLabel(item.status)}
                    </Badge>
                  </td>
                {/if}

                {#if visibleColumns.author}
                  <td class="px-4 py-3 align-middle text-xs text-muted-foreground">
                    {item.author}
                  </td>
                {/if}

                {#if visibleColumns.date}
                  <td class="px-4 py-3 align-middle text-xs text-muted-foreground whitespace-nowrap">
                    {new Date(item.date).toLocaleDateString()}
                  </td>
                {/if}

                {#if visibleColumns.actions}
                  <td class="px-4 py-3 text-right align-middle whitespace-nowrap">
                    {#if item.targetPath}
                      <button
                        type="button"
                        class="h-7 px-2.5 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs"
                        onclick={(e) => { e.stopPropagation(); handleRowClick(item); }}
                      >
                        {t('common.edit')}
                      </button>
                    {/if}
                  </td>
                {/if}
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <!-- Pagination & Row Count Footer (Exact shadcn specification) -->
      <div class="flex flex-col sm:flex-row items-center justify-between gap-4 px-4 py-3 border-t border-border text-xs text-muted-foreground">
        <div>
          <span>{t('dashboard.table.selected', { selected: selectedIds.length, total: filteredItems.length })}</span>
        </div>

        <div class="flex items-center gap-6">
          <div class="flex items-center gap-2">
            <span>{t('dashboard.table.rowsPerPage')}</span>
            <select
              bind:value={pageSize}
              onchange={() => { pageIndex = 0; }}
              class="h-7 rounded border border-input bg-background px-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring cursor-pointer"
            >
              <option value={10}>10</option>
              <option value={20}>20</option>
              <option value={50}>50</option>
            </select>
          </div>

          <span class="font-medium">
            {t('dashboard.table.pageOf', { page: pageIndex + 1, total: totalPages })}
          </span>

          <div class="flex items-center gap-1">
            <button
              type="button"
              disabled={pageIndex === 0}
              class="size-7 inline-flex items-center justify-center rounded border border-input bg-background hover:bg-accent text-foreground disabled:opacity-40 disabled:cursor-not-allowed transition-colors cursor-pointer shadow-xs"
              onclick={() => { if (pageIndex > 0) pageIndex--; }}
              aria-label={t('dashboard.table.prev')}
            >
              ‹
            </button>
            <button
              type="button"
              disabled={pageIndex >= totalPages - 1}
              class="size-7 inline-flex items-center justify-center rounded border border-input bg-background hover:bg-accent text-foreground disabled:opacity-40 disabled:cursor-not-allowed transition-colors cursor-pointer shadow-xs"
              onclick={() => { if (pageIndex < totalPages - 1) pageIndex++; }}
              aria-label={t('dashboard.table.next')}
            >
              ›
            </button>
          </div>
        </div>
      </div>
    {/if}
  </div>
</div>
