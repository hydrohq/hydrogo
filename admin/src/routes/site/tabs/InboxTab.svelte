<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';

  interface SubmissionItem {
    id: string;
    form_key: string;
    email: string;
    values: Record<string, string>;
    status: string;
    lead_stage?: string;
    assigned_to?: string;
    location?: {
      country_code?: string;
      city?: string;
      local?: boolean;
    };
    created_at: string;
  }

  interface SubmissionNote {
    id: string;
    author: string;
    note: string;
    created_at: string;
  }

  let items = $state<SubmissionItem[]>([]);
  let loading = $state(true);
  let statusFilter = $state('');
  let formFilter = $state('');
  let formsList = $state<string[]>([]);
  let statusMsg = $state<string | null>(null);
  let statusErr = $state<string | null>(null);

  // Selected Submission Detail
  let selectedItem = $state<SubmissionItem | null>(null);
  let notes = $state<SubmissionNote[]>([]);
  let loadingNotes = $state(false);
  let newNoteText = $state('');
  let addingNote = $state(false);

  // Purge Dialog
  let purgeOpen = $state(false);
  let purgeDays = $state(90);
  let purging = $state(false);

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

  async function loadInbox() {
    loading = true;
    try {
      const params = new URLSearchParams();
      if (statusFilter) params.set('status', statusFilter);
      if (formFilter) params.set('form', formFilter);
      params.set('limit', '100');

      const res = await api.get<ApiResponse<SubmissionItem[]>>(`/api/v1/admin/site/inbox?${params.toString()}`);
      if (res?.data && Array.isArray(res.data)) {
        items = res.data;
        const set = new Set<string>();
        for (const it of items) {
          if (it.form_key) set.add(it.form_key);
        }
        formsList = Array.from(set);
      }
    } catch (e: any) {
      notify(e.message || t('site.inbox.loadFailed'), true);
    } finally {
      loading = false;
    }
  }

  async function openDetail(item: SubmissionItem) {
    selectedItem = item;
    newNoteText = '';
    loadingNotes = true;
    try {
      const res = await api.get<ApiResponse<SubmissionNote[]>>(`/api/v1/admin/site/inbox/${item.id}/notes`);
      if (res?.data && Array.isArray(res.data)) {
        notes = res.data;
      } else {
        notes = [];
      }
    } catch {
      notes = [];
    } finally {
      loadingNotes = false;
    }
  }

  async function updateStatus(newStatus: string) {
    if (!selectedItem) return;
    try {
      await api.patch(`/api/v1/admin/site/inbox/${selectedItem.id}`, { status: newStatus });
      selectedItem.status = newStatus;
      const found = items.find((i) => i.id === selectedItem?.id);
      if (found) found.status = newStatus;
      notify(t('site.inbox.statusChanged', { status: statusLabel(newStatus) }));
    } catch (e: any) {
      notify(e.message || t('site.inbox.statusFailed'), true);
    }
  }

  async function handleAddNote() {
    if (!selectedItem || !newNoteText.trim()) return;
    addingNote = true;
    try {
      await api.post(`/api/v1/admin/site/inbox/${selectedItem.id}/notes`, { note: newNoteText.trim() });
      newNoteText = '';
      const res = await api.get<ApiResponse<SubmissionNote[]>>(`/api/v1/admin/site/inbox/${selectedItem.id}/notes`);
      if (res?.data && Array.isArray(res.data)) {
        notes = res.data;
      }
      notify(t('site.inbox.noteAdded'));
    } catch (e: any) {
      notify(e.message || t('site.inbox.noteFailed'), true);
    } finally {
      addingNote = false;
    }
  }

  async function handlePurge() {
    purging = true;
    try {
      const cutoff = new Date();
      cutoff.setDate(cutoff.getDate() - purgeDays);
      await api.post('/api/v1/admin/site/inbox/purge', { before: cutoff.toISOString() });
      notify(t('site.inbox.purged', { days: purgeDays }));
      purgeOpen = false;
      await loadInbox();
    } catch (e: any) {
      notify(e.message || t('site.inbox.purgeFailed'), true);
    } finally {
      purging = false;
    }
  }

  function getStatusBadgeVariant(st: string): 'default' | 'outline' | 'secondary' | 'success' | 'destructive' {
    if (st === 'new') return 'success';
    if (st === 'processed') return 'secondary';
    if (st === 'archived') return 'outline';
    return 'default';
  }

  onMount(() => {
    loadInbox();
  });
  function statusLabel(status: string): string {
    switch (status) {
      case 'new': return t('site.inbox.new');
      case 'processed': return t('site.inbox.processed');
      case 'archived': return t('site.inbox.archived');
      default: return status;
    }
  }
</script>

<div class="space-y-4">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-2 border-b border-border">
    <div>
      <h3 class="text-sm font-semibold text-foreground">{t('site.inbox.title')}</h3>
      <p class="text-xs text-muted-foreground">{t('site.inbox.help')}</p>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <!-- Export buttons -->
      <a
        href="/api/v1/admin/site/inbox/export.csv{formFilter ? '?form=' + formFilter : ''}"
        download
        class="h-8 px-2.5 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors inline-flex items-center gap-1.5 shadow-xs"
        title={t('site.inbox.downloadCsv')}
      >
        <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>
        <span>CSV</span>
      </a>
      <a
        href="/api/v1/admin/site/inbox/export.xlsx{formFilter ? '?form=' + formFilter : ''}"
        download
        class="h-8 px-2.5 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors inline-flex items-center gap-1.5 shadow-xs"
        title={t('site.inbox.downloadExcel')}
      >
        <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>
        <span>Excel</span>
      </a>

      <!-- Purge button -->
      <Button variant="outline" class="text-xs gap-1.5" onclick={() => { purgeOpen = true; }}>
        <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
        <span>{t('site.inbox.purge')}</span>
      </Button>
    </div>
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

  <!-- Filters -->
  <div class="flex flex-wrap items-center gap-3">
    <div class="flex items-center gap-1.5 text-xs">
      <span class="text-muted-foreground">{t('site.inbox.statusLabel')}</span>
      <select
        bind:value={statusFilter}
        onchange={() => loadInbox()}
        class="rounded-md border border-input bg-background px-2.5 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
      >
        <option value="">{t('site.inbox.allStatuses')}</option>
        <option value="new">{t('site.inbox.new')}</option>
        <option value="processed">{t('site.inbox.processed')}</option>
        <option value="archived">{t('site.inbox.archived')}</option>
      </select>
    </div>

    {#if formsList.length > 0}
      <div class="flex items-center gap-1.5 text-xs">
        <span class="text-muted-foreground">{t('site.inbox.form')}</span>
        <select
          bind:value={formFilter}
          onchange={() => loadInbox()}
          class="rounded-md border border-input bg-background px-2.5 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
        >
          <option value="">{t('site.inbox.allForms')}</option>
          {#each formsList as f}
            <option value={f}>{f}</option>
          {/each}
        </select>
      </div>
    {/if}

    <span class="text-xs text-muted-foreground ml-auto">{t('site.inbox.count', { count: items.length })}</span>
  </div>

  <!-- Submissions Table -->
  {#if loading}
    <div class="p-8 text-center text-xs text-muted-foreground">{t('site.inbox.loading')}</div>
  {:else if items.length === 0}
    <div class="rounded-xl border border-dashed border-border p-8 text-center text-xs text-muted-foreground">
      <p>{t('site.inbox.empty')}</p>
    </div>
  {:else}
    <div class="rounded-xl border border-border bg-card shadow-xs overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="border-b border-border bg-muted/40 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
            <tr>
              <th class="px-4 py-2.5">{t('common.date')}</th>
              <th class="px-4 py-2.5">{t('site.inbox.formCol')}</th>
              <th class="px-4 py-2.5">{t('site.inbox.sender')}</th>
              <th class="px-4 py-2.5">{t('site.inbox.preview')}</th>
              <th class="px-4 py-2.5">{t('common.status')}</th>
              <th class="px-4 py-2.5 text-right">{t('dashboard.table.action')}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            {#each items as it (it.id)}
              <tr
                class="hover:bg-muted/20 transition-colors cursor-pointer"
                onclick={() => openDetail(it)}
              >
                <td class="px-4 py-2.5 text-muted-foreground whitespace-nowrap">
                  {new Date(it.created_at).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                </td>
                <td class="px-4 py-2.5 font-mono text-[11px] text-foreground font-medium whitespace-nowrap">
                  {it.form_key}
                </td>
                <td class="px-4 py-2.5 text-foreground whitespace-nowrap">
                  {it.email || it.values?.email || it.values?.name || t('site.inbox.anonymous')}
                </td>
                <td class="px-4 py-2.5 text-muted-foreground truncate max-w-xs">
                  {it.values?.message || it.values?.subject || Object.values(it.values || {})[0] || '—'}
                </td>
                <td class="px-4 py-2.5 whitespace-nowrap">
                  <Badge variant={getStatusBadgeVariant(it.status)} class="capitalize text-[10px]">
                    {statusLabel(it.status)}
                  </Badge>
                </td>
                <td class="px-4 py-2.5 text-right whitespace-nowrap">
                  <button
                    type="button"
                    class="text-xs text-primary hover:underline font-medium"
                    onclick={(e) => { e.stopPropagation(); openDetail(it); }}
                  >
                    {t('site.inbox.view')}
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {/if}
</div>

<!-- Submission Detail Modal -->
<Modal open={!!selectedItem} onclose={() => { selectedItem = null; }} title={t('site.inbox.submission', { form: selectedItem?.form_key ?? '' })}>
  {#if selectedItem}
    <div class="space-y-4 max-h-[75vh] overflow-y-auto px-1">
      <!-- Status & Metadata Bar -->
      <div class="flex items-center justify-between pb-3 border-b border-border">
        <div>
          <span class="text-xs text-muted-foreground">{t('site.inbox.submitted')} </span>
          <span class="text-xs font-medium text-foreground">{new Date(selectedItem.created_at).toLocaleString()}</span>
        </div>

        <div class="flex items-center gap-1.5">
          <span class="text-xs text-muted-foreground mr-1">{t('site.inbox.statusLabel')}</span>
          {#each ['new', 'processed', 'archived'] as st}
            <button
              type="button"
              onclick={() => updateStatus(st)}
              class="px-2 py-0.5 rounded text-[11px] font-medium capitalize transition-colors cursor-pointer {selectedItem.status === st ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground hover:bg-muted/80'}"
            >
              {statusLabel(st)}
            </button>
          {/each}
        </div>
      </div>

      <!-- Values Grid -->
      <div class="rounded-lg border border-border bg-card p-3.5 space-y-2.5">
        <div class="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">{t('site.inbox.fields')}</div>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
          {#each Object.entries(selectedItem.values || {}) as [k, v]}
            <div class="space-y-0.5">
              <span class="text-muted-foreground font-mono text-[11px]">{k}:</span>
              <p class="font-medium text-foreground break-words">{v || '—'}</p>
            </div>
          {/each}
        </div>
      </div>

      <!-- Location / Tech info -->
      {#if selectedItem.location}
        <div class="text-xs text-muted-foreground flex items-center gap-2">
          <span>{t('site.inbox.location', { place: `${selectedItem.location.city || ''} ${selectedItem.location.country_code || ''}`.trim() })}</span>
          {#if selectedItem.location.local}
            <Badge variant="outline" class="text-[9px]">{t('site.inbox.local')}</Badge>
          {/if}
        </div>
      {/if}

      <!-- Internal Team Notes -->
      <div class="space-y-2 pt-2 border-t border-border">
        <div class="text-xs font-semibold text-foreground">{t('site.inbox.notes', { count: notes.length })}</div>

        {#if loadingNotes}
          <div class="text-xs text-muted-foreground">{t('site.inbox.notesLoading')}</div>
        {:else if notes.length === 0}
          <div class="text-xs text-muted-foreground">{t('site.inbox.notesEmpty')}</div>
        {:else}
          <div class="space-y-1.5 max-h-40 overflow-y-auto">
            {#each notes as n}
              <div class="rounded-md border border-border bg-muted/20 p-2 text-xs space-y-1">
                <div class="flex items-center justify-between text-[10px] text-muted-foreground">
                  <span class="font-medium text-foreground">{n.author || t('site.inbox.staff')}</span>
                  <span>{new Date(n.created_at).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}</span>
                </div>
                <p class="text-foreground">{n.note}</p>
              </div>
            {/each}
          </div>
        {/if}

        <div class="flex gap-2 pt-1">
          <input
            bind:value={newNoteText}
            placeholder={t('site.inbox.notePlaceholder')}
            class="flex-1 rounded-md border border-input bg-background px-2.5 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            onkeydown={(e) => { if (e.key === 'Enter') handleAddNote(); }}
          />
          <Button onclick={handleAddNote} disabled={addingNote || !newNoteText.trim()} class="text-xs">
            {addingNote ? t('site.inbox.adding') : t('site.inbox.addNote')}
          </Button>
        </div>
      </div>
    </div>
  {/if}
</Modal>

<!-- Purge Confirm Modal -->
<ConfirmDialog
  open={purgeOpen}
  title={t('site.inbox.purgeTitle')}
  description={t('site.inbox.purgeDescription', { days: purgeDays })}
  confirmLabel={purging ? t('site.inbox.purging') : t('site.inbox.purgeConfirm')}
  variant="destructive"
  loading={purging}
  onconfirm={handlePurge}
  oncancel={() => { purgeOpen = false; }}
/>
