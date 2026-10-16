<script lang="ts">
  import { tick } from 'svelte';
  import { router } from '$lib/router.svelte';
  import { schema } from '$lib/stores/schema.svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';

  interface Props {
    open: boolean;
    onclose: () => void;
  }

  let { open = $bindable(false), onclose }: Props = $props();

  let query = $state('');
  let searchInput = $state<HTMLInputElement | null>(null);
  let activeIndex = $state(0);
  let liveLoading = $state(false);
  let liveResults = $state<Array<{ title: string; subtitle: string; path: string; icon: string }>>([]);

  const baseNavItems = [
    { title: 'Dashboard', category: 'Navigation', path: '/dashboard', icon: 'dashboard', desc: 'Overview & metrics' },
    { title: 'Pages', category: 'Navigation', path: '/pages', icon: 'pages', desc: 'Manage hierarchical pages' },
    { title: 'News', category: 'Navigation', path: '/news', icon: 'news', desc: 'Articles & publications' },
    { title: 'Media Library', category: 'Navigation', path: '/media', icon: 'media', desc: 'Images, assets & files' },
    { title: 'Members', category: 'Navigation', path: '/members', icon: 'members', desc: 'Subscriber & member directory' },
    { title: 'Site Settings', category: 'Settings', path: '/site', icon: 'site', desc: 'Configuration, maintenance & redirects' },
    { title: 'Users & Roles', category: 'Settings', path: '/users', icon: 'users', desc: 'Staff users, access keys & permissions' },
    { title: 'System Telemetry', category: 'Settings', path: '/system', icon: 'system', desc: 'Runtime metrics, heap & health' },
    { title: 'My Account', category: 'Account', path: '/account', icon: 'account', desc: 'Security, password & personal tokens' },
    { title: 'Documentation', category: 'Help', path: '/admin/docs', icon: 'docs', desc: 'Architecture, guides & API references' },
  ];

  const quickActions = [
    { title: 'Create New Page', category: 'Quick Actions', path: '/pages/new', icon: 'add', desc: 'Draft a new page' },
    { title: 'Write News Article', category: 'Quick Actions', path: '/news/new', icon: 'add', desc: 'Draft a new article' },
    { title: 'Upload Media', category: 'Quick Actions', path: '/media', icon: 'upload', desc: 'Add files to media library' },
  ];

  const collectionItems = $derived.by(() => {
    return schema.collections.map((col) => ({
      title: col.plural || col.label || col.id,
      category: 'Collections',
      path: `/content/${col.id}`,
      icon: 'collection',
      desc: col.description || `Manage ${col.plural || col.id}`,
    }));
  });

  const filteredItems = $derived.by(() => {
    const q = query.trim().toLowerCase();
    const all = [...baseNavItems, ...collectionItems, ...quickActions];
    if (!q) {
      return all;
    }
    const matched = all.filter((item) =>
      item.title.toLowerCase().includes(q) ||
      item.desc.toLowerCase().includes(q) ||
      item.category.toLowerCase().includes(q)
    );

    const live = liveResults.map((r) => ({
      title: r.title,
      category: 'Content',
      path: r.path,
      icon: r.icon,
      desc: r.subtitle,
    }));

    return [...matched, ...live];
  });

  // Debounced live content search
  let searchTimeout: any = null;
  $effect(() => {
    const q = query.trim();
    if (q.length >= 2) {
      clearTimeout(searchTimeout);
      searchTimeout = setTimeout(async () => {
        liveLoading = true;
        try {
          const results: Array<{ title: string; subtitle: string; path: string; icon: string }> = [];

          // Search pages
          try {
            const pRes = await api.get<ApiResponse<any[]>>(`/api/v1/admin/pages?search=${encodeURIComponent(q)}&per_page=5`);
            if (pRes?.data && Array.isArray(pRes.data)) {
              for (const p of pRes.data) {
                results.push({
                  title: p.title || p.slug || 'Untitled Page',
                  subtitle: `Page • /${p.slug || ''}`,
                  path: `/pages/${p.id}`,
                  icon: 'pages',
                });
              }
            }
          } catch {}

          // Search news
          try {
            const nRes = await api.get<ApiResponse<any[]>>(`/api/v1/admin/news?search=${encodeURIComponent(q)}&per_page=5`);
            if (nRes?.data && Array.isArray(nRes.data)) {
              for (const n of nRes.data) {
                results.push({
                  title: n.title || n.slug || 'Untitled Article',
                  subtitle: `Article • ${n.published_at ? 'Published' : 'Draft'}`,
                  path: `/news/${n.id}`,
                  icon: 'news',
                });
              }
            }
          } catch {}

          liveResults = results;
        } finally {
          liveLoading = false;
        }
      }, 250);
    } else {
      liveResults = [];
    }
  });

  $effect(() => {
    if (open) {
      activeIndex = 0;
      query = '';
      liveResults = [];
      tick().then(() => {
        searchInput?.focus();
      });
    }
  });

  function handleSelect(item: { path: string }) {
    if (item.path.startsWith('http') || item.path === '/admin/docs') {
      window.open('/admin/docs', '_blank');
    } else {
      router.navigate(item.path);
    }
    onclose();
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (!open) return;

    if (e.key === 'Escape') {
      e.preventDefault();
      onclose();
      return;
    }

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      activeIndex = (activeIndex + 1) % Math.max(1, filteredItems.length);
      return;
    }

    if (e.key === 'ArrowUp') {
      e.preventDefault();
      activeIndex = (activeIndex - 1 + filteredItems.length) % Math.max(1, filteredItems.length);
      return;
    }

    if (e.key === 'Enter') {
      e.preventDefault();
      const current = filteredItems[activeIndex];
      if (current) {
        handleSelect(current);
      }
    }
  }
</script>

<svelte:window onkeydown={handleKeyDown} />

{#if open}
  <div class="fixed inset-0 z-50 flex items-start justify-center p-4 pt-16 sm:pt-24">
    <!-- Backdrop button -->
    <button
      type="button"
      class="fixed inset-0 bg-black/60 backdrop-blur-xs cursor-default transition-opacity border-0"
      onclick={onclose}
      aria-label="Close command search"
    ></button>

    <!-- Dialog Card -->
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Command search"
      class="relative z-10 w-full max-w-xl rounded-xl border border-border bg-popover text-popover-foreground shadow-2xl overflow-hidden flex flex-col max-h-[75vh] animate-in zoom-in-95 duration-150"
    >
      <!-- Search Input Header -->
      <div class="flex items-center gap-3 px-4 border-b border-border bg-card/50">
        <svg class="size-4 text-muted-foreground shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"></circle>
          <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        </svg>
        <input
          bind:this={searchInput}
          bind:value={query}
          type="text"
          placeholder="Search navigation, collections, content..."
          class="flex-1 h-12 bg-transparent text-sm text-foreground placeholder:text-muted-foreground outline-hidden"
        />
        {#if liveLoading}
          <svg class="size-4 animate-spin text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12a9 9 0 1 1-6.219-8.56"></path>
          </svg>
        {/if}
        <kbd class="px-1.5 py-0.5 text-[10px] font-mono bg-muted text-muted-foreground border border-border rounded">ESC</kbd>
      </div>

      <!-- Results List -->
      <div class="flex-1 overflow-y-auto p-2 divide-y divide-border/40 space-y-1">
        {#if filteredItems.length === 0}
          <div class="py-12 text-center text-sm text-muted-foreground">
            No matching navigation or content found for "{query}".
          </div>
        {:else}
          {#each filteredItems as item, idx}
            <button
              type="button"
              onclick={() => handleSelect(item)}
              onmouseenter={() => (activeIndex = idx)}
              class="w-full flex items-center justify-between gap-3 px-3 py-2.5 rounded-lg text-left text-sm transition-colors cursor-pointer {idx === activeIndex ? 'bg-accent text-accent-foreground' : 'text-foreground hover:bg-accent/60'}"
            >
              <div class="flex items-center gap-3 min-w-0">
                <div class="size-7 rounded-md bg-muted/60 flex items-center justify-center text-muted-foreground shrink-0">
                  {#if item.icon === 'dashboard'}
                    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="9" rx="1"></rect><rect x="14" y="3" width="7" height="5" rx="1"></rect><rect x="14" y="12" width="7" height="9" rx="1"></rect><rect x="3" y="16" width="7" height="5" rx="1"></rect></svg>
                  {:else if item.icon === 'pages'}
                    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline></svg>
                  {:else if item.icon === 'news'}
                    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"></path></svg>
                  {:else if item.icon === 'media'}
                    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg>
                  {:else if item.icon === 'members'}
                    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path></svg>
                  {:else if item.icon === 'site'}
                    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>
                  {:else if item.icon === 'users'}
                    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><line x1="19" y1="8" x2="19" y2="14"></line><line x1="22" y1="11" x2="16" y2="11"></line></svg>
                  {:else if item.icon === 'system'}
                    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline></svg>
                  {:else if item.icon === 'account'}
                    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg>
                  {:else if item.icon === 'docs'}
                    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"></path><path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"></path></svg>
                  {:else if item.icon === 'add'}
                    <svg class="size-4 text-primary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="16"></line><line x1="8" y1="12" x2="16" y2="12"></line></svg>
                  {:else}
                    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 6h16M4 12h16M4 18h16"></path></svg>
                  {/if}
                </div>
                <div class="grid text-left leading-tight min-w-0">
                  <span class="font-medium truncate">{item.title}</span>
                  <span class="text-xs text-muted-foreground truncate">{item.desc}</span>
                </div>
              </div>
              <span class="text-[10px] font-mono uppercase px-2 py-0.5 rounded bg-muted/60 text-muted-foreground border border-border/50 shrink-0">
                {item.category}
              </span>
            </button>
          {/each}
        {/if}
      </div>

      <!-- Footer Guide -->
      <div class="px-4 py-2 border-t border-border bg-card/40 flex items-center justify-between text-[11px] text-muted-foreground">
        <div class="flex items-center gap-3">
          <span class="flex items-center gap-1"><kbd class="font-mono bg-muted px-1.5 py-0.5 rounded border border-border">↑</kbd><kbd class="font-mono bg-muted px-1.5 py-0.5 rounded border border-border">↓</kbd> navigate</span>
          <span class="flex items-center gap-1"><kbd class="font-mono bg-muted px-1.5 py-0.5 rounded border border-border">Enter</kbd> select</span>
          <span class="flex items-center gap-1"><kbd class="font-mono bg-muted px-1.5 py-0.5 rounded border border-border">Esc</kbd> close</span>
        </div>
        <span class="font-mono">{filteredItems.length} results</span>
      </div>
    </div>
  </div>
{/if}
