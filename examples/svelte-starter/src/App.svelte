<script lang="ts">
  import { onMount } from 'svelte';
  import Header from './lib/Header.svelte';
  import HomeView from './lib/HomeView.svelte';
  import DocsView from './lib/DocsView.svelte';
  import SearchDialog from './lib/components/SearchDialog.svelte';
  import { router } from './lib/router.svelte';
  import { theme } from './lib/theme.svelte';
  import { ui } from './lib/ui.svelte';
  import { setDefaultSEO } from './lib/seo';

  onMount(() => {
    theme.init();
    router.init();

    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        ui.searchOpen = !ui.searchOpen;
        return;
      }
      const el = e.target as HTMLElement | null;
      const typing = el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable);
      if (e.key === '/' && !typing && !e.ctrlKey && !e.metaKey && !e.altKey) {
        e.preventDefault();
        ui.searchOpen = true;
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  });

  $effect(() => {
    if (router.current.name === 'home') {
      setDefaultSEO();
      window.scrollTo({ top: 0, behavior: 'instant' as ScrollBehavior });
    }
  });
</script>

<!-- A hash link would be read by the router, so the skip link moves focus itself. -->
<button
  type="button"
  class="sr-only z-[70] rounded-md bg-accent px-3 py-2 text-accent-fg focus:not-sr-only focus:fixed focus:top-2 focus:left-2"
  onclick={() => document.getElementById('main')?.focus()}>Skip to content</button
>

<div class="flex min-h-screen flex-col bg-bg text-fg">
  <Header />
  <main id="main" class="flex-1" tabindex="-1">
    {#if router.current.name === 'docs'}
      <DocsView />
    {:else}
      <HomeView />
    {/if}
  </main>
  <SearchDialog />
</div>
