<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';

  interface FormField {
    id: string;
    label: string;
    type: string;
    required?: boolean;
    options?: string[];
  }

  interface CustomForm {
    id: string;
    title: string;
    fields: FormField[];
    notification_recipients?: string[];
    autoresponder?: boolean;
    autoresponder_subject?: string;
    autoresponder_body?: string;
  }

  let forms = $state<CustomForm[]>([]);
  let loading = $state(true);
  let statusMsg = $state<string | null>(null);
  let statusErr = $state<string | null>(null);

  // Form modal
  let modalOpen = $state(false);
  let editingId = $state<string | null>(null);
  let formTitle = $state('');
  let formId = $state('');
  let formRecipients = $state('');
  let formFields = $state<FormField[]>([
    { id: 'name', label: 'Full Name', type: 'text', required: true },
    { id: 'email', label: 'Email Address', type: 'email', required: true },
    { id: 'message', label: 'Message', type: 'textarea', required: true },
  ]);
  let autoresponder = $state(false);
  let autoSubject = $state('');
  let autoBody = $state('');
  let saving = $state(false);

  // Delete dialog
  let deleteTarget = $state<CustomForm | null>(null);
  let deleting = $state(false);

  function notify(text: string, isErr = false) {
    if (isErr) {
      statusErr = text;
      statusMsg = null;
    } else {
      statusMsg = text;
      statusErr = null;
    }
    setTimeout(() => {
      statusMsg = null;
      statusErr = null;
    }, 3500);
  }

  function slugify(text: string): string {
    return text
      .toLowerCase()
      .trim()
      .replace(/[^a-z0-9_]+/g, '_')
      .replace(/^_+|_+$/g, '');
  }

  async function loadForms() {
    loading = true;
    try {
      const res = await api.get<ApiResponse<CustomForm[]>>('/api/v1/admin/site/forms');
      if (res?.data && Array.isArray(res.data)) {
        forms = res.data;
      }
    } catch (e: any) {
      notify(e.message || 'Failed to load forms', true);
    } finally {
      loading = false;
    }
  }

  function openCreate() {
    editingId = null;
    formTitle = '';
    formId = '';
    formRecipients = '';
    formFields = [
      { id: 'name', label: 'Full Name', type: 'text', required: true },
      { id: 'email', label: 'Email Address', type: 'email', required: true },
      { id: 'message', label: 'Message', type: 'textarea', required: true },
    ];
    autoresponder = false;
    autoSubject = '';
    autoBody = '';
    modalOpen = true;
  }

  function openEdit(f: CustomForm) {
    editingId = f.id;
    formTitle = f.title;
    formId = f.id;
    formRecipients = (f.notification_recipients || []).join(', ');
    formFields = f.fields && f.fields.length > 0 ? JSON.parse(JSON.stringify(f.fields)) : [
      { id: 'email', label: 'Email', type: 'email', required: true },
    ];
    autoresponder = !!f.autoresponder;
    autoSubject = f.autoresponder_subject || '';
    autoBody = f.autoresponder_body || '';
    modalOpen = true;
  }

  function addField() {
    formFields = [
      ...formFields,
      { id: '', label: '', type: 'text', required: false },
    ];
  }

  function removeField(index: number) {
    if (formFields.length <= 1) return;
    formFields = formFields.filter((_, i) => i !== index);
  }

  async function handleSaveForm() {
    const id = slugify(formId || formTitle);
    if (!id || !formTitle.trim()) {
      notify('Form ID and Title are required.', true);
      return;
    }

    const cleanedFields = formFields
      .filter((f) => f.label.trim() !== '')
      .map((f) => ({
        id: slugify(f.id || f.label),
        label: f.label.trim(),
        type: f.type,
        required: !!f.required,
      }));

    if (cleanedFields.length === 0) {
      notify('At least one form field is required.', true);
      return;
    }

    saving = true;
    try {
      const payload: CustomForm = {
        id,
        title: formTitle.trim(),
        fields: cleanedFields,
        notification_recipients: formRecipients
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean),
        autoresponder,
        autoresponder_subject: autoresponder ? autoSubject.trim() : undefined,
        autoresponder_body: autoresponder ? autoBody.trim() : undefined,
      };

      if (editingId) {
        await api.put(`/api/v1/admin/site/forms/${editingId}`, payload);
        notify(`Form '${payload.title}' updated successfully.`);
      } else {
        await api.post('/api/v1/admin/site/forms', payload);
        notify(`Form '${payload.title}' created successfully.`);
      }

      modalOpen = false;
      await loadForms();
    } catch (e: any) {
      notify(e.message || 'Failed to save form', true);
    } finally {
      saving = false;
    }
  }

  async function handleDeleteForm() {
    if (!deleteTarget) return;
    deleting = true;
    try {
      await api.delete(`/api/v1/admin/site/forms/${deleteTarget.id}`);
      notify(`Form '${deleteTarget.title}' deleted.`);
      deleteTarget = null;
      await loadForms();
    } catch (e: any) {
      notify(e.message || 'Failed to delete form', true);
    } finally {
      deleting = false;
    }
  }

  onMount(() => {
    loadForms();
  });
</script>

<div class="space-y-4">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-2 border-b border-border">
    <div>
      <h3 class="text-sm font-semibold text-foreground">Custom Forms Builder</h3>
      <p class="text-xs text-muted-foreground">Build contact, inquiry, survey and lead capture forms.</p>
    </div>

    <Button onclick={openCreate} class="gap-1.5 text-xs shadow-xs">
      <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
      <span>New Form</span>
    </Button>
  </div>

  {#if statusMsg}
    <div class="p-2.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs">
      {statusMsg}
    </div>
  {/if}
  {#if statusErr}
    <div class="p-2.5 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs">
      {statusErr}
    </div>
  {/if}

  {#if loading}
    <div class="p-8 text-center text-xs text-muted-foreground">Loading forms...</div>
  {:else if forms.length === 0}
    <div class="rounded-xl border border-dashed border-border p-8 text-center text-xs text-muted-foreground space-y-2">
      <p>No custom forms configured yet.</p>
      <Button variant="outline" class="text-xs" onclick={openCreate}>Create First Form</Button>
    </div>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each forms as f (f.id)}
        <div class="flex flex-col justify-between rounded-xl border border-border bg-card p-4 shadow-xs hover:border-primary/50 transition-colors">
          <div>
            <div class="flex items-start justify-between gap-2">
              <div class="flex items-center gap-2">
                <div class="size-8 rounded-lg bg-primary/10 text-primary flex items-center justify-center shrink-0">
                  <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect width="18" height="18" x="3" y="3" rx="2" ry="2"></rect>
                    <line x1="3" y1="9" x2="21" y2="9"></line>
                    <line x1="9" y1="21" x2="9" y2="9"></line>
                  </svg>
                </div>
                <div>
                  <h4 class="text-sm font-semibold text-foreground">{f.title}</h4>
                  <span class="font-mono text-[11px] text-muted-foreground">key: {f.id}</span>
                </div>
              </div>

              <div class="flex items-center gap-1">
                <button
                  type="button"
                  class="size-7 rounded hover:bg-muted text-muted-foreground hover:text-foreground flex items-center justify-center cursor-pointer"
                  onclick={() => openEdit(f)}
                  title="Edit Form"
                >
                  <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"></path></svg>
                </button>
                <button
                  type="button"
                  class="size-7 rounded hover:bg-muted text-muted-foreground hover:text-destructive flex items-center justify-center cursor-pointer"
                  onclick={() => { deleteTarget = f; }}
                  title="Delete Form"
                >
                  <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                </button>
              </div>
            </div>

            <div class="flex flex-wrap gap-1.5 mt-3">
              <Badge variant="outline" class="text-[10px]">{f.fields?.length || 0} fields</Badge>
              {#if f.notification_recipients && f.notification_recipients.length > 0}
                <Badge variant="secondary" class="text-[10px]">Email alert</Badge>
              {/if}
              {#if f.autoresponder}
                <Badge variant="success" class="text-[10px]">Autoresponder</Badge>
              {/if}
            </div>
          </div>

          <div class="mt-4 pt-3 border-t border-border flex items-center justify-between text-[11px] text-muted-foreground">
            <span>POST /api/v1/forms/{f.id}/submit</span>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Form Create/Edit Modal -->
<Modal open={modalOpen} onclose={() => { modalOpen = false; }} title={editingId ? `Edit Form: ${formTitle}` : 'Create New Form'}>
  <div class="space-y-4 max-h-[75vh] overflow-y-auto px-1">
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <Input
        label="Form Title"
        placeholder="e.g. Contact Us, Request a Quote"
        bind:value={formTitle}
        required
      />
      <Input
        label="Form ID / Key"
        placeholder="e.g. contact, quote"
        bind:value={formId}
        disabled={!!editingId}
        required
      />
    </div>

    <Input
      label="Notification Emails"
      placeholder="Comma-separated: team@example.com, sales@example.com"
      bind:value={formRecipients}
    />

    <!-- Fields Builder -->
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <span class="text-xs font-semibold text-foreground">Form Fields ({formFields.length})</span>
        <button
          type="button"
          class="text-xs text-primary hover:underline flex items-center gap-1 cursor-pointer font-medium"
          onclick={addField}
        >
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
          <span>Add Field</span>
        </button>
      </div>

      <div class="space-y-2">
        {#each formFields as field, idx (idx)}
          <div class="flex items-center gap-2 rounded-lg border border-border bg-card p-2.5 shadow-xs">
            <input
              bind:value={field.label}
              placeholder="Field Label (e.g. Phone)"
              class="flex-1 rounded-md border border-input bg-background px-2.5 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            />
            <input
              bind:value={field.id}
              placeholder="ID"
              class="w-24 rounded-md border border-input bg-background px-2 py-1 text-xs font-mono text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            />
            <select
              bind:value={field.type}
              class="rounded-md border border-input bg-background px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            >
              <option value="text">Text</option>
              <option value="email">Email</option>
              <option value="textarea">Textarea</option>
              <option value="phone">Phone</option>
              <option value="number">Number</option>
              <option value="checkbox">Checkbox</option>
              <option value="select">Select</option>
            </select>
            <label class="flex items-center gap-1 text-xs text-muted-foreground cursor-pointer select-none">
              <input type="checkbox" bind:checked={field.required} class="size-3.5 rounded text-primary" />
              <span>Req</span>
            </label>
            {#if formFields.length > 1}
              <button
                type="button"
                onclick={() => removeField(idx)}
                class="size-6 rounded hover:bg-muted text-muted-foreground hover:text-destructive flex items-center justify-center cursor-pointer"
                title="Remove field"
                aria-label="Remove field"
              >
                <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
              </button>
            {/if}
          </div>
        {/each}
      </div>
    </div>

    <!-- Autoresponder -->
    <div class="rounded-lg border border-border p-3 space-y-2 bg-muted/20">
      <label class="flex items-center gap-2 cursor-pointer text-xs select-none font-medium">
        <input type="checkbox" bind:checked={autoresponder} class="size-4 rounded text-primary" />
        <span>Enable Automatic Email Reply (Autoresponder)</span>
      </label>

      {#if autoresponder}
        <div class="space-y-2 pt-2">
          <Input
            label="Email Subject"
            placeholder="Thank you for reaching out!"
            bind:value={autoSubject}
          />
          <div>
            <label for="auto-body" class="text-xs font-medium text-foreground">Email Body</label>
            <textarea
              id="auto-body"
              bind:value={autoBody}
              rows="3"
              placeholder="We received your submission and will get back to you shortly..."
              class="w-full rounded-md border border-input bg-background p-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            ></textarea>
          </div>
        </div>
      {/if}
    </div>
  </div>

  <div class="mt-4 pt-3 border-t border-border flex justify-end gap-2">
    <Button variant="outline" onclick={() => { modalOpen = false; }} disabled={saving}>Cancel</Button>
    <Button onclick={handleSaveForm} disabled={saving || !formTitle}>
      {saving ? 'Saving...' : 'Save Form'}
    </Button>
  </div>
</Modal>

<!-- Delete Form Dialog -->
<ConfirmDialog
  open={!!deleteTarget}
  title={`Delete Form '${deleteTarget?.title}'?`}
  description="Deleting this form will prevent new submissions. Existing submission data will remain preserved."
  confirmLabel={deleting ? 'Deleting...' : 'Delete Form'}
  variant="destructive"
  loading={deleting}
  onconfirm={handleDeleteForm}
  oncancel={() => { deleteTarget = null; }}
/>
