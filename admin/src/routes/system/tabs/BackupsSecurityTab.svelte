<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';

  interface BackupItem {
    name: string;
    created_at: string;
    bytes: number;
  }

  interface BackupStatus {
    enabled: boolean;
    public_key?: string;
    recent: BackupItem[];
  }

  interface IPBan {
    address: string;
    expires_at?: string;
  }

  let backupStatus = $state<BackupStatus | null>(null);
  let ipBans = $state<IPBan[]>([]);
  let loadingBackups = $state(true);
  let loadingBans = $state(true);

  // Download backup modal
  let backupModalOpen = $state(false);
  let adminPassword = $state('');
  let downloadingBackup = $state(false);
  let backupError = $state<string | null>(null);

  // IP Ban modal
  let banModalOpen = $state(false);
  let newAddress = $state('');
  let newDuration = $state('24h');
  let banning = $state(false);
  let banError = $state<string | null>(null);

  // IP Unban confirmation
  let unbanTarget = $state<IPBan | null>(null);
  let unbanning = $state(false);

  function formatBytes(bytes: number): string {
    if (!bytes) return '0 B';
    const mb = bytes / (1024 * 1024);
    if (mb < 1024) return `${mb.toFixed(1)} MB`;
    return `${(mb / 1024).toFixed(2)} GB`;
  }

  async function loadBackups() {
    loadingBackups = true;
    try {
      const res = await api.get<ApiResponse<BackupStatus>>('/api/v1/admin/backups');
      if (res?.data) {
        backupStatus = res.data;
      }
    } catch (e) {
      console.warn('Load backups failed:', e);
    } finally {
      loadingBackups = false;
    }
  }

  async function loadBans() {
    loadingBans = true;
    try {
      const res = await api.get<ApiResponse<IPBan[]>>('/api/v1/admin/security/ip-bans');
      if (res?.data && Array.isArray(res.data)) {
        ipBans = res.data;
      } else {
        ipBans = [];
      }
    } catch (e) {
      console.warn('Load IP bans failed:', e);
      ipBans = [];
    } finally {
      loadingBans = false;
    }
  }

  async function handleDownloadBackup(e: SubmitEvent) {
    e.preventDefault();
    if (!adminPassword) return;
    downloadingBackup = true;
    backupError = null;
    try {
      const res = await fetch('/api/v1/admin/backups', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest',
        },
        body: JSON.stringify({ password: adminPassword }),
      });

      if (!res.ok) {
        let errMsg = t('system.backup.failed');
        try {
          const errData = await res.json();
          errMsg = errData.message || errMsg;
        } catch {
          // not json
        }
        throw new Error(errMsg);
      }

      const blob = await res.blob();
      const disposition = res.headers.get('Content-Disposition') || '';
      let filename = 'hydrogo-backup.tar.gz';
      const match = disposition.match(/filename="?([^";]+)"?/);
      if (match && match[1]) {
        filename = match[1];
      }

      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);

      backupModalOpen = false;
      adminPassword = '';
      await loadBackups();
    } catch (err: any) {
      backupError = err.message || t('system.backup.creationFailed');
    } finally {
      downloadingBackup = false;
    }
  }

  async function handleAddBan(e: SubmitEvent) {
    e.preventDefault();
    if (!newAddress.trim()) return;
    banning = true;
    banError = null;
    try {
      await api.post('/api/v1/admin/security/ip-bans', {
        address: newAddress.trim(),
        duration: newDuration || undefined,
      });
      banModalOpen = false;
      newAddress = '';
      newDuration = '24h';
      await loadBans();
    } catch (err: any) {
      banError = err.message || t('system.ban.failed');
    } finally {
      banning = false;
    }
  }

  async function handleUnban() {
    if (!unbanTarget) return;
    unbanning = true;
    try {
      await api.delete('/api/v1/admin/security/ip-bans', {
        body: JSON.stringify({ address: unbanTarget.address }),
        headers: { 'Content-Type': 'application/json' },
      });
      unbanTarget = null;
      await loadBans();
    } catch (e) {
      console.warn('Unban IP failed:', e);
    } finally {
      unbanning = false;
    }
  }

  onMount(() => {
    loadBackups();
    loadBans();
  });
</script>

<div class="space-y-8">
  <!-- Section 1: Backups -->
  <div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-border pb-4">
      <div>
        <div class="flex items-center gap-2">
          <h3 class="text-base font-semibold text-foreground">{t('system.backup.title')}</h3>
          {#if backupStatus?.enabled}
            <Badge variant="default" class="bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20 text-[10px]">{t('system.backup.cloud')}</Badge>
          {:else}
            <Badge variant="secondary" class="text-[10px]">{t('system.backup.local')}</Badge>
          {/if}
        </div>
        <p class="text-xs text-muted-foreground mt-0.5">{t('system.backup.help')}</p>
      </div>

      <Button onclick={() => { backupModalOpen = true; backupError = null; adminPassword = ''; }}>
        <svg class="size-4 mr-1.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
          <polyline points="7 10 12 15 17 10"></polyline>
          <line x1="12" y1="15" x2="12" y2="3"></line>
        </svg>
        {t('system.backup.download')}
      </Button>
    </div>

    {#if backupStatus?.public_key}
      <div class="rounded-lg border border-border bg-muted/30 p-3.5 text-xs flex items-center justify-between gap-4">
        <div>
          <span class="font-medium text-foreground block">{t('system.backup.key')}</span>
          <span class="font-mono text-muted-foreground text-[11px] select-all break-all">{backupStatus.public_key}</span>
        </div>
        <button
          type="button"
          onclick={() => navigator.clipboard.writeText(backupStatus?.public_key || '')}
          class="text-xs text-primary hover:underline font-medium shrink-0 cursor-pointer"
        >
          {t('system.backup.copyKey')}
        </button>
      </div>
    {/if}

    <div class="space-y-3">
      <h4 class="text-xs font-semibold text-foreground uppercase tracking-wider">{t('system.backup.recent')}</h4>
      {#if loadingBackups}
        <div class="p-8 text-center text-xs text-muted-foreground animate-pulse">{t('system.backup.loading')}</div>
      {:else if !backupStatus?.recent || backupStatus.recent.length === 0}
        <div class="rounded-lg border border-dashed border-border p-6 text-center text-xs text-muted-foreground">
          {t('system.backup.empty')}
        </div>
      {:else}
        <div class="rounded-lg border border-border divide-y divide-border overflow-hidden">
          {#each backupStatus.recent as snapshot}
            <div class="p-3.5 flex items-center justify-between gap-4 hover:bg-muted/30 transition-colors text-xs">
              <div class="min-w-0">
                <span class="font-mono font-medium text-foreground block truncate">{snapshot.name}</span>
                <span class="text-muted-foreground text-[11px]">{new Date(snapshot.created_at).toLocaleString()}</span>
              </div>
              <Badge variant="outline" class="font-mono text-[11px]">{formatBytes(snapshot.bytes)}</Badge>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>

  <!-- Section 2: IP Blocklist & Bans -->
  <div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-6">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-border pb-4">
      <div>
        <h3 class="text-base font-semibold text-foreground">{t('system.ban.title')}</h3>
        <p class="text-xs text-muted-foreground mt-0.5">{t('system.ban.help')}</p>
      </div>

      <Button onclick={() => { banModalOpen = true; banError = null; newAddress = ''; }}>
        <svg class="size-4 mr-1.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="4.93" y1="4.93" x2="19.07" y2="19.07"></line>
        </svg>
        {t('system.ban.action')}
      </Button>
    </div>

    {#if loadingBans}
      <div class="p-8 text-center text-xs text-muted-foreground animate-pulse">{t('system.ban.loading')}</div>
    {:else if ipBans.length === 0}
      <div class="rounded-lg border border-dashed border-border p-8 text-center text-xs text-muted-foreground">
        {t('system.ban.empty')}
      </div>
    {:else}
      <div class="rounded-lg border border-border divide-y divide-border overflow-hidden">
        {#each ipBans as ban}
          <div class="p-4 flex items-center justify-between gap-4 hover:bg-muted/30 transition-colors">
            <div class="min-w-0">
              <span class="font-mono text-xs font-semibold text-foreground block">{ban.address}</span>
              <span class="text-[11px] text-muted-foreground">
                {#if ban.expires_at}
                  {t('users.expires', { date: new Date(ban.expires_at).toLocaleString() })}
                {:else}
                  <span class="text-amber-500 font-medium">{t('system.ban.permanent')}</span>
                {/if}
              </span>
            </div>

            <Button variant="outline" onclick={() => (unbanTarget = ban)}>
              {t('system.ban.unban')}
            </Button>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<!-- Modal: Download Backup Confirmation -->
<Modal open={backupModalOpen} title={t('system.backup.modalTitle')} onclose={() => (backupModalOpen = false)}>
  <form onsubmit={handleDownloadBackup} class="space-y-4">
    <p class="text-xs text-muted-foreground">
      {t('system.backup.modalHelp')}
    </p>

    {#if backupError}
      <div class="rounded-lg border border-destructive/20 bg-destructive/10 p-3 text-xs text-destructive">
        {backupError}
      </div>
    {/if}

    <div>
      <label for="admin-pwd" class="block text-xs font-medium text-foreground mb-1">{t('system.backup.password')}</label>
      <Input id="admin-pwd" type="password" bind:value={adminPassword} placeholder={t('system.backup.passwordPlaceholder')} required />
    </div>

    <div class="flex items-center justify-end gap-2 pt-2 border-t border-border">
      <Button type="button" variant="outline" onclick={() => (backupModalOpen = false)}>{t('common.cancel')}</Button>
      <Button type="submit" disabled={downloadingBackup || !adminPassword}>
        {downloadingBackup ? t('system.backup.creating') : t('system.backup.authorize')}
      </Button>
    </div>
  </form>
</Modal>

<!-- Modal: Ban IP Address -->
<Modal open={banModalOpen} title={t('system.ban.modalTitle')} onclose={() => (banModalOpen = false)}>
  <form onsubmit={handleAddBan} class="space-y-4">
    <p class="text-xs text-muted-foreground">
      {t('system.ban.modalHelp')}
    </p>

    {#if banError}
      <div class="rounded-lg border border-destructive/20 bg-destructive/10 p-3 text-xs text-destructive">
        {banError}
      </div>
    {/if}

    <div>
      <label for="ban-ip" class="block text-xs font-medium text-foreground mb-1">{t('system.ban.address')}</label>
      <Input id="ban-ip" type="text" bind:value={newAddress} placeholder={t('system.ban.addressPlaceholder')} required />
    </div>

    <div>
      <label for="ban-dur" class="block text-xs font-medium text-foreground mb-1">{t('system.ban.duration')}</label>
      <select
        id="ban-dur"
        bind:value={newDuration}
        class="h-9 w-full rounded-md border border-input bg-background px-3 text-xs text-foreground focus:outline-hidden focus:ring-1 focus:ring-ring"
      >
        <option value="1h">{t('system.ban.d1h')}</option>
        <option value="24h">{t('system.ban.d24h')}</option>
        <option value="168h">{t('system.ban.d7d')}</option>
        <option value="720h">{t('system.ban.d30d')}</option>
        <option value="">{t('system.ban.dperm')}</option>
      </select>
    </div>

    <div class="flex items-center justify-end gap-2 pt-2 border-t border-border">
      <Button type="button" variant="outline" onclick={() => (banModalOpen = false)}>{t('common.cancel')}</Button>
      <Button type="submit" disabled={banning || !newAddress.trim()}>
        {banning ? t('system.ban.enforcing') : t('system.ban.confirm')}
      </Button>
    </div>
  </form>
</Modal>

<!-- Unban Confirm Dialog -->
<ConfirmDialog
  open={!!unbanTarget}
  title={t('system.ban.removeTitle')}
  description={t('system.ban.removeDescription', { address: unbanTarget?.address ?? '' })}
  confirmLabel={t('system.ban.lift')}
  loading={unbanning}
  onconfirm={handleUnban}
  oncancel={() => (unbanTarget = null)}
/>
