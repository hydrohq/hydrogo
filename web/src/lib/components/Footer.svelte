<script lang="ts">
  const currentYear = new Date().getFullYear();
</script>

<footer class="mt-16 bg-zinc-950 border-t border-zinc-800 text-zinc-400 text-xs">
  <div class="max-w-7xl mx-auto px-4 py-12 grid grid-cols-1 md:grid-cols-4 gap-8">
    <div class="space-y-3 md:col-span-1">
      <div class="flex items-center gap-2">
        <div class="w-7 h-7 rounded-lg bg-red-600 flex items-center justify-center font-black text-white text-sm">
          H
        </div>
        <span class="text-lg font-black text-white tracking-tight">HYDROPOST</span>
      </div>
      <p class="text-zinc-500 leading-relaxed text-[11px]">
        HydroGo motoru üzerinde çalışan bağımsız, ultra hızlı ve güvenilir haber portalı.
      </p>
    </div>

    <div class="space-y-2">
      <h4 class="text-white font-bold uppercase tracking-wider text-xs">Kategoriler</h4>
      <ul class="space-y-1.5 text-zinc-400">
        <li><a href="/gundem" class="hover:text-white transition-colors">Gündem</a></li>
        <li><a href="/dunya" class="hover:text-white transition-colors">Dünya</a></li>
        <li><a href="/ekonomi" class="hover:text-white transition-colors">Ekonomi</a></li>
        <li><a href="/teknoloji" class="hover:text-white transition-colors">Teknoloji</a></li>
        <li><a href="/spor" class="hover:text-white transition-colors">Spor</a></li>
      </ul>
    </div>

    <div class="space-y-2">
      <h4 class="text-white font-bold uppercase tracking-wider text-xs">Medya & İçerik</h4>
      <ul class="space-y-1.5 text-zinc-400">
        <li><a href="/yazarlar" class="hover:text-white transition-colors">Köşe Yazarları</a></li>
        <li><a href="/galeri" class="hover:text-white transition-colors">Foto Galeriler</a></li>
        <li><a href="/videolar" class="hover:text-white transition-colors">Video Haberler</a></li>
        <li><a href="/rss.xml" class="hover:text-white transition-colors flex items-center gap-1">RSS Beslemesi ↗</a></li>
      </ul>
    </div>

    <div class="space-y-2">
      <h4 class="text-white font-bold uppercase tracking-wider text-xs">Kurumsal</h4>
      <ul class="space-y-1.5 text-zinc-400">
        <li><a href="/kunye" class="hover:text-white transition-colors">Künye</a></li>
        <li><a href="/iletisim" class="hover:text-white transition-colors">İletişim & Reklam</a></li>
        <li><a href="/gizlilik" class="hover:text-white transition-colors">Gizlilik & Çerez Politikası</a></li>
        <li><a href="/yayin-ilkeleri" class="hover:text-white transition-colors">Basın Meslek İlkeleri</a></li>
      </ul>
    </div>
  </div>

  <div class="border-t border-zinc-900 py-4 px-4 text-center text-zinc-600 text-[11px]">
    © {currentYear} HydroPost Haber Portalı. Tüm hakları saklıdır. Zero-build & High Performance Go CMS.
  </div>
</footer>
