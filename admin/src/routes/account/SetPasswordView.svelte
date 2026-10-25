<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { api, ApiClientError } from '$lib/api/client';
  import { router } from '$lib/router.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import BrandLogo from '$lib/components/ui/BrandLogo.svelte';

  // The two e-mailed links land here: a password reset and an invitation. Both
  // are a token plus a new password, posted to the same kind of endpoint.
  let { mode }: { mode: 'reset' | 'invite' } = $props();

  const token = $derived(router.query.get('token') ?? '');
  const endpoint = $derived(mode === 'reset' ? '/auth/password-reset/confirm' : '/auth/invites/accept');
  const done = $derived(mode === 'reset' ? 'password_reset=1' : 'invite_accepted=1');

  let password = $state('');
  let confirm = $state('');
  let loading = $state(false);
  let error = $state<string | null>(null);

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    error = null;
    if (password.length < 8) return void (error = t('setpw.tooShort'));
    if (password !== confirm) return void (error = t('setpw.mismatch'));
    loading = true;
    try {
      await api.post(endpoint, new URLSearchParams({ token, password }).toString(), {
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        noAuthRedirect: true,
      });
      window.location.assign(`/admin/login?${done}`);
    } catch (err) {
      const status = err instanceof ApiClientError ? err.status : 0;
      error = status === 401 || status === 404 ? t('setpw.invalidLink') : status === 400 ? t('setpw.weak') : status === 429 ? t('setpw.rate') : t('setpw.failed');
    } finally {
      loading = false;
    }
  }
</script>

<div class="min-h-screen w-full flex flex-col items-center justify-center p-6 md:p-10 bg-muted/40 text-foreground">
  <div class="w-full max-w-sm flex flex-col gap-6">
    <div class="self-center"><BrandLogo size="md" /></div>
    <div class="rounded-xl border border-border bg-card text-card-foreground shadow-xs p-6 sm:p-8">
      <div class="flex flex-col space-y-1.5 text-center mb-6">
        <h1 class="text-2xl font-semibold tracking-tight">{mode === 'reset' ? t('setpw.resetTitle') : t('setpw.inviteTitle')}</h1>
        <p class="text-sm text-muted-foreground">{mode === 'reset' ? t('setpw.resetHelp') : t('setpw.inviteHelp')}</p>
      </div>

      {#if !token}
        <div class="p-3 rounded-md bg-destructive/10 border border-destructive/20 text-destructive text-xs" role="alert">{t('setpw.invalidLink')}</div>
        <a class="mt-4 text-center text-sm underline" href="/admin/login">{t('setpw.backToSignIn')}</a>
      {:else}
        <form onsubmit={submit} class="flex flex-col gap-4">
          {#if error}
            <div class="p-3 rounded-md bg-destructive/10 border border-destructive/20 text-destructive text-xs" role="alert">{error}</div>
          {/if}
          <Input label={t('setpw.password')} type="password" autocomplete="new-password" bind:value={password} required autofocus />
          <Input label={t('setpw.confirm')} type="password" autocomplete="new-password" bind:value={confirm} required />
          <Button type="submit" variant="primary" size="md" class="w-full mt-2 font-medium" {loading}>{t('setpw.submit')}</Button>
        </form>
      {/if}
    </div>
  </div>
</div>
