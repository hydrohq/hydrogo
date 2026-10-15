<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';

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

  async function loadMembers() {
    loading = true;
    try {
      const res = await api.get<ApiResponse<Member[]>>('/api/v1/admin/members', {
        params: {
          q: search || undefined,
          status: statusFilter === 'all' ? undefined : statusFilter,
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
      await api.post(`/api/v1/admin/members/${member.id}`, { active: !member.active }, { method: 'PATCH' });
      member.active = !member.active;
    } catch (err: any) {
      alert(`Update status failed: ${err.message}`);
    }
  }

  async function verifyEmail() {
    if (!selectedMember) return;
    actionLoading = true;
    try {
      await api.post(`/api/v1/admin/members/${selectedMember.id}/verify-email`);
      selectedMember.email_verified_at = new Date().toISOString();
      await loadMembers();
    } catch (err: any) {
      alert(`Verify email failed: ${err.message}`);
    } finally {
      actionLoading = false;
    }
  }

  async function revokeAllSessions() {
    if (!selectedMember || !confirm('Revoke all active sessions for this member?')) return;
    actionLoading = true;
    try {
      await api.post(`/api/v1/admin/members/${selectedMember.id}/sessions/revoke-all`);
      alert('All sessions revoked.');
    } catch (err: any) {
      alert(`Revoke failed: ${err.message}`);
    } finally {
      actionLoading = false;
    }
  }

  onMount(() => {
    loadMembers();
  });
</script>

<!-- Member Detail Modal -->
<Modal
  open={detailOpen}
  title={selectedMember?.email || 'Member Detail'}
  description={`ID: ${selectedMember?.id || ''}`}
  onclose={() => { detailOpen = false; }}
>
  {#if selectedMember}
    <div class="space-y-4">
      <div class="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-2 text-xs">
        <div class="flex justify-between">
          <span class="text-slate-400">Status:</span>
          <Badge variant={selectedMember.active ? 'success' : 'danger'} size="sm">
            {selectedMember.active ? 'Active' : 'Disabled'}
          </Badge>
        </div>
        <div class="flex justify-between">
          <span class="text-slate-400">Email Verification:</span>
          {#if selectedMember.email_verified_at}
            <span class="text-emerald-400 font-medium">Verified ({new Date(selectedMember.email_verified_at).toLocaleDateString()})</span>
          {:else}
            <span class="text-amber-400 font-medium">Unverified</span>
          {/if}
        </div>
        <div class="flex justify-between">
          <span class="text-slate-400">Member Since:</span>
          <span class="text-slate-200">{new Date(selectedMember.created_at).toLocaleDateString()}</span>
        </div>
      </div>

      <div class="space-y-2 pt-2 border-t border-slate-800">
        {#if !selectedMember.email_verified_at}
          <Button
            variant="outline"
            size="sm"
            class="w-full"
            loading={actionLoading}
            onclick={verifyEmail}
          >
            Mark Email as Verified
          </Button>
        {/if}
        <Button
          variant="danger"
          size="sm"
          class="w-full"
          loading={actionLoading}
          onclick={revokeAllSessions}
        >
          Revoke All Active Sessions
        </Button>
      </div>
    </div>
  {/if}
  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => { detailOpen = false; }}>Close</Button>
  {/snippet}
</Modal>

<div class="space-y-6 max-w-6xl mx-auto">
  <!-- Header -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 pb-4 border-b border-slate-800">
    <div>
      <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2">
        <span>👥</span>
        <span>Members</span>
      </h2>
      <p class="text-xs text-slate-400 mt-1">
        Manage customer accounts, membership status, and authentication sessions.
      </p>
    </div>

    <a
      href="/api/v1/admin/members/export?format=csv"
      class="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-xs font-medium text-slate-200 border border-slate-700 transition-colors"
      download="members.csv"
    >
      <span>📥</span>
      <span>Export CSV</span>
    </a>
  </div>

  <!-- Filters -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
    <div class="flex items-center rounded-xl bg-slate-900 border border-slate-800 p-1 text-xs">
      {#each ['all', 'active', 'inactive', 'verified', 'unverified'] as filter}
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg capitalize transition-colors {statusFilter === filter ? 'bg-slate-800 text-white font-semibold' : 'text-slate-400 hover:text-slate-200'}"
          onclick={() => { statusFilter = filter; loadMembers(); }}
        >
          {filter}
        </button>
      {/each}
    </div>

    <input
      type="search"
      placeholder="Search by email…"
      bind:value={search}
      oninput={() => loadMembers()}
      class="bg-slate-900 border border-slate-800 rounded-lg px-3.5 py-1.5 text-xs text-slate-100 placeholder:text-slate-500 focus:outline-none focus:border-sky-500 w-full sm:w-64"
    />
  </div>

  <!-- Table -->
  <div class="rounded-xl border border-slate-800 bg-slate-900/60 overflow-hidden shadow-sm">
    {#if loading}
      <div class="py-16 text-center text-slate-500 text-sm">
        <span class="animate-spin inline-block mr-2">⏳</span> Loading members…
      </div>
    {:else if members.length === 0}
      <div class="py-16 text-center text-slate-400 text-sm">
        No registered members found.
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm text-slate-300">
          <thead class="bg-slate-950/80 text-[11px] uppercase tracking-wider text-slate-400 border-b border-slate-800">
            <tr>
              <th class="px-5 py-3 font-semibold">Email</th>
              <th class="px-5 py-3 font-semibold">Status</th>
              <th class="px-5 py-3 font-semibold">Verification</th>
              <th class="px-5 py-3 font-semibold">Joined Date</th>
              <th class="px-5 py-3 font-semibold text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/60">
            {#each members as m (m.id)}
              <tr
                class="hover:bg-slate-800/40 transition-colors cursor-pointer"
                onclick={() => { selectedMember = m; detailOpen = true; }}
              >
                <td class="px-5 py-3.5 font-medium text-slate-200">
                  {m.email}
                </td>
                <td class="px-5 py-3.5">
                  <Badge variant={m.active ? 'success' : 'danger'} size="sm">
                    {m.active ? 'Active' : 'Disabled'}
                  </Badge>
                </td>
                <td class="px-5 py-3.5">
                  {#if m.email_verified_at}
                    <span class="text-xs text-emerald-400 font-medium">✓ Verified</span>
                  {:else}
                    <span class="text-xs text-amber-400 font-medium">Unverified</span>
                  {/if}
                </td>
                <td class="px-5 py-3.5 text-xs text-slate-500">
                  {new Date(m.created_at).toLocaleDateString()}
                </td>
                <td class="px-5 py-3.5 text-right whitespace-nowrap">
                  <button
                    type="button"
                    class="px-2.5 py-1 text-xs text-slate-300 hover:text-white rounded hover:bg-slate-800 transition-colors"
                    onclick={(e) => toggleActive(m, e)}
                  >
                    {m.active ? 'Disable' : 'Enable'}
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>
