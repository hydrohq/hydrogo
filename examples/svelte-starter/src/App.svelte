<script lang="ts">
  import Header from './lib/Header.svelte';
  import HomeView from './lib/HomeView.svelte';
  import PageView from './lib/PageView.svelte';
  import NewsView from './lib/NewsView.svelte';

  let route = $state('/');
  let activeSlug = $state('');

  function navigateTo(path: string) {
    route = path;
    activeSlug = '';
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

<div class="min-h-screen bg-slate-950 flex flex-col">
  <Header currentRoute={route} onnavigate={navigateTo} />

  <main class="flex-1 max-w-6xl w-full mx-auto px-6 py-8">
    {#if route === '/'}
      <HomeView onopenpage={openPage} onopennews={openNews} />
    {:else if route === '/page'}
      <PageView slug={activeSlug} onback={() => navigateTo('/')} />
    {:else if route === '/news' || route === '/news-article'}
      {#if activeSlug}
        <NewsView slug={activeSlug} onback={() => { activeSlug = ''; route = '/'; }} />
      {:else}
        <HomeView onopenpage={openPage} onopennews={openNews} />
      {/if}
    {/if}
  </main>

  <footer class="border-t border-slate-800/80 py-8 text-center text-xs text-slate-500">
    <p>HydroGo v2 Headless Frontend Example • Pure JSON API</p>
  </footer>
</div>
