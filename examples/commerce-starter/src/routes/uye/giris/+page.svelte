<script lang="ts">
  import type { ActionData } from './$types';

  let { form }: { form: ActionData } = $props();
</script>

<svelte:head><title>Giriş yap</title></svelte:head>

<h1>Giriş yap</h1>

{#if form?.message}<p class="form-error">{form.message}</p>{/if}

<form method="POST" class="checkout-form">
  <div>
    <label for="email">E-posta</label>
    <input id="email" name="email" type="email" required value={form?.email ?? ''} />
  </div>
  <div>
    <label for="password">Şifre</label>
    <input id="password" name="password" type="password" required />
  </div>
  <button class="btn-primary" type="submit">Giriş yap</button>
</form>

<p class="meta">Hesabınız yok mu? <a href="/uye/kayit">Kayıt olun</a>.</p>
