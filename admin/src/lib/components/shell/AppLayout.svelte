<script lang="ts">
  import type { Snippet } from 'svelte';
  import Sidebar from './Sidebar.svelte';
  import Header from './Header.svelte';

  interface Props {
    children?: Snippet;
  }

  let { children }: Props = $props();
</script>

<div class="flex h-screen w-full bg-sidebar text-sidebar-foreground overflow-hidden font-sans">
  <Sidebar />
  <div class="relative flex min-w-0 flex-1 flex-col bg-background md:m-2 md:ml-0 md:rounded-xl md:border md:border-border md:shadow-xs overflow-hidden">
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
