<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { api } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import type { ApiResponse, ContentItem } from '$lib/api/types';
  import {
    parseRelationKeys,
    recordTitle,
    resolveRecordMedia,
    serializeRelationKeys,
  } from './relations';

  // Picks related record(s) visually by title and thumbnail, storing either id(s)
  // or text keys such as product slugs (`valueField`).
  interface Props {
    label: string;
    target: string;
    many?: boolean;
    valueField?: string;
    value?: string | string[];
    required?: boolean;
    id: string;
  }

  let {
    label,
    target,
    many = false,
    valueField = 'id',
    value = $bindable(),
    required = false,
    id,
  }: Props = $props();

  const targetCollection = $derived(schema.getCollection(target));
  const titleField = $derived(targetCollection?.title_field);
  const endpoint = $derived(targetCollection?.api_path || `/api/v1/admin/content/${target}`);

  let query = $state('');
  let options = $state<ContentItem[]>([]);
  let known = $state<Record<string, ContentItem>>({});
  let failed = $state(false);
  let searching = $state(false);
  let timer: ReturnType<typeof setTimeout> | undefined;

  function recordKey(record: ContentItem): string {
    const raw = record[valueField];
    if (raw !== undefined && raw !== null && String(raw).trim() !== '') {
      return String(raw).trim();
    }
    return String(record.id ?? '');
  }

  const selectedKeys = $derived(parseRelationKeys(value, valueField, many));

  function remember(records: ContentItem[]) {
    const next = { ...known };
    for (const record of records) {
      const k = recordKey(record);
      if (k) next[k] = record;
      if (valueField === 'id' && record.id) next[record.id] = record;
    }
    known = next;
  }

  async function search(text: string) {
    searching = true;
    try {
      const res = await api.get<ApiResponse<ContentItem[]>>(endpoint, {
        params: { limit: 50, ...(text.trim() ? { search: text.trim() } : {}) },
      });
      options = Array.isArray(res?.data) ? res.data : [];
      remember(options);
      failed = false;
    } catch {
      options = [];
      failed = true;
    } finally {
      searching = false;
    }
  }

  async function nameSelected() {
    for (const key of selectedKeys) {
      if (known[key]) continue;
      try {
        if (valueField === 'id') {
          const res = await api.get<ApiResponse<ContentItem>>(`${endpoint}/${encodeURIComponent(key)}`);
          if (res?.data) remember([res.data]);
        } else {
          const res = await api.get<ApiResponse<ContentItem[]>>(endpoint, {
            params: { [`filter[${valueField}]`]: key, limit: 5 },
          });
          if (Array.isArray(res?.data) && res.data.length > 0) {
            remember(res.data);
          }
        }
      } catch {
        // Record may be gone or target lacks filter index on valueField; key remains visible.
      }
    }
  }

  $effect(() => {
    void endpoint;
    const text = query;
    clearTimeout(timer);
    timer = setTimeout(() => void search(text), text ? 220 : 0);
    return () => clearTimeout(timer);
  });

  $effect(() => {
    void endpoint;
    void valueField;
    void selectedKeys.join(',');
    void nameSelected();
  });

  const displayTitle = (key: string) => {
    const rec = known[key];
    return rec ? recordTitle(rec, titleField) : key;
  };

  const secondaryMeta = (record: ContentItem): string => {
    const key = recordKey(record);
    const title = recordTitle(record, titleField);
    if (typeof record.sku === 'string' && record.sku.trim()) return record.sku.trim();
    if (typeof record.slug === 'string' && record.slug.trim() && record.slug.trim() !== title) {
      return record.slug.trim();
    }
    return key !== title ? key : '';
  };

  const choices = $derived(
    options.filter((record) => {
      const k = recordKey(record);
      return k !== '' && !selectedKeys.includes(k);
    })
  );

  function selectKey(key: string) {
    if (!key) return;
    if (many) {
      if (selectedKeys.includes(key)) return;
      value = serializeRelationKeys([...selectedKeys, key], valueField, true);
    } else {
      value = serializeRelationKeys([key], valueField, false);
    }
  }

  function removeKey(key: string) {
    const next = selectedKeys.filter((item) => item !== key);
    value = serializeRelationKeys(next, valueField, many);
  }
</script>

<div class="flex flex-col gap-2" data-relation-field={id}>
  <div class="flex items-center justify-between gap-2">
    <label for={`relation-${id}`} class="text-xs font-medium text-foreground">
      {label}{#if required}<span class="text-destructive"> *</span>{/if}
    </label>
    {#if selectedKeys.length > 0}
      <span class="text-[11px] text-muted-foreground tabular-nums">
        {t('schema.relationSelected', { count: selectedKeys.length })}
      </span>
    {/if}
  </div>

  {#if selectedKeys.length > 0}
    <ul class="flex flex-wrap gap-1.5">
      {#each selectedKeys as key (key)}
        {@const rec = known[key]}
        {@const thumb = resolveRecordMedia(rec)}
        {@const name = displayTitle(key)}
        <li class="inline-flex items-center gap-2 rounded-md border border-border bg-muted/40 px-2 py-1 text-xs max-w-full">
          {#if thumb}
            <img
              src={thumb}
              alt={name}
              class="size-6 rounded object-cover border border-border shrink-0"
              onerror={(e) => { (e.currentTarget as HTMLElement).style.display = 'none'; }}
            />
          {/if}
          <span class="truncate font-medium text-foreground">{name}</span>
          {#if name !== key}
            <span class="hidden sm:inline font-mono text-[10px] text-muted-foreground truncate max-w-28">{key}</span>
          {/if}
          <button
            type="button"
            class="ml-0.5 text-muted-foreground hover:text-destructive cursor-pointer shrink-0 leading-none px-1"
            aria-label={t('schema.relationRemove', { name })}
            onclick={() => removeKey(key)}
          >×</button>
        </li>
      {/each}
    </ul>
  {/if}

  <input
    id={`relation-${id}`}
    type="search"
    bind:value={query}
    placeholder={t('schema.relationSearch', { name: targetCollection?.label || target })}
    aria-label={t('schema.relationSearch', { name: targetCollection?.label || target })}
    class="h-8 w-full rounded-md border border-input bg-background px-3 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
  />

  {#if failed}
    <p class="text-xs text-destructive">{t('schema.relationFailed')}</p>
  {:else if choices.length > 0}
    <div
      class="max-h-44 overflow-y-auto rounded-md border border-border bg-card divide-y divide-border"
      role="listbox"
      aria-label={t('schema.relationMatches')}
    >
      {#each choices as record (record.id || recordKey(record))}
        {@const key = recordKey(record)}
        {@const thumb = resolveRecordMedia(record)}
        {@const title = recordTitle(record, titleField)}
        {@const meta = secondaryMeta(record)}
        <button
          type="button"
          role="option"
          aria-selected="false"
          class="w-full flex items-center justify-between gap-2.5 px-2.5 py-1.5 text-left text-xs hover:bg-accent/70 transition-colors cursor-pointer"
          onclick={() => selectKey(key)}
        >
          <span class="flex items-center gap-2 min-w-0">
            {#if thumb}
              <img
                src={thumb}
                alt={title}
                class="size-7 rounded object-cover border border-border shrink-0"
                onerror={(e) => { (e.currentTarget as HTMLElement).style.display = 'none'; }}
              />
            {/if}
            <span class="truncate font-medium text-foreground">{title}</span>
          </span>
          <span class="flex items-center gap-2 shrink-0">
            {#if meta}
              <span class="font-mono text-[10px] text-muted-foreground truncate max-w-28">{meta}</span>
            {/if}
            <span class="inline-flex size-5 items-center justify-center rounded border border-border bg-background text-muted-foreground">+</span>
          </span>
        </button>
      {/each}
    </div>
  {:else if !searching}
    <p class="text-[11px] text-muted-foreground px-1">{t('schema.relationNoMatch')}</p>
  {/if}
</div>
