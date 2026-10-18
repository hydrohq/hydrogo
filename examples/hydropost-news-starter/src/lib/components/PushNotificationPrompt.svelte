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
        // Only offer once the reader has scrolled past the headline
        // carousel, so the prompt never covers its nav arrows or number bar.
        const timer = setTimeout(() => {
          if (typeof window !== 'undefined' && window.scrollY > 200) {
            showPrompt = true;
          } else if (typeof window !== 'undefined') {
            const onScroll = () => {
              if (window.scrollY > 200) {
                showPrompt = true;
                window.removeEventListener('scroll', onScroll);
              }
            };
            window.addEventListener('scroll', onScroll, { passive: true });
          }
        }, 4000);
        return () => clearTimeout(timer);
      }
    }
  });
</script>

{#if showPrompt}
  <div class="fixed bottom-5 right-5 z-40 max-w-sm p-4 rounded-xl bg-[var(--hp-surface)] border border-[var(--hp-border-strong)] shadow-lg text-xs space-y-3 animate-in fade-in slide-in-from-bottom-4 duration-300">
    <div class="flex items-start gap-3">
      <div class="w-8 h-8 rounded-full bg-[var(--hp-primary-light)] text-[var(--hp-primary)] flex items-center justify-center shrink-0">
        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9" stroke-linecap="round" stroke-linejoin="round" />
          <path d="M13.73 21a2 2 0 0 1-3.46 0" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </div>
      <div>
        <h4 class="font-bold text-[var(--hp-ink)] text-xs">Son Dakika Bildirimleri</h4>
        <p class="text-[var(--hp-muted)] text-[11px] mt-0.5 leading-relaxed">
          Önemli gelişmelerden ve son dakika haberlerinden anında haberdar olmak ister misiniz?
        </p>
      </div>
    </div>

    <div class="flex items-center gap-2 justify-end pt-1">
      <button
        type="button"
        onclick={dismissPush}
        class="px-3 py-1.5 rounded-lg text-[var(--hp-muted)] hover:text-[var(--hp-ink)] hover:bg-[var(--hp-surface-sunken)] transition-colors cursor-pointer text-[11px] font-semibold"
      >
        Daha Sonra
      </button>
      <button
        type="button"
        onclick={acceptPush}
        class="px-3 py-1.5 rounded-lg bg-[var(--hp-primary)] hover:bg-[var(--hp-primary-dark)] text-white font-bold transition-colors cursor-pointer text-[11px]"
      >
        Bildirimleri Aç
      </button>
    </div>
  </div>
{/if}
