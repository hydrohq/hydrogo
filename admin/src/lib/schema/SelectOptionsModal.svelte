<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';

  interface ContentTypeField {
    id: string;
    label: string;
    kind: string;
    options?: string[];
    [key: string]: unknown;
  }

  interface ContentTypeDef {
    id: string;
    label: string;
    fields: ContentTypeField[];
    [key: string]: unknown;
  }

  interface Props {
    open: boolean;
    collectionId: string;
    fieldId: string;
    fieldLabel: string;
    layer?: 'base' | 'top';
    onclose: () => void;
    onupdated?: (options: string[]) => void;
  }

  let {
    open = false,
    collectionId,
    fieldId,
    fieldLabel,
    layer = 'top',
    onclose,
    onupdated,
  }: Props = $props();

  let loading = $state(false);
  let saving = $state(false);
  let error = $state<string | null>(null);
  let draftInput = $state('');
  let options = $state<string[]>([]);
  let existingDef = $state<ContentTypeDef | null>(null);

  async function loadDefinition() {
    loading = true;
    error = null;
    try {
      const res = await api.get<ApiResponse<ContentTypeDef[]>>('/api/v1/admin/content-types');
      const list = Array.isArray(res?.data) ? res.data : [];
      const found = list.find((d) => d.id === collectionId) ?? null;
      existingDef = found;
      const targetField = found?.fields?.find((f) => f.id === fieldId);
      options = Array.isArray(targetField?.options) ? [...targetField.options] : [];
    } catch (err) {
      error = err instanceof ApiClientError ? err.message : t('ct.err.load');
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    if (open && collectionId && fieldId) {
      draftInput = '';
      void loadDefinition();
    }
  });

  function addOption() {
    const clean = draftInput.trim();
    if (!clean) return;
    if (options.some((o) => o.toLowerCase() === clean.toLowerCase())) {
      error = t('schema.options.duplicate');
      return;
    }
    error = null;
    options = [...options, clean];
    draftInput = '';
  }

  function removeOption(idx: number) {
    options = options.filter((_, i) => i !== idx);
  }

  function moveOption(idx: number, delta: -1 | 1) {
    const target = idx + delta;
    if (target < 0 || target >= options.length) return;
    const copy = [...options];
    const [moved] = copy.splice(idx, 1);
    copy.splice(target, 0, moved);
    options = copy;
  }

  async function saveOptions() {
    if (!existingDef) return;
    saving = true;
    error = null;
    try {
      const cleaned = options.map((s) => s.trim()).filter(Boolean);
      const updatedDef: ContentTypeDef = {
        ...existingDef,
        fields: existingDef.fields.map((f) =>
          f.id === fieldId ? { ...f, options: cleaned } : f
        ),
      };
      await api.put(`/api/v1/admin/content-types/${collectionId}`, updatedDef);
      await schema.loadSchema(true);
      onupdated?.(cleaned);
      onclose();
    } catch (err) {
      error = err instanceof ApiClientError ? err.message : t('ct.err.update');
    } finally {
      saving = false;
    }
  }
</script>

<Modal
  {open}
  {layer}
  size="md"
  title={t('schema.options.title', { label: fieldLabel })}
  description={t('schema.options.description')}
  {onclose}
>
  {#if loading}
    <div class="py-10 flex items-center justify-center text-xs text-muted-foreground gap-2">
      <div class="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
      <span>{t('common.loading')}</span>
    </div>
  {:else}
    <div class="space-y-4">
      {#if error}
        <div class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive" role="alert">
          {error}
        </div>
      {/if}

      <div class="flex items-center gap-2">
        <input
          type="text"
          bind:value={draftInput}
          placeholder={t('schema.options.placeholder')}
          aria-label={t('schema.options.placeholder')}
          onkeydown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault();
              addOption();
            }
          }}
          class="h-8 flex-1 rounded-md border border-input bg-background px-3 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
        />
        <Button type="button" variant="outline" size="sm" onclick={addOption}>
          {t('schema.options.add')}
        </Button>
      </div>

      {#if options.length === 0}
        <p class="rounded-lg border border-dashed border-border py-6 text-center text-xs text-muted-foreground">
          {t('schema.options.empty')}
        </p>
      {:else}
        <ul class="rounded-lg border border-border divide-y divide-border bg-background/50">
          {#each options as opt, idx (`${idx}-${opt}`)}
            <li class="flex items-center justify-between gap-2 px-3 py-2 text-xs">
              <span class="font-medium text-foreground truncate">{opt}</span>
              <div class="flex items-center gap-1 shrink-0">
                <button
                  type="button"
                  disabled={idx === 0}
                  aria-label={t('schema.options.moveUp')}
                  title={t('schema.options.moveUp')}
                  class="size-6 inline-flex items-center justify-center rounded border border-border bg-card text-muted-foreground hover:text-foreground disabled:opacity-40 cursor-pointer"
                  onclick={() => moveOption(idx, -1)}
                >↑</button>
                <button
                  type="button"
                  disabled={idx === options.length - 1}
                  aria-label={t('schema.options.moveDown')}
                  title={t('schema.options.moveDown')}
                  class="size-6 inline-flex items-center justify-center rounded border border-border bg-card text-muted-foreground hover:text-foreground disabled:opacity-40 cursor-pointer"
                  onclick={() => moveOption(idx, 1)}
                >↓</button>
                <button
                  type="button"
                  aria-label={t('schema.relationRemove', { name: opt })}
                  class="size-6 inline-flex items-center justify-center rounded border border-border bg-card text-muted-foreground hover:text-destructive cursor-pointer"
                  onclick={() => removeOption(idx)}
                >×</button>
              </div>
            </li>
          {/each}
        </ul>
      {/if}
    </div>
  {/if}

  {#snippet footer()}
    <Button variant="ghost" size="sm" onclick={onclose} disabled={saving}>
      {t('common.cancel')}
    </Button>
    <Button variant="primary" size="sm" loading={saving} disabled={loading || !existingDef} onclick={saveOptions}>
      {t('common.save')}
    </Button>
  {/snippet}
</Modal>
