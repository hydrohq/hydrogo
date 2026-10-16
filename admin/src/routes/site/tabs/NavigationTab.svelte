<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';

  interface NavigationItem {
    id: string;
    locale: string;
    label: string;
    href: string;
    target?: string;
    position: number;
    visible: boolean;
  }

  interface NavigationResponse {
    locale: string;
    locales: string[];
    items: NavigationItem[];
  }

  let selectedLocale = $state('tr');
  let availableLocales = $state<string[]>(['tr', 'en']);
  let items = $state<NavigationItem[]>([]);
  let loading = $state(true);
  let saving = $state(false);
  let msg = $state<string | null>(null);
  let err = $state<string | null>(null);

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

  async function loadNavigation(loc: string) {
    loading = true;
    try {
      const res = await api.get<ApiResponse<NavigationResponse>>(`/api/v1/admin/site/navigation/${loc}`);
      if (res?.data) {
        items = res.data.items || [];
        availableLocales = res.data.locales || ['tr', 'en'];
        selectedLocale = res.data.locale || loc;
      }
    } catch (e: any) {
      notify(e.message || 'Failed to load navigation', true);
    } finally {
      loading = false;
    }
  }

  function addItem() {
    items = [
      ...items,
      {
        id: 'new_' + Math.random().toString(36).substring(2, 9),
        locale: selectedLocale,
        label: '',
        href: '/',
        target: '_self',
        position: items.length,
        visible: true,
      },
    ];
  }

  function removeItem(index: number) {
    items = items.filter((_, i) => i !== index);
  }

  function moveItem(index: number, direction: 'up' | 'down') {
    if (direction === 'up' && index > 0) {
      const updated = [...items];
      const temp = updated[index - 1];
      updated[index - 1] = updated[index];
      updated[index] = temp;
      items = updated;
    } else if (direction === 'down' && index < items.length - 1) {
      const updated = [...items];
      const temp = updated[index + 1];
      updated[index + 1] = updated[index];
      updated[index] = temp;
      items = updated;
    }
  }

  async function handleSave() {
    saving = true;
    try {
      const cleanedItems = items.map((it, idx) => ({
        id: it.id.startsWith('new_') ? '' : it.id,
        locale: selectedLocale,
        label: it.label.trim(),
        href: it.href.trim(),
        target: it.target || '_self',
        position: idx,
        visible: it.visible !== false,
      }));

      await api.put(`/api/v1/admin/site/navigation/${selectedLocale}`, { items: cleanedItems });
      notify('Navigation saved successfully!');
      await loadNavigation(selectedLocale);
    } catch (e: any) {
      notify(e.message || 'Failed to save navigation', true);
    } finally {
      saving = false;
    }
  }

  onMount(() => {
    loadNavigation(selectedLocale);
  });
</script>

<div class="space-y-4">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-2 border-b border-border">
    <div>
      <h3 class="text-sm font-semibold text-foreground">Menu & Navigation Tree</h3>
      <p class="text-xs text-muted-foreground">Configure the site header and footer menu links per locale.</p>
    </div>

    <div class="flex items-center gap-2">
      <!-- Locale Switcher -->
      <div class="flex items-center rounded-lg border border-border bg-muted/40 p-0.5 text-xs">
        {#each availableLocales as loc}
          <button
            type="button"
            onclick={() => { selectedLocale = loc; loadNavigation(loc); }}
            class="px-2.5 py-1 rounded-md uppercase font-medium transition-colors cursor-pointer {selectedLocale === loc ? 'bg-background text-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
          >
            {loc}
          </button>
        {/each}
      </div>

      <Button variant="outline" class="text-xs gap-1.5" onclick={addItem}>
        <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
        <span>Add Link</span>
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

  {#if loading}
    <div class="p-8 text-center text-xs text-muted-foreground">Loading menu items...</div>
  {:else if items.length === 0}
    <div class="rounded-xl border border-dashed border-border p-8 text-center text-xs text-muted-foreground space-y-2">
      <p>No navigation links configured for {selectedLocale.toUpperCase()}.</p>
      <Button variant="outline" class="text-xs" onclick={addItem}>Add First Link</Button>
    </div>
  {:else}
    <div class="space-y-2">
      {#each items as item, index (item.id)}
        <div class="flex items-center gap-2 rounded-lg border border-border bg-card p-2.5 shadow-xs">
          <!-- Reorder buttons -->
          <div class="flex flex-col gap-0.5 shrink-0">
            <button
              type="button"
              disabled={index === 0}
              onclick={() => moveItem(index, 'up')}
              class="size-5 rounded hover:bg-muted text-muted-foreground disabled:opacity-30 flex items-center justify-center cursor-pointer"
              title="Move Up"
            >
              <svg class="size-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="18 15 12 9 6 15"></polyline></svg>
            </button>
            <button
              type="button"
              disabled={index === items.length - 1}
              onclick={() => moveItem(index, 'down')}
              class="size-5 rounded hover:bg-muted text-muted-foreground disabled:opacity-30 flex items-center justify-center cursor-pointer"
              title="Move Down"
            >
              <svg class="size-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"></polyline></svg>
            </button>
          </div>

          <!-- Label -->
          <div class="flex-1 min-w-[120px]">
            <input
              bind:value={item.label}
              placeholder="Menu Label (e.g. About, Contact)"
              class="w-full rounded-md border border-input bg-background px-2.5 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            />
          </div>

          <!-- URL / Href -->
          <div class="flex-1 min-w-[140px]">
            <input
              bind:value={item.href}
              placeholder="Target URL (e.g. /about, https://...)"
              class="w-full rounded-md border border-input bg-background px-2.5 py-1 text-xs font-mono text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            />
          </div>

          <!-- Target Window -->
          <select
            bind:value={item.target}
            class="rounded-md border border-input bg-background px-2 py-1 text-xs text-foreground shrink-0 focus:outline-none focus:ring-1 focus:ring-ring"
          >
            <option value="_self">Same tab (_self)</option>
            <option value="_blank">New tab (_blank)</option>
          </select>

          <!-- Visible Toggle -->
          <label class="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer shrink-0 select-none px-1">
            <input type="checkbox" bind:checked={item.visible} class="size-3.5 rounded text-primary" />
            <span class="hidden sm:inline">Visible</span>
          </label>

          <!-- Remove -->
          <button
            type="button"
            onclick={() => removeItem(index)}
            class="size-7 rounded hover:bg-muted text-muted-foreground hover:text-destructive flex items-center justify-center shrink-0 cursor-pointer"
            title="Remove Item"
          >
            <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
          </button>
        </div>
      {/each}
    </div>

    <div class="flex justify-end pt-3">
      <Button onclick={handleSave} disabled={saving}>
        {saving ? 'Saving...' : 'Save Navigation'}
      </Button>
    </div>
  {/if}
</div>
