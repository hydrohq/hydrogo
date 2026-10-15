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

<div class="min-h-screen w-full flex items-center justify-center p-4 bg-background bg-[radial-gradient(ellipse_80%_80%_at_50%_-20%,rgba(37,99,235,0.2),rgba(255,255,255,0))]">
  <div class="w-full max-w-sm flex flex-col items-center">
    <!-- Brand -->
    <div class="flex flex-col items-center mb-8 text-center">
      <div class="w-12 h-12 rounded-xl bg-primary/15 border border-primary/30 flex items-center justify-center text-primary mb-3 shadow-lg shadow-blue-500/10">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"></path>
        </svg>
      </div>
      <h1 class="text-xl font-bold tracking-tight text-foreground">Welcome to Hydro<span class="text-primary font-black">Go</span></h1>
      <p class="text-xs text-muted-foreground mt-1">Sign in to access your headless CMS studio</p>
    </div>

    <!-- Login Card -->
    <div class="w-full rounded-xl border border-border bg-card/90 backdrop-blur-xl p-6 shadow-2xl">
      <form onsubmit={handleLogin} class="flex flex-col gap-4">
        {#if errorMessage}
          <div class="p-3 rounded-md bg-destructive/15 border border-destructive/30 text-rose-300 text-xs flex items-start gap-2">
            <span>⚠️</span>
            <span>{errorMessage}</span>
          </div>
        {/if}

        <Input
          label="Email Address"
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
          class="w-full mt-2 font-medium"
          {loading}
        >
          {mfaRequired ? 'Verify & Sign In' : 'Sign In'}
        </Button>
      </form>
    </div>

    <!-- Security Info -->
    <p class="text-[11px] text-muted-foreground text-center mt-6">
      Protected with CSRF double-submit token & HttpOnly session credentials.
    </p>
  </div>
</div>
