<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';

  interface StaffUser {
    id: string;
    email: string;
    role: string;
    active: boolean;
    permissions: string[];
    last_login_at?: string;
  }

  interface APIToken {
    id: string;
    name: string;
    prefix: string;
    scope: string;
    expires_at?: string;
    created_at: string;
    last_used_at?: string;
  }

  let users = $state<StaffUser[]>([]);
  let tokens = $state<APIToken[]>([]);
  let loadingUsers = $state(true);
  let loadingTokens = $state(false);
  let activeTab = $state<'users' | 'tokens'>('users');

  // Invite Modal
  let inviteOpen = $state(false);
  let inviteEmail = $state('');
  let inviteRole = $state('editor');
  let inviting = $state(false);

  // New Token Modal
  let tokenModalOpen = $state(false);
  let tokenName = $state('');
  let tokenDays = $state(30);
  let creatingToken = $state(false);
  let newlyIssuedToken = $state<string | null>(null);

  async function loadUsers() {
    loadingUsers = true;
    try {
      const res = await api.get<ApiResponse<StaffUser[]>>('/api/v1/admin/users');
      if (res && res.data) {
        users = Array.isArray(res.data) ? res.data : [];
      }
    } catch (e) {
      console.warn('Users load error:', e);
      users = [];
    } finally {
      loadingUsers = false;
    }
  }

  async function loadTokens() {
    loadingTokens = true;
    try {
      const res = await api.get<ApiResponse<APIToken[]>>('/api/v1/admin/account/tokens');
      if (res && res.data) {
        tokens = Array.isArray(res.data) ? res.data : [];
      }
    } catch (e) {
      console.warn('Tokens load error:', e);
      tokens = [];
    } finally {
      loadingTokens = false;
    }
  }

  async function handleInvite(e: SubmitEvent) {
    e.preventDefault();
    if (!inviteEmail) return;
    inviting = true;
    try {
      await api.post('/api/v1/admin/users/invites', { email: inviteEmail, role: inviteRole });
      alert(`Invitation sent to ${inviteEmail}.`);
      inviteEmail = '';
      inviteOpen = false;
      await loadUsers();
    } catch (err: any) {
      alert(`Invite failed: ${err.message}`);
    } finally {
      inviting = false;
    }
  }

  async function handleCreateToken(e: SubmitEvent) {
    e.preventDefault();
    if (!tokenName) return;
    creatingToken = true;
    try {
      const res = await api.post<ApiResponse<{ token: string }>>('/api/v1/admin/account/tokens', {
        name: tokenName,
        expires_in_days: Number(tokenDays) || 30,
      });
      if (res && res.data && res.data.token) {
        newlyIssuedToken = res.data.token;
      }
      tokenName = '';
      await loadTokens();
    } catch (err: any) {
      alert(`Create token failed: ${err.message}`);
    } finally {
      creatingToken = false;
    }
  }

  async function handleRevokeToken(id: string) {
    if (!confirm('Revoke this personal access token immediately?')) return;
    try {
      await api.delete(`/api/v1/admin/account/tokens/${id}`);
      await loadTokens();
    } catch (err: any) {
      alert(`Revoke failed: ${err.message}`);
    }
  }

  onMount(() => {
    loadUsers();
  });
</script>

<!-- Invite User Modal -->
<Modal
  open={inviteOpen}
  title="Invite Staff Member"
  description="Send an invitation email to add an admin or editor to HydroGo."
  onclose={() => { inviteOpen = false; }}
>
  <form onsubmit={handleInvite} class="space-y-4">
    <Input
      label="Email Address"
      type="email"
      placeholder="colleague@example.com"
      bind:value={inviteEmail}
      required
      autofocus
    />
    <div>
      <label for="invite-role-select" class="text-xs font-semibold uppercase tracking-wider text-slate-300">Role</label>
      <select
        id="invite-role-select"
        bind:value={inviteRole}
        class="w-full mt-1 rounded-lg bg-slate-900 border border-slate-800 px-3 py-2 text-sm text-slate-100 focus:outline-none focus:border-sky-500"
      >
        <option value="editor">Editor (Content manager)</option>
        <option value="author">Author (Create/edit own posts)</option>
        <option value="admin">Admin (Full administrative control)</option>
      </select>
    </div>
    <div class="flex justify-end gap-2 pt-2">
      <Button variant="ghost" size="sm" onclick={() => { inviteOpen = false; }}>Cancel</Button>
      <Button type="submit" variant="primary" size="sm" loading={inviting}>Send Invitation</Button>
    </div>
  </form>
</Modal>

<!-- New Token Modal -->
<Modal
  open={tokenModalOpen}
  title={newlyIssuedToken ? "Personal Access Token Generated" : "Generate Personal Access Token"}
  description={newlyIssuedToken ? "Copy this token now. It will not be shown again." : "Issue a personal Bearer token for CLI, scripts, or API integrations."}
  onclose={() => { tokenModalOpen = false; newlyIssuedToken = null; }}
>
  {#if newlyIssuedToken}
    <div class="space-y-4">
      <div class="p-3.5 rounded-xl bg-amber-950/40 border border-amber-800/80 text-amber-300 text-xs font-mono break-all select-all">
        {newlyIssuedToken}
      </div>
      <Button
        variant="primary"
        size="sm"
        class="w-full"
        onclick={() => {
          if (newlyIssuedToken) navigator.clipboard.writeText(newlyIssuedToken);
          tokenModalOpen = false;
          newlyIssuedToken = null;
        }}
      >
        Copy to Clipboard & Close
      </Button>
    </div>
  {:else}
    <form onsubmit={handleCreateToken} class="space-y-4">
      <Input
        label="Token Name"
        placeholder="e.g. CI Deployment, Backup Script"
        bind:value={tokenName}
        required
        autofocus
      />
      <Input
        label="Expires in (Days)"
        type="number"
        min="1"
        max="365"
        bind:value={tokenDays}
        required
      />
      <div class="flex justify-end gap-2 pt-2">
        <Button variant="ghost" size="sm" onclick={() => { tokenModalOpen = false; }}>Cancel</Button>
        <Button type="submit" variant="primary" size="sm" loading={creatingToken}>Generate Token</Button>
      </div>
    </form>
  {/if}
</Modal>

<div class="space-y-6 max-w-6xl mx-auto">
  <!-- Header -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 pb-4 border-b border-slate-800">
    <div>
      <h2 class="text-xl font-bold text-slate-100 flex items-center gap-2">
        <span>🔐</span>
        <span>Users & Access Control</span>
      </h2>
      <p class="text-xs text-slate-400 mt-1">
        Manage staff accounts, assign roles, and issue personal API access tokens.
      </p>
    </div>

    <div class="flex items-center gap-2">
      {#if activeTab === 'users'}
        <Button variant="primary" size="md" onclick={() => { inviteOpen = true; }}>
          + Invite User
        </Button>
      {:else}
        <Button variant="primary" size="md" onclick={() => { tokenModalOpen = true; }}>
          + New Access Token
        </Button>
      {/if}
    </div>
  </div>

  <!-- Tabs Navigation -->
  <div class="flex items-center gap-2 border-b border-slate-800 pb-px">
    <button
      type="button"
      class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {activeTab === 'users' ? 'border-sky-500 text-sky-400 font-semibold' : 'border-transparent text-slate-400 hover:text-slate-200'}"
      onclick={() => { activeTab = 'users'; loadUsers(); }}
    >
      Staff Users ({users.length})
    </button>
    <button
      type="button"
      class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {activeTab === 'tokens' ? 'border-sky-500 text-sky-400 font-semibold' : 'border-transparent text-slate-400 hover:text-slate-200'}"
      onclick={() => { activeTab = 'tokens'; loadTokens(); }}
    >
      Personal Access Tokens
    </button>
  </div>

  <!-- Tab Content: Users -->
  {#if activeTab === 'users'}
    <div class="rounded-xl border border-slate-800 bg-slate-900/60 overflow-hidden shadow-sm">
      {#if loadingUsers}
        <div class="py-16 text-center text-slate-500 text-sm">
          <span class="animate-spin inline-block mr-2">⏳</span> Loading staff directory…
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-left text-sm text-slate-300">
            <thead class="bg-slate-950/80 text-[11px] uppercase tracking-wider text-slate-400 border-b border-slate-800">
              <tr>
                <th class="px-5 py-3 font-semibold">User</th>
                <th class="px-5 py-3 font-semibold">Role</th>
                <th class="px-5 py-3 font-semibold">Status</th>
                <th class="px-5 py-3 font-semibold">Permissions</th>
                <th class="px-5 py-3 font-semibold">Last Login</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-800/60">
              {#each users as u (u.id)}
                <tr class="hover:bg-slate-800/40 transition-colors">
                  <td class="px-5 py-3.5">
                    <p class="font-medium text-slate-100">{u.email}</p>
                    <span class="text-[10px] text-slate-500 font-mono">ID: {u.id}</span>
                  </td>
                  <td class="px-5 py-3.5">
                    <Badge variant={u.role === 'admin' ? 'info' : 'default'} size="sm">
                      {u.role}
                    </Badge>
                  </td>
                  <td class="px-5 py-3.5">
                    <Badge variant={u.active ? 'success' : 'danger'} size="sm">
                      {u.active ? 'Active' : 'Disabled'}
                    </Badge>
                  </td>
                  <td class="px-5 py-3.5 text-xs text-slate-400">
                    {u.permissions?.length || 0} granted
                  </td>
                  <td class="px-5 py-3.5 text-xs text-slate-500">
                    {u.last_login_at ? new Date(u.last_login_at).toLocaleString() : 'Never'}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>
  {:else}
    <!-- Tab Content: API Tokens -->
    <div class="rounded-xl border border-slate-800 bg-slate-900/60 overflow-hidden shadow-sm">
      {#if loadingTokens}
        <div class="py-16 text-center text-slate-500 text-sm">
          <span class="animate-spin inline-block mr-2">⏳</span> Loading access tokens…
        </div>
      {:else if tokens.length === 0}
        <div class="py-16 text-center text-slate-400 text-sm flex flex-col items-center justify-center gap-3">
          <p>No active personal access tokens.</p>
          <Button variant="secondary" size="sm" onclick={() => { tokenModalOpen = true; }}>
            Generate First Token
          </Button>
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-left text-sm text-slate-300">
            <thead class="bg-slate-950/80 text-[11px] uppercase tracking-wider text-slate-400 border-b border-slate-800">
              <tr>
                <th class="px-5 py-3 font-semibold">Name</th>
                <th class="px-5 py-3 font-semibold">Prefix</th>
                <th class="px-5 py-3 font-semibold">Scope</th>
                <th class="px-5 py-3 font-semibold">Created</th>
                <th class="px-5 py-3 font-semibold">Expires</th>
                <th class="px-5 py-3 font-semibold text-right">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-800/60">
              {#each tokens as t (t.id)}
                <tr class="hover:bg-slate-800/40 transition-colors">
                  <td class="px-5 py-3.5 font-medium text-slate-100">
                    {t.name}
                  </td>
                  <td class="px-5 py-3.5 font-mono text-xs text-sky-400">
                    {t.prefix || 'hyd_'}…
                  </td>
                  <td class="px-5 py-3.5">
                    <Badge variant="default" size="sm">{t.scope || 'admin'}</Badge>
                  </td>
                  <td class="px-5 py-3.5 text-xs text-slate-500">
                    {new Date(t.created_at).toLocaleDateString()}
                  </td>
                  <td class="px-5 py-3.5 text-xs text-slate-500">
                    {t.expires_at ? new Date(t.expires_at).toLocaleDateString() : 'Never'}
                  </td>
                  <td class="px-5 py-3.5 text-right whitespace-nowrap">
                    <button
                      type="button"
                      class="px-2.5 py-1 text-xs text-rose-400 hover:text-rose-300 rounded hover:bg-rose-950/40 transition-colors cursor-pointer"
                      onclick={() => handleRevokeToken(t.id)}
                    >
                      Revoke
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>
  {/if}
</div>
