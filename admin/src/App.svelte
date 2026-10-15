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
  import MembersView from './routes/members/MembersView.svelte';
  import UsersView from './routes/users/UsersView.svelte';
  import SiteView from './routes/site/SiteView.svelte';
  import AuditView from './routes/audit/AuditView.svelte';
  import SystemView from './routes/system/SystemView.svelte';

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

  $effect(() => {
    if (auth.isAuthenticated && !schema.schema && !schema.isLoading) {
      schema.loadSchema();
    }
  });

  const routeInfo = $derived.by(() => {
    const p = router.path;
    if (p === '/' || p === '/dashboard' || p === '/admin' || p === '/admin/' || p === '') {
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
  <div class="h-screen w-full bg-background flex flex-col items-center justify-center text-muted-foreground gap-3">
    <div class="size-9 rounded-lg bg-primary text-primary-foreground flex items-center justify-center font-bold shadow-xs animate-pulse">
      <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="10"></circle>
        <polyline points="12 6 12 12 16 14"></polyline>
      </svg>
    </div>
    <p class="text-xs font-mono tracking-wider text-muted-foreground uppercase">Starting HydroGo Admin…</p>
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
    {:else if routeInfo.module === 'members'}
      <MembersView />
    {:else if routeInfo.module === 'users'}
      <UsersView />
    {:else if routeInfo.module === 'site'}
      <SiteView />
    {:else if routeInfo.module === 'audit'}
      <AuditView />
    {:else if routeInfo.module === 'system'}
      <SystemView />
    {:else}
      <!-- Module screen fallback (renders generic collection if schema matches or info panel) -->
      {#if schema.getCollection(routeInfo.module || '')}
        <GenericListView type={routeInfo.module!} />
      {:else}
        <div class="p-8 max-w-2xl mx-auto rounded-xl border border-border bg-card text-card-foreground text-center space-y-3 shadow-xs">
          <h3 class="text-lg font-semibold capitalize">{routeInfo.module}</h3>
          <p class="text-sm text-muted-foreground">
            Dedicated module interface will be rendered here.
          </p>
        </div>
      {/if}
    {/if}
  </AppLayout>
{/if}
