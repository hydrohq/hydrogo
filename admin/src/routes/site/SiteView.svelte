<script lang="ts">
  import { onMount } from 'svelte';
  import { t, i18n, isLocale } from '$lib/i18n/index.svelte';
  import { api } from '$lib/api/client';
  import { brandingStore } from '$lib/stores/branding.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
  import NavigationTab from './tabs/NavigationTab.svelte';
  import TranslationsTab from './tabs/TranslationsTab.svelte';
  import FormsTab from './tabs/FormsTab.svelte';
  import InboxTab from './tabs/InboxTab.svelte';

  interface SettingItem {
    key: string;
    value: string;
  }

  interface RedirectRule {
    id: string;
    pattern?: string;
    source?: string;
    target_path?: string;
    target?: string;
    status?: number;
    code?: number;
  }

  interface NotFoundEntry {
    path: string;
    hits: number;
    last_seen: string;
  }

  interface SocialProfile {
    platform: string;
    url: string;
  }

  type TabKey = 'general' | 'navigation' | 'forms' | 'inbox' | 'translations' | 'maintenance' | 'notice' | 'business' | 'redirects' | 'notfound';
  let activeTab = $state<TabKey>('general');
  let loading = $state(false);
  let saving = $state(false);
  let saveMsg = $state<string | null>(null);
  let saveErr = $state<string | null>(null);

  // General settings
  let settings = $state<Record<string, string>>({
    'site.name': 'HydroGo',
    'site.title': 'HydroGo Headless CMS',
    'site.description': '',
    'site.home_path': '/',
    'appearance.favicon_url': '',
    'appearance.social_image_url': '',
    'admin.default_locale': '',
    'admin.brand_name': '',
    'admin.brand_logo_url': '',
    'admin.brand_color': '',
  });

  // Maintenance settings
  let maintenance = $state({
    enabled: false,
    headline: '',
    message: '',
    allowlistStr: '',
    starts_at: '',
    ends_at: '',
  });

  // Staff notice
  let staffNotice = $state({
    enabled: false,
    message: '',
    severity: 'info',
  });

  // Business profile
  let business = $state({
    company_name: '',
    legal_name: '',
    tax_id: '',
    email: '',
    phone: '',
    address: '',
    city: '',
    country: '',
    postal_code: '',
  });

  // Social profiles
  let socialProfiles = $state<SocialProfile[]>([
    { platform: 'twitter', url: '' },
    { platform: 'github', url: '' },
    { platform: 'linkedin', url: '' },
    { platform: 'youtube', url: '' },
  ]);

  // Redirects
  let redirects = $state<RedirectRule[]>([]);
  let notFounds = $state<NotFoundEntry[]>([]);
  let newPattern = $state('');
  let newTarget = $state('');
  let newStatus = $state(301);
  let deleteRedirectTarget = $state<RedirectRule | null>(null);
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
          // An unset home path keeps the "/" default instead of blanking the required field.
          if (item.key === 'site.home_path' && !item.value) continue;
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
        staffNotice.enabled = Boolean(res.data.enabled);
        staffNotice.message = res.data.message || '';
        staffNotice.severity = res.data.severity || 'info';
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
        business = { ...business, ...bRes.value.data };
      }
      if (sRes.status === 'fulfilled' && sRes.value?.data?.profiles) {
        socialProfiles = sRes.value.data.profiles;
      }
    } catch (e) {
      console.warn('Load business/social failed:', e);
    }
  }

  async function loadRedirects() {
    try {
      const res = await api.get<ApiResponse<RedirectRule[]>>('/api/v1/admin/site/redirects');
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
      if (isLocale(settings['admin.default_locale'])) i18n.setProjectDefault(settings['admin.default_locale']);
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
        starts_at: maintenance.starts_at || undefined,
        ends_at: maintenance.ends_at || undefined,
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
      await api.put('/api/v1/admin/site/staff-notice', {
        enabled: staffNotice.enabled,
        message: staffNotice.message,
        severity: staffNotice.severity,
      });
      showNotification(t('site.notify.notice'));
    } catch (err: any) {
      showNotification(err.message || t('site.err.notice'), true);
    } finally {
      saving = false;
    }
  }

  async function handleSaveBusinessAndSocial(e: SubmitEvent) {
    e.preventDefault();
    saving = true;
    try {
      await Promise.all([
        api.put('/api/v1/admin/site/business-profile', business),
        api.put('/api/v1/admin/site/social-links', { profiles: socialProfiles }),
      ]);
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
      showNotification('404 converted to 301 redirect!');
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
  <div class="@container/main flex flex-1 flex-col gap-4 p-4 lg:p-6 max-w-5xl mx-auto w-full">
    <!-- Header -->
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

    <!-- Tabs -->
    <div class="flex items-center gap-1 border-b border-border overflow-x-auto pb-px">
      <button
        type="button"
        onclick={() => (activeTab = 'general')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'general' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        {t('site.tab.general')}
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'navigation')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'navigation' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        {t('site.tab.navigation')}
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'forms')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'forms' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        {t('site.tab.forms')}
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'inbox')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'inbox' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        {t('site.tab.inbox')}
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'translations')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'translations' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        {t('site.tab.translations')}
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'maintenance')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'maintenance' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        {t('site.tab.maintenance')} {maintenance.enabled ? '●' : ''}
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'notice')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'notice' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        {t('site.tab.notice')} {staffNotice.enabled ? '●' : ''}
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'business')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'business' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        {t('site.tab.business')}
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'redirects')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'redirects' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        {t('site.tab.redirectsCount', { count: redirects.length })}
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'notfound')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'notfound' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        404 Tracker ({notFounds.length})
      </button>
    </div>

    <!-- Content: Tab 1 - General -->
    {#if activeTab === 'general'}
      <form onsubmit={handleSaveGeneral} class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-5">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label for="site-name" class="block text-xs font-medium text-foreground mb-1">{t('site.general.name')}</label>
            <Input id="site-name" type="text" bind:value={settings['site.name']} placeholder="e.g. HydroGo" required />
          </div>
          <div>
            <label for="site-title" class="block text-xs font-medium text-foreground mb-1">{t('site.general.title')}</label>
            <Input id="site-title" type="text" bind:value={settings['site.title']} placeholder="e.g. Modern Headless CMS" required />
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
            <label for="favicon-url" class="block text-xs font-medium text-foreground mb-1">{t('site.general.favicon')}</label>
            <Input id="favicon-url" type="text" bind:value={settings['appearance.favicon_url']} placeholder="/favicon.ico or https://..." />
          </div>
        </div>

        <div>
          <label for="social-img" class="block text-xs font-medium text-foreground mb-1">{t('site.general.socialImage')}</label>
          <Input id="social-img" type="text" bind:value={settings['appearance.social_image_url']} placeholder="https://..." />
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

        <div class="space-y-3 rounded-lg border border-border p-4">
          <p class="text-xs font-semibold text-foreground">{t('brand.title')}</p>
          <p class="text-[11px] text-muted-foreground">{t('brand.help')}</p>
          <div>
            <label for="brand-name" class="block text-xs font-medium text-foreground mb-1">{t('brand.name')}</label>
            <Input id="brand-name" type="text" bind:value={settings['admin.brand_name']} placeholder="HydroGo" maxlength={60} />
          </div>
          <div>
            <label for="brand-logo" class="block text-xs font-medium text-foreground mb-1">{t('brand.logo')}</label>
            <Input id="brand-logo" type="text" bind:value={settings['admin.brand_logo_url']} placeholder="https://..." />
          </div>
          <div class="flex items-center gap-3">
            <label for="brand-color" class="text-xs font-medium text-foreground">{t('brand.color')}</label>
            <input id="brand-color" type="color" value={settings['admin.brand_color'] || '#18181b'} oninput={(e) => (settings['admin.brand_color'] = e.currentTarget.value)} class="h-8 w-12 cursor-pointer rounded border border-input bg-background" />
            <Button type="button" variant="outline" size="sm" onclick={() => (settings['admin.brand_color'] = '')}>{t('brand.reset')}</Button>
          </div>
        </div>

        <div class="pt-3 border-t border-border flex justify-end">
          <Button type="submit" disabled={saving}>{saving ? 'Saving...' : 'Save Settings'}</Button>
        </div>
      </form>

    <!-- Content: Tab 2 - Maintenance Mode -->
    {:else if activeTab === 'maintenance'}
      <form onsubmit={handleSaveMaintenance} class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-5">
        <div class="flex items-center justify-between p-4 rounded-lg border border-border bg-muted/20">
          <div>
            <span class="text-sm font-semibold text-foreground block">{t('site.maint.title')}</span>
            <span class="text-xs text-muted-foreground block mt-0.5">{t('site.maint.help')}</span>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" bind:checked={maintenance.enabled} class="sr-only peer" />
            <div class="w-11 h-6 bg-muted peer-focus:outline-hidden rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
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

        <div class="pt-3 border-t border-border flex justify-end">
          <Button type="submit" disabled={saving}>{saving ? t('common.saving') : t('site.maint.update')}</Button>
        </div>
      </form>

    <!-- Content: Tab 3 - Staff Notice -->
    {:else if activeTab === 'notice'}
      <form onsubmit={handleSaveStaffNotice} class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-5">
        <div class="flex items-center justify-between p-4 rounded-lg border border-border bg-muted/20">
          <div>
            <span class="text-sm font-semibold text-foreground block">{t('site.notice.title')}</span>
            <span class="text-xs text-muted-foreground block mt-0.5">{t('site.notice.help')}</span>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" bind:checked={staffNotice.enabled} class="sr-only peer" />
            <div class="w-11 h-6 bg-muted peer-focus:outline-hidden rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
          </label>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div class="sm:col-span-2">
            <label for="sn-msg" class="block text-xs font-medium text-foreground mb-1">{t('site.notice.message')}</label>
            <Input id="sn-msg" type="text" bind:value={staffNotice.message} placeholder={t('site.notice.messagePlaceholder')} />
          </div>
          <div>
            <label for="sn-sev" class="block text-xs font-medium text-foreground mb-1">{t('site.notice.severity')}</label>
            <select
              id="sn-sev"
              bind:value={staffNotice.severity}
              class="w-full h-9 rounded-md border border-input bg-background px-3 text-xs text-foreground focus:outline-hidden focus:ring-1 focus:ring-primary"
            >
              <option value="info">{t('site.notice.info')}</option>
              <option value="warning">{t('site.notice.warning')}</option>
              <option value="critical">{t('site.notice.critical')}</option>
            </select>
          </div>
        </div>

        {#if staffNotice.enabled && staffNotice.message}
          <div class="p-3 rounded-lg border flex items-center gap-2 text-xs {staffNotice.severity === 'critical' ? 'bg-destructive/10 border-destructive/20 text-destructive' : staffNotice.severity === 'warning' ? 'bg-amber-500/10 border-amber-500/20 text-amber-600 dark:text-amber-400' : 'bg-primary/10 border-primary/20 text-primary'}">
            <span class="font-bold uppercase tracking-wider text-[10px]">{t('site.notice.preview')}</span>
            <span>{staffNotice.message}</span>
          </div>
        {/if}

        <div class="pt-3 border-t border-border flex justify-end">
          <Button type="submit" disabled={saving}>{saving ? t('common.saving') : t('site.notice.broadcast')}</Button>
        </div>
      </form>

    <!-- Content: Tab 4 - Business & Social -->
    {:else if activeTab === 'business'}
      <form onsubmit={handleSaveBusinessAndSocial} class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-6">
        <div>
          <h3 class="text-sm font-semibold text-foreground">{t('site.biz.title')}</h3>
          <p class="text-xs text-muted-foreground mt-0.5">{t('site.biz.help')}</p>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label for="b-name" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.company')}</label>
            <Input id="b-name" type="text" bind:value={business.company_name} placeholder="Acme Labs" />
          </div>
          <div>
            <label for="b-legal" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.legal')}</label>
            <Input id="b-legal" type="text" bind:value={business.legal_name} placeholder="Acme Labs LLC" />
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <label for="b-tax" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.tax')}</label>
            <Input id="b-tax" type="text" bind:value={business.tax_id} placeholder="US-12345678" />
          </div>
          <div>
            <label for="b-email" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.email')}</label>
            <Input id="b-email" type="email" bind:value={business.email} placeholder="contact@example.com" />
          </div>
          <div>
            <label for="b-phone" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.phone')}</label>
            <Input id="b-phone" type="tel" bind:value={business.phone} placeholder="+1 555-0199" />
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div class="md:col-span-2">
            <label for="b-addr" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.address')}</label>
            <Input id="b-addr" type="text" bind:value={business.address} placeholder="123 Innovation Way" />
          </div>
          <div>
            <label for="b-city" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.city')}</label>
            <Input id="b-city" type="text" bind:value={business.city} placeholder="San Francisco" />
          </div>
          <div>
            <label for="b-country" class="block text-xs font-medium text-foreground mb-1">{t('site.biz.country')}</label>
            <Input id="b-country" type="text" bind:value={business.country} placeholder="United States" />
          </div>
        </div>

        <div class="pt-4 border-t border-border">
          <h3 class="text-sm font-semibold text-foreground">{t('site.biz.social')}</h3>
          <p class="text-xs text-muted-foreground mt-0.5">{t('site.biz.socialHelp')}</p>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
            {#each socialProfiles as profile}
              <div>
                <label for={`soc-${profile.platform}`} class="block text-xs font-medium text-foreground capitalize mb-1">{profile.platform}</label>
                <Input id={`soc-${profile.platform}`} type="url" bind:value={profile.url} placeholder={`https://${profile.platform}.com/...`} />
              </div>
            {/each}
          </div>
        </div>

        <div class="pt-3 border-t border-border flex justify-end">
          <Button type="submit" disabled={saving}>{saving ? t('common.saving') : t('site.biz.save')}</Button>
        </div>
      </form>

    <!-- Content: Tab 5 - Redirects -->
    {:else if activeTab === 'redirects'}
      <div class="space-y-6">
        <!-- Add Redirect Form -->
        <form onsubmit={handleCreateRedirect} class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-4">
          <h3 class="text-sm font-semibold text-foreground">{t('site.redir.add')}</h3>
          <div class="grid grid-cols-1 md:grid-cols-12 gap-3 items-end">
            <div class="md:col-span-5">
              <label for="red-from" class="block text-xs font-medium text-foreground mb-1">{t('site.redir.from')}</label>
              <Input id="red-from" type="text" bind:value={newPattern} placeholder="/old-blog-post" required />
            </div>
            <div class="md:col-span-5">
              <label for="red-to" class="block text-xs font-medium text-foreground mb-1">{t('site.redir.to')}</label>
              <Input id="red-to" type="text" bind:value={newTarget} placeholder="/news/new-blog-post" required />
            </div>
            <div class="md:col-span-2">
              <Button type="submit" class="w-full" disabled={saving}>{t('site.redir.addRule')}</Button>
            </div>
          </div>
        </form>

        <!-- Redirects List -->
        <div class="rounded-xl border border-border bg-card overflow-hidden shadow-xs">
          <div class="p-4 border-b border-border flex items-center justify-between">
            <h3 class="text-sm font-semibold text-foreground">{t('site.redir.active', { count: redirects.length })}</h3>
          </div>
          <div class="divide-y divide-border">
            {#if redirects.length === 0}
              <div class="p-8 text-center text-xs text-muted-foreground">{t('site.redir.empty')}</div>
            {:else}
              {#each redirects as r}
                <div class="p-4 flex items-center justify-between gap-4 hover:bg-muted/30 transition-colors">
                  <div class="grid text-left min-w-0">
                    <div class="flex items-center gap-2">
                      <span class="font-mono text-xs font-semibold text-foreground">{r.pattern || r.source}</span>
                      <span class="text-muted-foreground text-xs">→</span>
                      <span class="font-mono text-xs text-primary">{r.target_path || r.target}</span>
                      <Badge variant="outline" class="text-[10px]">{r.status || r.code || 301}</Badge>
                    </div>
                  </div>
                  <Button variant="danger" onclick={() => (deleteRedirectTarget = r)}>{t('common.delete')}</Button>
                </div>
              {/each}
            {/if}
          </div>
        </div>
      </div>

    <!-- Content: Tab 6 - 404 Tracker -->
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
            <div class="text-xs font-mono text-foreground">
              {t('site.nf.redirect')} <span class="text-destructive font-semibold">{convertNotFoundPath}</span> {t('site.nf.to')}
            </div>
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
            {#each notFounds as item}
              <div class="p-4 flex items-center justify-between gap-4 hover:bg-muted/30 transition-colors">
                <div class="grid text-left min-w-0">
                  <span class="font-mono text-xs font-semibold text-foreground">{item.path}</span>
                  <span class="text-[11px] text-muted-foreground mt-0.5">{t(item.hits === 1 ? 'site.nf.hit' : 'site.nf.hits', { count: item.hits })} • {t('site.nf.lastSeen', { when: item.last_seen ? new Date(item.last_seen).toLocaleString() : t('site.nf.recent') })}</span>
                </div>
                <Button variant="outline" onclick={() => { convertNotFoundPath = item.path; convertTarget = ''; }}>
                  {t('site.nf.create')}
                </Button>
              </div>
            {/each}
          {/if}
        </div>
      </div>
    {:else if activeTab === 'navigation'}
      <NavigationTab />
    {:else if activeTab === 'forms'}
      <FormsTab />
    {:else if activeTab === 'inbox'}
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
