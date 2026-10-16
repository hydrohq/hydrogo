import { onMount, onDestroy } from 'svelte';
import { api, getCsrfToken } from '$lib/api/client';
import type { ActiveEditor } from '$lib/types/presence';

export function createPresence(
  getEndpoint: () => string,
  getId: () => string | null | undefined
) {
  let activeEditors = $state<ActiveEditor[]>([]);
  let dismissedBanner = $state(false);
  let timer: any = null;

  async function sendHeartbeat() {
    const endpoint = getEndpoint();
    const id = getId();
    if (!endpoint || !id || id === 'new') return;
    try {
      const res = await api.post<{ data: ActiveEditor[] }>(endpoint, {});
      if (res && Array.isArray(res.data)) {
        activeEditors = res.data;
      }
    } catch {
      // Quiet fail if offline or unauthorized
    }
  }

  function sendLeave() {
    const endpoint = getEndpoint();
    const id = getId();
    if (!endpoint || !id || id === 'new') return;
    try {
      api.post(`${endpoint}/leave`, {}).catch(() => {});
    } catch {
      // Ignore
    }
  }

  function handleBeforeUnload() {
    const endpoint = getEndpoint();
    const id = getId();
    if (!endpoint || !id || id === 'new') return;
    try {
      const token = getCsrfToken();
      fetch(`${endpoint}/leave`, {
        method: 'POST',
        keepalive: true,
        credentials: 'same-origin',
        headers: token ? { 'X-CSRF-Token': token } : {},
      }).catch(() => {});
    } catch {
      // Ignore
    }
  }

  $effect(() => {
    const endpoint = getEndpoint();
    const id = getId();

    if (timer) {
      clearInterval(timer);
      timer = null;
    }

    if (endpoint && id && id !== 'new') {
      sendHeartbeat();
      timer = setInterval(sendHeartbeat, 10000);
    } else {
      activeEditors = [];
    }

    return () => {
      if (timer) {
        clearInterval(timer);
        timer = null;
      }
      sendLeave();
    };
  });

  onMount(() => {
    if (typeof window !== 'undefined') {
      window.addEventListener('beforeunload', handleBeforeUnload);
    }
  });

  onDestroy(() => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('beforeunload', handleBeforeUnload);
    }
  });

  return {
    get activeEditors() {
      return activeEditors;
    },
    get count() {
      return activeEditors.length;
    },
    get dismissedBanner() {
      return dismissedBanner;
    },
    set dismissedBanner(v: boolean) {
      dismissedBanner = v;
    },
    get names() {
      return activeEditors.map((e) => e.name || e.email || 'Bir editör').join(', ');
    },
    dismiss() {
      dismissedBanner = true;
    },
  };
}

export type PresenceState = ReturnType<typeof createPresence>;
