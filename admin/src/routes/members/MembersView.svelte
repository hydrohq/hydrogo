<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';

  interface Member {
    id: string;
    email: string;
    active: boolean;
    email_verified_at?: string;
    created_at: string;
    last_login_at?: string;
  }

  let members = $state<Member[]>([]);
  let loading = $state(true);
  let search = $state('');
  let statusFilter = $state('all');
  let selectedMember = $state<Member | null>(null);
  let detailOpen = $state(false);
  let actionLoading = $state(false);
  let revokeConfirmOpen = $state(false);
  let feedbackMessage = $state<{ type: 'success' | 'error'; text: string } | null>(null);

  const filteredMembers = $derived.by(() => {
    let list = members;
    if (statusFilter === 'active') {
      list = list.filter((m) => m.active);
    } else if (statusFilter === 'inactive') {
      list = list.filter((m) => !m.active);
    } else if (statusFilter === 'verified') {
      list = list.filter((m) => !!m.email_verified_at);
    } else if (statusFilter === 'unverified') {
      list = list.filter((m) => !m.email_verified_at);
    }
    if (search.trim()) {
      const q = search.toLowerCase();
      list = list.filter((m) => m.email.toLowerCase().includes(q));
    }
    return list;
  });

  async function loadMembers() {
    loading = true;
    try {
      const res = await api.get<ApiResponse<Member[]>>('/api/v1/admin/members', {
        params: {
          limit: 100,
        },
      });
      if (res && res.data) {
        members = Array.isArray(res.data) ? res.data : [];
      }
    } catch (e) {
      console.warn('Members load error:', e);
      members = [];
    } finally {
      loading = false;
    }
  }

  async function toggleActive(member: Member, e: MouseEvent) {
    e.stopPropagation();
    try {
      await api.patch(`/api/v1/admin/members/${member.id}`, { active: !member.active });
      member.active = !member.active;
    } catch (err: any) {
      console.error('Update status failed:', err);
    }
  }

  async function verifyEmail() {
    if (!selectedMember) return;
    actionLoading = true;
    feedbackMessage = null;
    try {
      await api.post(`/api/v1/admin/members/${selectedMember.id}/verify-email`);
      selectedMember.email_verified_at = new Date().toISOString();
      await loadMembers();
      feedbackMessage = { type: 'success', text: t('members.verifyDone') };
    } catch (err: any) {
      feedbackMessage = { type: 'error', text: err?.message || t('members.verifyFailed') };
    } finally {
      actionLoading = false;
    }
  }

  async function confirmRevokeSessions() {
    if (!selectedMember) return;
    actionLoading = true;
    feedbackMessage = null;
    try {
      await api.post(`/api/v1/admin/members/${selectedMember.id}/sessions/revoke-all`);
      revokeConfirmOpen = false;
      feedbackMessage = { type: 'success', text: t('members.revokeDone') };
    } catch (err: any) {
      feedbackMessage = { type: 'error', text: err?.message || t('members.revokeFailed') };
    } finally {
      actionLoading = false;
    }
  }

  function filterLabel(filter: string): string {
    switch (filter) {
      case 'active': return t('common.active');
      case 'inactive': return t('members.filter.inactive');
      case 'verified': return t('members.filter.verified');
      case 'unverified': return t('members.verification.unverified');
      default: return t('common.all');
    }
  }

  onMount(() => {
    loadMembers();
  });
</script>

<ConfirmDialog
  open={revokeConfirmOpen}
  title={t('members.revokeTitle')}
  description={t('members.revokeDescription', { email: selectedMember?.email ?? '' })}
  confirmLabel={t('members.revokeConfirm')}
  variant="destructive"
  loading={actionLoading}
  onconfirm={confirmRevokeSessions}
  oncancel={() => { if (!actionLoading) revokeConfirmOpen = false; }}
/>

<!-- Member Detail Modal -->
<Modal
  open={detailOpen}
  title={selectedMember?.email || t('members.detail')}
  description={`ID: ${selectedMember?.id || ''}`}
  onclose={() => { detailOpen = false; }}
>
  {#if selectedMember}
    <div class="space-y-4">
      {#if feedbackMessage}
        <div class="p-3 rounded-lg text-xs {feedbackMessage.type === 'success' ? 'bg-emerald-500/10 border border-emerald-500/20 text-emerald-500' : 'bg-destructive/10 border border-destructive/20 text-destructive'} flex items-center justify-between">
          <span>{feedbackMessage.text}</span>
          <button type="button" class="hover:underline cursor-pointer" onclick={() => { feedbackMessage = null; }}>{t('common.dismiss')}</button>
        </div>
      {/if}

      <div class="p-4 rounded-lg bg-muted/50 border border-border space-y-2.5 text-xs">
        <div class="flex justify-between items-center">
          <span class="text-muted-foreground font-medium">{t('members.status')}</span>
          <Badge variant={selectedMember.active ? 'success' : 'destructive'} size="sm">
            {selectedMember.active ? t('common.active') : t('common.disabled')}
          </Badge>
        </div>
        <div class="flex justify-between items-center">
          <span class="text-muted-foreground font-medium">{t('members.emailVerification')}</span>
          {#if selectedMember.email_verified_at}
            <span class="text-emerald-500 font-medium">{t('members.verifiedOn', { date: new Date(selectedMember.email_verified_at).toLocaleDateString() })}</span>
          {:else}
            <span class="text-amber-500 font-medium">{t('members.verification.unverified')}</span>
          {/if}
        </div>
        <div class="flex justify-between items-center">
          <span class="text-muted-foreground font-medium">{t('members.since')}</span>
          <span class="text-foreground">{new Date(selectedMember.created_at).toLocaleDateString()}</span>
        </div>
      </div>

      <div class="space-y-2 pt-2 border-t border-border">
        {#if !selectedMember.email_verified_at}
          <Button
            variant="outline"
            size="sm"
            class="w-full"
            loading={actionLoading}
            onclick={verifyEmail}
          >
            {t('members.markVerified')}
          </Button>
        {/if}
        <Button
          variant="danger"
          size="sm"
          class="w-full"
          loading={actionLoading}
          onclick={() => { revokeConfirmOpen = true; }}
        >
          {t('members.revokeAll')}
        </Button>
      </div>
    </div>
  {/if}
  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => { detailOpen = false; }}>{t('common.close')}</Button>
  {/snippet}
</Modal>

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-2">
    <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <!-- Toolbar (exact shadcn Tabs + Search + CTA layout) -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 px-4 lg:px-6">
        <!-- Segmented Status Tabs -->
        <div class="inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground overflow-x-auto max-w-full">
          {#each ['all', 'active', 'inactive', 'verified', 'unverified'] as filter}
            <button
              type="button"
              class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer capitalize {statusFilter === filter ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
              onclick={() => { statusFilter = filter; }}
            >
              {filterLabel(filter)}
            </button>
          {/each}
        </div>

        <!-- Search & Export CTA -->
        <div class="flex items-center gap-2">
          <div class="relative w-48 sm:w-64">
            <svg class="absolute left-2.5 top-2.5 size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"></circle>
              <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
            </svg>
            <input
              type="search"
              placeholder={t('members.search')}
              bind:value={search}
              class="h-8 w-full rounded-md border border-input bg-background pl-8 pr-3 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
            />
          </div>

          <a
            href="/api/v1/admin/members/export?format=csv"
            download="members.csv"
            class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer shrink-0 shadow-xs"
          >
            <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
              <polyline points="7 10 12 15 17 10"></polyline>
              <line x1="12" y1="15" x2="12" y2="3"></line>
            </svg>
            <span>{t('members.export')}</span>
          </a>
        </div>
      </div>

      <!-- Table Container -->
      <div class="px-4 lg:px-6">
        <div class="overflow-hidden rounded-xl border border-border bg-card shadow-xs">
          {#if loading}
            <div class="p-12 text-center text-xs text-muted-foreground">
              {t('members.loading')}
            </div>
          {:else if filteredMembers.length === 0}
            <div class="p-12 text-center text-xs text-muted-foreground">
              {t('members.empty')}
            </div>
          {:else}
            <div class="overflow-x-auto">
              <table class="w-full text-left text-sm text-foreground caption-bottom">
                <thead class="sticky top-0 z-10 bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
                  <tr>
                    <th class="h-10 px-4 font-medium">{t('common.email')}</th>
                    <th class="h-10 px-4 font-medium">{t('common.status')}</th>
                    <th class="h-10 px-4 font-medium">{t('members.col.verification')}</th>
                    <th class="h-10 px-4 font-medium">{t('members.col.joined')}</th>
                    <th class="h-10 w-24 px-4 font-medium text-right">{t('common.actions')}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-border">
                  {#each filteredMembers as m (m.id)}
                    <tr
                      class="hover:bg-muted/50 transition-colors cursor-pointer"
                      onclick={() => { selectedMember = m; detailOpen = true; }}
                    >
                      <td class="px-4 py-3 font-medium text-foreground">
                        {m.email}
                      </td>
                      <td class="px-4 py-3">
                        <Badge variant={m.active ? 'success' : 'destructive'} size="sm">
                          {m.active ? t('common.active') : t('common.disabled')}
                        </Badge>
                      </td>
                      <td class="px-4 py-3">
                        {#if m.email_verified_at}
                          <span class="text-xs text-emerald-500 font-medium">{t('members.verifiedShort')}</span>
                        {:else}
                          <span class="text-xs text-amber-500 font-medium">{t('members.verification.unverified')}</span>
                        {/if}
                      </td>
                      <td class="px-4 py-3 text-xs text-muted-foreground">
                        {new Date(m.created_at).toLocaleDateString()}
                      </td>
                      <td class="px-4 py-3 text-right">
                        <button
                          type="button"
                          class="px-2.5 py-1 text-xs rounded-md border border-input bg-background hover:bg-accent text-foreground transition-colors cursor-pointer shadow-xs"
                          onclick={(e) => toggleActive(m, e)}
                        >
                          {m.active ? t('members.disable') : t('members.enable')}
                        </button>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>

            <!-- Footer Pagination / Stats -->
            <div class="flex items-center justify-between px-4 py-3 border-t border-border text-xs text-muted-foreground">
              <span>{t('members.showing', { shown: filteredMembers.length, total: members.length })}</span>
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>
