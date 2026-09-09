# HydroGo Architecture Decision Backlog

**Güncelleme:** 2026-09-09. Karar durumu ile uygulama durumu ayrı izlenir. Bazı kararların uygulaması ve contract kanıtı mevcut olsa da gerçek ortam kapıları ayrıca açık olabilir; güncel durum [roadmap](../ROADMAP.md) ve doğrulama raporlarında tutulur. Kanıt planı [uygulama öncesi incelemede](../PRE_IMPLEMENTATION_REVIEW.md), aktif ADR dizini [README](README.md) içindedir.

## Kapanan temel kararlar

| Konu | Güncel cevap | Karar kaynağı |
|---|---|---|
| İlk ürün | Site Edition; Pages ilk dilim, News ikinci doğrulama modülü | [ADR-0009](0009-site-edition-and-delivery-scope.md) |
| Tenancy | Müşteri başına ayrı deployment/veri; ilk sürüm multi-tenant değil | [ADR-0001](0001-product-and-deployment-model.md); zaten kabul edilmişti |
| Veritabanı | İlk sertifikasyon SQLite/PostgreSQL; sonraki motorlar test kitiyle | [ADR-0006](0006-extensible-database-adapters.md) |
| Şema/özel kod | Build spec'inden standart CRUD; karmaşık iş akışı modül; runtime otomatik DDL yok | [ADR-0009](0009-site-edition-and-delivery-scope.md) |
| SDK/Edition | Dış Go module, dar extension, deneysel v0, iki modül/iki Edition sonrası v1 | [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md) |
| Generated/custom | Deterministik diff; custom ve yayımlanmış migration korunur | [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md) |
| Modül yaşam döngüsü | Seçili build, kontrollü kapatma, işler park edilir, veri otomatik silinmez | [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md) |
| Release manifest | Edition niyeti, dependency lock ve build kanıtı ayrıdır; sırlar harici | [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md) |
| Migration/rollback | Per-dialect metadata, schema ve payload uyumluluğu; binary rollback ayrı, restore ayrı | [ADR-0011](0011-consistency-upgrade-and-recovery.md) |
| Cache | L0–L3; instance fan-out/generation, stale bütçesi, bounded arıza fallback | [ADR-0003](0003-cache-architecture.md) + [ADR-0011](0011-consistency-upgrade-and-recovery.md) |
| Queue/outbox | DB-backed en az bir kez, transaction scope, lease/token, idempotency/karantina | [ADR-0011](0011-consistency-upgrade-and-recovery.md) |
| İçerik/locale | Tipli ilişkisel model, ayrı revision/locale, versioned rich-text | [ADR-0012](0012-content-media-and-portability.md) |
| Medya | Local ilk; S3 ilgili profil kapısı; immutable/pending-ready/private-public | [ADR-0012](0012-content-media-and-portability.md) |
| Export/arama | Mantıksal export; arama yeniden üretilebilir ve yetkiyle filtrelenen projection | [ADR-0012](0012-content-media-and-portability.md) |
| Auth/güvenlik | Local auth, DB session, pilot MFA, kayıt/alan yetkisi; dış kimlik ileride | [ADR-0013](0013-identity-and-security-boundaries.md) |

## Açık işler ve engelledikleri teslim

P0: ilk uygulama ayrıntısı; P1: pilot/v1 teslimi; P2: talep ile genişleme. Bu tablodaki açık işlerin hepsi ilk kodu engellemez.

| Öncelik | Konu | Öneri / karar için gereken | Kapı | Durum |
|---|---|---|---|---|
| P0 | Go module namespace | github.com/ardaozyildirim/hydrogo; go.mod/toolchain R1 deneyinde oluşturulur | İlk go.mod | Uygulandı; go.mod, CI ve import guard ile doğrulandı |
| P0 | Driver/toolchain/migration aracı | İki DB + Windows dev/Linux amd64 fixture; sürüm/build/lisans değerlendirmesi | Skeleton S02/S04 | Core/CI ve migration contract'ları uygulandı; provider sertifikasyonu ayrıca açık |
| P0 | SDK imzaları/transaction executor | Küçük registration ve transaction scope; geniş interface dondurulmasın | S01/S02, kararlılık S08 | Uygulandı; extension/v1 ve transaction contract'ları doğrulandı |
| P0 | UI bağımlılıkları/view slot | Templ + aday Quill/HTMX/Alpine CSP; iki tema ile doğrulama | S03 | Uygulandı; sürümlü asset, CSP ve tema contract'ları doğrulandı |
| P0 | Slug/email canonicalization | Normalize yardımcı alanlar; Unicode/case/locale fixture'ları; veri taşınabilirliği | İlk kalıcı şema | Detay açık |
| P1 | Sayısal kaynak/güvenlik/cache varsayılanları | Ölçülmüş limitler ve toplam max_stale; sessiz sonsuz varsayılan yok | Pilot/profil | Typed profile budgets, bounded cache/queue/body/upload limitleri uygulandı; gerçek kapasite ölçümü bekliyor |
| P1 | Restore ve RPO/RTO | İki DB gerçek restore; medya/key ve egress karantinası | Pilot S07 | Kanıt bekliyor |
| P1 | Restore silme geçmişi | ADR-0014 dış dayanıklı niyet günlüğü, güncel checkpoint, retention ve crash deneyi | Pilot R3 | Provider/protokol kanıtı bekliyor |
| P1 | DB/OS/provider destek sürümleri | Gerçek CI matrisi, backup/encoder gereksinimleri ve sertifikasyon | İlgili release | Açık |
| P1 | Lisans/çevrimdışı davranış | Öneri: public site süre dolunca kapanmaz; destek/güncelleme hakkı ayrı | Satış öncesi | Ticari karar açık |
| P1 | Kaynak kod/yeniden dağıtım | Müşteri teslimi ve bağımlılık lisansları açıkça tanımlansın | Satış öncesi | Ticari karar açık |
| P1 | LTS/deprecation/SLA | Aynı-major SDK ilkesi kabul; somut süreler/support penceresi seçilmeli | Kararlı v1/satış | Ticari karar açık |
| P1 | Marka/domain | HydroGo/HydroCore için ayrı uygunluk araştırması | Dış pazarlama | Doğrulanmadı |
| P1 | Taşınabilir export/import | İki DB round-trip; format/modül/secret kuralları | v1 S09 | Allowlist, SQLite/PostgreSQL round-trip ve CLI uygulandı; gerçek müşteri donanımı benchmark'ı bekliyor |
| P2 | Gelişmiş arama/object storage | Arayüz ilkeleri kabul; provider semantiği ve yük testi gerekli | İlgili özellik/Distributed | Uygulama bekliyor |
| P2 | OIDC/SAML/özel mobil API | Ayrı akış, threat model, token/revoke ve adapter ADR'si | Özellik açılmadan | Ertelendi |
| P2 | Fleet | İlk müşterilerde CI + sürüm envanteri; ihtiyaç kanıtlanınca kontrol paneli | Operasyon talebi | Ertelendi |
| P2 | Commerce | Ayrı edition ve para/stok/ledger/transaction kararları | Yeni ürün | Ertelendi |
| P2 | Multi-region/multi-tenant | İlk ürün kapsamı dışı; gerçek talep ve yeni karar gerektirir | Yeni topoloji | Ertelendi |

## Geliştirme hızı hedefi

Standart müşteri işi için edition başlatma, tema, collection, özel validation, test, build/deploy ve hata düzeltme süreleri ölçülür. 1–2 saat hedefi ancak tanımlı işte tekrarlı pilot ölçümüyle ürün iddiası olabilir. Hazır generator veya SDK varmış gibi süre vaadi verilmez.
