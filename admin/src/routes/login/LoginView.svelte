<script lang="ts">
  import { auth } from '$lib/stores/auth.svelte';
  import { router } from '$lib/router.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';

  let email = $state('');
  let password = $state('');
  let code = $state('');
  let mfaRequired = $state(false);
  let loading = $state(false);
  let errorMessage = $state<string | null>(null);

  async function handleLogin(e: SubmitEvent) {
    e.preventDefault();
    if (!email || !password) return;

    loading = true;
    errorMessage = null;

    try {
      const ok = await auth.login({
        email,
        password,
        code: code || undefined,
      });

      if (ok) {
        const redirect = router.query.get('redirect') || '/dashboard';
        router.navigate(redirect);
      } else {
        mfaRequired = true;
      }
    } catch (err: any) {
      errorMessage = err.message || 'Login failed. Please check your credentials.';
    } finally {
      loading = false;
    }
  }
</script>

<div class="min-h-screen w-full flex items-center justify-center p-4 bg-slate-950 bg-[radial-gradient(ellipse_80%_80%_at_50%_-20%,rgba(14,165,233,0.15),rgba(255,255,255,0))]">
  <div class="w-full max-w-sm flex flex-col items-center">
    <!-- Brand -->
    <div class="flex flex-col items-center mb-8 text-center">
      <div class="w-12 h-12 rounded-2xl bg-sky-500/10 border border-sky-500/30 flex items-center justify-center text-sky-400 mb-3 shadow-lg shadow-sky-500/10">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
          <polyline points="2 17 12 22 22 17"></polyline>
          <polyline points="2 12 12 17 22 12"></polyline>
        </svg>
      </div>
      <h1 class="text-xl font-bold text-slate-100">Welcome to Hydro<span class="text-sky-400">Go</span></h1>
      <p class="text-xs text-slate-400 mt-1">Sign in to access your headless CMS admin panel</p>
    </div>

    <!-- Login Card -->
    <div class="w-full rounded-2xl border border-slate-800 bg-slate-900/80 backdrop-blur-xl p-6 shadow-2xl">
      <form onsubmit={handleLogin} class="flex flex-col gap-4">
        {#if errorMessage}
          <div class="p-3 rounded-lg bg-rose-950/60 border border-rose-800/80 text-rose-300 text-xs flex items-start gap-2">
            <span>⚠️</span>
            <span>{errorMessage}</span>
          </div>
        {/if}

        <Input
          label="Email"
          type="email"
          placeholder="admin@example.com"
          bind:value={email}
          required
          autofocus
        />

        <Input
          label="Password"
          type="password"
          placeholder="••••••••"
          bind:value={password}
          required
        />

        {#if mfaRequired}
          <Input
            label="MFA Authenticator Code"
            type="text"
            placeholder="6-digit code"
            bind:value={code}
            required
            maxlength={6}
          />
        {/if}

        <Button
          type="submit"
          variant="primary"
          size="md"
          class="w-full mt-2 font-semibold"
          {loading}
        >
          {mfaRequired ? 'Verify & Sign In' : 'Sign In'}
        </Button>
      </form>
    </div>

    <!-- Security Info -->
    <p class="text-[11px] text-slate-500 text-center mt-6">
      Protected with CSRF double-submit token & HttpOnly session credentials.
    </p>
  </div>
</div>
