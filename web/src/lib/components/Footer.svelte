<script lang="ts">
  const currentYear = new Date().getFullYear();
</script>

<footer class="mt-16 bg-zinc-50 border-t border-zinc-200 text-zinc-600 text-xs">
  <div class="max-w-7xl mx-auto px-4 py-12 grid grid-cols-1 md:grid-cols-4 gap-8">
    <div class="space-y-3 md:col-span-1">
      <div class="flex items-center gap-2">
        <div class="w-7 h-7 rounded-lg bg-red-600 flex items-center justify-center font-black text-white text-sm">
          H
        </div>
        <span class="text-lg font-black text-zinc-900 tracking-tight">HYDROPOST</span>
      </div>
      <p class="text-zinc-500 leading-relaxed text-[11px]">
        HydroGo motoru üzerinde çalışan bağımsız, ultra hızlı ve güvenilir haber portalı.
      </p>
    </div>

    <div class="space-y-2">
      <h4 class="text-zinc-900 font-bold uppercase tracking-wider text-xs">Kategoriler</h4>
      <ul class="space-y-1.5 text-zinc-600">
        <li><a href="/gundem" class="hover:text-red-600 transition-colors">Gündem</a></li>
        <li><a href="/dunya" class="hover:text-red-600 transition-colors">Dünya</a></li>
        <li><a href="/ekonomi" class="hover:text-red-600 transition-colors">Ekonomi</a></li>
        <li><a href="/teknoloji" class="hover:text-red-600 transition-colors">Teknoloji</a></li>
        <li><a href="/spor" class="hover:text-red-600 transition-colors">Spor</a></li>
      </ul>
    </div>

    <div class="space-y-2">
      <h4 class="text-zinc-900 font-bold uppercase tracking-wider text-xs">Medya & İçerik</h4>
      <ul class="space-y-1.5 text-zinc-600">
        <li><a href="/yazarlar" class="hover:text-red-600 transition-colors">Köşe Yazarları</a></li>
        <li><a href="/galeri" class="hover:text-red-600 transition-colors">Foto Galeriler</a></li>
        <li><a href="/videolar" class="hover:text-red-600 transition-colors">Video Haberler</a></li>
        <li><a href="/rss.xml" class="hover:text-red-600 transition-colors flex items-center gap-1">RSS Beslemesi ↗</a></li>
      </ul>
    </div>

    <div class="space-y-2">
      <h4 class="text-zinc-900 font-bold uppercase tracking-wider text-xs">Kurumsal</h4>
      <ul class="space-y-1.5 text-zinc-600">
        <li><a href="/kunye" class="hover:text-red-600 transition-colors">Künye</a></li>
        <li><a href="/iletisim" class="hover:text-red-600 transition-colors">İletişim & Reklam</a></li>
        <li><a href="/gizlilik" class="hover:text-red-600 transition-colors">Gizlilik & Çerez Politikası</a></li>
        <li><a href="/yayin-ilkeleri" class="hover:text-red-600 transition-colors">Basın Meslek İlkeleri</a></li>
      </ul>
    </div>
  </div>

  <div class="border-t border-zinc-200 py-4 px-4 text-center text-zinc-500 text-[11px]">
    © {currentYear} HydroPost Haber Portalı. Tüm hakları saklıdır. Zero-build & High Performance Go CMS.
  </div>
</footer>
