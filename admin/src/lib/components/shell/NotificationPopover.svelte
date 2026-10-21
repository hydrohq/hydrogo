<script lang="ts">
  import { schema } from '$lib/stores/schema.svelte';
  import { t } from '$lib/i18n/index.svelte';
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { router } from '$lib/router.svelte';

  interface NotificationItem {
    id: string;
    title: string;
    message: string;
    href?: string;
    target_path?: string;
    created_at: string;
    read: boolean;
  }

  let open = $state(false);
  let notifications = $state<NotificationItem[]>([]);
  let unreadCount = $state(0);

  async function loadNotifications() {
    if (schema.schema && schema.modules.pages !== true) return;
    try {
      const res = await api.get<any>('/api/v1/admin/notifications');
      if (res?.data && Array.isArray(res.data)) {
        notifications = res.data;
        unreadCount = Number(res?.metadata?.unread ?? res?.meta?.unread ?? res.data.filter((n: NotificationItem) => !n.read).length);
      } else if (res?.data?.items && Array.isArray(res.data.items)) {
        notifications = res.data.items;
        unreadCount = Number(res?.data?.unread_count ?? 0);
      }
    } catch (e) {
      // Quiet fail if not supported or not logged in
    }
  }

  async function markAllRead() {
    try {
      await api.post('/api/v1/admin/notifications/read-all', {});
      unreadCount = 0;
      notifications = notifications.map((n) => ({ ...n, read: true }));
    } catch (e) {
      console.warn('Mark all read failed:', e);
    }
  }

  async function dismissNotification(id: string, e: MouseEvent) {
    e.stopPropagation();
    try {
      await api.post(`/api/v1/admin/notifications/${id}/dismiss`, {});
      notifications = notifications.filter((n) => n.id !== id);
      unreadCount = Math.max(0, unreadCount - 1);
    } catch (e) {
      console.warn('Dismiss notification failed:', e);
    }
  }

  function handleNavigate(targetPath?: string) {
    open = false;
    if (targetPath) {
      router.navigate(targetPath);
    }
  }

  function toggleOpen() {
    open = !open;
    if (open) {
      loadNotifications();
    }
  }

  onMount(() => {
    loadNotifications();
    // Poll notifications every 45s
    const timer = setInterval(loadNotifications, 45000);
    return () => clearInterval(timer);
  });
</script>

<svelte:window
  onclick={(e) => {
    if (open) {
      const target = e.target as HTMLElement;
      if (!target.closest('.notification-container')) {
        open = false;
      }
    }
  }}
  onkeydown={(e) => {
    if (e.key === 'Escape') open = false;
  }}
/>

<div class="relative notification-container">
  <!-- Bell button -->
  <button
    type="button"
    onclick={toggleOpen}
    class="relative flex size-8 items-center justify-center rounded-md border border-input bg-background hover:bg-accent text-muted-foreground hover:text-foreground transition-colors cursor-pointer shadow-xs"
    title={t('notif.title')}
    aria-label={t('notif.title')}
  >
    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"></path>
      <path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"></path>
    </svg>

    {#if unreadCount > 0}
      <span class="absolute -top-1 -right-1 flex size-4 items-center justify-center rounded-full bg-primary text-[9px] font-bold text-primary-foreground shadow-xs animate-in zoom-in-50">
        {unreadCount > 9 ? '9+' : unreadCount}
      </span>
    {/if}
  </button>

  <!-- Popover dropdown -->
  {#if open}
    <div
      class="absolute right-0 mt-2 w-80 sm:w-96 rounded-xl border border-border bg-card p-0 shadow-xl z-50 overflow-hidden animate-in fade-in zoom-in-95 duration-100"
    >
      <div class="flex items-center justify-between px-4 py-3 border-b border-border bg-muted/40">
        <div class="flex items-center gap-2">
          <span class="text-xs font-semibold text-foreground">{t('notif.title')}</span>
          {#if unreadCount > 0}
            <span class="px-1.5 py-0.5 rounded-full bg-primary/10 text-primary text-[10px] font-medium">
              {t('notif.unread', { count: unreadCount })}
            </span>
          {/if}
        </div>

        {#if unreadCount > 0}
          <button
            type="button"
            onclick={markAllRead}
            class="text-[11px] text-primary hover:underline font-medium cursor-pointer"
          >
            {t('notif.markAll')}
          </button>
        {/if}
      </div>

      <div class="max-h-80 overflow-y-auto divide-y divide-border/60">
        {#if notifications.length === 0}
          <div class="p-8 text-center text-xs text-muted-foreground">
            <svg class="size-6 mx-auto mb-2 opacity-40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"></path>
              <path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"></path>
            </svg>
            {t('notif.empty')}
          </div>
        {:else}
          {#each notifications as item}
            <div
              role="button"
              tabindex="0"
              onclick={() => handleNavigate(item.href || item.target_path)}
              onkeydown={(e) => { if (e.key === 'Enter') handleNavigate(item.href || item.target_path); }}
              class="w-full text-left p-3.5 hover:bg-muted/40 transition-colors cursor-pointer relative group flex items-start gap-3 {!item.read ? 'bg-primary/5' : ''}"
            >
              <div class="size-2 rounded-full mt-1.5 shrink-0 {!item.read ? 'bg-primary' : 'bg-transparent'}"></div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center justify-between gap-2">
                  <span class="text-xs font-semibold text-foreground truncate">{item.title}</span>
                  <span class="text-[10px] text-muted-foreground shrink-0">{new Date(item.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                </div>
                <p class="text-xs text-muted-foreground mt-0.5 line-clamp-2 leading-relaxed">{item.message}</p>
              </div>

              <button
                type="button"
                onclick={(e) => dismissNotification(item.id, e)}
                class="opacity-0 group-hover:opacity-100 p-1 text-muted-foreground hover:text-foreground rounded transition-opacity cursor-pointer shrink-0"
                title={t('common.dismiss')}
                aria-label={t('notif.dismiss')}
              >
                <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"></line>
                  <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
              </button>
            </div>
          {/each}
        {/if}
      </div>
    </div>
  {/if}
</div>
