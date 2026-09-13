# Strict Modular Monolith Uygulama Planı

Bu plan HydroGo’yu fiziksel microservice sistemine çevirmeden; Core, modüller ve
dış Client Edition’lar arasında sert, ölçülebilir ve gerektiğinde process sınırına
taşınabilir sözleşmeler oluşturur.

## Hedef

- Core yalnızca ortak altyapı ve versioned contract sağlar.
- Modüller kendi domain, service, repository, route, migration ve worker
  yaşam döngüsüne sahip olur.
- Modüller birbirlerinin repository veya SQL tablolarına doğrudan erişemez.
- İnaktif modül route, migration, asset, worker veya zorunlu kaynak oluşturmaz.
- Client Edition’lar Core’un internal paketlerine erişemez.
- İleride Commerce/Search gibi modüller process dışına alınabilecek capability ve
  event sınırlarına sahip olur.

## Aşamalar

1. **Mimari sözleşme:** ADR, dependency matrix, import allowlist ve yasak
   bağımlılık testleri.
2. **Core sınırları:** ortak hata, pagination, authorization, transaction, audit
   ve provider interface’lerini tekil Core sözleşmelerine toplama.
3. **Lifecycle/registry:** modül manifest’i, merkezi registry, dependency ve
   capability kaydı; inaktif modül fail-closed davranışı.
4. **Capability/event erişimi:** modüller arası concrete servis/repository/SQL
   çağrılarını küçük interface ve versioned event’lere taşıma.
5. **Migration sahipliği:** migration metadata’sına module/dependency/profile
   sahipliği ve SQLite/PostgreSQL uyumluluk kontrolleri ekleme.
6. **Runtime kaynak izolasyonu:** worker, queue, cache, DB, timeout ve media
   budget’larının modül bazında uygulanması.
7. **Admin/frontend izolasyonu:** aktif modül manifest’inden navigation,
   dashboard, command palette ve public route üretme.
8. **API/SDK izolasyonu:** aktif modüllerden OpenAPI/SDK üretme, legacy alias’ları
   merkezi router’da tutma ve contract consistency testleri.
9. **CI/test sertleştirmesi:** import, lifecycle, inactive resource, migration,
   capability, event, E2E ve blue/green contract testleri.
10. **Process split hazırlığı:** module-owned health, metrics, queue topic,
    config, serialization ve external capability adapter sözleşmeleri.

## Teslim kuralı

Her aşama dar kapsamlı değişiklik, ilgili test ve gerekiyorsa ADR/validation
notuyla tamamlanır. Aşama yeşil olmadan sonraki aşamaya geçilmez. Gerçek
müşteri kapasitesi, reverse proxy, external backup/key escrow, provider
credentials ve production signing gerektiren doğrulamalar yerel başarı olarak
işaretlenmez; production gate olarak kalır.
