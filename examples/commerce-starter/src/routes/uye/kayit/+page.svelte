<script lang="ts">
  import type { ActionData } from './$types';

  let { form }: { form: ActionData } = $props();
</script>

<svelte:head><title>Kayıt ol</title></svelte:head>

<h1>Hesap oluştur</h1>

{#if form?.message}<p class="form-error">{form.message}</p>{/if}

<form method="POST" class="checkout-form">
  <div>
    <label for="email">E-posta</label>
    <input id="email" name="email" type="email" required value={form?.email ?? ''} />
  </div>
  <div>
    <label for="password">Şifre</label>
    <input id="password" name="password" type="password" required minlength="10" />
  </div>
  <div>
    <label for="password_confirm">Şifre (tekrar)</label>
    <input id="password_confirm" name="password_confirm" type="password" required minlength="10" />
  </div>
  <button class="btn-primary" type="submit">Hesap oluştur</button>
</form>

<p class="meta">Zaten hesabınız var mı? <a href="/uye/giris">Giriş yapın</a>.</p>
