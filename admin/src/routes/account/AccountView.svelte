<script lang="ts">
  import { onMount } from 'svelte';
  import { auth } from '$lib/stores/auth.svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';

  interface SessionItem {
    id: string;
    ip_address?: string;
    user_agent?: string;
    current?: boolean;
    created_at?: string;
    last_active_at?: string;
  }

  interface PersonalToken {
    id: string;
    name: string;
    prefix: string;
    created_at: string;
    expires_at?: string;
    last_used_at?: string;
  }

  type ActiveTab = 'profile' | 'security' | 'mfa' | 'sessions' | 'tokens' | 'export';
  let activeTab = $state<ActiveTab>('profile');

  // Password change state
  let currentPassword = $state('');
  let newPassword = $state('');
  let confirmPassword = $state('');
  let passwordLoading = $state(false);
  let passwordSuccess = $state<string | null>(null);
  let passwordError = $state<string | null>(null);

  // MFA state
  let mfaEnrolling = $state(false);
  let mfaSecret = $state<string | null>(null);
  let mfaQrData = $state<string | null>(null);
  let mfaCode = $state('');
  let mfaDisablePassword = $state('');
  let mfaDisableCode = $state('');
  let mfaLoading = $state(false);
  let mfaError = $state<string | null>(null);
  let mfaSuccess = $state<string | null>(null);

  // Sessions state
  let sessions = $state<SessionItem[]>([]);
  let loadingSessions = $state(false);
  let revokingSessions = $state(false);
  let revokeSuccess = $state<string | null>(null);

  // Tokens state
  let tokens = $state<PersonalToken[]>([]);
  let loadingTokens = $state(false);
  let tokenModalOpen = $state(false);
  let newTokenName = $state('');
  let newTokenDays = $state(30);
  let creatingToken = $state(false);
  let tokenError = $state<string | null>(null);
  let issuedTokenValue = $state<string | null>(null);
  let copiedToken = $state(false);

  // Token revoke confirm
  let deleteTokenTarget = $state<PersonalToken | null>(null);
  let revokingToken = $state(false);

  // Data export state
  let exporting = $state(false);

  async function handlePasswordChange(e: SubmitEvent) {
    e.preventDefault();
    passwordError = null;
    passwordSuccess = null;

    if (newPassword !== confirmPassword) {
      passwordError = 'New passwords do not match.';
      return;
    }
    if (newPassword.length < 8) {
      passwordError = 'New password must be at least 8 characters long.';
      return;
    }

    passwordLoading = true;
    try {
      await api.post('/auth/me/credentials', {
        current_password: currentPassword,
        new_password: newPassword,
      });
      passwordSuccess = 'Password updated successfully.';
      currentPassword = '';
      newPassword = '';
      confirmPassword = '';
    } catch (err: any) {
      passwordError = err.message || 'Failed to update password. Verify your current password.';
    } finally {
      passwordLoading = false;
    }
  }

  async function handleStartMFAEnroll() {
    mfaLoading = true;
    mfaError = null;
    mfaSuccess = null;
    try {
      const res = await api.post<ApiResponse<{ secret: string; qr_svg?: string; url?: string }>>('/api/v1/admin/account/mfa/enroll', {});
      if (res?.data) {
        mfaSecret = res.data.secret;
        mfaQrData = res.data.qr_svg || res.data.url || null;
        mfaEnrolling = true;
      }
    } catch (err: any) {
      mfaError = err.message || 'Failed to initiate 2FA enrollment.';
    } finally {
      mfaLoading = false;
    }
  }

  async function handleConfirmMFAEnable(e: SubmitEvent) {
    e.preventDefault();
    if (!mfaCode.trim()) return;
    mfaLoading = true;
    mfaError = null;
    try {
      await api.post('/api/v1/admin/account/mfa/enable', {
        code: mfaCode.trim(),
      });
      mfaSuccess = 'Two-factor authentication has been successfully enabled!';
      mfaEnrolling = false;
      mfaSecret = null;
      mfaCode = '';
      await auth.checkAuth();
    } catch (err: any) {
      mfaError = err.message || 'Invalid verification code. Please check your authenticator app.';
    } finally {
      mfaLoading = false;
    }
  }

  async function handleDisableMFA(e: SubmitEvent) {
    e.preventDefault();
    if (!mfaDisablePassword || !mfaDisableCode) return;
    mfaLoading = true;
    mfaError = null;
    mfaSuccess = null;
    try {
      await api.post('/api/v1/admin/account/mfa/disable', {
        password: mfaDisablePassword,
        code: mfaDisableCode,
      });
      mfaSuccess = 'Two-factor authentication has been disabled.';
      mfaDisablePassword = '';
      mfaDisableCode = '';
      await auth.checkAuth();
    } catch (err: any) {
      mfaError = err.message || 'Failed to disable 2FA. Verify your password and 6-digit code.';
    } finally {
      mfaLoading = false;
    }
  }

  async function loadSessions() {
    loadingSessions = true;
    try {
      const res = await api.get<ApiResponse<SessionItem[]>>('/auth/me/sessions');
      if (res?.data && Array.isArray(res.data)) {
        sessions = res.data;
      } else {
        sessions = [];
      }
    } catch (e) {
      console.warn('Load sessions failed:', e);
      sessions = [];
    } finally {
      loadingSessions = false;
    }
  }

  async function handleRevokeOtherSessions() {
    revokingSessions = true;
    revokeSuccess = null;
    try {
      await api.post('/auth/me/sessions/revoke-others', {});
      revokeSuccess = 'All other active sessions have been signed out.';
      await loadSessions();
    } catch (e: any) {
      console.warn('Revoke sessions failed:', e);
    } finally {
      revokingSessions = false;
    }
  }

  async function loadTokens() {
    loadingTokens = true;
    try {
      const res = await api.get<ApiResponse<PersonalToken[]>>('/api/v1/admin/account/tokens');
      if (res?.data && Array.isArray(res.data)) {
        tokens = res.data;
      } else {
        tokens = [];
      }
    } catch (e) {
      console.warn('Load tokens failed:', e);
      tokens = [];
    } finally {
      loadingTokens = false;
    }
  }

  async function handleCreateToken(e: SubmitEvent) {
    e.preventDefault();
    if (!newTokenName.trim()) return;
    creatingToken = true;
    tokenError = null;
    try {
      const res = await api.post<ApiResponse<{ token: string; key?: string }>>('/api/v1/admin/account/tokens', {
        name: newTokenName.trim(),
        days: Number(newTokenDays) || 30,
      });
      issuedTokenValue = res?.data?.token || res?.data?.key || null;
      await loadTokens();
    } catch (err: any) {
      tokenError = err.message || 'Failed to issue personal token.';
    } finally {
      creatingToken = false;
    }
  }

  async function handleRevokeToken() {
    if (!deleteTokenTarget) return;
    revokingToken = true;
    try {
      await api.delete(`/api/v1/admin/account/tokens/${deleteTokenTarget.id}`);
      deleteTokenTarget = null;
      await loadTokens();
    } catch (e) {
      console.warn('Revoke token failed:', e);
    } finally {
      revokingToken = false;
    }
  }

  async function handleExportUserData() {
    exporting = true;
    try {
      const res = await api.get<any>('/auth/me/export');
      const blob = new Blob([JSON.stringify(res, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `user-export-${auth.user?.id || 'me'}.json`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      console.error('Data export failed:', e);
    } finally {
      exporting = false;
    }
  }

  onMount(() => {
    loadSessions();
    loadTokens();
  });
</script>

<div class="flex flex-1 flex-col">
  <div class="@container/main flex flex-1 flex-col gap-4 p-4 lg:p-6 max-w-5xl mx-auto w-full">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-border pb-4">
      <div>
        <h2 class="text-xl font-semibold tracking-tight text-foreground">Account & Security</h2>
        <p class="text-xs text-muted-foreground mt-0.5">Manage your credentials, 2FA, sessions, and personal access tokens.</p>
      </div>

      <div class="flex items-center gap-2">
        <Badge variant="outline" class="font-mono text-xs uppercase">
          Role: {auth.user?.role || 'admin'}
        </Badge>
        <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
          <span class="size-1.5 rounded-full bg-emerald-500"></span> Active Staff
        </span>
      </div>
    </div>

    <!-- Navigation Tabs -->
    <div class="flex items-center gap-1 border-b border-border overflow-x-auto pb-px">
      <button
        type="button"
        onclick={() => (activeTab = 'profile')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'profile' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Profile
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'security')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'security' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Password
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'mfa')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'mfa' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Two-Factor Auth
      </button>
      <button
        type="button"
        onclick={() => { activeTab = 'sessions'; loadSessions(); }}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'sessions' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Active Sessions ({sessions.length})
      </button>
      <button
        type="button"
        onclick={() => { activeTab = 'tokens'; loadTokens(); }}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'tokens' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Personal Tokens ({tokens.length})
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'export')}
        class="px-3.5 py-2 text-xs font-medium border-b-2 transition-colors cursor-pointer {activeTab === 'export' ? 'border-primary text-foreground font-semibold' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        Data Export
      </button>
    </div>

    <!-- Tab 1: Profile -->
    {#if activeTab === 'profile'}
      <div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-6">
        <div class="flex items-center gap-4">
          <div class="size-16 rounded-full bg-primary/10 border border-primary/20 flex items-center justify-center text-primary font-bold text-2xl uppercase">
            {auth.user?.email?.[0] || 'U'}
          </div>
          <div>
            <h3 class="text-base font-semibold text-foreground">{auth.user?.email}</h3>
            <p class="text-xs text-muted-foreground">ID: <code class="font-mono">{auth.user?.id || 'default'}</code></p>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4 pt-4 border-t border-border/60">
          <div>
            <span class="block text-xs font-medium text-muted-foreground mb-1">Email Address</span>
            <div class="p-2.5 rounded-lg border border-border bg-muted/30 text-xs font-mono text-foreground">
              {auth.user?.email}
            </div>
          </div>

          <div>
            <span class="block text-xs font-medium text-muted-foreground mb-1">Assigned Role</span>
            <div class="p-2.5 rounded-lg border border-border bg-muted/30 text-xs text-foreground capitalize">
              {auth.user?.role || 'admin'}
            </div>
          </div>
        </div>
      </div>

    <!-- Tab 2: Security & Password -->
    {:else if activeTab === 'security'}
      <div class="rounded-xl border border-border bg-card p-6 shadow-xs max-w-xl space-y-4">
        <div>
          <h3 class="text-sm font-semibold text-foreground">Change Password</h3>
          <p class="text-xs text-muted-foreground mt-0.5">Ensure your account uses a strong, unique password.</p>
        </div>

        {#if passwordSuccess}
          <div class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs">
            {passwordSuccess}
          </div>
        {/if}

        {#if passwordError}
          <div class="p-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs">
            {passwordError}
          </div>
        {/if}

        <form onsubmit={handlePasswordChange} class="space-y-4 pt-2">
          <div>
            <label for="current-pwd" class="block text-xs font-medium text-foreground mb-1">Current Password</label>
            <Input id="current-pwd" type="password" bind:value={currentPassword} required placeholder="••••••••••••" />
          </div>

          <div>
            <label for="new-pwd" class="block text-xs font-medium text-foreground mb-1">New Password</label>
            <Input id="new-pwd" type="password" bind:value={newPassword} required minlength={8} placeholder="At least 8 characters" />
          </div>

          <div>
            <label for="confirm-pwd" class="block text-xs font-medium text-foreground mb-1">Confirm New Password</label>
            <Input id="confirm-pwd" type="password" bind:value={confirmPassword} required minlength={8} placeholder="Repeat new password" />
          </div>

          <div class="pt-2">
            <Button type="submit" disabled={passwordLoading}>
              {passwordLoading ? 'Updating Password...' : 'Save Password'}
            </Button>
          </div>
        </form>
      </div>

    <!-- Tab 3: MFA / 2FA -->
    {:else if activeTab === 'mfa'}
      <div class="rounded-xl border border-border bg-card p-6 shadow-xs max-w-xl space-y-5">
        <div>
          <h3 class="text-sm font-semibold text-foreground">Two-Factor Authentication (TOTP)</h3>
          <p class="text-xs text-muted-foreground mt-0.5">Protect your account using standard TOTP authenticator apps (1Password, Google Authenticator, Authy).</p>
        </div>

        {#if mfaSuccess}
          <div class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs">
            {mfaSuccess}
          </div>
        {/if}

        {#if mfaError}
          <div class="p-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs">
            {mfaError}
          </div>
        {/if}

        {#if !mfaEnrolling}
          <div class="flex items-center justify-between p-4 rounded-lg border border-border bg-muted/20">
            <div>
              <p class="text-xs font-medium text-foreground">Authenticator App</p>
              <p class="text-[11px] text-muted-foreground mt-0.5">Use a mobile authenticator to generate one-time codes on login.</p>
            </div>
            <Button variant="outline" onclick={handleStartMFAEnroll} disabled={mfaLoading}>
              {mfaLoading ? 'Loading...' : 'Setup Authenticator'}
            </Button>
          </div>

          <!-- Disable Form -->
          <div class="pt-4 border-t border-border">
            <h4 class="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">Disable 2FA</h4>
            <form onsubmit={handleDisableMFA} class="space-y-3">
              <div>
                <label for="mfa-disable-pwd" class="block text-xs font-medium text-foreground mb-1">Confirm Password</label>
                <Input id="mfa-disable-pwd" type="password" bind:value={mfaDisablePassword} placeholder="Your current password" required />
              </div>
              <div>
                <label for="mfa-disable-code" class="block text-xs font-medium text-foreground mb-1">Current 6-digit Code</label>
                <Input id="mfa-disable-code" type="text" bind:value={mfaDisableCode} placeholder="123456" maxlength={6} required />
              </div>
              <Button type="submit" variant="danger" disabled={mfaLoading}>
                Disable 2FA
              </Button>
            </form>
          </div>
        {:else}
          <!-- Enrollment Flow -->
          <div class="p-4 rounded-xl border border-primary/30 bg-primary/5 space-y-4">
            <h4 class="text-xs font-semibold text-foreground">1. Scan QR Code or Enter Secret</h4>
            {#if mfaQrData}
              <div class="p-3 bg-white rounded-lg inline-block shadow-xs">
                {#if mfaQrData.startsWith('<svg')}
                  <!-- SVG format -->
                  {@html mfaQrData}
                {:else}
                  <p class="text-xs text-black font-mono break-all">{mfaQrData}</p>
                {/if}
              </div>
            {/if}
            {#if mfaSecret}
              <div>
                <span class="block text-[11px] font-medium text-muted-foreground mb-1">Manual Secret Key</span>
                <code class="px-2.5 py-1 rounded bg-muted border border-border font-mono text-xs select-all text-foreground">{mfaSecret}</code>
              </div>
            {/if}

            <h4 class="text-xs font-semibold text-foreground pt-2">2. Enter 6-digit Verification Code</h4>
            <form onsubmit={handleConfirmMFAEnable} class="space-y-3">
              <Input type="text" bind:value={mfaCode} placeholder="000000" maxlength={6} class="font-mono tracking-widest text-base text-center w-40" required />
              <div class="flex items-center gap-2">
                <Button type="submit" disabled={mfaLoading}>Verify & Activate</Button>
                <Button type="button" variant="outline" onclick={() => { mfaEnrolling = false; }}>Cancel</Button>
              </div>
            </form>
          </div>
        {/if}
      </div>

    <!-- Tab 4: Active Sessions -->
    {:else if activeTab === 'sessions'}
      <div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="text-sm font-semibold text-foreground">Active Sessions</h3>
            <p class="text-xs text-muted-foreground mt-0.5">Devices currently authenticated to your account.</p>
          </div>
          <Button variant="outline" onclick={handleRevokeOtherSessions} disabled={revokingSessions || sessions.length <= 1}>
            {revokingSessions ? 'Revoking...' : 'Sign Out Other Sessions'}
          </Button>
        </div>

        {#if revokeSuccess}
          <div class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs">
            {revokeSuccess}
          </div>
        {/if}

        <div class="divide-y divide-border border border-border rounded-lg overflow-hidden">
          {#if loadingSessions}
            <div class="p-6 text-center text-xs text-muted-foreground">Loading sessions...</div>
          {:else if sessions.length === 0}
            <div class="p-6 text-center text-xs text-muted-foreground">No session history available.</div>
          {:else}
            {#each sessions as s}
              <div class="p-4 flex items-center justify-between gap-4 bg-card hover:bg-muted/30 transition-colors">
                <div class="flex items-center gap-3">
                  <div class="size-8 rounded-lg bg-muted flex items-center justify-center text-muted-foreground">
                    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect><line x1="8" y1="21" x2="16" y2="21"></line><line x1="12" y1="17" x2="12" y2="21"></line></svg>
                  </div>
                  <div class="grid text-left leading-tight min-w-0">
                    <div class="flex items-center gap-2">
                      <span class="text-xs font-semibold text-foreground truncate">{s.user_agent || 'Browser session'}</span>
                      {#if s.current}
                        <Badge variant="outline" class="text-[10px] bg-primary/10 text-primary border-primary/20">This Device</Badge>
                      {/if}
                    </div>
                    <span class="text-[11px] text-muted-foreground font-mono mt-0.5">{s.ip_address || '127.0.0.1'} • Last active: {s.last_active_at ? new Date(s.last_active_at).toLocaleString() : 'Just now'}</span>
                  </div>
                </div>
              </div>
            {/each}
          {/if}
        </div>
      </div>

    <!-- Tab 5: Personal Tokens -->
    {:else if activeTab === 'tokens'}
      <div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="text-sm font-semibold text-foreground">Personal Access Tokens</h3>
            <p class="text-xs text-muted-foreground mt-0.5">Programmatic API tokens scoped to your staff identity.</p>
          </div>
          <Button onclick={() => { tokenModalOpen = true; newTokenName = ''; issuedTokenValue = null; }}>
            Generate New Token
          </Button>
        </div>

        <div class="divide-y divide-border border border-border rounded-lg overflow-hidden">
          {#if loadingTokens}
            <div class="p-6 text-center text-xs text-muted-foreground">Loading tokens...</div>
          {:else if tokens.length === 0}
            <div class="p-6 text-center text-xs text-muted-foreground">No personal access tokens generated yet.</div>
          {:else}
            {#each tokens as tok}
              <div class="p-4 flex items-center justify-between gap-4 bg-card hover:bg-muted/30 transition-colors">
                <div class="grid text-left leading-tight min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="text-xs font-semibold text-foreground">{tok.name}</span>
                    <code class="text-[10px] px-1.5 py-0.5 rounded bg-muted border border-border font-mono">{tok.prefix}...</code>
                  </div>
                  <span class="text-[11px] text-muted-foreground mt-0.5">Created: {new Date(tok.created_at).toLocaleDateString()} {tok.expires_at ? `• Expires: ${new Date(tok.expires_at).toLocaleDateString()}` : ''}</span>
                </div>
                <Button variant="danger" onclick={() => (deleteTokenTarget = tok)}>Revoke</Button>
              </div>
            {/each}
          {/if}
        </div>
      </div>

    <!-- Tab 6: Data Export -->
    {:else if activeTab === 'export'}
      <div class="rounded-xl border border-border bg-card p-6 shadow-xs max-w-xl space-y-4">
        <div>
          <h3 class="text-sm font-semibold text-foreground">GDPR & Account Data Archive</h3>
          <p class="text-xs text-muted-foreground mt-0.5">Download a complete machine-readable JSON archive of your personal profile, activity logs, and authorization data.</p>
        </div>

        <div class="pt-2">
          <Button onclick={handleExportUserData} disabled={exporting}>
            {exporting ? 'Generating JSON Export...' : 'Download My Data (JSON)'}
          </Button>
        </div>
      </div>
    {/if}
  </div>
</div>

<!-- Modal: New Token -->
<Modal bind:open={tokenModalOpen} title="Generate Personal Access Token" onclose={() => (tokenModalOpen = false)}>
  {#if issuedTokenValue}
    <div class="space-y-4">
      <div class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs">
        Make sure to copy your new personal access token now. You won't be able to see it again!
      </div>
      <div class="flex items-center gap-2">
        <Input type="text" readonly value={issuedTokenValue} class="font-mono text-xs select-all" />
        <Button
          variant="outline"
          onclick={() => {
            navigator.clipboard.writeText(issuedTokenValue || '');
            copiedToken = true;
            setTimeout(() => (copiedToken = false), 2000);
          }}
        >
          {copiedToken ? 'Copied!' : 'Copy'}
        </Button>
      </div>
      <div class="flex justify-end pt-2">
        <Button onclick={() => (tokenModalOpen = false)}>Done</Button>
      </div>
    </div>
  {:else}
    <form onsubmit={handleCreateToken} class="space-y-4">
      {#if tokenError}
        <div class="p-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs">
          {tokenError}
        </div>
      {/if}
      <div>
        <label for="tok-name" class="block text-xs font-medium text-foreground mb-1">Token Name</label>
        <Input id="tok-name" type="text" bind:value={newTokenName} placeholder="e.g. CLI Sync, CI Deploy" required />
      </div>
      <div>
        <label for="tok-days" class="block text-xs font-medium text-foreground mb-1">Expiration (Days)</label>
        <Input id="tok-days" type="number" bind:value={newTokenDays} min={1} max={365} required />
      </div>
      <div class="flex justify-end gap-2 pt-2">
        <Button type="button" variant="outline" onclick={() => (tokenModalOpen = false)}>Cancel</Button>
        <Button type="submit" disabled={creatingToken}>Generate Token</Button>
      </div>
    </form>
  {/if}
</Modal>

<!-- Modal: Revoke Token Confirm -->
<ConfirmDialog
  open={!!deleteTokenTarget}
  title="Revoke Personal Access Token"
  description="Are you sure you want to revoke the token '{deleteTokenTarget?.name}'? Any automation or client relying on it will immediately stop working."
  confirmLabel="Revoke Token"
  loading={revokingToken}
  onconfirm={handleRevokeToken}
  oncancel={() => (deleteTokenTarget = null)}
/>
