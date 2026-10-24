<script lang="ts">
  import './layout.css';
  import { getAssetUrl } from '#lib';
  import Header from '#lib/components/Header.svelte';
  import WhatsAppTipButton from '#lib/components/WhatsAppTipButton.svelte';
  import PushNotificationPrompt from '#lib/components/PushNotificationPrompt.svelte';
  import Footer from '#lib/components/Footer.svelte';
  import { navigating } from '$app/state';
  import { onMount } from 'svelte';
  import type { LayoutProps } from './$types';

  let { children, data }: LayoutProps = $props();
  let favicon = $derived(getAssetUrl(data.siteSettings.favicon) || '/favicon.svg');

  onMount(() => {
    const loader = document.getElementById('hp-initial-loader');
    if (loader) {
      loader.remove();
    }
  });

  $effect(() => {
    // Secondary safety: ensure loader is gone as soon as reactive state settles
    if (typeof document !== 'undefined') {
      const loader = document.getElementById('hp-initial-loader');
      if (loader) loader.remove();
    }
  });
</script>

<svelte:head>
  <title>{data.siteSettings.seo_title || data.siteSettings.site_name}</title>
  <meta name="description" content={data.siteSettings.seo_description || ''} />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <link rel="icon" href={favicon} />
</svelte:head>

{#if navigating.to}
  <div class="fixed top-0 left-0 right-0 h-1 z-[100] bg-red-100 overflow-hidden">
    <div class="h-full bg-[var(--hp-primary)] animate-hp-progress w-full"></div>
  </div>
{/if}

<div class="min-h-screen bg-[var(--hp-bg)] text-[var(--hp-ink-soft)] antialiased flex flex-col justify-between">
  <div>
    <Header nav={data.headerNav} breakingNews={data.breakingNews} siteName={data.siteSettings.site_name} />
    <main class="max-w-7xl mx-auto px-3 sm:px-4 py-4 w-full transition-opacity duration-150 {navigating.to ? 'opacity-40 pointer-events-none' : 'opacity-100'}">
      {@render children()}
    </main>
  </div>
  {#if data.whatsappPhone}
    <WhatsAppTipButton phone={data.whatsappPhone} />
  {/if}
  <PushNotificationPrompt />
  <Footer settings={data.siteSettings} pagesNav={data.footerPagesNav} socialNav={data.footerSocialNav} />
</div>
