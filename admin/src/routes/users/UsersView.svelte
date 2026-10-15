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
      <label for="invite-role-select" class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Role</label>
      <select
        id="invite-role-select"
        bind:value={inviteRole}
        class="w-full mt-1 rounded-md bg-background border border-input px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
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
      <div class="p-3.5 rounded-lg bg-muted border border-border text-foreground text-xs font-mono break-all select-all">
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

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-2">
    <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
      <!-- Toolbar (exact shadcn Tabs + Action CTA) -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 px-4 lg:px-6">
        <!-- Segmented Navigation Tabs -->
        <div class="inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground">
          <button
            type="button"
            class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {activeTab === 'users' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
            onclick={() => { activeTab = 'users'; loadUsers(); }}
          >
            Staff Directory ({users.length})
          </button>
          <button
            type="button"
            class="inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium ring-offset-background transition-all cursor-pointer {activeTab === 'tokens' ? 'bg-background text-foreground shadow-xs' : 'hover:bg-background/50 hover:text-foreground'}"
            onclick={() => { activeTab = 'tokens'; loadTokens(); }}
          >
            Personal Access Tokens
          </button>
        </div>

        <!-- Action CTAs -->
        <div class="flex items-center gap-2">
          {#if activeTab === 'users'}
            <button
              type="button"
              class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs cursor-pointer shrink-0 transition-colors"
              onclick={() => { inviteOpen = true; }}
            >
              <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="12" y1="5" x2="12" y2="19"></line>
                <line x1="5" y1="12" x2="19" y2="12"></line>
              </svg>
              <span>Invite Staff</span>
            </button>
          {:else}
            <button
              type="button"
              class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs cursor-pointer shrink-0 transition-colors"
              onclick={() => { tokenModalOpen = true; }}
            >
              <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="12" y1="5" x2="12" y2="19"></line>
                <line x1="5" y1="12" x2="19" y2="12"></line>
              </svg>
              <span>Generate Token</span>
            </button>
          {/if}
        </div>
      </div>

      <!-- Table Container -->
      <div class="px-4 lg:px-6">
        {#if activeTab === 'users'}
          <div class="overflow-hidden rounded-xl border border-border bg-card shadow-xs">
            {#if loadingUsers}
              <div class="p-12 text-center text-xs text-muted-foreground">
                Loading staff directory…
              </div>
            {:else}
              <div class="overflow-x-auto">
                <table class="w-full text-left text-sm text-foreground caption-bottom">
                  <thead class="sticky top-0 z-10 bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
                    <tr>
                      <th class="h-10 px-4 font-medium">User</th>
                      <th class="h-10 px-4 font-medium">Role</th>
                      <th class="h-10 px-4 font-medium">Status</th>
                      <th class="h-10 px-4 font-medium">Permissions</th>
                      <th class="h-10 px-4 font-medium">Last Login</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-border">
                    {#each users as u (u.id)}
                      <tr class="hover:bg-muted/50 transition-colors">
                        <td class="px-4 py-3">
                          <p class="font-medium text-foreground">{u.email}</p>
                          <span class="text-[11px] text-muted-foreground font-mono">ID: {u.id}</span>
                        </td>
                        <td class="px-4 py-3">
                          <Badge variant={u.role === 'admin' ? 'default' : 'outline'} size="sm">
                            {u.role}
                          </Badge>
                        </td>
                        <td class="px-4 py-3">
                          <Badge variant={u.active ? 'success' : 'destructive'} size="sm">
                            {u.active ? 'Active' : 'Disabled'}
                          </Badge>
                        </td>
                        <td class="px-4 py-3 text-xs text-muted-foreground">
                          {u.permissions?.length || 0} granted
                        </td>
                        <td class="px-4 py-3 text-xs text-muted-foreground">
                          {u.last_login_at ? new Date(u.last_login_at).toLocaleString() : 'Never'}
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>

              <!-- Footer Stats -->
              <div class="flex items-center justify-between px-4 py-3 border-t border-border text-xs text-muted-foreground">
                <span>{users.length} staff members total</span>
              </div>
            {/if}
          </div>
        {:else}
          <div class="overflow-hidden rounded-xl border border-border bg-card shadow-xs">
            {#if loadingTokens}
              <div class="p-12 text-center text-xs text-muted-foreground">
                Loading access tokens…
              </div>
            {:else if tokens.length === 0}
              <div class="p-12 text-center text-xs text-muted-foreground flex flex-col items-center justify-center gap-3">
                <p>No active personal access tokens.</p>
                <button
                  type="button"
                  class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors cursor-pointer"
                  onclick={() => { tokenModalOpen = true; }}
                >
                  Generate First Token
                </button>
              </div>
            {:else}
              <div class="overflow-x-auto">
                <table class="w-full text-left text-sm text-foreground caption-bottom">
                  <thead class="sticky top-0 z-10 bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
                    <tr>
                      <th class="h-10 px-4 font-medium">Name</th>
                      <th class="h-10 px-4 font-medium">Prefix</th>
                      <th class="h-10 px-4 font-medium">Scope</th>
                      <th class="h-10 px-4 font-medium">Created</th>
                      <th class="h-10 px-4 font-medium">Expires</th>
                      <th class="h-10 w-24 px-4 font-medium text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-border">
                    {#each tokens as t (t.id)}
                      <tr class="hover:bg-muted/50 transition-colors">
                        <td class="px-4 py-3 font-medium text-foreground">
                          {t.name}
                        </td>
                        <td class="px-4 py-3 font-mono text-xs text-muted-foreground">
                          {t.prefix || 'hyd_'}…
                        </td>
                        <td class="px-4 py-3">
                          <Badge variant="outline" size="sm">{t.scope || 'admin'}</Badge>
                        </td>
                        <td class="px-4 py-3 text-xs text-muted-foreground">
                          {new Date(t.created_at).toLocaleDateString()}
                        </td>
                        <td class="px-4 py-3 text-xs text-muted-foreground">
                          {t.expires_at ? new Date(t.expires_at).toLocaleDateString() : 'Never'}
                        </td>
                        <td class="px-4 py-3 text-right">
                          <button
                            type="button"
                            class="px-2.5 py-1 text-xs text-destructive hover:bg-destructive/10 rounded transition-colors cursor-pointer"
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

              <!-- Footer Stats -->
              <div class="flex items-center justify-between px-4 py-3 border-t border-border text-xs text-muted-foreground">
                <span>{tokens.length} active tokens</span>
              </div>
            {/if}
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>
