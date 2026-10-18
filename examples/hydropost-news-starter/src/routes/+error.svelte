<script lang="ts">
  import { page } from '$app/state';
</script>

<svelte:head>
  <title>{page.status} — Sayfa Bulunamadı — HydroPost</title>
</svelte:head>

<div class="max-w-xl mx-auto my-16 p-8 sm:p-12 rounded-3xl bg-[var(--hp-surface)] border border-[var(--hp-border)] text-center space-y-5 shadow-sm">
  <div class="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-[var(--hp-primary)]/10 border border-[var(--hp-primary)]/20 text-[var(--hp-primary)] font-black text-4xl">
    {page.status}
  </div>
  <h1 class="text-2xl sm:text-3xl font-extrabold text-[var(--hp-ink)] tracking-tight">
    {page.status === 404 ? 'Aradığınız Sayfa Bulunamadı' : 'Bir Hata Oluştu'}
  </h1>
  <p class="text-sm text-[var(--hp-muted)] max-w-md mx-auto leading-relaxed">
    {page.error?.message || 'İstediğiniz sayfa veya kategori mevcut değil, kaldırılmış ya da adı değiştirilmiş olabilir.'}
  </p>
  <div class="pt-4 flex items-center justify-center gap-3">
    <a
      href="/"
      class="px-6 py-2.5 rounded-xl bg-[var(--hp-primary)] hover:bg-[var(--hp-primary-dark)] text-white text-xs font-bold transition-all shadow-md shadow-red-600/20 hover:scale-105"
    >
      Ana Sayfaya Dön
    </a>
  </div>
</div>
