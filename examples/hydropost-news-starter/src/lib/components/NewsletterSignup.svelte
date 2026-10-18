<script lang="ts">
  let email = $state('');
  let status = $state<'idle' | 'submitting' | 'done' | 'error'>('idle');

  async function subscribe(e: SubmitEvent) {
    e.preventDefault();
    if (!email.includes('@')) {
      status = 'error';
      return;
    }
    status = 'submitting';
    // No newsletter backend wired up in this starter — this just simulates
    // the request so the component is usable as a drop-in reference.
    await new Promise((r) => setTimeout(r, 400));
    status = 'done';
  }
</script>

<div class="rounded-xl bg-[var(--hp-surface-sunken)] p-5 sm:p-6 flex flex-col sm:flex-row sm:items-center gap-4">
  <div class="w-11 h-11 rounded-full bg-[var(--hp-primary-light)] text-[var(--hp-primary)] flex items-center justify-center shrink-0">
    <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <rect x="3" y="5" width="18" height="14" rx="2" /><path d="m3 7 9 6 9-6" stroke-linecap="round" stroke-linejoin="round" />
    </svg>
  </div>

  <div class="flex-1 min-w-0">
    <h3 class="font-bold text-[var(--hp-ink)] text-sm">Bülten Aboneliği</h3>
    <p class="text-xs text-[var(--hp-muted)] mt-0.5">Son haberleri ve ilginizi çeken konulardaki güncellemeleri almak için kaydolun.</p>
  </div>

  {#if status === 'done'}
    <span class="text-sm font-semibold text-emerald-700 shrink-0">Kaydınız alındı.</span>
  {:else}
    <form onsubmit={subscribe} class="flex items-center gap-2 w-full sm:w-auto">
      <input
        type="email"
        required
        bind:value={email}
        placeholder="email@example.com"
        class="flex-1 sm:w-56 px-3 py-2 text-sm rounded-lg border border-[var(--hp-border-strong)] bg-[var(--hp-surface)] text-[var(--hp-ink)] outline-none focus:border-[var(--hp-primary)] transition-colors"
      />
      <button
        type="submit"
        disabled={status === 'submitting'}
        class="shrink-0 px-4 py-2 rounded-lg bg-[var(--hp-primary)] hover:bg-[var(--hp-primary-dark)] disabled:opacity-60 text-white text-sm font-bold transition-colors cursor-pointer"
      >
        {status === 'submitting' ? '...' : 'Kayıt Ol'}
      </button>
    </form>
  {/if}
</div>
{#if status === 'error'}
  <p class="text-xs text-[var(--hp-primary)] mt-1.5">Geçerli bir e-posta adresi girin.</p>
{/if}
