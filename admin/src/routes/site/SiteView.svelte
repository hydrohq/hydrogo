<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';

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
  let redirectError = $state<string | null>(null);
  let addingRedirect = $state(false);

  // Delete Redirect Dialog
  let deleteTargetRule = $state<RedirectRule | null>(null);
  let deleteLoading = $state(false);

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
      console.error('Save failed:', err);
    } finally {
      saving = false;
    }
  }

  async function handleAddRedirect(e: SubmitEvent) {
    e.preventDefault();
    if (!newSource.trim() || !newTarget.trim()) return;
    addingRedirect = true;
    redirectError = null;
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
      redirectError = err?.message || 'Failed to add redirect rule';
    } finally {
      addingRedirect = false;
    }
  }

  async function confirmDeleteRedirect() {
    if (!deleteTargetRule) return;
    deleteLoading = true;
    try {
      await api.delete(`/api/v1/admin/site/redirects/${deleteTargetRule.id}`);
      deleteTargetRule = null;
      await loadRedirects();
    } catch (err: any) {
      console.error('Delete failed:', err);
    } finally {
      deleteLoading = false;
    }
  }

  onMount(() => {
    loadSettings();
    loadRedirects();
  });
</script>

<ConfirmDialog
  open={!!deleteTargetRule}
  title="Delete Redirect Rule"
  description={`Permanently remove redirect from "${deleteTargetRule?.source}" to "${deleteTargetRule?.target}"? Incoming visitors will no longer be redirected.`}
  confirmLabel="Delete Rule"
  variant="destructive"
  loading={deleteLoading}
  onconfirm={confirmDeleteRedirect}
  oncancel={() => { if (!deleteLoading) deleteTargetRule = null; }}
/>

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

              {#if redirectError}
                <div class="rounded-lg border border-destructive/30 bg-destructive/10 p-2.5 text-xs text-destructive flex items-center justify-between">
                  <span>{redirectError}</span>
                  <button type="button" class="hover:underline cursor-pointer" onclick={() => { redirectError = null; }}>Dismiss</button>
                </div>
              {/if}

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
                  disabled={addingRedirect}
                  class="h-8 px-3.5 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs transition-colors cursor-pointer disabled:opacity-50 inline-flex items-center gap-1.5"
                >
                  {#if addingRedirect}
                    <div class="size-3 animate-spin rounded-full border-2 border-current border-t-transparent"></div>
                  {/if}
                  <span>Add Rule</span>
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
                              onclick={() => { deleteTargetRule = r; }}
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
