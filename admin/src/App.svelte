<script lang="ts">
  import { onMount } from 'svelte';
  import { auth } from '$lib/stores/auth.svelte';
  import { schema } from '$lib/stores/schema.svelte';
  import { router } from '$lib/router.svelte';
  import AppLayout from '$lib/components/shell/AppLayout.svelte';
  import LoginView from './routes/login/LoginView.svelte';
  import DashboardView from './routes/dashboard/DashboardView.svelte';
  import GenericListView from '$lib/schema/GenericListView.svelte';
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
  import AccountView from './routes/account/AccountView.svelte';
  import ContentTypesView from './routes/contenttypes/ContentTypesView.svelte';
  import DashboardSkeleton from '$lib/components/shell/DashboardSkeleton.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';

  import { i18n, isLocale } from '$lib/i18n/index.svelte';
  import { applyBrandColor } from '$lib/branding';
  import { brandingStore } from '$lib/stores/branding.svelte';

  let initializing = $state(true);

  onMount(async () => {
    void brandingStore.load();
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

  // Pick the admin language: the user's own choice, else the project default.
  $effect(() => {
    if (auth.user && schema.schema) {
      i18n.init(auth.user.id, schema.schema.locales?.admin_default ?? null);
    }
  });

  // Before sign-in there is no user choice yet: the sign-in page follows the project default.
  $effect(() => {
    const fallback = brandingStore.defaultLocale;
    if (!auth.user && fallback && isLocale(fallback)) i18n.setProjectDefault(fallback);
  });

  $effect(() => {
    const branding = brandingStore.branding ?? schema.schema?.branding;
    applyBrandColor(branding?.color);
    const name = branding?.name;
    if (name) document.title = name;
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
        return { view: 'list', type, modalId: 'new' };
      }
      if (actionOrId) {
        return { view: 'list', type, modalId: actionOrId };
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
  <DashboardSkeleton />
{:else if !auth.isAuthenticated}
  <LoginView />
{:else}
  <AppLayout>
    <div class="animate-fade-in w-full">
      {#if routeInfo.view === 'dashboard'}
        <DashboardView />
      {:else if routeInfo.view === 'list'}
        <GenericListView type={routeInfo.type!} initialItemId={routeInfo.modalId} />
      {:else if routeInfo.view === 'form'}
        <GenericListView type={routeInfo.type!} initialItemId={routeInfo.id} />
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
      {:else if routeInfo.module === 'account'}
        <AccountView />
      {:else if routeInfo.module === 'content-types'}
        <ContentTypesView />
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
    </div>
  </AppLayout>

  <!-- Global Navigation Guard Dialog for Unsaved Changes -->
  <ConfirmDialog
    open={router.showUnsavedModal}
    title="Kaydedilmemiş Değişiklikler"
    description={router.dirtyMessage}
    confirmLabel="Yine de Ayrıl"
    cancelLabel="Düzenlemeye Devam Et"
    variant="destructive"
    onconfirm={() => router.confirmLeave()}
    oncancel={() => router.cancelLeave()}
  />
{/if}
