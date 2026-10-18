<script lang="ts">
  const categories = [
    { name: 'Ana Sayfa', href: '/' },
    { name: 'Gündem', href: '/gundem' },
    { name: 'Dünya', href: '/dunya' },
    { name: 'Ekonomi', href: '/ekonomi' },
    { name: 'Siyaset', href: '/siyaset' },
    { name: 'Teknoloji', href: '/teknoloji' },
    { name: 'Spor', href: '/spor' },
    { name: 'Yazarlar', href: '/yazarlar' },
    { name: 'Resmi İlanlar', href: '/resmi-ilanlar' },
    { name: 'Galeriler', href: '/galeri' },
  ];

  const currentDate = new Date().toLocaleDateString('tr-TR', {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
</script>

<header class="w-full bg-black border-b border-zinc-800 select-none">
  <!-- Topmost Utility Bar -->
  <div class="border-b border-zinc-900 bg-zinc-950/80 px-4 py-1.5 text-xs text-zinc-400">
    <div class="max-w-7xl mx-auto flex items-center justify-between">
      <div class="flex items-center gap-4">
        <span class="font-medium text-zinc-300">{currentDate}</span>
        <div class="hidden md:flex items-center gap-3 border-l border-zinc-800 pl-4 text-[11px] font-mono">
          <span class="text-zinc-300">USD/TRY: <b class="text-emerald-400">34.25</b></span>
          <span class="text-zinc-300">EUR/TRY: <b class="text-emerald-400">37.10</b></span>
          <span class="text-zinc-300">BIST 100: <b class="text-emerald-400">9.120</b></span>
        </div>
      </div>

      <div class="flex items-center gap-3">
        <a href="/en" class="hover:text-white transition-colors text-[11px] font-bold">English</a>
        <span class="text-zinc-700">|</span>
        <a href="/admin" target="_blank" class="text-red-400 hover:text-red-300 text-[11px] font-semibold">
          Yönetici Paneli ↗
        </a>
      </div>
    </div>
  </div>

  <!-- Main Branding Row -->
  <div class="px-4 py-4 md:py-6 max-w-7xl mx-auto flex items-center justify-between">
    <div class="flex items-center gap-3">
      <a href="/" class="flex items-center gap-2.5 group">
        <div class="w-10 h-10 rounded-xl bg-gradient-to-tr from-red-600 to-rose-500 flex items-center justify-center font-black text-white text-xl shadow-lg shadow-red-600/30 group-hover:scale-105 transition-transform">
          H
        </div>
        <div class="flex flex-col">
          <span class="text-2xl sm:text-3xl font-black text-white tracking-tight leading-none">
            HYDRO<span class="text-red-500">POST</span>
          </span>
          <span class="text-[9px] uppercase tracking-widest text-zinc-400 font-bold mt-0.5">
            Bağımsız Haber Portalı
          </span>
        </div>
      </a>
    </div>

    <!-- Quick Search or Live Status -->
    <div class="hidden sm:flex items-center gap-2">
      <span class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs font-semibold">
        <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
        Canlı Yayın & Haber Akışı
      </span>
    </div>
  </div>

  <!-- Primary Navigation Categories -->
  <nav class="bg-zinc-950 border-t border-zinc-800/80 px-4">
    <div class="max-w-7xl mx-auto flex items-center gap-1 overflow-x-auto scrollbar-none py-1">
      {#each categories as cat}
        <a
          href={cat.href}
          class="px-3.5 py-2 rounded-lg text-xs md:text-sm font-bold text-zinc-300 hover:text-white hover:bg-zinc-800/60 whitespace-nowrap transition-colors"
        >
          {cat.name}
        </a>
      {/each}
    </div>
  </nav>
</header>
