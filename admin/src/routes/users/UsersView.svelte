<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { auth } from '$lib/stores/auth.svelte';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';

  interface StaffUser {
    id: string;
    email: string;
    role: string;
    active: boolean;
    permissions?: string[];
    last_login_at?: string;
  }

  interface APIKey {
    id: string;
    name: string;
    prefix: string;
    scope?: string;
    created_at: string;
    expires_at?: string;
    last_used_at?: string;
  }

  let users = $state<StaffUser[]>([]);
  let apiKeys = $state<APIKey[]>([]);
  let loading = $state(true);
  let activeTab = $state<'users' | 'apikeys'>('users');
  let statusMsg = $state<string | null>(null);
  let statusErr = $state<string | null>(null);

  // Invite Modal
  let inviteOpen = $state(false);
  let inviteEmail = $state('');
  let inviteRole = $state('editor');
  let inviting = $state(false);

  // Change Role Modal
  let roleModalUser = $state<StaffUser | null>(null);
  let selectedRole = $state('editor');
  let savingRole = $state(false);

  // Granular Permissions Modal
  let permModalUser = $state<StaffUser | null>(null);
  let userPerms = $state<Record<string, boolean>>({});
  let savingPerms = $state(false);

  // Password Reset Link Modal
  let resetLinkModalUser = $state<StaffUser | null>(null);
  let generatedResetLink = $state<string | null>(null);
  let generatingResetLink = $state(false);
  let copiedResetLink = $state(false);

  // Disable / Enable User Dialog
  let toggleActiveTarget = $state<StaffUser | null>(null);
  let togglingActive = $state(false);

  // New API Key Modal
  let keyModalOpen = $state(false);
  let newKeyName = $state('');
  let creatingKey = $state(false);
  let issuedKeyValue = $state<string | null>(null);
  let copiedKey = $state(false);

  // Revoke Key Dialog
  let deleteKeyTarget = $state<APIKey | null>(null);
  let revokingKey = $state(false);

  const availablePermissions = [
    { id: 'content.write', label: 'Content Management', desc: 'Create, edit, publish and delete articles and pages' },
    { id: 'media.upload', label: 'Media Library Access', desc: 'Upload, delete, and organize files in media storage' },
    { id: 'site.manage', label: 'Site & Engine Configuration', desc: 'Modify global site settings, maintenance mode and redirects' },
    { id: 'users.manage', label: 'User & Staff Administration', desc: 'Invite staff members, assign roles, and alter permissions' },
    { id: 'telemetry.read', label: 'Diagnostics & Telemetry', desc: 'View runtime engine metrics, memory heap, and health' },
  ];

  function notify(msg: string, isErr = false) {
    if (isErr) {
      statusErr = msg;
      statusMsg = null;
    } else {
      statusMsg = msg;
      statusErr = null;
    }
    setTimeout(() => {
      statusMsg = null;
      statusErr = null;
    }, 3500);
  }

  async function loadUsers() {
    try {
      const res = await api.get<ApiResponse<StaffUser[]>>('/api/v1/admin/users');
      if (res?.data && Array.isArray(res.data)) {
        users = res.data;
      }
    } catch (e) {
      console.warn('Load users failed:', e);
    }
  }

  async function loadAPIKeys() {
    try {
      const res = await api.get<ApiResponse<APIKey[]>>('/api/v1/admin/api-keys');
      if (res?.data && Array.isArray(res.data)) {
        apiKeys = res.data;
      }
    } catch (e) {
      console.warn('Load api-keys failed:', e);
    }
  }

  async function handleInvite(e: SubmitEvent) {
    e.preventDefault();
    if (!inviteEmail.trim()) return;
    inviting = true;
    try {
      await api.post('/api/v1/admin/users/invites', { email: inviteEmail.trim(), role: inviteRole });
      notify(`Invitation dispatched to ${inviteEmail}.`);
      inviteEmail = '';
      inviteOpen = false;
      await loadUsers();
    } catch (err: any) {
      notify(err.message || 'Failed to dispatch invitation', true);
    } finally {
      inviting = false;
    }
  }

  async function handleSaveRole(e: SubmitEvent) {
    e.preventDefault();
    if (!roleModalUser) return;
    savingRole = true;
    try {
      const form = new URLSearchParams();
      form.append('role', selectedRole);
      await api.post(`/api/v1/admin/users/${roleModalUser.id}/role`, form.toString(), {
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      });
      notify(`Role updated to ${selectedRole} for ${roleModalUser.email}`);
      roleModalUser = null;
      await loadUsers();
    } catch (err: any) {
      notify(err.message || 'Failed to update user role', true);
    } finally {
      savingRole = false;
    }
  }

  async function handleSavePermissions(e: SubmitEvent) {
    e.preventDefault();
    if (!permModalUser) return;
    savingPerms = true;
    try {
      const form = new URLSearchParams();
      for (const [key, allowed] of Object.entries(userPerms)) {
        form.append(`perm_${key}`, allowed ? 'allow' : 'deny');
      }
      await api.post(`/api/v1/admin/users/${permModalUser.id}/permissions`, form.toString(), {
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      });
      notify(`Custom permissions saved for ${permModalUser.email}`);
      permModalUser = null;
      await loadUsers();
    } catch (err: any) {
      notify(err.message || 'Failed to save permissions', true);
    } finally {
      savingPerms = false;
    }
  }

  async function handleGenerateResetLink(user: StaffUser) {
    resetLinkModalUser = user;
    generatedResetLink = null;
    generatingResetLink = true;
    try {
      const res = await api.post<{ url?: string; link?: string }>(`/api/v1/admin/users/${user.id}/password-reset-link`, {});
      generatedResetLink = res?.url || res?.link || 'Reset link generated successfully.';
    } catch (err: any) {
      notify(err.message || 'Failed to create password reset link', true);
      resetLinkModalUser = null;
    } finally {
      generatingResetLink = false;
    }
  }

  async function handleToggleUserActive() {
    if (!toggleActiveTarget) return;
    togglingActive = true;
    try {
      await api.post(`/api/v1/admin/users/${toggleActiveTarget.id}/disable`, {});
      notify(`Account state toggled for ${toggleActiveTarget.email}`);
      toggleActiveTarget = null;
      await loadUsers();
    } catch (err: any) {
      notify(err.message || 'Failed to toggle account state', true);
    } finally {
      togglingActive = false;
    }
  }

  async function handleCreateAPIKey(e: SubmitEvent) {
    e.preventDefault();
    if (!newKeyName.trim()) return;
    creatingKey = true;
    try {
      const res = await api.post<ApiResponse<{ token?: string; key?: string }>>('/api/v1/admin/api-keys', {
        name: newKeyName.trim(),
      });
      issuedKeyValue = res?.data?.key || res?.data?.token || 'Key generated';
      await loadAPIKeys();
    } catch (err: any) {
      notify(err.message || 'Failed to issue API key', true);
    } finally {
      creatingKey = false;
    }
  }

  async function handleRevokeAPIKey() {
    if (!deleteKeyTarget) return;
    revokingKey = true;
    try {
      await api.delete(`/api/v1/admin/api-keys/${deleteKeyTarget.id}`);
      deleteKeyTarget = null;
      notify('System API key revoked.');
      await loadAPIKeys();
    } catch (err: any) {
      notify(err.message || 'Failed to revoke API key', true);
    } finally {
      revokingKey = false;
    }
  }

  onMount(async () => {
    loading = true;
    try {
      await Promise.allSettled([loadUsers(), loadAPIKeys()]);
    } finally {
      loading = false;
    }
  });
</script>

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-4 p-4 lg:p-6 max-w-5xl mx-auto w-full">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-border pb-4">
      <div class="flex items-center gap-3">
        <div>
          <h2 class="text-xl font-semibold tracking-tight text-foreground">Staff & Access Control</h2>
          <p class="text-xs text-muted-foreground mt-0.5">Manage administrative team members, roles, granular permissions, and system API keys.</p>
        </div>
        {#if loading}
          <svg class="size-4 animate-spin text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12a9 9 0 1 1-6.219-8.56"></path>
          </svg>
        {/if}
      </div>

      {#if statusMsg}
        <div class="p-2.5 px-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs flex items-center gap-1.5 animate-in fade-in-0">
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"></polyline></svg>
          <span>{statusMsg}</span>
        </div>
      {/if}
      {#if statusErr}
        <div class="p-2.5 px-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-center gap-1.5 animate-in fade-in-0">
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"></line></svg>
          <span>{statusErr}</span>
        </div>
      {/if}
    </div>

    <!-- Tabs -->
    <div class="flex items-center justify-between border-b border-border overflow-x-auto pb-px">
      <div class="flex items-center gap-1">
        <button
          type="button"
          onclick={() => (activeTab = 'users')}
          class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'users' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
        >
          Staff Users ({users.length})
        </button>
        <button
          type="button"
          onclick={() => (activeTab = 'apikeys')}
          class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'apikeys' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
        >
          System API Keys ({apiKeys.length})
        </button>
      </div>

      {#if activeTab === 'users'}
        <Button onclick={() => (inviteOpen = true)}>Invite Staff Member</Button>
      {:else}
        <Button onclick={() => { keyModalOpen = true; newKeyName = ''; issuedKeyValue = null; }}>
          Generate API Key
        </Button>
      {/if}
    </div>

    <!-- Content: Tab 1 - Users -->
    {#if activeTab === 'users'}
      <div class="rounded-xl border border-border bg-card overflow-hidden shadow-xs">
        <div class="divide-y divide-border">
          {#if users.length === 0}
            <div class="p-8 text-center text-xs text-muted-foreground">No staff users found.</div>
          {:else}
            {#each users as u}
              <div class="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4 hover:bg-muted/30 transition-colors">
                <div class="flex items-center gap-3 min-w-0">
                  <div class="size-10 rounded-full bg-primary/10 border border-primary/20 flex items-center justify-center text-primary font-bold text-sm uppercase shrink-0">
                    {u.email?.[0] || 'U'}
                  </div>
                  <div class="grid text-left leading-tight min-w-0">
                    <div class="flex items-center gap-2">
                      <span class="text-sm font-semibold text-foreground truncate">{u.email}</span>
                      <Badge variant="outline" class="text-[10px] capitalize font-mono">
                        {u.role}
                      </Badge>
                      {#if !u.active}
                        <Badge variant="danger" class="text-[10px]">Disabled</Badge>
                      {:else}
                        <span class="size-1.5 rounded-full bg-emerald-500 inline-block" title="Active"></span>
                      {/if}
                    </div>
                    <span class="text-[11px] text-muted-foreground mt-0.5 truncate">
                      ID: <code class="font-mono">{u.id}</code> {u.last_login_at ? `• Last login: ${new Date(u.last_login_at).toLocaleDateString()}` : ''}
                    </span>
                  </div>
                </div>

                <!-- User Actions -->
                <div class="flex items-center gap-1.5 flex-wrap">
                  {#if u.id !== auth.user?.id}
                    <Button
                      variant="outline"
                      onclick={() => { roleModalUser = u; selectedRole = u.role; }}
                      title="Change user role"
                    >
                      Role
                    </Button>
                    <Button
                      variant="outline"
                      onclick={() => {
                        permModalUser = u;
                        userPerms = {};
                        if (Array.isArray(u.permissions)) {
                          for (const p of u.permissions) userPerms[p] = true;
                        }
                      }}
                      title="Configure permissions"
                    >
                      Permissions
                    </Button>
                    <Button
                      variant="outline"
                      onclick={() => handleGenerateResetLink(u)}
                      title="Issue one-time password reset link"
                    >
                      Reset Link
                    </Button>
                    <Button
                      variant="danger"
                      onclick={() => (toggleActiveTarget = u)}
                      title={u.active ? 'Disable account' : 'Reactivate account'}
                    >
                      {u.active ? 'Disable' : 'Enable'}
                    </Button>
                  {:else}
                    <Badge variant="outline" class="text-xs text-muted-foreground">Current Account</Badge>
                  {/if}
                </div>
              </div>
            {/each}
          {/if}
        </div>
      </div>

    <!-- Content: Tab 2 - System API Keys -->
    {:else if activeTab === 'apikeys'}
      <div class="rounded-xl border border-border bg-card overflow-hidden shadow-xs">
        <div class="divide-y divide-border">
          {#if apiKeys.length === 0}
            <div class="p-8 text-center text-xs text-muted-foreground">No system API keys active. Create one to authenticate external CLI tools or CI workflows.</div>
          {:else}
            {#each apiKeys as key}
              <div class="p-4 flex items-center justify-between gap-4 hover:bg-muted/30 transition-colors">
                <div class="grid text-left min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="text-xs font-semibold text-foreground">{key.name}</span>
                    <code class="text-[10px] px-1.5 py-0.5 rounded bg-muted border border-border font-mono">{key.prefix}...</code>
                  </div>
                  <span class="text-[11px] text-muted-foreground mt-0.5">Created: {new Date(key.created_at).toLocaleDateString()} {key.expires_at ? `• Expires: ${new Date(key.expires_at).toLocaleDateString()}` : ''}</span>
                </div>
                <Button variant="danger" onclick={() => (deleteKeyTarget = key)}>Revoke</Button>
              </div>
            {/each}
          {/if}
        </div>
      </div>
    {/if}
  </div>
</div>

<!-- Modal: Invite Staff -->
<Modal bind:open={inviteOpen} title="Invite Staff Member" onclose={() => (inviteOpen = false)}>
  <form onsubmit={handleInvite} class="space-y-4">
    <div>
      <label for="inv-email" class="block text-xs font-medium text-foreground mb-1">Email Address</label>
      <Input id="inv-email" type="email" bind:value={inviteEmail} placeholder="colleague@example.com" required />
    </div>
    <div>
      <label for="inv-role" class="block text-xs font-medium text-foreground mb-1">Initial Role</label>
      <select
        id="inv-role"
        bind:value={inviteRole}
        class="w-full h-9 rounded-md border border-input bg-background px-3 text-xs text-foreground focus:outline-hidden focus:ring-1 focus:ring-primary"
      >
        <option value="editor">Editor (Create, edit, publish content)</option>
        <option value="author">Author (Create and edit own content)</option>
        <option value="viewer">Viewer (Read-only studio access)</option>
        <option value="admin">Administrator (Full engine control)</option>
      </select>
    </div>
    <div class="flex justify-end gap-2 pt-2">
      <Button type="button" variant="outline" onclick={() => (inviteOpen = false)}>Cancel</Button>
      <Button type="submit" disabled={inviting}>{inviting ? 'Dispatching...' : 'Send Invitation'}</Button>
    </div>
  </form>
</Modal>

<!-- Modal: Change Role -->
<Modal open={!!roleModalUser} title="Change Staff Role" onclose={() => (roleModalUser = null)}>
  {#if roleModalUser}
    <form onsubmit={handleSaveRole} class="space-y-4">
      <p class="text-xs text-muted-foreground">Updating role for <span class="font-semibold text-foreground">{roleModalUser.email}</span></p>
      <div>
        <label for="ch-role" class="block text-xs font-medium text-foreground mb-1">Select Role</label>
        <select
          id="ch-role"
          bind:value={selectedRole}
          class="w-full h-9 rounded-md border border-input bg-background px-3 text-xs text-foreground focus:outline-hidden focus:ring-1 focus:ring-primary"
        >
          <option value="editor">Editor</option>
          <option value="author">Author</option>
          <option value="viewer">Viewer</option>
          <option value="admin">Administrator</option>
        </select>
      </div>
      <div class="flex justify-end gap-2 pt-2">
        <Button type="button" variant="outline" onclick={() => (roleModalUser = null)}>Cancel</Button>
        <Button type="submit" disabled={savingRole}>Save Role</Button>
      </div>
    </form>
  {/if}
</Modal>

<!-- Modal: Custom Permissions -->
<Modal open={!!permModalUser} title="Granular Staff Permissions" onclose={() => (permModalUser = null)}>
  {#if permModalUser}
    <form onsubmit={handleSavePermissions} class="space-y-4">
      <p class="text-xs text-muted-foreground">Customize explicit permission overrides for <span class="font-semibold text-foreground">{permModalUser.email}</span>.</p>
      <div class="divide-y divide-border border border-border rounded-lg overflow-hidden">
        {#each availablePermissions as perm}
          <div class="p-3 flex items-center justify-between gap-3 bg-card hover:bg-muted/20">
            <div>
              <span class="text-xs font-semibold text-foreground block">{perm.label}</span>
              <span class="text-[11px] text-muted-foreground block">{perm.desc}</span>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={userPerms[perm.id] ?? false}
                onchange={(e) => { userPerms[perm.id] = (e.target as HTMLInputElement).checked; }}
                class="sr-only peer"
              />
              <div class="w-9 h-5 bg-muted peer-focus:outline-hidden rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-primary"></div>
            </label>
          </div>
        {/each}
      </div>
      <div class="flex justify-end gap-2 pt-2">
        <Button type="button" variant="outline" onclick={() => (permModalUser = null)}>Cancel</Button>
        <Button type="submit" disabled={savingPerms}>Save Overrides</Button>
      </div>
    </form>
  {/if}
</Modal>

<!-- Modal: Password Reset Link -->
<Modal open={!!resetLinkModalUser} title="One-Time Password Reset Link" onclose={() => (resetLinkModalUser = null)}>
  <div class="space-y-4">
    <p class="text-xs text-muted-foreground">Share this temporary password reset link directly with <span class="font-semibold text-foreground">{resetLinkModalUser?.email}</span>. It can be used only once.</p>
    {#if generatingResetLink}
      <div class="p-4 text-center text-xs text-muted-foreground animate-pulse">Generating secure reset token...</div>
    {:else if generatedResetLink}
      <div class="flex items-center gap-2">
        <Input type="text" readonly value={generatedResetLink} class="font-mono text-xs select-all" />
        <Button
          variant="outline"
          onclick={() => {
            navigator.clipboard.writeText(generatedResetLink || '');
            copiedResetLink = true;
            setTimeout(() => (copiedResetLink = false), 2000);
          }}
        >
          {copiedResetLink ? 'Copied!' : 'Copy'}
        </Button>
      </div>
    {/if}
    <div class="flex justify-end pt-2">
      <Button onclick={() => (resetLinkModalUser = null)}>Done</Button>
    </div>
  </div>
</Modal>

<!-- Modal: New API Key -->
<Modal bind:open={keyModalOpen} title="Generate System API Key" onclose={() => (keyModalOpen = false)}>
  {#if issuedKeyValue}
    <div class="space-y-4">
      <div class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs">
        Copy your system API key now. It will not be shown again!
      </div>
      <div class="flex items-center gap-2">
        <Input type="text" readonly value={issuedKeyValue} class="font-mono text-xs select-all" />
        <Button
          variant="outline"
          onclick={() => {
            navigator.clipboard.writeText(issuedKeyValue || '');
            copiedKey = true;
            setTimeout(() => (copiedKey = false), 2000);
          }}
        >
          {copiedKey ? 'Copied!' : 'Copy'}
        </Button>
      </div>
      <div class="flex justify-end pt-2">
        <Button onclick={() => (keyModalOpen = false)}>Done</Button>
      </div>
    </div>
  {:else}
    <form onsubmit={handleCreateAPIKey} class="space-y-4">
      <div>
        <label for="ak-name" class="block text-xs font-medium text-foreground mb-1">Key Name / Service</label>
        <Input id="ak-name" type="text" bind:value={newKeyName} placeholder="e.g. GitHub Actions, Nightly Backup" required />
      </div>
      <div class="flex justify-end gap-2 pt-2">
        <Button type="button" variant="outline" onclick={() => (keyModalOpen = false)}>Cancel</Button>
        <Button type="submit" disabled={creatingKey}>Generate Key</Button>
      </div>
    </form>
  {/if}
</Modal>

<!-- Modal: Revoke API Key Confirm -->
<ConfirmDialog
  open={!!deleteKeyTarget}
  title="Revoke System API Key"
  description={`Are you sure you want to revoke the API key '${deleteKeyTarget?.name}'?`}
  confirmLabel="Revoke Key"
  loading={revokingKey}
  onconfirm={handleRevokeAPIKey}
  oncancel={() => (deleteKeyTarget = null)}
/>

<!-- Modal: Toggle User Active Confirm -->
<ConfirmDialog
  open={!!toggleActiveTarget}
  title={toggleActiveTarget?.active ? 'Disable Staff Member' : 'Reactivate Staff Member'}
  description={toggleActiveTarget?.active
    ? `Are you sure you want to disable ${toggleActiveTarget?.email}? They will immediately lose access to the Admin Studio.`
    : `Are you sure you want to reactivate ${toggleActiveTarget?.email}?`}
  confirmLabel={toggleActiveTarget?.active ? 'Disable' : 'Activate'}
  loading={togglingActive}
  onconfirm={handleToggleUserActive}
  oncancel={() => (toggleActiveTarget = null)}
/>
