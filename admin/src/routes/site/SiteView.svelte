<script lang="ts">
  import { onMount } from 'svelte';
  import { t, i18n, isLocale } from '$lib/i18n/index.svelte';
  import { api } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import { brandingStore } from '$lib/stores/branding.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
  import GeneralTab from './tabs/GeneralTab.svelte';
  import ThemesTab from './tabs/ThemesTab.svelte';
  import BusinessSocialTab from './tabs/BusinessSocialTab.svelte';
  import RedirectsTab, { type RedirectRuleRow } from './tabs/RedirectsTab.svelte';
  import NavigationTab from './tabs/NavigationTab.svelte';
  import TranslationsTab from './tabs/TranslationsTab.svelte';
  import FormsTab from './tabs/FormsTab.svelte';
  import InboxTab from './tabs/InboxTab.svelte';
  import {
    mergeSocialPlatforms,
    normalizeSocialUrl,
    type BusinessProfileForm,
    type SocialProfile,
  } from './siteHelpers';

  interface SettingItem {
    key: string;
    value: string;
  }

  interface NotFoundEntry {
    path: string;
    hits: number;
    last_seen: string;
  }

  type TabKey = 'general' | 'themes' | 'navigation' | 'forms' | 'inbox' | 'translations' | 'maintenance' | 'notice' | 'business' | 'redirects' | 'notfound';
  let activeTab = $state<TabKey>('general');
  let loading = $state(false);
  let saving = $state(false);
  let saveMsg = $state<string | null>(null);
  let saveErr = $state<string | null>(null);

  let settings = $state<Record<string, string>>({
    'site.name': '',
    'site.title': '',
    'site.description': '',
    'site.home_path': '/',
    'appearance.logo_url': '',
    'appearance.favicon_url': '',
    'appearance.social_image_url': '',
    'storefront.announcement_bar': '',
    'storefront.footer_tagline': '',
    'storefront.whatsapp_number': '',
    'storefront.hero_eyebrow': '',
    'storefront.hero_title': '',
    'storefront.hero_subtitle': '',
    'storefront.hero_image_url': '',
    'storefront.hero_cta_label': '',
    'storefront.hero_cta_href': '',
    'storefront.popup_enabled': 'true',
    'storefront.popup_code': '',
    'storefront.popup_percent': '',
    'storefront.theme_id': 'minimal',
    'storefront.theme_tokens': '',
    'admin.default_locale': '',
    'admin.brand_name': '',
    'admin.brand_logo_url': '',
    'admin.brand_color': '',
  });

  let maintenance = $state({
    enabled: false,
    headline: '',
    message: '',
    allowlistStr: '',
    starts_at: '',
    ends_at: '',
  });

  let staffNotice = $state({
    active: false,
    message: '',
    severity: 'info',
    expires_at: '',
  });

  let business = $state<BusinessProfileForm>({
    name: '',
    legal_name: '',
    tax_id: '',
    tax_office: '',
    mersis_no: '',
    kep_address: '',
    email: '',
    phone: '',
    website: '',
    address: '',
    city: '',
    country: '',
    postal_code: '',
    timezone: 'Europe/Istanbul',
  });

  let socialPlatforms = $state<string[] | null>(null);
  let socialProfiles = $state<SocialProfile[]>(mergeSocialPlatforms());

  let redirects = $state<RedirectRuleRow[]>([]);
  let notFounds = $state<NotFoundEntry[]>([]);
  let newPattern = $state('');
  let newTarget = $state('');
  let newKind = $state('exact');
  let newStatus = $state(301);
  let deleteRedirectTarget = $state<RedirectRuleRow | null>(null);
  let convertNotFoundPath = $state<string | null>(null);
  let convertTarget = $state('');

  function showNotification(msg: string, isError = false) {
    if (isError) {
      saveErr = msg;
      saveMsg = null;
    } else {
      saveMsg = msg;
      saveErr = null;
    }
    setTimeout(() => {
      saveMsg = null;
      saveErr = null;
    }, 3500);
  }

  async function loadGeneralSettings() {
    try {
      const res = await api.get<ApiResponse<SettingItem[]>>('/api/v1/admin/site/settings');
      if (res?.data && Array.isArray(res.data)) {
        for (const item of res.data) {
          if (item.key === 'site.home_path' && !item.value) continue;
          if (item.key === 'storefront.popup_enabled' && !item.value) continue;
          settings[item.key] = item.value;
        }
      }
    } catch (e) {
      console.warn('Load settings failed:', e);
    }
  }

  async function loadMaintenance() {
    try {
      const res = await api.get<ApiResponse<any>>('/api/v1/admin/site/maintenance');
      if (res?.data) {
        maintenance.enabled = Boolean(res.data.enabled);
        maintenance.headline = res.data.headline || '';
        maintenance.message = res.data.message || '';
        maintenance.starts_at = res.data.starts_at || '';
        maintenance.ends_at = res.data.ends_at || '';
        maintenance.allowlistStr = Array.isArray(res.data.allowlist) ? res.data.allowlist.join(', ') : '';
      }
    } catch (e) {
      console.warn('Load maintenance failed:', e);
    }
  }

  async function loadStaffNotice() {
    try {
      const res = await api.get<ApiResponse<any>>('/api/v1/admin/site/staff-notice');
      if (res?.data) {
        staffNotice.active = Boolean(res.data.active);
        staffNotice.message = res.data.message || '';
        staffNotice.severity = res.data.severity || 'info';
        staffNotice.expires_at = res.data.expires_at || '';
      }
    } catch (e) {
      console.warn('Load staff notice failed:', e);
    }
  }

  async function loadBusinessAndSocial() {
    try {
      const [bRes, sRes] = await Promise.allSettled([
        api.get<ApiResponse<any>>('/api/v1/admin/site/business-profile'),
        api.get<ApiResponse<any>>('/api/v1/admin/site/social-links'),
      ]);
      if (bRes.status === 'fulfilled' && bRes.value?.data) {
        business = { ...business, ...bRes.value.data, timezone: bRes.value.data.timezone || business.timezone || 'Europe/Istanbul' };
      }
      if (sRes.status === 'fulfilled' && sRes.value?.data) {
        if (Array.isArray(sRes.value.data.platforms)) {
          socialPlatforms = sRes.value.data.platforms;
        }
        socialProfiles = mergeSocialPlatforms(socialPlatforms, sRes.value.data.profiles);
      }
    } catch (e) {
      console.warn('Load business/social failed:', e);
    }
  }

  async function loadRedirects() {
    try {
      const res = await api.get<ApiResponse<RedirectRuleRow[]>>('/api/v1/admin/site/redirects');
      if (res?.data && Array.isArray(res.data)) {
        redirects = res.data;
      }
    } catch (e) {
      console.warn('Load redirects failed:', e);
    }
  }

  async function loadNotFounds() {
    try {
      const res = await api.get<ApiResponse<NotFoundEntry[]>>('/api/v1/admin/site/not-found');
      if (res?.data && Array.isArray(res.data)) {
        notFounds = res.data;
      }
    } catch (e) {
      console.warn('Load not-found failed:', e);
    }
  }

  async function handleSaveGeneral(e: SubmitEvent) {
    e.preventDefault();
    saving = true;
    try {
      const payload = {
        settings: Object.entries(settings).map(([key, value]) => ({ key, value })),
      };
      await api.put('/api/v1/admin/site/settings', payload);
      await brandingStore.load();
      const nextDefault = settings['admin.default_locale'];
      const resolvedDefault = isLocale(nextDefault) ? nextDefault : null;
      if (resolvedDefault) {
        i18n.setProjectDefault(resolvedDefault);
      }
      await i18n.setUserLocale(resolvedDefault);
      showNotification(t('site.general.saved'));
    } catch (err: any) {
      showNotification(err.message || t('site.general.saveFailed'), true);
    } finally {
      saving = false;
    }
  }

  async function handleSaveMaintenance(e: SubmitEvent) {
    e.preventDefault();
    saving = true;
    try {
      const allowlist = maintenance.allowlistStr
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      await api.put('/api/v1/admin/site/maintenance', {
        enabled: maintenance.enabled,
        headline: maintenance.headline,
        message: maintenance.message,
        allowlist,
        starts_at: maintenance.starts_at.trim() || undefined,
        ends_at: maintenance.ends_at.trim() || undefined,
      });
      showNotification(t('site.notify.maint'));
    } catch (err: any) {
      showNotification(err.message || t('site.err.maint'), true);
    } finally {
      saving = false;
    }
  }

  async function handleSaveStaffNotice(e: SubmitEvent) {
    e.preventDefault();
    saving = true;
    try {
      const res = await api.put<ApiResponse<any>>('/api/v1/admin/site/staff-notice', {
        active: staffNotice.active,
        message: staffNotice.message.trim(),
        severity: staffNotice.severity,
        expires_at: staffNotice.expires_at.trim() || undefined,
      });
      if (res?.data?.expires_at) {
        staffNotice.expires_at = res.data.expires_at;
      }
      showNotification(t('site.notify.notice'));
    } catch (err: any) {
      showNotification(err.message || t('site.err.notice'), true);
    } finally {
      saving = false;
    }
  }

  async function handleSaveBusinessAndSocial(e: SubmitEvent) {
    e.preventDefault();
    const hasName = business.name.trim() !== '';
    const hasAnyOtherBizField = [
      business.legal_name,
      business.tax_id,
      business.tax_office,
      business.mersis_no,
      business.kep_address,
      business.email,
      business.phone,
      business.website,
      business.address,
      business.city,
      business.country,
      business.postal_code,
    ].some((v) => v.trim() !== '');

    if (!hasName && hasAnyOtherBizField) {
      showNotification(t('site.biz.nameRequired'), true);
      return;
    }

    saving = true;
    try {
      const normalizedSocial = socialProfiles.map((p) => ({
        platform: p.platform,
        url: normalizeSocialUrl(p.url),
      }));
      const calls: Promise<unknown>[] = [
        api.put('/api/v1/admin/site/social-links', { profiles: normalizedSocial }),
      ];
      if (hasName) {
        calls.push(
          api.put('/api/v1/admin/site/business-profile', {
            ...business,
            timezone: business.timezone.trim() || 'Europe/Istanbul',
          }),
        );
      }
      await Promise.all(calls);
      socialProfiles = mergeSocialPlatforms(socialPlatforms, normalizedSocial);
      showNotification(t('site.notify.biz'));
    } catch (err: any) {
      showNotification(err.message || t('site.err.biz'), true);
    } finally {
      saving = false;
    }
  }

  async function handleCreateRedirect(e: SubmitEvent) {
    e.preventDefault();
    if (!newPattern.trim() || !newTarget.trim()) return;
    saving = true;
    try {
      await api.post('/api/v1/admin/site/redirects', {
        pattern: newPattern.trim(),
        kind: newKind || 'exact',
        target_path: newTarget.trim(),
        status: Number(newStatus) || 301,
      });
      newPattern = '';
      newTarget = '';
      showNotification(t('site.notify.redirCreated'));
      await loadRedirects();
    } catch (err: any) {
      showNotification(err.message || t('site.err.redirCreate'), true);
    } finally {
      saving = false;
    }
  }

  async function handleDeleteRedirect() {
    if (!deleteRedirectTarget) return;
    saving = true;
    try {
      await api.delete(`/api/v1/admin/site/redirects/${deleteRedirectTarget.id}`);
      deleteRedirectTarget = null;
      showNotification(t('site.notify.redirDeleted'));
      await loadRedirects();
    } catch (err: any) {
      showNotification(err.message || t('site.err.redirDelete'), true);
    } finally {
      saving = false;
    }
  }

  async function handleConvertNotFound(e: SubmitEvent) {
    e.preventDefault();
    if (!convertNotFoundPath || !convertTarget.trim()) return;
    saving = true;
    try {
      await api.post('/api/v1/admin/site/not-found/convert', {
        path: convertNotFoundPath,
        target: convertTarget.trim(),
      });
      convertNotFoundPath = null;
      convertTarget = '';
      showNotification(t('site.notify.converted'));
      await Promise.all([loadNotFounds(), loadRedirects()]);
    } catch (err: any) {
      showNotification(err.message || t('site.err.convert'), true);
    } finally {
      saving = false;
    }
  }

  onMount(async () => {
    loading = true;
    try {
      await Promise.allSettled([
        loadGeneralSettings(),
        loadMaintenance(),
        loadStaffNotice(),
        loadBusinessAndSocial(),
        loadRedirects(),
        loadNotFounds(),
      ]);
    } finally {
      loading = false;
    }
  });
</script>

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-4 w-full">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-border pb-4">
      <div class="flex items-center gap-3">
        <div>
          <h2 class="text-xl font-semibold tracking-tight text-foreground">{t('site.title')}</h2>
          <p class="text-xs text-muted-foreground mt-0.5">{t('site.subtitle')}</p>
        </div>
        {#if loading}
          <svg class="size-4 animate-spin text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12a9 9 0 1 1-6.219-8.56"></path>
          </svg>
        {/if}
      </div>

      {#if saveMsg}
        <div class="p-2.5 px-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs flex items-center gap-1.5 animate-in fade-in-0">
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"></polyline></svg>
          <span>{saveMsg}</span>
        </div>
      {/if}
      {#if saveErr}
        <div class="p-2.5 px-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-center gap-1.5 animate-in fade-in-0">
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"></line></svg>
          <span>{saveErr}</span>
        </div>
      {/if}
    </div>

    <div class="flex items-center gap-1 border-b border-border overflow-x-auto pb-px">
      <button type="button" onclick={() => (activeTab = 'general')} class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'general' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}">{t('site.tab.general')}</button>
      <button type="button" onclick={() => (activeTab = 'themes')} class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'themes' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}">{t('site.tab.themes')}</button>
      <button type="button" onclick={() => (activeTab = 'navigation')} class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'navigation' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}">{t('site.tab.navigation')}</button>
      {#if schema.modules['forms']}
        <button type="button" onclick={() => (activeTab = 'forms')} class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'forms' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}">{t('site.tab.forms')}</button>
        <button type="button" onclick={() => (activeTab = 'inbox')} class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'inbox' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}">{t('site.tab.inbox')}</button>
      {/if}
      <button type="button" onclick={() => (activeTab = 'translations')} class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'translations' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}">{t('site.tab.translations')}</button>
      <button type="button" onclick={() => (activeTab = 'maintenance')} class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'maintenance' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}">{t('site.tab.maintenance')} {maintenance.enabled ? '●' : ''}</button>
      <button type="button" onclick={() => (activeTab = 'notice')} class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'notice' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}">{t('site.tab.notice')} {staffNotice.active ? '●' : ''}</button>
      <button type="button" onclick={() => (activeTab = 'business')} class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'business' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}">{t('site.tab.business')}</button>
      <button type="button" onclick={() => (activeTab = 'redirects')} class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'redirects' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}">{t('site.tab.redirectsCount', { count: redirects.length })}</button>
      <button type="button" onclick={() => (activeTab = 'notfound')} class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'notfound' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}">{t('site.tab.notfoundCount', { count: notFounds.length })}</button>
    </div>

    {#if activeTab === 'general'}
      <GeneralTab bind:settings {saving} onsubmit={handleSaveGeneral} />
    {:else if activeTab === 'themes'}
      <ThemesTab bind:settings {saving} onsubmit={handleSaveGeneral} />
    {:else if activeTab === 'maintenance'}
      <form onsubmit={handleSaveMaintenance} class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-5">
        <div class="flex items-center justify-between p-4 rounded-lg border border-border bg-muted/20">
          <div>
            <span class="text-sm font-semibold text-foreground block">{t('site.maint.title')}</span>
            <span class="text-xs text-muted-foreground block mt-0.5">{t('site.maint.help')}</span>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" bind:checked={maintenance.enabled} class="sr-only peer" />
            <div class="w-11 h-6 bg-muted peer-focus:outline-hidden rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-transform peer-checked:bg-primary"></div>
          </label>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label for="m-headline" class="block text-xs font-medium text-foreground mb-1">{t('site.maint.headline')}</label>
            <Input id="m-headline" type="text" bind:value={maintenance.headline} placeholder={t('site.maint.headlinePlaceholder')} />
          </div>
          <div>
            <label for="m-allowlist" class="block text-xs font-medium text-foreground mb-1">{t('site.maint.allowlist')}</label>
            <Input id="m-allowlist" type="text" bind:value={maintenance.allowlistStr} placeholder="127.0.0.1, 192.168.1.100" />
          </div>
        </div>

        <div>
          <label for="m-msg" class="block text-xs font-medium text-foreground mb-1">{t('site.maint.message')}</label>
          <Input id="m-msg" type="text" bind:value={maintenance.message} placeholder={t('site.maint.messagePlaceholder')} />
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label for="m-starts" class="block text-xs font-medium text-foreground mb-1">{t('site.maint.startsAt')}</label>
            <Input id="m-starts" type="text" bind:value={maintenance.starts_at} placeholder="2026-11-15T00:00:00Z" />
          </div>
          <div>
            <label for="m-ends" class="block text-xs font-medium text-foreground mb-1">{t('site.maint.endsAt')}</label>
            <Input id="m-ends" type="text" bind:value={maintenance.ends_at} placeholder="2026-11-15T02:00:00Z" />
          </div>
        </div>

        <div class="pt-3 border-t border-border flex justify-end">
          <Button type="submit" disabled={saving}>{saving ? t('common.saving') : t('site.maint.update')}</Button>
        </div>
      </form>
    {:else if activeTab === 'notice'}
      <form onsubmit={handleSaveStaffNotice} class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-5">
        <div class="flex items-center justify-between p-4 rounded-lg border border-border bg-muted/20">
          <div>
            <span class="text-sm font-semibold text-foreground block">{t('site.notice.title')}</span>
            <span class="text-xs text-muted-foreground block mt-0.5">{t('site.notice.help')}</span>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" bind:checked={staffNotice.active} class="sr-only peer" />
            <div class="w-11 h-6 bg-muted peer-focus:outline-hidden rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-transform peer-checked:bg-primary"></div>
          </label>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-4 gap-4">
          <div class="sm:col-span-2">
            <label for="sn-msg" class="block text-xs font-medium text-foreground mb-1">{t('site.notice.message')}</label>
            <Input id="sn-msg" type="text" bind:value={staffNotice.message} placeholder={t('site.notice.messagePlaceholder')} />
          </div>
          <div>
            <label for="sn-sev" class="block text-xs font-medium text-foreground mb-1">{t('site.notice.severity')}</label>
            <select id="sn-sev" bind:value={staffNotice.severity} class="w-full h-9 rounded-md border border-input bg-background px-3 text-xs text-foreground focus:outline-hidden focus:ring-1 focus:ring-primary">
              <option value="info">{t('site.notice.info')}</option>
              <option value="warning">{t('site.notice.warning')}</option>
              <option value="critical">{t('site.notice.critical')}</option>
            </select>
          </div>
          <div>
            <label for="sn-exp" class="block text-xs font-medium text-foreground mb-1">{t('site.notice.expiresAt')}</label>
            <Input id="sn-exp" type="text" bind:value={staffNotice.expires_at} placeholder="2026-12-31T23:59:59Z" />
          </div>
        </div>

        {#if staffNotice.active && staffNotice.message}
          <div class="p-3 rounded-lg border flex items-center gap-2 text-xs {staffNotice.severity === 'critical' ? 'bg-destructive/10 border-destructive/20 text-destructive' : staffNotice.severity === 'warning' ? 'bg-amber-500/10 border-amber-500/20 text-amber-600 dark:text-amber-400' : 'bg-primary/10 border-primary/20 text-primary'}">
            <span class="font-bold uppercase tracking-wider text-[10px]">{t('site.notice.preview')}</span>
            <span>{staffNotice.message}</span>
          </div>
        {/if}

        <div class="pt-3 border-t border-border flex justify-end">
          <Button type="submit" disabled={saving}>{saving ? t('common.saving') : t('site.notice.broadcast')}</Button>
        </div>
      </form>
    {:else if activeTab === 'business'}
      <BusinessSocialTab bind:business bind:socialProfiles {saving} onsubmit={handleSaveBusinessAndSocial} />
    {:else if activeTab === 'redirects'}
      <RedirectsTab {redirects} bind:newPattern bind:newTarget bind:newKind bind:newStatus {saving} oncreate={handleCreateRedirect} ondelete={(r) => (deleteRedirectTarget = r)} />
    {:else if activeTab === 'notfound'}
      <div class="rounded-xl border border-border bg-card overflow-hidden shadow-xs">
        <div class="p-4 border-b border-border flex items-center justify-between">
          <div>
            <h3 class="text-sm font-semibold text-foreground">{t('site.nf.title')}</h3>
            <p class="text-xs text-muted-foreground mt-0.5">{t('site.nf.help')}</p>
          </div>
          <Button variant="outline" onclick={loadNotFounds}>{t('system.refresh')}</Button>
        </div>

        {#if convertNotFoundPath}
          <form onsubmit={handleConvertNotFound} class="p-4 bg-muted/40 border-b border-border flex flex-col sm:flex-row items-center gap-3">
            <div class="text-xs font-mono text-foreground">{t('site.nf.redirect')} <span class="text-destructive font-semibold">{convertNotFoundPath}</span> {t('site.nf.to')}</div>
            <Input type="text" bind:value={convertTarget} placeholder={t('site.nf.destination')} class="flex-1" required />
            <div class="flex items-center gap-2">
              <Button type="submit" disabled={saving}>{t('site.nf.save')}</Button>
              <Button type="button" variant="outline" onclick={() => (convertNotFoundPath = null)}>{t('common.cancel')}</Button>
            </div>
          </form>
        {/if}

        <div class="divide-y divide-border">
          {#if notFounds.length === 0}
            <div class="p-8 text-center text-xs text-muted-foreground">{t('site.nf.empty')}</div>
          {:else}
            {#each notFounds as item (item.path)}
              <div class="p-4 flex items-center justify-between gap-4 hover:bg-muted/30 transition-colors">
                <div class="grid text-left min-w-0">
                  <span class="font-mono text-xs font-semibold text-foreground">{item.path}</span>
                  <span class="text-[11px] text-muted-foreground mt-0.5">{t(item.hits === 1 ? 'site.nf.hit' : 'site.nf.hits', { count: item.hits })} • {t('site.nf.lastSeen', { when: item.last_seen ? new Date(item.last_seen).toLocaleString() : t('site.nf.recent') })}</span>
                </div>
                <Button variant="outline" onclick={() => { convertNotFoundPath = item.path; convertTarget = ''; }}>{t('site.nf.create')}</Button>
              </div>
            {/each}
          {/if}
        </div>
      </div>
    {:else if activeTab === 'navigation'}
      <NavigationTab />
    {:else if activeTab === 'forms' && schema.modules['forms']}
      <FormsTab />
    {:else if activeTab === 'inbox' && schema.modules['forms']}
      <InboxTab />
    {:else if activeTab === 'translations'}
      <TranslationsTab />
    {/if}
  </div>
</div>

<ConfirmDialog
  open={!!deleteRedirectTarget}
  title={t('site.redir.deleteTitle')}
  description={t('site.redir.deleteDescription', { name: deleteRedirectTarget?.pattern || deleteRedirectTarget?.source || '' })}
  confirmLabel={t('common.delete')}
  loading={saving}
  onconfirm={handleDeleteRedirect}
  oncancel={() => (deleteRedirectTarget = null)}
/>
