<script lang="ts">
  import './layout.css';
  import Header from '#lib/components/Header.svelte';
  import LiveInfoBar from '#lib/components/LiveInfoBar.svelte';
  import WhatsAppTipButton from '#lib/components/WhatsAppTipButton.svelte';
  import PushNotificationPrompt from '#lib/components/PushNotificationPrompt.svelte';
  import Footer from '#lib/components/Footer.svelte';
  import type { LayoutProps } from './$types';

  let { children, data }: LayoutProps = $props();
</script>

<svelte:head>
  <title>HydroPost — Bağımsız ve Güçlü Haber Portalı</title>
  <meta name="description" content="HydroGo tekil binary mimarisiyle çalışan, ışık hızında haber portalı." />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
</svelte:head>

<div class="min-h-screen bg-white text-zinc-900 font-sans antialiased selection:bg-red-600 selection:text-white flex flex-col justify-between">
  <div>
    <Header />
    <LiveInfoBar />
    <main class="max-w-7xl mx-auto px-3 sm:px-4 py-4 w-full">
      {@render children()}
    </main>
  </div>
  {#if data.whatsappPhone}
    <WhatsAppTipButton phone={data.whatsappPhone} />
  {/if}
  <PushNotificationPrompt />
  <Footer />
</div>
