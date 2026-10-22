<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { api } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import type { ApiResponse, ContentItem } from '$lib/api/types';
  import { recordTitle } from './relations';

  // Picks the record(s) a relation field points at by name, so nobody has to
  // type a record id. The value is the id (one) or the list of ids (many).
  interface Props {
    label: string;
    target: string;
    many?: boolean;
    value?: string | string[];
    required?: boolean;
    id: string;
  }

  // No fallback on the bindable: the form is drawn once before the record has loaded,
  // when the bound value is still undefined.
  let { label, target, many = false, value = $bindable(), required = false, id }: Props = $props();

  const targetCollection = $derived(schema.getCollection(target));
  const titleField = $derived(targetCollection?.title_field);
  const endpoint = $derived(targetCollection?.api_path || `/api/v1/admin/content/${target}`);

  let query = $state('');
  let options = $state<ContentItem[]>([]);
  let known = $state<Record<string, ContentItem>>({});
  let failed = $state(false);
  let timer: ReturnType<typeof setTimeout> | undefined;

  const selectedIds = $derived(many ? (Array.isArray(value) ? value : []) : typeof value === 'string' && value ? [value] : []);

  function remember(records: ContentItem[]) {
    const next = { ...known };
    for (const record of records) next[record.id] = record;
    known = next;
  }

  async function search(text: string) {
    try {
      const res = await api.get<ApiResponse<ContentItem[]>>(endpoint, {
        params: { limit: 25, ...(text.trim() ? { search: text.trim() } : {}) },
      });
      options = Array.isArray(res?.data) ? res.data : [];
      remember(options);
      failed = false;
    } catch {
      options = [];
      failed = true;
    }
  }

  // Selected records that are not in the current options still need a name.
  async function nameSelected() {
    for (const selectedId of selectedIds) {
      if (known[selectedId]) continue;
      try {
        const res = await api.get<ApiResponse<ContentItem>>(`${endpoint}/${selectedId}`);
        if (res?.data) remember([res.data]);
      } catch {
        // The record may be gone or hidden: its id stays as the name.
      }
    }
  }

  $effect(() => {
    const text = query;
    clearTimeout(timer);
    timer = setTimeout(() => void search(text), text ? 250 : 0);
    return () => clearTimeout(timer);
  });

  $effect(() => {
    void selectedIds.join(',');
    void nameSelected();
  });

  const name = (recordId: string) => recordTitle(known[recordId] ?? { id: recordId }, titleField);
  const choices = $derived(options.filter((record) => !selectedIds.includes(record.id)));

  function pick(event: Event) {
    const chosen = (event.currentTarget as HTMLSelectElement).value;
    if (many) {
      if (chosen) value = [...selectedIds, chosen];
      (event.currentTarget as HTMLSelectElement).value = '';
    } else {
      value = chosen;
    }
  }

  function remove(recordId: string) {
    value = selectedIds.filter((item) => item !== recordId);
  }
</script>

<div class="flex flex-col gap-1.5" data-relation-field={id}>
  <label for={`relation-${id}`} class="text-xs font-medium text-foreground">{label}{#if required}<span class="text-destructive"> *</span>{/if}</label>

  {#if many && selectedIds.length > 0}
    <ul class="flex flex-wrap gap-1.5">
      {#each selectedIds as recordId (recordId)}
        <li class="inline-flex items-center gap-1 rounded-md border border-input bg-muted/40 px-2 py-0.5 text-xs">
          <span>{name(recordId)}</span>
          <button type="button" class="text-muted-foreground hover:text-foreground cursor-pointer" aria-label={t('schema.relationRemove', { name: name(recordId) })} onclick={() => remove(recordId)}>×</button>
        </li>
      {/each}
    </ul>
  {/if}

  <input
    type="search"
    bind:value={query}
    placeholder={t('schema.relationSearch', { name: targetCollection?.label || target })}
    aria-label={t('schema.relationSearch', { name: targetCollection?.label || target })}
    class="h-8 w-full rounded-md border border-input bg-background px-3 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
  />
  <select
    id={`relation-${id}`}
    value={many ? '' : (typeof value === 'string' ? value : '')}
    onchange={pick}
    required={required && !many}
    class="w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
  >
    <option value="">{many ? t('schema.relationAdd') : t('schema.relationNone')}</option>
    {#if !many && selectedIds[0] && !choices.some((record) => record.id === selectedIds[0])}
      <option value={selectedIds[0]}>{name(selectedIds[0])}</option>
    {/if}
    {#each many ? choices : options as record (record.id)}
      <option value={record.id}>{recordTitle(record, titleField)}</option>
    {/each}
  </select>
  {#if failed}
    <p class="text-xs text-destructive">{t('schema.relationFailed')}</p>
  {/if}
</div>
