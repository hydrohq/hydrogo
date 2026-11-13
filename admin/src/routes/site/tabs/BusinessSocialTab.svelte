<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import {
    SOCIAL_PLATFORM_LABELS,
    socialPlatformPlaceholder,
    type BusinessProfileForm,
    type SocialProfile,
  } from '../siteHelpers';

  let {
    business = $bindable<BusinessProfileForm>(),
    socialProfiles = $bindable<SocialProfile[]>([]),
    saving = false,
    onsubmit,
  }: {
    business: BusinessProfileForm;
    socialProfiles: SocialProfile[];
    saving: boolean;
    onsubmit: (e: SubmitEvent) => void;
  } = $props();
</script>

<form {onsubmit} class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-6">
  <div>
    <h3 class="text-sm font-semibold text-foreground">{t('site.biz.title')}</h3>
    <p class="text-xs text-muted-foreground mt-0.5">{t('site.biz.help')}</p>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
    <div>
      <label for="b-name" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.company')}</label>
      <Input id="b-name" type="text" bind:value={business.name} placeholder="BEYOND Leather" />
    </div>
    <div>
      <label for="b-legal" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.legal')}</label>
      <Input id="b-legal" type="text" bind:value={business.legal_name} placeholder="BEYOND Deri Ürünleri Ltd. Şti." />
    </div>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
    <div>
      <label for="b-tax-office" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.taxOffice')}</label>
      <Input id="b-tax-office" type="text" bind:value={business.tax_office} placeholder="Beyoğlu VD" />
    </div>
    <div>
      <label for="b-tax" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.tax')}</label>
      <Input id="b-tax" type="text" bind:value={business.tax_id} placeholder="1234567890" />
    </div>
    <div>
      <label for="b-mersis" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.mersis')}</label>
      <Input id="b-mersis" type="text" bind:value={business.mersis_no} placeholder="0123456789000001" />
    </div>
    <div>
      <label for="b-kep" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.kep')}</label>
      <Input id="b-kep" type="email" bind:value={business.kep_address} placeholder="firma@hs01.kep.tr" />
    </div>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
    <div>
      <label for="b-email" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.email')}</label>
      <Input id="b-email" type="email" bind:value={business.email} placeholder="destek@example.com" />
    </div>
    <div>
      <label for="b-phone" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.phone')}</label>
      <Input id="b-phone" type="tel" bind:value={business.phone} placeholder="+90 212 000 00 00" />
    </div>
    <div>
      <label for="b-website" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.website')}</label>
      <Input id="b-website" type="url" bind:value={business.website} placeholder="https://beyondleather.com" />
    </div>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
    <div class="md:col-span-2">
      <label for="b-addr" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.address')}</label>
      <Input id="b-addr" type="text" bind:value={business.address} placeholder="İstiklal Cad. No: 1" />
    </div>
    <div>
      <label for="b-city" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.city')}</label>
      <Input id="b-city" type="text" bind:value={business.city} placeholder="İstanbul" />
    </div>
    <div>
      <label for="b-country" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.country')}</label>
      <Input id="b-country" type="text" bind:value={business.country} placeholder="Türkiye" />
    </div>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
    <div>
      <label for="b-postal" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.postalCode')}</label>
      <Input id="b-postal" type="text" bind:value={business.postal_code} placeholder="34430" />
    </div>
    <div>
      <label for="b-tz" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.timezone')}</label>
      <Input id="b-tz" type="text" bind:value={business.timezone} placeholder="Europe/Istanbul" />
    </div>
  </div>

  <div class="pt-4 border-t border-border">
    <h3 class="text-sm font-semibold text-foreground">{t('site.biz.social')}</h3>
    <p class="text-xs text-muted-foreground mt-0.5">{t('site.biz.socialHelp')}</p>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
      {#each socialProfiles as profile (profile.platform)}
        <div>
          <label for={`soc-${profile.platform}`} class="block text-xs font-medium text-foreground mb-1">
            {SOCIAL_PLATFORM_LABELS[profile.platform] ?? profile.platform}
          </label>
          <Input
            id={`soc-${profile.platform}`}
            type="url"
            bind:value={profile.url}
            placeholder={socialPlatformPlaceholder(profile.platform)}
          />
        </div>
      {/each}
    </div>
  </div>

  <div class="pt-3 border-t border-border flex justify-end">
    <Button type="submit" disabled={saving}>{saving ? t('common.saving') : t('site.biz.save')}</Button>
  </div>
</form>
