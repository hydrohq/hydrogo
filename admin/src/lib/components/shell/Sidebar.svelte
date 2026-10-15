<script lang="ts">
  import { auth } from '$lib/stores/auth.svelte';
  import { schema } from '$lib/stores/schema.svelte';
  import { router } from '$lib/router.svelte';

  const currentPath = $derived(router.path);

  function isActive(target: string): boolean {
    if (target === '/' || target === '/dashboard') {
      return currentPath === '/' || currentPath === '/dashboard';
    }
    return currentPath.startsWith(target);
  }
</script>

<aside class="w-64 flex-shrink-0 bg-slate-950 border-r border-slate-800/80 flex flex-col h-full select-none">
  <!-- Brand -->
  <div class="h-16 px-6 flex items-center gap-3 border-b border-slate-800/80">
    <div class="w-8 h-8 rounded-lg bg-sky-500/10 border border-sky-500/25 flex items-center justify-center text-sky-400 font-bold">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
        <polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
        <polyline points="2 17 12 22 22 17"></polyline>
        <polyline points="2 12 12 17 22 12"></polyline>
      </svg>
    </div>
    <div class="flex flex-col">
      <span class="text-sm font-bold tracking-tight text-slate-100 flex items-center gap-1">
        Hydro<span class="text-sky-400">Go</span>
      </span>
      <span class="text-[10px] text-slate-500 font-mono tracking-wider">ADMIN v2.0</span>
    </div>
  </div>

  <!-- Navigation -->
  <div class="flex-1 overflow-y-auto px-3 py-4 space-y-6">
    <!-- Main -->
    <div>
      <div class="px-3 mb-2 text-[10px] font-bold uppercase tracking-wider text-slate-400">
        Overview
      </div>
      <nav class="space-y-0.5">
        <a
          href="#/dashboard"
          class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors {isActive('/dashboard') ? 'bg-sky-500/10 text-sky-400 font-semibold' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/60'}"
        >
          <span class="text-base">📊</span>
          Dashboard
        </a>
      </nav>
    </div>

    <!-- Schema-driven Content Types -->
    {#if schema.resources.length > 0}
      <div>
        <div class="px-3 mb-2 text-[10px] font-bold uppercase tracking-wider text-slate-400 flex items-center justify-between">
          <span>Content</span>
          <span class="text-[10px] bg-slate-900 border border-slate-800 text-slate-400 px-1.5 py-0.2 rounded-full font-mono">
            {schema.resources.length}
          </span>
        </div>
        <nav class="space-y-0.5">
          {#each schema.resources as res (res.id)}
            <a
              href={`#/content/${res.id}`}
              class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors {isActive(`/content/${res.id}`) ? 'bg-sky-500/10 text-sky-400 font-semibold' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/60'}"
            >
              <span class="text-base">{res.icon || '📄'}</span>
              <span class="truncate">{res.label || res.plural}</span>
            </a>
          {/each}
        </nav>
      </div>
    {/if}

    <!-- Core Modules -->
    <div>
      <div class="px-3 mb-2 text-[10px] font-bold uppercase tracking-wider text-slate-400">
        Modules
      </div>
      <nav class="space-y-0.5">
        {#if schema.modules['pages'] !== false}
          <a
            href="#/pages"
            class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors {isActive('/pages') ? 'bg-sky-500/10 text-sky-400 font-semibold' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/60'}"
          >
            <span class="text-base">📑</span>
            Pages
          </a>
        {/if}
        {#if schema.modules['news'] !== false}
          <a
            href="#/news"
            class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors {isActive('/news') ? 'bg-sky-500/10 text-sky-400 font-semibold' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/60'}"
          >
            <span class="text-base">📰</span>
            News
          </a>
        {/if}
        {#if schema.modules['media'] !== false}
          <a
            href="#/media"
            class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors {isActive('/media') ? 'bg-sky-500/10 text-sky-400 font-semibold' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/60'}"
          >
            <span class="text-base">🖼️</span>
            Media Library
          </a>
        {/if}
        {#if schema.modules['members'] !== false}
          <a
            href="#/members"
            class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors {isActive('/members') ? 'bg-sky-500/10 text-sky-400 font-semibold' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/60'}"
          >
            <span class="text-base">👥</span>
            Members
          </a>
        {/if}
        {#if schema.modules['site'] !== false}
          <a
            href="#/site"
            class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors {isActive('/site') ? 'bg-sky-500/10 text-sky-400 font-semibold' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/60'}"
          >
            <span class="text-base">⚙️</span>
            Site Settings
          </a>
        {/if}
      </nav>
    </div>

    <!-- Management -->
    {#if auth.user?.role === 'admin'}
      <div>
        <div class="px-3 mb-2 text-[10px] font-bold uppercase tracking-wider text-slate-400">
          Settings & Ops
        </div>
        <nav class="space-y-0.5">
          <a
            href="#/users"
            class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors {isActive('/users') ? 'bg-sky-500/10 text-sky-400 font-semibold' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/60'}"
          >
            <span class="text-base">🔐</span>
            Users & Roles
          </a>
          <a
            href="#/audit"
            class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors {isActive('/audit') ? 'bg-sky-500/10 text-sky-400 font-semibold' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/60'}"
          >
            <span class="text-base">📋</span>
            Audit Logs
          </a>
          <a
            href="#/system"
            class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors {isActive('/system') ? 'bg-sky-500/10 text-sky-400 font-semibold' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900/60'}"
          >
            <span class="text-base">⚡</span>
            System & Metrics
          </a>
        </nav>
      </div>
    {/if}
  </div>

  <!-- User Footer -->
  <div class="p-3 border-t border-slate-800/80 bg-slate-950/60">
    <div class="flex items-center justify-between p-2 rounded-lg hover:bg-slate-900 transition-colors">
      <div class="flex items-center gap-2.5 min-w-0">
        <div class="w-8 h-8 rounded-full bg-slate-800 border border-slate-700 flex items-center justify-center font-bold text-xs text-sky-400 flex-shrink-0">
          {auth.user?.email?.[0]?.toUpperCase() || 'U'}
        </div>
        <div class="min-w-0">
          <p class="text-xs font-semibold text-slate-200 truncate">{auth.user?.email || 'User'}</p>
          <span class="text-[10px] text-slate-400 uppercase tracking-wider">{auth.user?.role || 'editor'}</span>
        </div>
      </div>
      <button
        type="button"
        onclick={() => auth.logout()}
        class="text-slate-400 hover:text-rose-400 p-1.5 rounded-md hover:bg-slate-800/60 transition-colors cursor-pointer"
        title="Log out"
      >
        <span class="text-sm">🚪</span>
      </button>
    </div>
  </div>
</aside>
