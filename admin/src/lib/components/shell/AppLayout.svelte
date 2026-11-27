<script lang="ts">
  import type { Snippet } from 'svelte';
  import { t } from '$lib/i18n/index.svelte';
  import Sidebar from './Sidebar.svelte';
  import Header from './Header.svelte';

  interface Props {
    children?: Snippet;
  }

  let { children }: Props = $props();
  const isDemoMode = import.meta.env.VITE_DEMO_MODE === 'true';

  async function handleResetDemo() {
    const { resetDemoState } = await import('$lib/demo/mockApi');
    resetDemoState();
  }
</script>

<div class="flex h-screen w-full bg-sidebar text-sidebar-foreground overflow-hidden font-sans">
  <Sidebar />
  <div class="relative flex min-w-0 flex-1 flex-col bg-background md:m-2 md:ml-0 md:rounded-xl md:border md:border-border md:shadow-xs overflow-hidden">
    {#if isDemoMode}
      <div class="flex items-center justify-between gap-3 border-b border-border bg-muted/60 px-4 py-1.5 text-xs text-muted-foreground">
        <span>
          <strong class="font-medium text-foreground">{t('demo.liveBadge')}</strong> - {t('demo.runningInBrowser')} <code class="font-mono text-[11px]">localStorage</code> {t('demo.noDatabase')}
        </span>
        <div class="flex items-center gap-3 shrink-0">
          <a href="/" class="hover:text-foreground underline underline-offset-2">{t('demo.backToSite')}</a>
          <button
            type="button"
            onclick={handleResetDemo}
            class="rounded border border-border bg-background px-2 py-0.5 text-[11px] font-medium text-foreground hover:bg-accent cursor-pointer"
          >
            {t('demo.resetData')}
          </button>
        </div>
      </div>
    {/if}
    <Header />
    <main class="flex-1 overflow-y-auto bg-background min-w-0">
      <!-- One frame for every screen: same left edge and width, never centred. -->
      <div class="admin-page w-full px-4 py-4 lg:px-6 lg:py-6">
        {#if children}
          {@render children()}
        {/if}
      </div>
    </main>
  </div>
</div>
