<script lang="ts">
  import type { Snippet } from 'svelte';
  import Sidebar from './Sidebar.svelte';
  import Header from './Header.svelte';

  interface Props {
    children?: Snippet;
  }

  let { children }: Props = $props();
</script>

<div class="flex h-screen w-full bg-slate-950 text-slate-100 overflow-hidden font-sans">
  <Sidebar />
  <div class="flex-1 flex flex-col min-w-0 h-full overflow-hidden">
    <Header />
    <main class="flex-1 overflow-y-auto bg-slate-900/30 p-8">
      {#if children}
        {@render children()}
      {/if}
    </main>
  </div>
</div>
