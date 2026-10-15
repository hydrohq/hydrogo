<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
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
  let loading = $state(false);
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

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-2">
    <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <!-- Toolbar (exact shadcn Tabs layout) -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 px-4 lg:px-6">
        <div class="inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground">
          <button
            type="button"
            class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {activeTab === 'general' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
            onclick={() => { activeTab = 'general'; }}
          >
            General Settings
          </button>
          <button
            type="button"
            class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {activeTab === 'redirects' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
            onclick={() => { activeTab = 'redirects'; }}
          >
            Redirect Rules ({redirects.length})
          </button>
        </div>

        <div class="flex items-center gap-3">
          {#if loading}
            <span class="text-xs text-muted-foreground animate-pulse">Loading…</span>
          {/if}
          {#if successMsg}
            <div class="text-xs text-emerald-600 dark:text-emerald-400 font-medium animate-fade-in">
              ✓ {successMsg}
            </div>
          {/if}
        </div>
      </div>

      <!-- Tab Content -->
      <div class="px-4 lg:px-6">
        {#if activeTab === 'general'}
          <form onsubmit={handleSaveSettings} class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-4 max-w-2xl">
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
              <label for="site-desc" class="text-xs font-medium text-foreground">
                Site Meta Description
              </label>
              <textarea
                id="site-desc"
                rows="3"
                placeholder="Brief description for search engines…"
                bind:value={settings['site.description']}
                class="w-full rounded-md border border-input bg-transparent px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
              ></textarea>
            </div>

            <Input
              label="Home Path"
              placeholder="e.g. /"
              bind:value={settings['site.home_path']}
            />

            <div class="pt-3 border-t border-border flex justify-end">
              <button
                type="submit"
                disabled={saving}
                class="inline-flex items-center gap-1.5 h-8 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs transition-colors cursor-pointer"
              >
                {saving ? 'Saving…' : 'Save Settings'}
              </button>
            </div>
          </form>
        {:else}
          <div class="space-y-4 max-w-3xl">
            <!-- Add redirect form -->
            <form onsubmit={handleAddRedirect} class="p-4 rounded-xl border border-border bg-card shadow-xs space-y-3">
              <span class="text-xs font-medium text-foreground">Add Redirect Rule</span>
              <div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
                <input
                  type="text"
                  placeholder="From (e.g. /old-page)"
                  bind:value={newSource}
                  required
                  class="h-8 rounded-md border border-input bg-background px-3 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
                />
                <input
                  type="text"
                  placeholder="To (e.g. /new-page)"
                  bind:value={newTarget}
                  required
                  class="h-8 rounded-md border border-input bg-background px-3 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
                />
                <select
                  bind:value={newCode}
                  class="h-8 rounded-md border border-input bg-background px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
                >
                  <option value={301}>301 Permanent</option>
                  <option value={302}>302 Temporary</option>
                </select>
              </div>
              <div class="flex justify-end">
                <button
                  type="submit"
                  class="h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs transition-colors cursor-pointer"
                >
                  Add Rule
                </button>
              </div>
            </form>

            <!-- Redirects Table -->
            <div class="overflow-hidden rounded-xl border border-border bg-card shadow-xs">
              {#if redirects.length === 0}
                <div class="p-8 text-center text-xs text-muted-foreground">
                  No redirect rules configured.
                </div>
              {:else}
                <div class="overflow-x-auto">
                  <table class="w-full text-left text-sm text-foreground caption-bottom">
                    <thead class="sticky top-0 z-10 bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
                      <tr>
                        <th class="h-10 px-4 font-medium">Source</th>
                        <th class="h-10 px-4 font-medium">Destination</th>
                        <th class="h-10 px-4 font-medium">Code</th>
                        <th class="h-10 w-24 px-4 font-medium text-right">Action</th>
                      </tr>
                    </thead>
                    <tbody class="divide-y divide-border">
                      {#each redirects as r (r.id)}
                        <tr class="hover:bg-muted/50 transition-colors">
                          <td class="px-4 py-3 font-mono text-xs text-foreground">{r.source}</td>
                          <td class="px-4 py-3 font-mono text-xs text-muted-foreground">{r.target}</td>
                          <td class="px-4 py-3">
                            <Badge variant="outline" size="sm">{r.code}</Badge>
                          </td>
                          <td class="px-4 py-3 text-right">
                            <button
                              type="button"
                              class="text-xs text-destructive hover:bg-destructive/10 px-2 py-1 rounded transition-colors cursor-pointer"
                              onclick={() => handleDeleteRedirect(r.id)}
                            >
                              Delete
                            </button>
                          </td>
                        </tr>
                      {/each}
                    </tbody>
                  </table>
                </div>
              {/if}
            </div>
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>
