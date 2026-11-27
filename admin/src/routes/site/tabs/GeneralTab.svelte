<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { schema } from '$lib/stores/schema.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import MediaUrlInput from '$lib/components/media/MediaUrlInput.svelte';

  let {
    settings = $bindable<Record<string, string>>({}),
    saving = false,
    onsubmit,
  }: {
    settings: Record<string, string>;
    saving: boolean;
    onsubmit: (e: SubmitEvent) => void;
  } = $props();
</script>

<form {onsubmit} class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-6">
  <div class="space-y-4">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label for="site-name" class="block text-xs font-medium text-foreground mb-1">{t('site.general.name')}</label>
        <Input id="site-name" type="text" bind:value={settings['site.name']} placeholder="Acme Studio" />
      </div>
      <div>
        <label for="site-title" class="block text-xs font-medium text-foreground mb-1">{t('site.general.title')}</label>
        <Input id="site-title" type="text" bind:value={settings['site.title']} placeholder="Acme Studio | Modern Web Experiences" />
      </div>
    </div>

    <div>
      <label for="site-desc" class="block text-xs font-medium text-foreground mb-1">{t('site.general.description')}</label>
      <Input id="site-desc" type="text" bind:value={settings['site.description']} placeholder={t('site.descPlaceholder')} />
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label for="home-path" class="block text-xs font-medium text-foreground mb-1">{t('site.general.homePath')}</label>
        <Input id="home-path" type="text" bind:value={settings['site.home_path']} placeholder="/" required />
      </div>
      <div>
        <label for="site-logo-url" class="block text-xs font-medium text-foreground mb-1">{t('site.general.logo')}</label>
        <MediaUrlInput id="site-logo-url" bind:value={settings['appearance.logo_url']} placeholder="/brand/logo.svg or https://..." />
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label for="favicon-url" class="block text-xs font-medium text-foreground mb-1">{t('site.general.favicon')}</label>
        <MediaUrlInput id="favicon-url" bind:value={settings['appearance.favicon_url']} placeholder="/favicon.svg or https://..." />
      </div>
      <div>
        <label for="social-img" class="block text-xs font-medium text-foreground mb-1">{t('site.general.socialImage')}</label>
        <MediaUrlInput id="social-img" bind:value={settings['appearance.social_image_url']} placeholder="/media/... or https://..." />
      </div>
    </div>

    <div>
      <label for="admin-default-locale" class="block text-xs font-medium text-foreground mb-1">{t('language.project')}</label>
      <select
        id="admin-default-locale"
        bind:value={settings['admin.default_locale']}
        class="h-9 w-full max-w-xs rounded-md border border-input bg-background px-3 text-sm text-foreground"
      >
        <option value="">{t('language.auto')}</option>
        <option value="en">{t('language.en')}</option>
        <option value="tr">{t('language.tr')}</option>
      </select>
      <p class="text-[11px] text-muted-foreground mt-1">{t('language.projectHelp')}</p>
    </div>
  </div>

  <!-- Storefront & Campaign Controls (only shown for built-in commerce storefront) -->
  {#if schema.modules?.commerce === true && !schema.externalStorefront}
  <div class="space-y-4 rounded-lg border border-border p-4">
    <div>
      <p class="text-xs font-semibold text-foreground">{t('site.storefront.title')}</p>
      <p class="text-[11px] text-muted-foreground mt-0.5">{t('site.storefront.help')}</p>
    </div>

    <div>
      <label for="sf-announcement" class="block text-xs font-medium text-foreground mb-1">{t('site.storefront.announcement')}</label>
      <textarea
        id="sf-announcement"
        rows={3}
        bind:value={settings['storefront.announcement_bar']}
        placeholder={'Free shipping on all orders | /shipping\nSame-day dispatch | /shipping'}
        class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
      ></textarea>
      <p class="text-[11px] text-muted-foreground mt-1">{t('site.storefront.announcementHelp')}</p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label for="sf-whatsapp" class="block text-xs font-medium text-foreground mb-1">{t('site.storefront.whatsapp')}</label>
        <Input id="sf-whatsapp" type="tel" bind:value={settings['storefront.whatsapp_number']} placeholder="+1 555 000 0000" />
      </div>
      <div>
        <label for="sf-tagline" class="block text-xs font-medium text-foreground mb-1">{t('site.storefront.footerTagline')}</label>
        <Input id="sf-tagline" type="text" bind:value={settings['storefront.footer_tagline']} placeholder="Modern digital storefront." />
      </div>
    </div>

    <div class="space-y-3 pt-2 border-t border-border">
      <p class="text-xs font-medium text-foreground">{t('site.storefront.heroTitleSection')}</p>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label for="sf-hero-eyebrow" class="block text-xs font-medium text-foreground mb-1">{t('site.storefront.heroEyebrow')}</label>
          <Input id="sf-hero-eyebrow" type="text" bind:value={settings['storefront.hero_eyebrow']} placeholder="Acme Studio" />
        </div>
        <div>
          <label for="sf-hero-title" class="block text-xs font-medium text-foreground mb-1">{t('site.storefront.heroTitle')}</label>
          <Input id="sf-hero-title" type="text" bind:value={settings['storefront.hero_title']} placeholder="Built for speed | and scale" />
        </div>
      </div>
      <div>
        <label for="sf-hero-sub" class="block text-xs font-medium text-foreground mb-1">{t('site.storefront.heroSubtitle')}</label>
        <Input id="sf-hero-sub" type="text" bind:value={settings['storefront.hero_subtitle']} placeholder="Explore the new collection..." />
      </div>
      <div>
        <label for="sf-hero-img" class="block text-xs font-medium text-foreground mb-1">{t('site.storefront.heroImage')}</label>
        <MediaUrlInput id="sf-hero-img" bind:value={settings['storefront.hero_image_url']} placeholder="/media/hero.jpg" />
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label for="sf-hero-cta" class="block text-xs font-medium text-foreground mb-1">{t('site.storefront.heroCtaLabel')}</label>
          <Input id="sf-hero-cta" type="text" bind:value={settings['storefront.hero_cta_label']} placeholder="Browse Catalog" />
        </div>
        <div>
          <label for="sf-hero-href" class="block text-xs font-medium text-foreground mb-1">{t('site.storefront.heroCtaHref')}</label>
          <Input id="sf-hero-href" type="text" bind:value={settings['storefront.hero_cta_href']} placeholder="/products" />
        </div>
      </div>
    </div>
  </div>
  {/if}

  <!-- Admin Panel Branding -->
  <div class="space-y-3 rounded-lg border border-border p-4">
    <p class="text-xs font-semibold text-foreground">{t('brand.title')}</p>
    <p class="text-[11px] text-muted-foreground">{t('brand.help')}</p>
    <div>
      <label for="brand-name" class="block text-xs font-medium text-foreground mb-1">{t('brand.name')}</label>
      <Input id="brand-name" type="text" bind:value={settings['admin.brand_name']} placeholder="HydroGo" maxlength={60} />
    </div>
    <div>
      <label for="brand-logo" class="block text-xs font-medium text-foreground mb-1">{t('brand.logo')}</label>
      <MediaUrlInput id="brand-logo" bind:value={settings['admin.brand_logo_url']} placeholder="https://..." />
    </div>
    <div class="flex items-center gap-3">
      <label for="brand-color" class="text-xs font-medium text-foreground">{t('brand.color')}</label>
      <input
        id="brand-color"
        type="color"
        value={settings['admin.brand_color'] || '#18181b'}
        oninput={(e) => (settings['admin.brand_color'] = e.currentTarget.value)}
        class="h-8 w-12 cursor-pointer rounded border border-input bg-background"
      />
      <Button type="button" variant="outline" size="sm" onclick={() => (settings['admin.brand_color'] = '')}>{t('brand.reset')}</Button>
    </div>
  </div>

  <div class="pt-3 border-t border-border flex justify-end">
    <Button type="submit" disabled={saving}>{saving ? t('common.saving') : t('site.general.save')}</Button>
  </div>
</form>
