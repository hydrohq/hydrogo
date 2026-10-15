<script lang="ts">
  import { onMount } from 'svelte';
  import { auth } from '$lib/stores/auth.svelte';
  import { schema } from '$lib/stores/schema.svelte';
  import { router } from '$lib/router.svelte';
  import AppLayout from '$lib/components/shell/AppLayout.svelte';
  import LoginView from './routes/login/LoginView.svelte';
  import DashboardView from './routes/dashboard/DashboardView.svelte';
  import GenericListView from '$lib/schema/GenericListView.svelte';
  import GenericFormView from '$lib/schema/GenericFormView.svelte';
  import MediaView from './routes/media/MediaView.svelte';
  import NewsListView from './routes/news/NewsListView.svelte';
  import NewsEditorView from './routes/news/NewsEditorView.svelte';
  import PagesListView from './routes/pages/PagesListView.svelte';
  import PageEditorView from './routes/pages/PageEditorView.svelte';

  let initializing = $state(true);

  onMount(async () => {
    try {
      const isAuthed = await auth.checkAuth();
      if (isAuthed) {
        await schema.loadSchema();
        if (router.path === '/login') {
          router.navigate('/dashboard');
        }
      } else {
        if (!router.path.includes('/login')) {
          router.navigate('/login');
        }
      }
    } finally {
      initializing = false;
    }
  });

  const routeInfo = $derived.by(() => {
    const p = router.path;
    if (p === '/' || p === '/dashboard') {
      return { view: 'dashboard' };
    }
    if (p === '/login') {
      return { view: 'login' };
    }
    if (p.startsWith('/content/')) {
      const parts = p.split('/');
      const type = parts[2];
      const actionOrId = parts[3];
      if (actionOrId === 'new') {
        return { view: 'form', type, id: 'new' };
      }
      if (actionOrId) {
        return { view: 'form', type, id: actionOrId };
      }
      return { view: 'list', type };
    }
    if (p.startsWith('/news')) {
      const parts = p.split('/');
      const actionOrId = parts[2];
      if (actionOrId === 'new') {
        return { view: 'news-editor', id: 'new' };
      }
      if (actionOrId) {
        return { view: 'news-editor', id: actionOrId };
      }
      return { view: 'news-list' };
    }
    if (p.startsWith('/pages')) {
      const parts = p.split('/');
      const actionOrId = parts[2];
      if (actionOrId === 'new') {
        return { view: 'page-editor', id: 'new' };
      }
      if (actionOrId) {
        return { view: 'page-editor', id: actionOrId };
      }
      return { view: 'pages-list' };
    }
    return { view: 'custom', module: p.replace(/^\//, '') };
  });
</script>

{#if initializing}
  <div class="h-screen w-full bg-slate-950 flex flex-col items-center justify-center text-slate-400 gap-3">
    <div class="w-10 h-10 rounded-xl bg-sky-500/10 border border-sky-500/20 flex items-center justify-center text-sky-400 font-bold animate-pulse">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
        <polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
        <polyline points="2 17 12 22 22 17"></polyline>
        <polyline points="2 12 12 17 22 12"></polyline>
      </svg>
    </div>
    <p class="text-xs font-mono tracking-wider text-slate-500 uppercase">Starting HydroGo Admin…</p>
  </div>
{:else if !auth.isAuthenticated}
  <LoginView />
{:else}
  <AppLayout>
    {#if routeInfo.view === 'dashboard'}
      <DashboardView />
    {:else if routeInfo.view === 'list'}
      <GenericListView type={routeInfo.type!} />
    {:else if routeInfo.view === 'form'}
      <GenericFormView type={routeInfo.type!} id={routeInfo.id} />
    {:else if routeInfo.view === 'news-list'}
      <NewsListView />
    {:else if routeInfo.view === 'news-editor'}
      <NewsEditorView id={routeInfo.id!} />
    {:else if routeInfo.view === 'pages-list'}
      <PagesListView />
    {:else if routeInfo.view === 'page-editor'}
      <PageEditorView id={routeInfo.id!} />
    {:else if routeInfo.module === 'media'}
      <MediaView />
    {:else}
      <!-- Module screen fallback (renders generic collection if schema matches or info panel) -->
      {#if schema.getCollection(routeInfo.module || '')}
        <GenericListView type={routeInfo.module!} />
      {:else}
        <div class="p-8 max-w-2xl mx-auto rounded-2xl border border-slate-800 bg-slate-900/60 text-center space-y-3">
          <h3 class="text-lg font-bold text-slate-100 capitalize">{routeInfo.module}</h3>
          <p class="text-sm text-slate-400">
            Dedicated module interface will be rendered here.
          </p>
        </div>
      {/if}
    {/if}
  </AppLayout>
{/if}
