<script lang="ts">
  import './layout.css';
  import { getAssetUrl } from '#lib';
  import Header from '#lib/components/Header.svelte';
  import WhatsAppTipButton from '#lib/components/WhatsAppTipButton.svelte';
  import PushNotificationPrompt from '#lib/components/PushNotificationPrompt.svelte';
  import Footer from '#lib/components/Footer.svelte';
  import type { LayoutProps } from './$types';

  let { children, data }: LayoutProps = $props();
  let favicon = $derived(getAssetUrl(data.siteSettings.favicon) || '/favicon.svg');
</script>

<svelte:head>
  <title>{data.siteSettings.seo_title || data.siteSettings.site_name}</title>
  <meta name="description" content={data.siteSettings.seo_description || ''} />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <link rel="icon" href={favicon} />
</svelte:head>

<div class="min-h-screen bg-[var(--hp-bg)] text-[var(--hp-ink-soft)] antialiased flex flex-col justify-between">
  <div>
    <Header nav={data.headerNav} breakingNews={data.breakingNews} siteName={data.siteSettings.site_name} />
    <main class="max-w-7xl mx-auto px-3 sm:px-4 py-4 w-full">
      {@render children()}
    </main>
  </div>
  {#if data.whatsappPhone}
    <WhatsAppTipButton phone={data.whatsappPhone} />
  {/if}
  <PushNotificationPrompt />
  <Footer settings={data.siteSettings} pagesNav={data.footerPagesNav} socialNav={data.footerSocialNav} />
</div>
