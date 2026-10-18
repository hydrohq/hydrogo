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

<header class="w-full bg-white border-b border-zinc-200 select-none shadow-xs">
  <!-- Topmost Utility Bar -->
  <div class="border-b border-zinc-200 bg-zinc-50 px-4 py-1.5 text-xs text-zinc-600">
    <div class="max-w-7xl mx-auto flex items-center justify-between">
      <div class="flex items-center gap-4">
        <span class="font-medium text-zinc-700">{currentDate}</span>
        <div class="hidden md:flex items-center gap-3 border-l border-zinc-200 pl-4 text-[11px] font-mono">
          <span class="text-zinc-600">USD/TRY: <b class="text-emerald-600">34.25</b></span>
          <span class="text-zinc-600">EUR/TRY: <b class="text-emerald-600">37.10</b></span>
          <span class="text-zinc-600">BIST 100: <b class="text-emerald-600">9.120</b></span>
        </div>
      </div>

      <div class="flex items-center gap-3">
        <a href="/en" class="hover:text-zinc-900 transition-colors text-[11px] font-bold text-zinc-600">English</a>
      </div>
    </div>
  </div>

  <!-- Main Branding Row -->
  <div class="px-4 py-4 md:py-5 max-w-7xl mx-auto flex items-center justify-between">
    <div class="flex items-center gap-3">
      <a href="/" class="flex items-center gap-3 group">
        <div class="w-10 h-10 rounded-xl bg-gradient-to-tr from-red-600 via-rose-500 to-amber-500 flex items-center justify-center font-black text-white text-xl shadow-md shadow-red-500/20">
          H
        </div>
        <div class="flex flex-col">
          <span class="text-2xl sm:text-3xl font-black text-zinc-900 tracking-tight leading-none">
            HYDRO<span class="text-red-600">POST</span>
          </span>
          <span class="text-[9px] uppercase tracking-widest text-zinc-500 font-bold mt-1">
            Bağımsız Haber Portalı
          </span>
        </div>
      </a>
    </div>

    <!-- Quick Live Status -->
    <div class="hidden sm:flex items-center gap-2">
      <span class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-50 border border-emerald-200 text-emerald-700 text-xs font-semibold">
        <span class="w-2 h-2 rounded-full bg-emerald-600 animate-pulse"></span>
        Canlı Yayın & Haber Akışı
      </span>
    </div>
  </div>

  <!-- Primary Navigation Categories -->
  <nav class="bg-white border-t border-zinc-200 px-4">
    <div class="max-w-7xl mx-auto flex items-center gap-1 overflow-x-auto scrollbar-none py-1">
      {#each categories as cat}
        <a
          href={cat.href}
          class="px-3 py-1.5 rounded-lg text-xs md:text-sm font-bold text-zinc-700 hover:text-red-600 hover:bg-zinc-50 whitespace-nowrap transition-colors"
        >
          {cat.name}
        </a>
      {/each}
    </div>
  </nav>
</header>
