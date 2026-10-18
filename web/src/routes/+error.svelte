<script lang="ts">
  import { page } from '$app/state';
</script>

<svelte:head>
  <title>{page.status} — Sayfa Bulunamadı — HydroPost</title>
</svelte:head>

<div class="max-w-xl mx-auto my-20 p-8 sm:p-12 rounded-3xl bg-zinc-950 border border-zinc-800 text-center space-y-5 shadow-2xl">
  <div class="inline-flex items-center justify-center w-20 h-20 rounded-2xl bg-red-600/10 border border-red-500/20 text-red-500 font-black text-4xl">
    {page.status}
  </div>
  <h1 class="text-2xl sm:text-3xl font-extrabold text-white">
    {page.status === 404 ? 'Aradığınız Haber Bulunamadı' : 'Bir Hata Oluştu'}
  </h1>
  <p class="text-sm text-zinc-400 max-w-md mx-auto leading-relaxed">
    {page.error?.message || 'İstediğiniz sayfa kaldırılmış, adı değiştirilmiş veya geçici olarak kullanım dışı olabilir.'}
  </p>
  <div class="pt-4 flex items-center justify-center gap-3">
    <a
      href="/"
      class="px-6 py-2.5 rounded-xl bg-red-600 hover:bg-red-500 text-white text-xs font-bold transition-all shadow-lg shadow-red-600/30 hover:scale-105"
    >
      Ana Sayfaya Dön
    </a>
  </div>
</div>
