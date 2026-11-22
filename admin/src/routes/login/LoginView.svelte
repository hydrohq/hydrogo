<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { auth } from '$lib/stores/auth.svelte';
  import { schema } from '$lib/stores/schema.svelte';
  import { router } from '$lib/router.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import BrandLogo from '$lib/components/ui/BrandLogo.svelte';

  let email = $state(import.meta.env.VITE_DEMO_MODE === 'true' ? 'admin@example.com' : '');
  let password = $state(import.meta.env.VITE_DEMO_MODE === 'true' ? 'DemoPass-2026-Strong!' : '');
  let showPassword = $state(false);
  let code = $state('');
  let mfaRequired = $state(false);
  let loading = $state(false);
  let errorMessage = $state<string | null>(null);
  const isDemoHost = $derived(
    import.meta.env.VITE_DEMO_MODE === 'true' ||
      (typeof window !== 'undefined' && (window.location.hostname === 'demo.hydrogo.dev' || router.query.get('demo') === '1'))
  );
  // Shown after the page that sets a new password sends people back here.
  const notice = $derived(
    router.query.get('password_reset') === '1' ? t('login.passwordReset') : router.query.get('invite_accepted') === '1' ? t('login.inviteAccepted') : null
  );

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
        const isConsole = typeof window !== 'undefined' && (window.location.hostname.startsWith('console.'));
        const redirect = router.query.get('redirect') || (isConsole ? '/console' : '/dashboard');
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

<div class="min-h-screen w-full flex flex-col bg-background text-foreground">
  {#if isDemoHost}
    <div class="w-full border-b border-border bg-muted/50 px-4 py-2 text-center text-xs text-muted-foreground">
      This is a live demo of <strong class="font-medium text-foreground">HydroGo</strong>. Demo data resets periodically.
    </div>
  {/if}

  <div class="flex-1 flex flex-col items-center justify-center px-6 py-12">
    <div class="w-full max-w-sm flex flex-col items-center">
      <!-- Minimal Centered Brand -->
      <div class="mb-4">
        <BrandLogo size="lg" showText={false} />
      </div>

      <h1 class="text-lg font-medium tracking-tight text-foreground mb-7">
        {t('login.welcome')}
      </h1>

      {#if notice}
        <div class="w-full mb-4 px-3.5 py-2.5 rounded-lg bg-emerald-500/10 border border-emerald-500/25 text-xs text-center" role="status">{notice}</div>
      {/if}

      <form onsubmit={handleLogin} class="w-full flex flex-col gap-4">
        {#if errorMessage}
          <div class="px-3.5 py-2.5 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-start gap-2">
            <svg class="size-4 shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="12" y1="8" x2="12" y2="12"></line>
              <line x1="12" y1="16" x2="12.01" y2="16"></line>
            </svg>
            <span>{errorMessage}</span>
          </div>
        {/if}

        <Input
          label={t('login.email')}
          type="email"
          placeholder="admin@example.com"
          bind:value={email}
          required
          autofocus
        />

        <div class="relative">
          <Input
            label={t('login.password')}
            type={showPassword ? 'text' : 'password'}
            placeholder="••••••••"
            bind:value={password}
            required
          />
          <button
            type="button"
            onclick={() => (showPassword = !showPassword)}
            aria-label={showPassword ? 'Hide password' : 'Show password'}
            class="absolute right-2.5 bottom-2 p-1 text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
          >
            {#if showPassword}
              <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" />
                <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" />
                <line x1="1" y1="1" x2="23" y2="23" />
              </svg>
            {:else}
              <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                <circle cx="12" cy="12" r="3" />
              </svg>
            {/if}
          </button>
        </div>

        {#if mfaRequired}
          <Input
            label={t('login.mfa')}
            type="text"
            placeholder={t('login.mfaPlaceholder')}
            bind:value={code}
            required
            maxlength={6}
          />
        {/if}

        <Button
          type="submit"
          variant="primary"
          size="md"
          class="w-full mt-2 font-medium gap-2"
          {loading}
        >
          <span>{mfaRequired ? t('login.verify') : t('login.signIn')}</span>
          <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path d="M5 12h14" />
            <path d="m12 5 7 7-7 7" />
          </svg>
        </Button>
      </form>
    </div>
  </div>
</div>
