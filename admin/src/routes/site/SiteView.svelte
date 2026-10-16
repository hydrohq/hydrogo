<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
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
      await api.post('/api/v1/admin/site/settings', payload, { method: 'PUT' });
      showNotification('General settings saved successfully.');
    } catch (err: any) {
      showNotification(err.message || 'Failed to save general settings', true);
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
      await api.post('/api/v1/admin/site/maintenance', {
        enabled: maintenance.enabled,
        headline: maintenance.headline,
        message: maintenance.message,
        allowlist,
        starts_at: maintenance.starts_at || undefined,
        ends_at: maintenance.ends_at || undefined,
      }, { method: 'PUT' });
      showNotification('Maintenance mode configuration saved.');
    } catch (err: any) {
      showNotification(err.message || 'Failed to save maintenance settings', true);
    } finally {
      saving = false;
    }
  }

  async function handleSaveStaffNotice(e: SubmitEvent) {
    e.preventDefault();
    saving = true;
    try {
      await api.post('/api/v1/admin/site/staff-notice', {
        enabled: staffNotice.enabled,
        message: staffNotice.message,
        severity: staffNotice.severity,
      }, { method: 'PUT' });
      showNotification('Staff broadcast notice updated.');
    } catch (err: any) {
      showNotification(err.message || 'Failed to save staff notice', true);
    } finally {
      saving = false;
    }
  }

  async function handleSaveBusinessAndSocial(e: SubmitEvent) {
    e.preventDefault();
    saving = true;
    try {
      await Promise.all([
        api.post('/api/v1/admin/site/business-profile', business, { method: 'PUT' }),
        api.post('/api/v1/admin/site/social-links', { profiles: socialProfiles }, { method: 'PUT' }),
      ]);
      showNotification('Business profile & social links saved.');
    } catch (err: any) {
      showNotification(err.message || 'Failed to save profile & links', true);
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
      showNotification('Redirect rule created.');
      await loadRedirects();
    } catch (err: any) {
      showNotification(err.message || 'Failed to create redirect rule', true);
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
      showNotification('Redirect rule deleted.');
      await loadRedirects();
    } catch (err: any) {
      showNotification(err.message || 'Failed to delete redirect rule', true);
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
      showNotification(err.message || 'Failed to convert 404', true);
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
          <h2 class="text-xl font-semibold tracking-tight text-foreground">Site & Engine Settings</h2>
          <p class="text-xs text-muted-foreground mt-0.5">Global CMS configuration, maintenance mode, staff notice, and URL redirects.</p>
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
        General & SEO
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'navigation')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'navigation' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Navigation
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'forms')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'forms' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Forms
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'inbox')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'inbox' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Inbox (CRM)
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'translations')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'translations' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Translations
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'maintenance')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'maintenance' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Maintenance {maintenance.enabled ? '●' : ''}
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'notice')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'notice' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Staff Notice {staffNotice.enabled ? '●' : ''}
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'business')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'business' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Business & Social
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'redirects')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'redirects' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Redirects ({redirects.length})
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
            <label for="site-name" class="block text-xs font-medium text-foreground mb-1">Site Name</label>
            <Input id="site-name" type="text" bind:value={settings['site.name']} placeholder="e.g. HydroGo" required />
          </div>
          <div>
            <label for="site-title" class="block text-xs font-medium text-foreground mb-1">Default Meta Title</label>
            <Input id="site-title" type="text" bind:value={settings['site.title']} placeholder="e.g. Modern Headless CMS" required />
          </div>
        </div>

        <div>
          <label for="site-desc" class="block text-xs font-medium text-foreground mb-1">Site Description (Meta Description)</label>
          <Input id="site-desc" type="text" bind:value={settings['site.description']} placeholder="Brief description for search engines..." />
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label for="home-path" class="block text-xs font-medium text-foreground mb-1">Home Route Path</label>
            <Input id="home-path" type="text" bind:value={settings['site.home_path']} placeholder="/" required />
          </div>
          <div>
            <label for="favicon-url" class="block text-xs font-medium text-foreground mb-1">Favicon URL</label>
            <Input id="favicon-url" type="text" bind:value={settings['appearance.favicon_url']} placeholder="/favicon.ico or https://..." />
          </div>
        </div>

        <div>
          <label for="social-img" class="block text-xs font-medium text-foreground mb-1">Open Graph / Social Sharing Image URL</label>
          <Input id="social-img" type="text" bind:value={settings['appearance.social_image_url']} placeholder="https://..." />
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
            <span class="text-sm font-semibold text-foreground block">Maintenance Mode</span>
            <span class="text-xs text-muted-foreground block mt-0.5">When active, public API calls return 503 Service Unavailable, except for allowlisted IPs.</span>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" bind:checked={maintenance.enabled} class="sr-only peer" />
            <div class="w-11 h-6 bg-muted peer-focus:outline-hidden rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
          </label>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label for="m-headline" class="block text-xs font-medium text-foreground mb-1">Headline</label>
            <Input id="m-headline" type="text" bind:value={maintenance.headline} placeholder="Scheduled System Upgrade" />
          </div>
          <div>
            <label for="m-allowlist" class="block text-xs font-medium text-foreground mb-1">Allowlisted IP Addresses (comma separated)</label>
            <Input id="m-allowlist" type="text" bind:value={maintenance.allowlistStr} placeholder="127.0.0.1, 192.168.1.100" />
          </div>
        </div>

        <div>
          <label for="m-msg" class="block text-xs font-medium text-foreground mb-1">Maintenance Message</label>
          <Input id="m-msg" type="text" bind:value={maintenance.message} placeholder="We are performing essential database optimizations. Service will resume shortly." />
        </div>

        <div class="pt-3 border-t border-border flex justify-end">
          <Button type="submit" disabled={saving}>{saving ? 'Saving...' : 'Update Maintenance Mode'}</Button>
        </div>
      </form>

    <!-- Content: Tab 3 - Staff Notice -->
    {:else if activeTab === 'notice'}
      <form onsubmit={handleSaveStaffNotice} class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-5">
        <div class="flex items-center justify-between p-4 rounded-lg border border-border bg-muted/20">
          <div>
            <span class="text-sm font-semibold text-foreground block">Broadcast Notice to Staff</span>
            <span class="text-xs text-muted-foreground block mt-0.5">Renders an informative alert banner at the top of the Admin Studio for all editors.</span>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" bind:checked={staffNotice.enabled} class="sr-only peer" />
            <div class="w-11 h-6 bg-muted peer-focus:outline-hidden rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
          </label>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div class="sm:col-span-2">
            <label for="sn-msg" class="block text-xs font-medium text-foreground mb-1">Notice Message</label>
            <Input id="sn-msg" type="text" bind:value={staffNotice.message} placeholder="Content freeze in effect until 16:00 UTC for media CDN re-indexing." />
          </div>
          <div>
            <label for="sn-sev" class="block text-xs font-medium text-foreground mb-1">Severity Level</label>
            <select
              id="sn-sev"
              bind:value={staffNotice.severity}
              class="w-full h-9 rounded-md border border-input bg-background px-3 text-xs text-foreground focus:outline-hidden focus:ring-1 focus:ring-primary"
            >
              <option value="info">Info (Blue)</option>
              <option value="warning">Warning (Amber)</option>
              <option value="critical">Critical (Red)</option>
            </select>
          </div>
        </div>

        {#if staffNotice.enabled && staffNotice.message}
          <div class="p-3 rounded-lg border flex items-center gap-2 text-xs {staffNotice.severity === 'critical' ? 'bg-destructive/10 border-destructive/20 text-destructive' : staffNotice.severity === 'warning' ? 'bg-amber-500/10 border-amber-500/20 text-amber-600 dark:text-amber-400' : 'bg-primary/10 border-primary/20 text-primary'}">
            <span class="font-bold uppercase tracking-wider text-[10px]">Preview:</span>
            <span>{staffNotice.message}</span>
          </div>
        {/if}

        <div class="pt-3 border-t border-border flex justify-end">
          <Button type="submit" disabled={saving}>{saving ? 'Saving...' : 'Broadcast Notice'}</Button>
        </div>
      </form>

    <!-- Content: Tab 4 - Business & Social -->
    {:else if activeTab === 'business'}
      <form onsubmit={handleSaveBusinessAndSocial} class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-6">
        <div>
          <h3 class="text-sm font-semibold text-foreground">Company & Legal Information</h3>
          <p class="text-xs text-muted-foreground mt-0.5">Used for schema.org rich snippets and public legal disclosures.</p>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label for="b-name" class="block text-xs font-medium text-foreground mb-1">Company / Brand Name</label>
            <Input id="b-name" type="text" bind:value={business.company_name} placeholder="Acme Labs" />
          </div>
          <div>
            <label for="b-legal" class="block text-xs font-medium text-foreground mb-1">Legal Entity Name</label>
            <Input id="b-legal" type="text" bind:value={business.legal_name} placeholder="Acme Labs LLC" />
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <label for="b-tax" class="block text-xs font-medium text-foreground mb-1">Tax ID / VAT Number</label>
            <Input id="b-tax" type="text" bind:value={business.tax_id} placeholder="US-12345678" />
          </div>
          <div>
            <label for="b-email" class="block text-xs font-medium text-foreground mb-1">Public Contact Email</label>
            <Input id="b-email" type="email" bind:value={business.email} placeholder="contact@example.com" />
          </div>
          <div>
            <label for="b-phone" class="block text-xs font-medium text-foreground mb-1">Phone Number</label>
            <Input id="b-phone" type="tel" bind:value={business.phone} placeholder="+1 555-0199" />
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div class="md:col-span-2">
            <label for="b-addr" class="block text-xs font-medium text-foreground mb-1">Street Address</label>
            <Input id="b-addr" type="text" bind:value={business.address} placeholder="123 Innovation Way" />
          </div>
          <div>
            <label for="b-city" class="block text-xs font-medium text-foreground mb-1">City</label>
            <Input id="b-city" type="text" bind:value={business.city} placeholder="San Francisco" />
          </div>
          <div>
            <label for="b-country" class="block text-xs font-medium text-foreground mb-1">Country</label>
            <Input id="b-country" type="text" bind:value={business.country} placeholder="United States" />
          </div>
        </div>

        <div class="pt-4 border-t border-border">
          <h3 class="text-sm font-semibold text-foreground">Social Network Profiles</h3>
          <p class="text-xs text-muted-foreground mt-0.5">Links to your verified organizational channels.</p>

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
          <Button type="submit" disabled={saving}>{saving ? 'Saving...' : 'Save Business & Social'}</Button>
        </div>
      </form>

    <!-- Content: Tab 5 - Redirects -->
    {:else if activeTab === 'redirects'}
      <div class="space-y-6">
        <!-- Add Redirect Form -->
        <form onsubmit={handleCreateRedirect} class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-4">
          <h3 class="text-sm font-semibold text-foreground">Add URL Redirect Rule</h3>
          <div class="grid grid-cols-1 md:grid-cols-12 gap-3 items-end">
            <div class="md:col-span-5">
              <label for="red-from" class="block text-xs font-medium text-foreground mb-1">Source Path (From)</label>
              <Input id="red-from" type="text" bind:value={newPattern} placeholder="/old-blog-post" required />
            </div>
            <div class="md:col-span-5">
              <label for="red-to" class="block text-xs font-medium text-foreground mb-1">Target Path or URL (To)</label>
              <Input id="red-to" type="text" bind:value={newTarget} placeholder="/news/new-blog-post" required />
            </div>
            <div class="md:col-span-2">
              <Button type="submit" class="w-full" disabled={saving}>Add Rule</Button>
            </div>
          </div>
        </form>

        <!-- Redirects List -->
        <div class="rounded-xl border border-border bg-card overflow-hidden shadow-xs">
          <div class="p-4 border-b border-border flex items-center justify-between">
            <h3 class="text-sm font-semibold text-foreground">Active Redirect Rules ({redirects.length})</h3>
          </div>
          <div class="divide-y divide-border">
            {#if redirects.length === 0}
              <div class="p-8 text-center text-xs text-muted-foreground">No custom redirect rules configured yet.</div>
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
                  <Button variant="danger" onclick={() => (deleteRedirectTarget = r)}>Delete</Button>
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
            <h3 class="text-sm font-semibold text-foreground">404 Not Found Tracking</h3>
            <p class="text-xs text-muted-foreground mt-0.5">Live log of URLs visitors tried to access that returned 404. Convert frequent 404s to 301 redirects instantly.</p>
          </div>
          <Button variant="outline" onclick={loadNotFounds}>Refresh</Button>
        </div>

        {#if convertNotFoundPath}
          <form onsubmit={handleConvertNotFound} class="p-4 bg-muted/40 border-b border-border flex flex-col sm:flex-row items-center gap-3">
            <div class="text-xs font-mono text-foreground">
              Redirect <span class="text-destructive font-semibold">{convertNotFoundPath}</span> to:
            </div>
            <Input type="text" bind:value={convertTarget} placeholder="/new-destination-path" class="flex-1" required />
            <div class="flex items-center gap-2">
              <Button type="submit" disabled={saving}>Save 301</Button>
              <Button type="button" variant="outline" onclick={() => (convertNotFoundPath = null)}>Cancel</Button>
            </div>
          </form>
        {/if}

        <div class="divide-y divide-border">
          {#if notFounds.length === 0}
            <div class="p-8 text-center text-xs text-muted-foreground">No 404 hit records recorded. All traffic is resolving smoothly!</div>
          {:else}
            {#each notFounds as item}
              <div class="p-4 flex items-center justify-between gap-4 hover:bg-muted/30 transition-colors">
                <div class="grid text-left min-w-0">
                  <span class="font-mono text-xs font-semibold text-foreground">{item.path}</span>
                  <span class="text-[11px] text-muted-foreground mt-0.5">{item.hits} hit{item.hits === 1 ? '' : 's'} • Last seen: {item.last_seen ? new Date(item.last_seen).toLocaleString() : 'Recent'}</span>
                </div>
                <Button variant="outline" onclick={() => { convertNotFoundPath = item.path; convertTarget = ''; }}>
                  Create Redirect
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
  title="Delete Redirect Rule"
  description={`Are you sure you want to delete the redirect rule for '${deleteRedirectTarget?.pattern || deleteRedirectTarget?.source}'?`}
  confirmLabel="Delete"
  loading={saving}
  onconfirm={handleDeleteRedirect}
  oncancel={() => (deleteRedirectTarget = null)}
/>
