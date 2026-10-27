<script lang="ts">
  import { onMount } from 'svelte';
  import { auth } from '$lib/stores/auth.svelte';
  import { schema } from '$lib/stores/schema.svelte';
  import { router } from '$lib/router.svelte';
  import AppLayout from '$lib/components/shell/AppLayout.svelte';
  import LoginView from './routes/login/LoginView.svelte';
  import SetPasswordView from './routes/account/SetPasswordView.svelte';
  import DashboardSkeleton from '$lib/components/shell/DashboardSkeleton.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
  import LazyView from '$lib/components/shell/LazyView.svelte';

  // Each route view ships in its own chunk, downloaded only when visited —
  // keeps the app shell small (the Pages/News editors pull in TipTap).
  const loadDashboardView = () => import('./routes/dashboard/DashboardView.svelte');
  const loadGenericListView = () => import('$lib/schema/GenericListView.svelte');
  const loadMediaView = () => import('./routes/media/MediaView.svelte');
  const loadNewsListView = () => import('./routes/news/NewsListView.svelte');
  const loadNewsEditorView = () => import('./routes/news/NewsEditorView.svelte');
  const loadPagesListView = () => import('./routes/pages/PagesListView.svelte');
  const loadPageEditorView = () => import('./routes/pages/PageEditorView.svelte');
  const loadMembersView = () => import('./routes/members/MembersView.svelte');
  const loadCommerceOrdersView = () => import('./routes/commerce/CommerceOrdersView.svelte');
  const loadUsersView = () => import('./routes/users/UsersView.svelte');
  const loadSiteView = () => import('./routes/site/SiteView.svelte');
  const loadAuditView = () => import('./routes/audit/AuditView.svelte');
  const loadSystemView = () => import('./routes/system/SystemView.svelte');
  const loadAccountView = () => import('./routes/account/AccountView.svelte');
  const loadContentTypesView = () => import('./routes/contenttypes/ContentTypesView.svelte');

  import { i18n, isLocale } from '$lib/i18n/index.svelte';
  import { applyBrandColor } from '$lib/branding';
  import { brandingStore } from '$lib/stores/branding.svelte';

  let initializing = $state(true);
  // Pages a person reaches from an e-mail before they have a session.
  const publicPaths = ['/login', '/reset-password', '/accept-invite'];
  const isPublicPath = (p: string) => publicPaths.some((x) => p === x || p.startsWith(`${x}?`));

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
        if (!isPublicPath(router.path)) {
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
{:else if router.path.startsWith('/reset-password')}
  <SetPasswordView mode="reset" />
{:else if router.path.startsWith('/accept-invite')}
  <SetPasswordView mode="invite" />
{:else if !auth.isAuthenticated}
  <LoginView />
{:else}
  <AppLayout>
    <div class="animate-fade-in w-full">
      {#if routeInfo.view === 'dashboard'}
        <LazyView load={loadDashboardView} />
      {:else if routeInfo.view === 'list'}
        <LazyView load={loadGenericListView} props={{ type: routeInfo.type, initialItemId: routeInfo.modalId }} />
      {:else if routeInfo.view === 'form'}
        <LazyView load={loadGenericListView} props={{ type: routeInfo.type, initialItemId: routeInfo.id }} />
      {:else if routeInfo.view === 'news-list'}
        <LazyView load={loadNewsListView} />
      {:else if routeInfo.view === 'news-editor'}
        <LazyView load={loadNewsEditorView} props={{ id: routeInfo.id }} />
      {:else if routeInfo.view === 'pages-list'}
        <LazyView load={loadPagesListView} />
      {:else if routeInfo.view === 'page-editor'}
        <LazyView load={loadPageEditorView} props={{ id: routeInfo.id }} />
      {:else if routeInfo.module === 'media'}
        <LazyView load={loadMediaView} />
      {:else if routeInfo.module === 'members'}
        <LazyView load={loadMembersView} />
      {:else if routeInfo.module === 'commerce'}
        <LazyView load={loadCommerceOrdersView} />
      {:else if routeInfo.module === 'users'}
        <LazyView load={loadUsersView} />
      {:else if routeInfo.module === 'site'}
        <LazyView load={loadSiteView} />
      {:else if routeInfo.module === 'audit'}
        <LazyView load={loadAuditView} />
      {:else if routeInfo.module === 'system'}
        <LazyView load={loadSystemView} />
      {:else if routeInfo.module === 'account'}
        <LazyView load={loadAccountView} />
      {:else if routeInfo.module === 'content-types'}
        <LazyView load={loadContentTypesView} />
      {:else}
        <!-- Module screen fallback (renders generic collection if schema matches or info panel) -->
        {#if schema.getCollection(routeInfo.module || '')}
          <LazyView load={loadGenericListView} props={{ type: routeInfo.module }} />
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
