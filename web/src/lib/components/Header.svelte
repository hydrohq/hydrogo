<script lang="ts">
  const categories = [
    { name: 'Ana Sayfa', href: '/' },
    { name: 'Gündem', href: '/gundem' },
    { name: 'Dünya', href: '/dunya' },
    { name: 'Ekonomi', href: '/ekonomi' },
    { name: 'Teknoloji', href: '/teknoloji' },
    { name: 'Spor', href: '/spor' },
    { name: 'Yazarlar', href: '/#yazarlar' },
    { name: 'Resmi İlanlar', href: '/resmi-ilanlar' },
    { name: 'Nöbetçi Eczaneler', href: '/nobetci-eczaneler' },
    { name: 'Künye', href: '/kunye' },
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
      <a href="/" class="flex items-center gap-3 group">
        <div class="w-11 h-11 rounded-2xl bg-gradient-to-tr from-red-600 via-rose-500 to-amber-500 flex items-center justify-center font-black text-white text-2xl shadow-xl shadow-red-600/40 group-hover:scale-105 group-hover:rotate-1 transition-all duration-300">
          H
        </div>
        <div class="flex flex-col">
          <span class="text-2xl sm:text-3xl font-black text-white tracking-tight leading-none group-hover:text-zinc-100 transition-colors">
            HYDRO<span class="text-red-500">POST</span>
          </span>
          <span class="text-[9px] uppercase tracking-widest text-zinc-400 font-bold mt-1">
            Bağımsız Haber Portalı
          </span>
        </div>
      </a>
    </div>

    <!-- Quick Live Status -->
    <div class="hidden sm:flex items-center gap-2">
      <span class="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 text-xs font-semibold shadow-inner">
        <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
        Canlı Yayın & Haber Akışı
      </span>
    </div>
  </div>

  <!-- Primary Navigation Categories -->
  <nav class="bg-zinc-950 border-t border-zinc-800/80 px-4">
    <div class="max-w-7xl mx-auto flex items-center gap-1.5 overflow-x-auto scrollbar-none py-1.5">
      {#each categories as cat}
        <a
          href={cat.href}
          class="px-3.5 py-1.5 rounded-xl text-xs md:text-sm font-bold text-zinc-300 hover:text-white hover:bg-zinc-800/90 hover:border-zinc-700 border border-transparent whitespace-nowrap transition-all duration-200"
        >
          {cat.name}
        </a>
      {/each}
    </div>
  </nav>
</header>
