<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';

  interface SettingItem {
    key: string;
    value: string;
  }

  interface RedirectRule {
    id: string;
    source: string;
    target: string;
    code: number;
  }

  let settings = $state<Record<string, string>>({
    'site.name': 'HydroGo',
    'site.title': 'HydroGo Headless CMS',
    'site.description': '',
    'site.home_path': '/',
  });

  let redirects = $state<RedirectRule[]>([]);
  let activeTab = $state<'general' | 'redirects'>('general');
  let loading = $state(true);
  let saving = $state(false);
  let successMsg = $state<string | null>(null);

  // New Redirect
  let newSource = $state('');
  let newTarget = $state('');
  let newCode = $state(301);

  async function loadSettings() {
    loading = true;
    try {
      const res = await api.get<ApiResponse<SettingItem[]>>('/api/v1/admin/site/settings');
      if (res && res.data) {
        const map: Record<string, string> = {};
        for (const s of res.data) {
          map[s.key] = s.value;
        }
        settings = { ...settings, ...map };
      }
    } catch (e) {
      console.warn('Site settings load error:', e);
    } finally {
      loading = false;
    }
  }

  async function loadRedirects() {
    try {
      const res = await api.get<ApiResponse<RedirectRule[]>>('/api/v1/admin/site/redirects');
      if (res && res.data) {
        redirects = Array.isArray(res.data) ? res.data : [];
      }
    } catch (e) {
      console.warn('Redirects load error:', e);
    }
  }

  async function handleSaveSettings(e: SubmitEvent) {
    e.preventDefault();
    saving = true;
    try {
      for (const [key, value] of Object.entries(settings)) {
        await api.post(`/api/v1/admin/site/settings/${key}`, { value }, { method: 'PUT' });
      }
      successMsg = 'Settings saved successfully.';
      setTimeout(() => { successMsg = null; }, 3000);
    } catch (err: any) {
      alert(`Save failed: ${err.message}`);
    } finally {
      saving = false;
    }
  }

  async function handleAddRedirect(e: SubmitEvent) {
    e.preventDefault();
    if (!newSource || !newTarget) return;
    try {
      await api.post('/api/v1/admin/site/redirects', {
        source: newSource.trim(),
        target: newTarget.trim(),
        code: Number(newCode) || 301,
      });
      newSource = '';
      newTarget = '';
      await loadRedirects();
    } catch (err: any) {
      alert(`Add redirect failed: ${err.message}`);
    }
  }

  async function handleDeleteRedirect(id: string) {
    if (!confirm('Delete this redirect rule?')) return;
    try {
      await api.delete(`/api/v1/admin/site/redirects/${id}`);
      await loadRedirects();
    } catch (err: any) {
      alert(`Delete failed: ${err.message}`);
    }
  }

  onMount(() => {
    loadSettings();
    loadRedirects();
  });
</script>

<div class="space-y-6 max-w-4xl mx-auto">
  <!-- Header -->
  <div class="flex items-center justify-between pb-4 border-b border-slate-800">
    <div>
      <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2">
        <span>⚙️</span>
        <span>Site Configuration</span>
      </h2>
      <p class="text-xs text-slate-400 mt-1 flex items-center gap-2">
        <span>Configure public site identity, metadata, and HTTP redirects.</span>
        {#if loading}
          <span class="text-xs text-sky-400">Loading…</span>
        {/if}
      </p>
    </div>
  </div>

  <!-- Tabs Navigation -->
  <div class="flex items-center gap-2 border-b border-slate-800 pb-px">
    <button
      type="button"
      class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {activeTab === 'general' ? 'border-sky-500 text-sky-400 font-semibold' : 'border-transparent text-slate-400 hover:text-slate-200'}"
      onclick={() => { activeTab = 'general'; }}
    >
      General Settings
    </button>
    <button
      type="button"
      class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {activeTab === 'redirects' ? 'border-sky-500 text-sky-400 font-semibold' : 'border-transparent text-slate-400 hover:text-slate-200'}"
      onclick={() => { activeTab = 'redirects'; }}
    >
      Redirect Rules ({redirects.length})
    </button>
  </div>

  {#if successMsg}
    <div class="p-3.5 rounded-xl bg-emerald-950/60 border border-emerald-800/80 text-emerald-300 text-xs">
      ✓ {successMsg}
    </div>
  {/if}

  {#if activeTab === 'general'}
    <!-- General Settings Form -->
    <form onsubmit={handleSaveSettings} class="rounded-2xl border border-slate-800 bg-slate-900/60 p-6 space-y-5">
      <Input
        label="Site Name"
        placeholder="e.g. Acme Corp"
        bind:value={settings['site.name']}
        required
      />

      <Input
        label="Site Title (Default Page Title)"
        placeholder="e.g. Acme - Next Generation Cloud"
        bind:value={settings['site.title']}
        required
      />

      <div class="flex flex-col gap-1.5">
        <label for="site-desc" class="text-xs font-semibold uppercase tracking-wider text-slate-300">
          Site Meta Description
        </label>
        <textarea
          id="site-desc"
          rows="3"
          placeholder="Brief description for search engines…"
          bind:value={settings['site.description']}
          class="w-full rounded-lg bg-slate-900 border border-slate-800 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:border-sky-500"
        ></textarea>
      </div>

      <Input
        label="Home Path"
        placeholder="e.g. /"
        bind:value={settings['site.home_path']}
      />

      <div class="pt-3 border-t border-slate-800 flex justify-end">
        <Button type="submit" variant="primary" size="md" loading={saving}>
          Save Settings
        </Button>
      </div>
    </form>
  {:else}
    <!-- Redirects -->
    <div class="space-y-6">
      <form onsubmit={handleAddRedirect} class="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-3">
        <h4 class="text-xs font-semibold text-slate-300 uppercase tracking-wider">Add Redirect Rule</h4>
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
          <Input
            placeholder="From (e.g. /old-page)"
            bind:value={newSource}
            required
          />
          <Input
            placeholder="To (e.g. /new-page or https://...)"
            bind:value={newTarget}
            required
          />
          <select
            bind:value={newCode}
            class="rounded-lg bg-slate-900 border border-slate-800 px-3 py-2 text-xs text-slate-100 focus:outline-none focus:border-sky-500"
          >
            <option value={301}>301 Permanent</option>
            <option value={302}>302 Temporary</option>
          </select>
        </div>
        <div class="flex justify-end">
          <Button type="submit" variant="primary" size="sm">Add Rule</Button>
        </div>
      </form>

      <div class="rounded-xl border border-slate-800 bg-slate-900/60 overflow-hidden shadow-sm">
        {#if redirects.length === 0}
          <div class="py-12 text-center text-xs text-slate-500">
            No redirect rules configured.
          </div>
        {:else}
          <table class="w-full text-left text-sm text-slate-300">
            <thead class="bg-slate-950/80 text-[11px] uppercase tracking-wider text-slate-400 border-b border-slate-800">
              <tr>
                <th class="px-5 py-3 font-semibold">Source</th>
                <th class="px-5 py-3 font-semibold">Destination</th>
                <th class="px-5 py-3 font-semibold">Code</th>
                <th class="px-5 py-3 font-semibold text-right">Action</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-800/60">
              {#each redirects as r (r.id)}
                <tr class="hover:bg-slate-800/40 transition-colors">
                  <td class="px-5 py-3 font-mono text-xs text-slate-200">{r.source}</td>
                  <td class="px-5 py-3 font-mono text-xs text-sky-400">{r.target}</td>
                  <td class="px-5 py-3">
                    <Badge variant={r.code === 301 ? 'default' : 'info'} size="sm">{r.code}</Badge>
                  </td>
                  <td class="px-5 py-3 text-right">
                    <button
                      type="button"
                      class="text-xs text-rose-400 hover:text-rose-300 p-1"
                      onclick={() => handleDeleteRedirect(r.id)}
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      </div>
    </div>
  {/if}
</div>
