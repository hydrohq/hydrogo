<script lang="ts">
  import { onMount } from 'svelte';
  import Header from './lib/Header.svelte';
  import HomeView from './lib/HomeView.svelte';
  import PageView from './lib/PageView.svelte';
  import NewsView from './lib/NewsView.svelte';
  import DocsView from './lib/DocsView.svelte';

  import { setDefaultSEO, setDocsSEO } from './lib/seo';

  let route = $state('/');
  let activeSlug = $state('');

  function syncRouteFromHash() {
    const hash = window.location.hash.replace(/^#\/?/, '');
    if (hash === 'docs' || window.location.pathname === '/docs') {
      route = '/docs';
      setDocsSEO();
    } else if (route === '/') {
      setDefaultSEO();
    }
  }

  onMount(() => {
    syncRouteFromHash();
    window.addEventListener('hashchange', syncRouteFromHash);
    return () => {
      window.removeEventListener('hashchange', syncRouteFromHash);
    };
  });

  function navigateTo(path: string) {
    route = path;
    activeSlug = '';
    window.location.hash = path === '/' ? '' : path;
    if (path === '/') {
      setDefaultSEO();
    } else if (path === '/docs') {
      setDocsSEO();
    }
  }

  function openPage(slug: string) {
    activeSlug = slug;
    route = '/page';
  }

  function openNews(slug: string) {
    activeSlug = slug;
    route = '/news-article';
  }
</script>

{#if route === '/docs'}
  <DocsView onback={() => navigateTo('/')} />
{:else}
  <div class="min-h-screen bg-black flex flex-col text-white font-sans antialiased selection:bg-white selection:text-black">
    {#if route !== '/'}
      <Header currentRoute={route} onnavigate={navigateTo} />
    {/if}

    <main class="flex-1 w-full flex flex-col justify-center px-6 py-12">
      {#if route === '/'}
        <HomeView onopenpage={openPage} onopennews={openNews} onopendocs={() => navigateTo('/docs')} />
      {:else if route === '/page'}
        <PageView slug={activeSlug} onback={() => navigateTo('/')} />
      {:else if route === '/news' || route === '/news-article'}
        {#if activeSlug}
          <NewsView slug={activeSlug} onback={() => { activeSlug = ''; route = '/'; }} />
        {:else}
          <HomeView onopenpage={openPage} onopennews={openNews} onopendocs={() => navigateTo('/docs')} />
        {/if}
      {/if}
    </main>
  </div>
{/if}
