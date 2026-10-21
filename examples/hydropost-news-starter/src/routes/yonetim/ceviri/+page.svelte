<script lang="ts">
  import { enhance } from '$app/forms';
  import type { PageProps } from './$types';

  let { data, form }: PageProps = $props();

  let password = $state('');
  let newsId = $state('');
  let targetLocale = $state('en');
  let submitting = $state(false);
</script>

<svelte:head>
  <title>Çeviri Paneli</title>
  <meta name="robots" content="noindex, nofollow" />
</svelte:head>

<div class="py-10 max-w-xl mx-auto px-4">
  <h1 class="text-xl font-bold text-[var(--hp-ink)] mb-1">Haber Çeviri Paneli</h1>
  <p class="text-xs text-[var(--hp-muted)] mb-6">
    Bir haberin kimliğini (ID) girerek AI destekli çeviri akışını tetikleyin. Çeviri,
    <code class="px-1 rounded bg-[var(--hp-surface-sunken)]">/api/translate</code> üzerinden
    HydroGo admin API'sine yazılır.
  </p>

  {#if !data.configured}
    <div class="p-4 rounded-xl bg-amber-500/10 border border-amber-500/20 text-xs text-amber-800">
      Sunucuda <code>TRANSLATE_PANEL_PASSWORD</code> tanımlı değil. Bu paneli kullanmak için
      <code>.env</code> dosyasına bir değer ekleyin.
    </div>
  {:else if !data.authed}
    <form
      method="POST"
      action="?/login"
      use:enhance={() => {
        submitting = true;
        return async ({ update }) => {
          submitting = false;
          await update();
        };
      }}
      class="p-6 rounded-2xl bg-[var(--hp-surface)] border border-[var(--hp-border)] space-y-4"
    >
      <div>
        <label for="password" class="block text-xs font-semibold text-[var(--hp-ink)] mb-1">Panel şifresi</label>
        <input
          id="password"
          name="password"
          type="password"
          required
          bind:value={password}
          class="w-full px-3.5 py-2.5 rounded-xl border border-[var(--hp-border-strong)] bg-[var(--hp-surface)] text-sm text-[var(--hp-ink)] outline-none focus:border-[var(--hp-primary)] transition-colors"
        />
      </div>
      {#if form?.error}
        <p class="text-xs text-red-600">{form.error}</p>
      {/if}
      <button
        type="submit"
        disabled={submitting}
        class="px-6 py-2.5 rounded-xl bg-[var(--hp-primary)] hover:bg-[var(--hp-primary-dark)] text-white text-sm font-bold transition-colors disabled:opacity-50 cursor-pointer"
      >
        Giriş yap
      </button>
    </form>
  {:else}
    <form
      method="POST"
      action="?/translate"
      use:enhance={() => {
        submitting = true;
        return async ({ update }) => {
          submitting = false;
          await update();
        };
      }}
      class="p-6 rounded-2xl bg-[var(--hp-surface)] border border-[var(--hp-border)] space-y-4"
    >
      <div>
        <label for="id" class="block text-xs font-semibold text-[var(--hp-ink)] mb-1">Haber ID</label>
        <input
          id="id"
          name="id"
          type="text"
          required
          bind:value={newsId}
          placeholder="ör. 9f1c2e..."
          class="w-full px-3.5 py-2.5 rounded-xl border border-[var(--hp-border-strong)] bg-[var(--hp-surface)] text-sm text-[var(--hp-ink)] outline-none focus:border-[var(--hp-primary)] transition-colors"
        />
      </div>

      <div>
        <label for="targetLocale" class="block text-xs font-semibold text-[var(--hp-ink)] mb-1">Hedef dil</label>
        <select
          id="targetLocale"
          name="targetLocale"
          bind:value={targetLocale}
          class="w-full px-3.5 py-2.5 rounded-xl border border-[var(--hp-border-strong)] bg-[var(--hp-surface)] text-sm text-[var(--hp-ink)] outline-none focus:border-[var(--hp-primary)] transition-colors"
        >
          <option value="en">English</option>
          <option value="de">Deutsch</option>
          <option value="fr">Français</option>
          <option value="ar">العربية</option>
        </select>
      </div>

      {#if form?.error}
        <div class="p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-xs text-red-700">
          {form.error}
          {#if form?.details}
            <pre class="mt-1 whitespace-pre-wrap break-all">{form.details}</pre>
          {/if}
        </div>
      {/if}

      {#if form?.success}
        <div class="p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-xs text-emerald-800">
          Çeviri {form.result?.action === 'updated' ? 'güncellendi' : 'oluşturuldu'} ({form.result?.locale}).
        </div>
      {/if}

      <button
        type="submit"
        disabled={submitting}
        class="px-6 py-2.5 rounded-xl bg-[var(--hp-primary)] hover:bg-[var(--hp-primary-dark)] text-white text-sm font-bold transition-colors disabled:opacity-50 cursor-pointer"
      >
        {submitting ? 'Çevriliyor…' : 'Çeviriyi Başlat'}
      </button>
    </form>

    <form method="POST" action="?/logout" class="pt-3">
      <button type="submit" class="text-xs text-[var(--hp-muted)] underline cursor-pointer">Çıkış yap</button>
    </form>
  {/if}
</div>
