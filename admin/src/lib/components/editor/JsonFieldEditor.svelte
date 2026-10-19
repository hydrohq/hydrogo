<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  interface Props {
    value?: any;
    label?: string;
    required?: boolean;
    placeholder?: string;
  }

  let {
    value = $bindable(),
    label,
    required = false,
  }: Props = $props();

  type Mode = 'form' | 'raw';
  let mode = $state<Mode>('form');

  // Key-Value rows for objects
  interface KeyValueRow {
    id: string;
    key: string;
    val: string;
  }

  // Item rows for arrays
  interface ArrayRow {
    id: string;
    val: string;
  }

  let isArray = $state(false);
  let objectRows = $state<KeyValueRow[]>([]);
  let arrayRows = $state<ArrayRow[]>([]);
  let rawJsonText = $state('');
  let parseError = $state<string | null>(null);

  function generateId(): string {
    return Math.random().toString(36).substring(2, 9);
  }

  function parseInitial() {
    let parsed: any = null;
    let rawStr = '';

    if (typeof value === 'object' && value !== null) {
      parsed = value;
      rawStr = JSON.stringify(value, null, 2);
    } else if (typeof value === 'string' && value.trim()) {
      rawStr = value;
      try {
        parsed = JSON.parse(value);
      } catch {
        // Not valid JSON yet, stay in raw mode
        mode = 'raw';
        rawJsonText = value;
        parseError = t('json.notValid');
        return;
      }
    } else {
      // Empty -> initialize empty object
      parsed = {};
      rawStr = '{}';
    }

    rawJsonText = rawStr;
    parseError = null;

    if (Array.isArray(parsed)) {
      isArray = true;
      arrayRows = parsed.map((item) => ({
        id: generateId(),
        val: typeof item === 'object' ? JSON.stringify(item) : String(item ?? ''),
      }));
    } else if (typeof parsed === 'object' && parsed !== null) {
      isArray = false;
      objectRows = Object.entries(parsed).map(([k, v]) => ({
        id: generateId(),
        key: k,
        val: typeof v === 'object' ? JSON.stringify(v) : String(v ?? ''),
      }));
    }
  }

  let lastSyncedValue: any = undefined;

  $effect(() => {
    if (value !== lastSyncedValue) {
      lastSyncedValue = value;
      parseInitial();
    }
  });

  function syncFromForm() {
    if (isArray) {
      const arr = arrayRows.map((r) => {
        const trimmed = r.val.trim();
        if (!isNaN(Number(trimmed)) && trimmed !== '') return Number(trimmed);
        if (trimmed === 'true') return true;
        if (trimmed === 'false') return false;
        return r.val;
      });
      const str = JSON.stringify(arr, null, 2);
      rawJsonText = str;
      value = typeof value === 'object' ? arr : str;
    } else {
      const obj: Record<string, any> = {};
      for (const row of objectRows) {
        if (!row.key.trim()) continue;
        const trimmed = row.val.trim();
        if (!isNaN(Number(trimmed)) && trimmed !== '') {
          obj[row.key.trim()] = Number(trimmed);
        } else if (trimmed === 'true') {
          obj[row.key.trim()] = true;
        } else if (trimmed === 'false') {
          obj[row.key.trim()] = false;
        } else {
          obj[row.key.trim()] = row.val;
        }
      }
      const str = JSON.stringify(obj, null, 2);
      rawJsonText = str;
      value = typeof value === 'object' ? obj : str;
    }
    lastSyncedValue = value;
    parseError = null;
  }

  function handleRawChange(newText: string) {
    rawJsonText = newText;
    try {
      const parsed = JSON.parse(newText);
      parseError = null;
      value = typeof value === 'object' ? parsed : newText;
      lastSyncedValue = value;
      // Also update form rows
      if (Array.isArray(parsed)) {
        isArray = true;
        arrayRows = parsed.map((item) => ({
          id: generateId(),
          val: typeof item === 'object' ? JSON.stringify(item) : String(item ?? ''),
        }));
      } else if (typeof parsed === 'object' && parsed !== null) {
        isArray = false;
        objectRows = Object.entries(parsed).map(([k, v]) => ({
          id: generateId(),
          key: k,
          val: typeof v === 'object' ? JSON.stringify(v) : String(v ?? ''),
        }));
      }
    } catch (e: any) {
      parseError = e?.message || t('json.invalid');
      value = newText;
      lastSyncedValue = value;
    }
  }

  function addObjectRow() {
    objectRows = [...objectRows, { id: generateId(), key: '', val: '' }];
    syncFromForm();
  }

  function removeObjectRow(id: string) {
    objectRows = objectRows.filter((r) => r.id !== id);
    syncFromForm();
  }

  function addArrayRow() {
    arrayRows = [...arrayRows, { id: generateId(), val: '' }];
    syncFromForm();
  }

  function removeArrayRow(id: string) {
    arrayRows = arrayRows.filter((r) => r.id !== id);
    syncFromForm();
  }

  function prettifyJson() {
    try {
      const parsed = JSON.parse(rawJsonText);
      rawJsonText = JSON.stringify(parsed, null, 2);
      parseError = null;
    } catch {}
  }
</script>

<div class="space-y-2">
  <!-- Header with Label and Mode Switcher -->
  <div class="flex items-center justify-between gap-2">
    <span class="text-xs font-semibold text-foreground flex items-center gap-1.5">
      <span>{label ?? t('json.data')}</span>
      {#if required}
        <span class="text-destructive">*</span>
      {/if}
      <span class="text-[10px] font-mono px-1.5 py-0.2 rounded bg-muted text-muted-foreground border border-border/50">
        JSON
      </span>
    </span>

    <!-- Switch between Visual Form and Raw JSON -->
    <div class="inline-flex items-center rounded-lg border border-border bg-background p-0.5 text-xs">
      <button
        type="button"
        class="px-2.5 py-1 rounded-md font-medium transition-colors cursor-pointer {mode === 'form' ? 'bg-muted text-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { mode = 'form'; }}
      >
        {t('json.formMode')}
      </button>
      <button
        type="button"
        class="px-2.5 py-1 rounded-md font-medium transition-colors cursor-pointer {mode === 'raw' ? 'bg-muted text-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { mode = 'raw'; }}
      >
        {t('json.rawMode')}
      </button>
    </div>
  </div>

  {#if mode === 'form'}
    <!-- Visual Key-Value / Array Form Inputs -->
    <div class="rounded-xl border border-border bg-card p-3.5 space-y-3 shadow-xs">
      {#if isArray}
        <!-- Array Item Inputs -->
        {#if arrayRows.length === 0}
          <p class="text-xs text-muted-foreground py-2 text-center">{t('json.noItems')}</p>
        {:else}
          <div class="space-y-2">
            {#each arrayRows as row, idx (row.id)}
              <div class="flex items-center gap-2">
                <span class="w-6 text-[11px] font-mono font-semibold text-muted-foreground text-center">#{idx + 1}</span>
                <input
                  type="text"
                  placeholder={t('json.valuePlaceholder')}
                  bind:value={row.val}
                  oninput={syncFromForm}
                  class="flex-1 h-8 rounded-md border border-input bg-background px-3 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
                />
                <button
                  type="button"
                  class="size-8 inline-flex items-center justify-center rounded-md text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors cursor-pointer"
                  onclick={() => removeArrayRow(row.id)}
                  title={t('json.removeItem')}
                >
                  <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M3 6h18m-2 0v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                  </svg>
                </button>
              </div>
            {/each}
          </div>
        {/if}

        <button
          type="button"
          onclick={addArrayRow}
          class="inline-flex items-center gap-1.5 h-7 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs"
        >
          <svg class="size-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          <span>{t('json.addItem')}</span>
        </button>
      {:else}
        <!-- Object Key-Value Table Inputs -->
        {#if objectRows.length === 0}
          <p class="text-xs text-muted-foreground py-2 text-center">{t('json.noFields')}</p>
        {:else}
          <div class="space-y-2">
            <div class="grid grid-cols-12 gap-2 text-[11px] font-semibold text-muted-foreground px-1 uppercase tracking-wider">
              <span class="col-span-5">{t('json.colKey')}</span>
              <span class="col-span-6">{t('json.colValue')}</span>
              <span class="col-span-1 text-center">{t('json.colAction')}</span>
            </div>
            {#each objectRows as row (row.id)}
              <div class="grid grid-cols-12 gap-2 items-center">
                <div class="col-span-5">
                  <input
                    type="text"
                    placeholder={t('json.keyExample')}
                    bind:value={row.key}
                    oninput={syncFromForm}
                    class="w-full h-8 rounded-md border border-input bg-background px-3 text-xs font-medium text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
                  />
                </div>
                <div class="col-span-6">
                  <input
                    type="text"
                    placeholder={t('json.valueExample')}
                    bind:value={row.val}
                    oninput={syncFromForm}
                    class="w-full h-8 rounded-md border border-input bg-background px-3 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
                  />
                </div>
                <div class="col-span-1 flex justify-center">
                  <button
                    type="button"
                    class="size-8 inline-flex items-center justify-center rounded-md text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors cursor-pointer"
                    onclick={() => removeObjectRow(row.id)}
                    title={t('json.removeField')}
                  >
                    <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M3 6h18m-2 0v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                    </svg>
                  </button>
                </div>
              </div>
            {/each}
          </div>
        {/if}

        <button
          type="button"
          onclick={addObjectRow}
          class="inline-flex items-center gap-1.5 h-7 px-3 rounded-md border border-dashed border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shadow-xs mt-1"
        >
          <svg class="size-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          <span>{t('json.addField')}</span>
        </button>
      {/if}
    </div>
  {:else}
    <!-- Raw JSON Textarea with Format Button -->
    <div class="space-y-1.5">
      <div class="relative">
        <textarea
          rows="6"
          value={rawJsonText}
          oninput={(e) => handleRawChange((e.target as HTMLTextAreaElement).value)}
          placeholder={`{\n  "${t('json.exampleKey')}": "${t('json.exampleValue')}"\n}`}
          class="w-full rounded-md border border-input bg-background p-3 font-mono text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs leading-relaxed"
        ></textarea>
        <button
          type="button"
          onclick={prettifyJson}
          class="absolute right-2.5 top-2.5 px-2 py-0.5 rounded border border-border bg-muted text-[10px] font-mono text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
        >
          {t('json.format')}
        </button>
      </div>

      {#if parseError}
        <p class="text-[11px] font-medium text-destructive flex items-center gap-1">
          <svg class="size-3 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="12" y1="8" x2="12" y2="12"></line>
            <line x1="12" y1="16" x2="12.01" y2="16"></line>
          </svg>
          <span>{t('json.error', { message: parseError })}</span>
        </p>
      {/if}
    </div>
  {/if}
</div>
