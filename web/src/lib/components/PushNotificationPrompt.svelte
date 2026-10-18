<script lang="ts">
  import { onMount } from 'svelte';

  let showPrompt = $state(false);

  function acceptPush() {
    showPrompt = false;
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('push_permission', 'granted');
    }
    if (typeof Notification !== 'undefined') {
      Notification.requestPermission().then((permission) => {
        if (permission === 'granted') {
          console.log('Web push notifications enabled');
        }
      });
    }
  }

  function dismissPush() {
    showPrompt = false;
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('push_permission', 'dismissed');
    }
  }

  onMount(() => {
    if (typeof localStorage !== 'undefined') {
      const status = localStorage.getItem('push_permission');
      if (!status) {
        // Show after 3 seconds of reading to avoid intrusive popups
        const timer = setTimeout(() => {
          showPrompt = true;
        }, 3000);
        return () => clearTimeout(timer);
      }
    }
  });
</script>

{#if showPrompt}
  <div class="fixed bottom-5 left-5 z-40 max-w-sm p-4 rounded-2xl bg-zinc-900 border border-zinc-800 shadow-2xl text-xs space-y-3 animate-in fade-in slide-in-from-bottom-4 duration-300">
    <div class="flex items-start gap-3">
      <div class="w-8 h-8 rounded-full bg-red-600/20 border border-red-500/40 text-red-500 flex items-center justify-center text-sm shrink-0">
        🔔
      </div>
      <div>
        <h4 class="font-bold text-white text-xs">Son Dakika Bildirimleri</h4>
        <p class="text-zinc-400 text-[11px] mt-0.5 leading-relaxed">
          Önemli gelişmelerden ve son dakika haberlerinden anında haberdar olmak ister misiniz?
        </p>
      </div>
    </div>

    <div class="flex items-center gap-2 justify-end pt-1">
      <button
        type="button"
        onclick={dismissPush}
        class="px-3 py-1.5 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors cursor-pointer text-[11px] font-medium"
      >
        Daha Sonra
      </button>
      <button
        type="button"
        onclick={acceptPush}
        class="px-3 py-1.5 rounded-lg bg-red-600 hover:bg-red-500 text-white font-bold transition-colors cursor-pointer text-[11px] shadow-md shadow-red-600/20"
      >
        Bildirimleri Aç
      </button>
    </div>
  </div>
{/if}
