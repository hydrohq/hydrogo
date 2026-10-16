<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';

  interface TranslationRow {
    key: string;
    default?: string;
    value: string;
    overridden?: boolean;
  }

  interface TranslationsViewResponse {
    locale: string;
    translations: TranslationRow[];
  }

  let selectedLocale = $state('tr');
  let locales = $state<string[]>(['tr', 'en']);
  let items = $state<TranslationRow[]>([]);
  let search = $state('');
  let loading = $state(true);
  let saving = $state(false);
  let msg = $state<string | null>(null);
  let err = $state<string | null>(null);

  // New translation entry
  let newKey = $state('');
  let newValue = $state('');

  function notify(text: string, isErr = false) {
    if (isErr) {
      err = text;
      msg = null;
    } else {
      msg = text;
      err = null;
    }
    setTimeout(() => {
      msg = null;
      err = null;
    }, 3500);
  }

  async function loadTranslations(loc: string) {
    loading = true;
    try {
      const res = await api.get<ApiResponse<TranslationsViewResponse>>(`/api/v1/admin/site/translations/${loc}`);
      if (res?.data) {
        items = res.data.translations || [];
        selectedLocale = res.data.locale || loc;
      }
    } catch (e: any) {
      notify(e.message || 'Failed to load translations', true);
    } finally {
      loading = false;
    }
  }

  const filteredItems = $derived.by(() => {
    if (!search.trim()) return items;
    const q = search.toLowerCase();
    return items.filter(
      (it) => it.key.toLowerCase().includes(q) || it.value.toLowerCase().includes(q) || (it.default || '').toLowerCase().includes(q)
    );
  });

  async function handleSave() {
    saving = true;
    try {
      const values: Record<string, string> = {};
      for (const it of items) {
        if (it.value !== undefined) {
          values[it.key] = it.value;
        }
      }
      if (newKey.trim()) {
        values[newKey.trim()] = newValue.trim();
      }

      await api.put(`/api/v1/admin/site/translations/${selectedLocale}`, { values });
      notify('Translations saved successfully!');
      newKey = '';
      newValue = '';
      await loadTranslations(selectedLocale);
    } catch (e: any) {
      notify(e.message || 'Failed to save translations', true);
    } finally {
      saving = false;
    }
  }

  onMount(() => {
    loadTranslations(selectedLocale);
  });
</script>

<div class="space-y-4">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-2 border-b border-border">
    <div>
      <h3 class="text-sm font-semibold text-foreground">Site Translations (i18n)</h3>
      <p class="text-xs text-muted-foreground">Manage locale string overrides and dictionary translations.</p>
    </div>

    <div class="flex items-center gap-2">
      <div class="flex items-center rounded-lg border border-border bg-muted/40 p-0.5 text-xs">
        {#each locales as loc}
          <button
            type="button"
            onclick={() => { selectedLocale = loc; loadTranslations(loc); }}
            class="px-2.5 py-1 rounded-md uppercase font-medium transition-colors cursor-pointer {selectedLocale === loc ? 'bg-background text-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
          >
            {loc}
          </button>
        {/each}
      </div>

      <Button onclick={handleSave} disabled={saving} class="text-xs">
        {saving ? 'Saving...' : 'Save Translations'}
      </Button>
    </div>
  </div>

  {#if msg}
    <div class="p-2.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs">
      {msg}
    </div>
  {/if}
  {#if err}
    <div class="p-2.5 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs">
      {err}
    </div>
  {/if}

  <!-- Search & Add bar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
    <div class="relative flex-1 max-w-sm">
      <input
        type="search"
        placeholder="Filter translation keys..."
        bind:value={search}
        class="w-full rounded-md border border-input bg-background pl-8 pr-3 py-1.5 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
      />
      <svg class="size-3.5 absolute left-2.5 top-2.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="11" cy="11" r="8"></circle>
        <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
      </svg>
    </div>

    <span class="text-xs text-muted-foreground">{filteredItems.length} translation strings</span>
  </div>

  {#if loading}
    <div class="p-8 text-center text-xs text-muted-foreground">Loading translations...</div>
  {:else}
    <div class="rounded-xl border border-border bg-card shadow-xs overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="border-b border-border bg-muted/40 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
            <tr>
              <th class="px-4 py-2.5 w-1/3">Key</th>
              <th class="px-4 py-2.5 w-1/4">Default Catalog</th>
              <th class="px-4 py-2.5">Value ({selectedLocale.toUpperCase()})</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            {#each filteredItems as item (item.key)}
              <tr class="hover:bg-muted/20 transition-colors">
                <td class="px-4 py-2.5 font-mono text-[11px] text-foreground font-medium">
                  {item.key}
                  {#if item.overridden}
                    <Badge variant="outline" class="ml-1 text-[9px] py-0 px-1">custom</Badge>
                  {/if}
                </td>
                <td class="px-4 py-2.5 text-muted-foreground truncate max-w-xs">
                  {item.default || '—'}
                </td>
                <td class="px-4 py-2">
                  <input
                    bind:value={item.value}
                    placeholder={item.default || 'Translation value...'}
                    class="w-full rounded-md border border-input bg-background px-2.5 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                  />
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {/if}
</div>
