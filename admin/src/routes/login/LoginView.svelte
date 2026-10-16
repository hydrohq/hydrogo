<script lang="ts">
  import { auth } from '$lib/stores/auth.svelte';
  import { schema } from '$lib/stores/schema.svelte';
  import { router } from '$lib/router.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import BrandLogo from '$lib/components/ui/BrandLogo.svelte';

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
        await schema.loadSchema(true);
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

<div class="min-h-screen w-full flex flex-col items-center justify-center p-6 md:p-10 bg-muted/40 text-foreground">
  <div class="w-full max-w-sm flex flex-col gap-6">
    <!-- Brand -->
    <div class="self-center">
      <BrandLogo size="md" />
    </div>

    <!-- Login Card -->
    <div class="rounded-xl border border-border bg-card text-card-foreground shadow-xs p-6 sm:p-8">
      <div class="flex flex-col space-y-1.5 text-center mb-6">
        <h1 class="text-2xl font-semibold tracking-tight">Welcome back</h1>
        <p class="text-sm text-muted-foreground">Enter your credentials to sign in to your studio</p>
      </div>

      <form onsubmit={handleLogin} class="flex flex-col gap-4">
        {#if errorMessage}
          <div class="p-3 rounded-md bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-start gap-2">
            <svg class="size-4 shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="12" y1="8" x2="12" y2="12"></line>
              <line x1="12" y1="16" x2="12.01" y2="16"></line>
            </svg>
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
          placeholder="â€¢â€¢â€¢â€¢â€¢â€¢â€¢â€¢"
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
          {mfaRequired ? 'Verify & Sign In' : 'Sign in'}
        </Button>
      </form>
    </div>
  </div>
</div>
