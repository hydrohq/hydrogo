<script lang="ts">
  import { untrack } from 'svelte';
  import { t } from '$lib/i18n/index.svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { schema } from '$lib/stores/schema.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';

  export interface ContentTypeField {
    id: string;
    label: string;
    kind: string;
    required?: boolean;
    unique?: boolean;
    list?: boolean;
    searchable?: boolean;
    filterable?: boolean;
    options?: string[];
    to?: string;
    many?: boolean;
    value_field?: string;
    widget?: string;
    private?: boolean;
    [key: string]: unknown;
  }

  export interface ContentTypeDefinition {
    id: string;
    label: string;
    description?: string;
    group?: string;
    title_field?: string;
    public?: boolean;
    drafts?: boolean;
    revisions?: boolean;
    localization?: boolean;
    realtime?: boolean;
    singleton?: boolean;
    access_rules?: Record<string, string>;
    fields: ContentTypeField[];
    [key: string]: unknown;
  }

  interface EditableField {
    id: string;
    label: string;
    kind: string;
    widget: string;
    required: boolean;
    list: boolean;
    searchable: boolean;
    filterable: boolean;
    optionsText: string;
    to: string;
    many: boolean;
    value_field: string;
    locked: boolean;
  }

  interface Props {
    open: boolean;
    editing: ContentTypeDefinition | null;
    onclose: () => void;
    onsaved: (msg: string, createdId?: string) => Promise<void> | void;
  }

  let { open = false, editing = null, onclose, onsaved }: Props = $props();

  const isEdit = $derived(!!editing);
  let saving = $state(false);
  let error = $state<string | null>(null);

  let label = $state('');
  let id = $state('');
  let idTouched = $state(false);
  let group = $state('');
  let description = $state('');
  let titleField = $state('');
  let isPublic = $state(true);
  let drafts = $state(true);
  let revisions = $state(true);
  let realtime = $state(false);
  let singleton = $state(false);
  let showAccessRules = $state(false);
  let ruleList = $state('');
  let ruleRead = $state('');
  let ruleCreate = $state('');
  let ruleUpdate = $state('');
  let ruleDelete = $state('');
  let fields = $state<EditableField[]>([]);

  function slugifyKey(text: string): string {
    return text
      .toLowerCase()
      .trim()
      .replace(/[^a-z0-9_]+/g, '_')
      .replace(/^_+|_+$/g, '');
  }

  function resetFromProps() {
    error = null;
    if (editing) {
      label = editing.label || '';
      id = editing.id || '';
      idTouched = true;
      group = editing.group || '';
      description = editing.description || '';
      titleField = editing.title_field || '';
      isPublic = editing.public !== false;
      drafts = !!editing.drafts;
      revisions = !!editing.revisions;
      realtime = !!editing.realtime;
      singleton = !!editing.singleton;
      const rules = editing.access_rules ?? {};
      ruleList = rules.list || '';
      ruleRead = rules.read || '';
      ruleCreate = rules.create || '';
      ruleUpdate = rules.update || '';
      ruleDelete = rules.delete || '';
      showAccessRules = Object.values(rules).some(Boolean);
      fields = (editing.fields || []).map((f) => ({
        id: f.id,
        label: f.label,
        kind: f.kind,
        widget: f.widget || '',
        required: !!f.required,
        list: !!f.list,
        searchable: !!f.searchable,
        filterable: !!f.filterable,
        optionsText: Array.isArray(f.options) ? f.options.join(', ') : '',
        to: f.to || '',
        many: !!f.many,
        value_field: f.value_field || '',
        locked: true,
      }));
    } else {
      label = '';
      id = '';
      idTouched = false;
      group = '';
      description = '';
      titleField = '';
      isPublic = true;
      drafts = true;
      revisions = true;
      realtime = false;
      singleton = false;
      ruleList = '';
      ruleRead = '';
      ruleCreate = '';
      ruleUpdate = '';
      ruleDelete = '';
      showAccessRules = false;
      fields = [
        {
          id: 'title',
          label: 'Title',
          kind: 'text',
          widget: '',
          required: true,
          list: true,
          searchable: true,
          filterable: false,
          optionsText: '',
          to: '',
          many: false,
          value_field: '',
          locked: false,
        },
      ];
    }
  }

  $effect(() => {
    const active = open;
    const targetId = editing?.id ?? '';
    void targetId;
    if (active) {
      untrack(() => resetFromProps());
    }
  });

  function handleLabelInput(val: string) {
    label = val;
    if (!isEdit && !idTouched) {
      id = slugifyKey(val);
    }
  }

  function addField() {
    fields = [
      ...fields,
      {
        id: '',
        label: '',
        kind: 'text',
        widget: '',
        required: false,
        list: true,
        searchable: false,
        filterable: false,
        optionsText: '',
        to: '',
        many: false,
        value_field: '',
        locked: false,
      },
    ];
  }

  function addFieldKind(kind: string) {
    const cap = kind.charAt(0).toUpperCase() + kind.slice(1);
    const count = fields.filter((f) => f.kind === kind).length + 1;
    const fieldId = count > 1 ? `${kind}_${count}` : kind;
    fields = [
      ...fields,
      {
        id: fieldId,
        label: count > 1 ? `${cap} ${count}` : cap,
        kind,
        widget: '',
        required: false,
        list: true,
        searchable: kind === 'text' || kind === 'richtext' || kind === 'textarea',
        filterable: kind === 'text' || kind === 'number' || kind === 'select' || kind === 'boolean',
        optionsText: '',
        to: '',
        many: false,
        value_field: '',
        locked: false,
      },
    ];
  }

  function removeField(idx: number) {
    if (fields.length <= 1) return;
    fields = fields.filter((_, i) => i !== idx);
  }

  async function handleSubmit() {
    const cleanId = slugifyKey(id);
    if (!cleanId || !label.trim()) {
      error = t('ct.err.required');
      return;
    }

    const origById = new Map<string, ContentTypeField>(
      (editing?.fields ?? []).map((f) => [f.id, f])
    );

    const cleanedFields: ContentTypeField[] = fields
      .filter((f) => f.label.trim() !== '')
      .map((f) => {
        const fid = slugifyKey(f.id || f.label);
        const orig = origById.get(fid);
        const out: ContentTypeField = orig
          ? { ...orig }
          : { id: fid, label: f.label.trim(), kind: f.kind };
        out.id = fid;
        out.label = f.label.trim();
        out.kind = orig ? orig.kind : f.kind;
        out.required = !!f.required;
        out.list = !!f.list;
        out.searchable = !!f.searchable;
        out.filterable = !!f.filterable;

        if (out.kind === 'select') {
          out.options = f.optionsText.split(',').map((s) => s.trim()).filter(Boolean);
        } else {
          delete out.options;
        }

        if (out.kind === 'text') {
          if (f.widget) out.widget = f.widget;
          else delete out.widget;
          if (f.widget === 'relation') {
            out.to = f.to.trim();
            if (f.many) out.many = true;
            else delete out.many;
            if (f.value_field.trim()) out.value_field = f.value_field.trim();
            else delete out.value_field;
          } else {
            delete out.to;
            delete out.many;
            delete out.value_field;
          }
        } else if (out.kind === 'relation') {
          delete out.widget;
          delete out.value_field;
          out.to = f.to.trim();
          if (f.many) out.many = true;
          else delete out.many;
        } else {
          delete out.to;
          delete out.many;
          delete out.value_field;
        }
        return out;
      });

    if (cleanedFields.length === 0) {
      error = t('ct.err.field');
      return;
    }

    const accessRules: Record<string, string> = {};
    if (ruleList.trim()) accessRules.list = ruleList.trim();
    if (ruleRead.trim()) accessRules.read = ruleRead.trim();
    if (ruleCreate.trim()) accessRules.create = ruleCreate.trim();
    if (ruleUpdate.trim()) accessRules.update = ruleUpdate.trim();
    if (ruleDelete.trim()) accessRules.delete = ruleDelete.trim();

    const payload: ContentTypeDefinition = editing
      ? { ...editing }
      : { id: cleanId, label: label.trim(), fields: cleanedFields };

    payload.id = cleanId;
    payload.label = label.trim();
    payload.description = description.trim() || undefined;
    payload.group = group.trim() || undefined;
    payload.title_field = titleField.trim() || cleanedFields[0].id;
    payload.public = isPublic;
    payload.drafts = drafts;
    payload.revisions = revisions || drafts;
    payload.realtime = realtime;
    payload.singleton = singleton || undefined;
    payload.access_rules = Object.keys(accessRules).length > 0 ? accessRules : undefined;
    payload.fields = cleanedFields;

    saving = true;
    error = null;
    try {
      if (isEdit && editing) {
        await api.put(`/api/v1/admin/content-types/${editing.id}`, payload);
        await schema.loadSchema(true);
        await onsaved(t('ct.updated', { label: payload.label }));
      } else {
        await api.post('/api/v1/admin/content-types', payload);
        await schema.loadSchema(true);
        await onsaved(t('ct.created', { label: payload.label }), cleanId);
      }
      onclose();
    } catch (err) {
      error = err instanceof ApiClientError ? err.message : t(isEdit ? 'ct.err.update' : 'ct.err.create');
    } finally {
      saving = false;
    }
  }
</script>

<Modal
  {open}
  size="2xl"
  {onclose}
  title={isEdit ? t('ct.editModalTitle', { label: editing?.label ?? '' }) : t('ct.modalTitle')}
>
  <div class="space-y-4 max-h-[75vh] overflow-y-auto px-1">
    {#if error}
      <div class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive" role="alert">
        {error}
      </div>
    {/if}

    <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
      <Input
        label={t('ct.label')}
        placeholder={t('ct.labelPlaceholder')}
        value={label}
        oninput={(e: Event) => handleLabelInput((e.currentTarget as HTMLInputElement).value)}
        required
      />
      <Input
        label={t('ct.id')}
        placeholder={t('ct.idPlaceholder')}
        bind:value={id}
        disabled={isEdit}
        oninput={() => { idTouched = true; }}
        required
      />
      <Input
        label={t('ct.group')}
        placeholder={t('ct.groupPlaceholder')}
        bind:value={group}
      />
    </div>

    <div>
      <label for="ct-desc" class="text-xs font-medium text-foreground">{t('ct.description')}</label>
      <input
        id="ct-desc"
        bind:value={description}
        placeholder={t('ct.descriptionPlaceholder')}
        class="mt-1 w-full rounded-md border border-input bg-background px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
      />
    </div>

    <!-- Features -->
    <div class="rounded-lg border border-border p-3 space-y-2 bg-muted/20">
      <div class="text-xs font-semibold text-muted-foreground">{t('ct.features')}</div>
      <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 pt-1">
        <label class="flex items-center gap-2 cursor-pointer text-xs select-none">
          <input type="checkbox" bind:checked={isPublic} class="size-4 rounded border-input text-primary" />
          <span>{t('ct.featApi')}</span>
        </label>
        <label class="flex items-center gap-2 cursor-pointer text-xs select-none">
          <input type="checkbox" bind:checked={drafts} class="size-4 rounded border-input text-primary" />
          <span>{t('ct.featDrafts')}</span>
        </label>
        <label class="flex items-center gap-2 cursor-pointer text-xs select-none">
          <input type="checkbox" bind:checked={revisions} class="size-4 rounded border-input text-primary" />
          <span>{t('ct.featRevisions')}</span>
        </label>
        <label class="flex items-center gap-2 cursor-pointer text-xs select-none">
          <input type="checkbox" bind:checked={realtime} class="size-4 rounded border-input text-primary" />
          <span>{t('ct.featRealtime')}</span>
        </label>
        <label class="flex items-center gap-2 cursor-pointer text-xs select-none">
          <input type="checkbox" bind:checked={singleton} class="size-4 rounded border-input text-primary" />
          <span>{t('ct.singleton')}</span>
        </label>
      </div>
    </div>

    <!-- Access Rules (RLS) -->
    <div class="rounded-lg border border-border p-3 space-y-2 bg-muted/20">
      <div class="flex items-center justify-between">
        <div class="text-xs font-semibold text-muted-foreground">{t('ct.accessRules')}</div>
        <button
          type="button"
          class="text-xs text-primary hover:underline font-medium cursor-pointer"
          onclick={() => { showAccessRules = !showAccessRules; }}
        >{showAccessRules ? t('ct.hide') : t('ct.configure')}</button>
      </div>
      {#if showAccessRules}
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 pt-1 text-xs">
          <div>
            <label for="rule-list" class="text-[11px] text-muted-foreground">{t('ct.rule.list')}</label>
            <input id="rule-list" bind:value={ruleList} placeholder="true" class="w-full rounded border border-input bg-background px-2.5 py-1 text-xs font-mono text-foreground" />
          </div>
          <div>
            <label for="rule-read" class="text-[11px] text-muted-foreground">{t('ct.rule.read')}</label>
            <input id="rule-read" bind:value={ruleRead} placeholder="true" class="w-full rounded border border-input bg-background px-2.5 py-1 text-xs font-mono text-foreground" />
          </div>
          <div>
            <label for="rule-create" class="text-[11px] text-muted-foreground">{t('ct.rule.create')}</label>
            <input id="rule-create" bind:value={ruleCreate} placeholder="auth.is_admin" class="w-full rounded border border-input bg-background px-2.5 py-1 text-xs font-mono text-foreground" />
          </div>
          <div>
            <label for="rule-update" class="text-[11px] text-muted-foreground">{t('ct.rule.update')}</label>
            <input id="rule-update" bind:value={ruleUpdate} placeholder="auth.is_admin" class="w-full rounded border border-input bg-background px-2.5 py-1 text-xs font-mono text-foreground" />
          </div>
          <div class="sm:col-span-2">
            <label for="rule-delete" class="text-[11px] text-muted-foreground">{t('ct.rule.delete')}</label>
            <input id="rule-delete" bind:value={ruleDelete} placeholder="auth.is_admin" class="w-full rounded border border-input bg-background px-2.5 py-1 text-xs font-mono text-foreground" />
          </div>
        </div>
      {/if}
    </div>

    <!-- Field Definitions -->
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <span class="text-xs font-semibold text-foreground">{t('ct.fields', { count: fields.length })}</span>
        <button
          type="button"
          class="text-xs text-primary hover:underline flex items-center gap-1 cursor-pointer font-medium"
          onclick={addField}
        >+ {t('ct.addField')}</button>
      </div>

      {#if isEdit}
        <p class="text-[11px] text-muted-foreground">{t('ct.fieldLockedHint')}</p>
      {/if}

      <div class="flex flex-wrap items-center gap-1.5 pt-1 pb-1">
        <span class="text-[11px] font-medium text-muted-foreground mr-1">{t('ct.palette')}</span>
        {#each ['text', 'number', 'boolean', 'select', 'richtext', 'datetime', 'media', 'relation'] as kind}
          <button
            type="button"
            class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-muted hover:bg-muted/80 text-foreground transition-colors cursor-pointer border border-border"
            onclick={() => addFieldKind(kind)}
          >+ {kind}</button>
        {/each}
      </div>

      <div class="space-y-2.5">
        {#each fields as f, idx (`${idx}-${f.locked ? f.id : 'new'}`)}
          <div class="rounded-lg border border-border bg-card p-3 shadow-xs space-y-2">
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-2">
              <input
                bind:value={f.label}
                placeholder={t('ct.fieldLabel')}
                class="rounded-md border border-input bg-background px-2.5 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              />
              <input
                bind:value={f.id}
                disabled={f.locked}
                placeholder={t('ct.fieldId')}
                class="rounded-md border border-input bg-background px-2.5 py-1 text-xs font-mono text-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-60"
              />
              <select
                bind:value={f.kind}
                disabled={f.locked}
                class="rounded-md border border-input bg-background px-2.5 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:opacity-60"
              >
                <option value="text">{t('ct.kind.text')}</option>
                <option value="textarea">{t('ct.kind.textarea')}</option>
                <option value="richtext">{t('ct.kind.richtext')}</option>
                <option value="number">{t('ct.kind.number')}</option>
                <option value="boolean">{t('ct.kind.boolean')}</option>
                <option value="select">{t('ct.kind.select')}</option>
                <option value="date">{t('ct.kind.date')}</option>
                <option value="datetime">{t('ct.kind.datetime')}</option>
                <option value="media">{t('ct.kind.media')}</option>
                <option value="relation">{t('ct.kind.relation')}</option>
              </select>
            </div>

            {#if f.kind === 'select'}
              <input
                bind:value={f.optionsText}
                placeholder={t('ct.options')}
                class="w-full rounded-md border border-input bg-background px-2.5 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              />
            {:else if f.kind === 'text'}
              <div class="grid grid-cols-1 sm:grid-cols-3 gap-2">
                <select
                  aria-label={t('ct.widget')}
                  bind:value={f.widget}
                  class="rounded-md border border-input bg-background px-2.5 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                >
                  <option value="">{t('ct.widget.default')}</option>
                  <option value="textarea">{t('ct.kind.textarea')}</option>
                  <option value="media">{t('ct.widget.media')}</option>
                  <option value="relation">{t('ct.widget.relation')}</option>
                </select>
                {#if f.widget === 'relation'}
                  <input
                    bind:value={f.to}
                    placeholder={t('ct.target')}
                    class="rounded-md border border-input bg-background px-2.5 py-1 text-xs font-mono text-foreground"
                  />
                  <input
                    bind:value={f.value_field}
                    placeholder={t('ct.valueField')}
                    class="rounded-md border border-input bg-background px-2.5 py-1 text-xs font-mono text-foreground"
                  />
                {/if}
              </div>
            {:else if f.kind === 'relation'}
              <input
                bind:value={f.to}
                placeholder={t('ct.target')}
                class="w-full rounded-md border border-input bg-background px-2.5 py-1 text-xs font-mono text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              />
            {/if}

            <div class="flex items-center justify-between pt-1 gap-2 flex-wrap">
              <div class="flex flex-wrap items-center gap-3 text-[11px] text-muted-foreground select-none">
                <label class="flex items-center gap-1.5 cursor-pointer">
                  <input type="checkbox" bind:checked={f.required} class="size-3.5 rounded text-primary" />
                  <span>{t('ct.required')}</span>
                </label>
                <label class="flex items-center gap-1.5 cursor-pointer">
                  <input type="checkbox" bind:checked={f.list} class="size-3.5 rounded text-primary" />
                  <span>{t('ct.showInTable')}</span>
                </label>
                <label class="flex items-center gap-1.5 cursor-pointer">
                  <input type="checkbox" bind:checked={f.searchable} class="size-3.5 rounded text-primary" />
                  <span>{t('ct.searchable')}</span>
                </label>
                <label class="flex items-center gap-1.5 cursor-pointer">
                  <input type="checkbox" bind:checked={f.filterable} class="size-3.5 rounded text-primary" />
                  <span>{t('ct.filterable')}</span>
                </label>
                {#if f.kind === 'relation' || (f.kind === 'text' && f.widget === 'relation')}
                  <label class="flex items-center gap-1.5 cursor-pointer">
                    <input type="checkbox" bind:checked={f.many} class="size-3.5 rounded text-primary" />
                    <span>{t('ct.many')}</span>
                  </label>
                {/if}
              </div>

              {#if fields.length > 1 && !f.locked}
                <button
                  type="button"
                  onclick={() => removeField(idx)}
                  class="text-xs text-muted-foreground hover:text-destructive cursor-pointer"
                  title={t('ct.removeField')}
                >×</button>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    </div>
  </div>

  {#snippet footer()}
    <Button variant="ghost" size="sm" onclick={onclose} disabled={saving}>
      {t('common.cancel')}
    </Button>
    <Button size="sm" loading={saving} onclick={handleSubmit} disabled={saving || !label || !id}>
      {#if isEdit}
        {saving ? t('ct.saving') : t('ct.saveChanges')}
      {:else}
        {saving ? t('ct.generating') : t('ct.modalTitle')}
      {/if}
    </Button>
  {/snippet}
</Modal>
