<script lang="ts">
  import './layout.css';
  import Header from '#lib/components/Header.svelte';
  import Footer from '#lib/components/Footer.svelte';
  import type { LayoutProps } from './$types';

  let { children }: LayoutProps = $props();
</script>

<svelte:head>
  <title>HydroPost — Bağımsız ve Güçlü Haber Portalı</title>
  <meta name="description" content="HydroGo tekil binary mimarisiyle çalışan, ışık hızında haber portalı." />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
</svelte:head>

<div class="min-h-screen bg-black text-white font-sans antialiased selection:bg-red-600 selection:text-white flex flex-col justify-between">
  <div>
    <Header />
    <main class="max-w-7xl mx-auto px-3 sm:px-4 py-4 w-full">
      {@render children()}
    </main>
  </div>
  <Footer />
</div>
