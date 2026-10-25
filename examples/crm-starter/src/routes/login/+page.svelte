<script lang="ts">
  import { enhance } from '$app/forms';
  import { messages } from '$lib/i18n';
  let { data, form } = $props();
  const t = $derived(messages[data.lang]);
  const error = $derived(form?.error as keyof typeof t | undefined);
</script>

<svelte:head><title>{t.signIn} · {t.appName}</title></svelte:head>

<main class="login">
  <form class="card box" method="POST" use:enhance>
    <span class="mark"><svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="9" r="4" /><circle cx="16" cy="13" r="4" /></svg></span>
    <h1>{t.appName}</h1>
    {#if form?.mfa}<p class="muted">{t.mfaHint}</p>{:else if error}<p class="error" role="alert">{t[error] ?? t.error}</p>{/if}
    <label>{t.email}<input name="email" type="email" autocomplete="username" required value={form?.email ?? ''} /></label>
    <label>{t.password}<input name="password" type="password" autocomplete="current-password" required /></label>
    {#if form?.mfa}
      <label>{t.mfaCode}<input name="mfa_code" inputmode="numeric" autocomplete="one-time-code" /></label>
    {/if}
    <button class="primary">{t.signIn}</button>
  </form>
</main>
