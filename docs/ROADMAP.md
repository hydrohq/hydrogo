# HydroGo uygulama roadmap'i

**Tarih:** 2026-09-08 · **Durum:** R0, R1 S03 ve R2 tamamlandı; R3 güvenlik başlangıç kapısı tamamlandı, ticari pilot kapıları açık.

Bu dosya iş sırasını ve teslim kapılarını izler. Mimari hükümler ADR'lerde, ayrıntılı kapsam [ana planda](implementation_plan.md), deney tanımları [hazırlık incelemesindedir](PRE_IMPLEMENTATION_REVIEW.md). Takvim tahmini ilk teknik deneyler tamamlandıktan sonra yapılır. Bir kapı yalnız kod revizyonu ve doğrulama raporuyla kapanır.

## R0 — Depo ve kararların hazırlanması

- [x] Taslak route/publish ve silme geçmişiyle restore sınırını ADR-0014 ile tanımla.
- [x] SEO/redirects'i pilot kapsamına yerleştir; ilk dilimde route kimliği ve publish atomikliği kalsın.
- [x] Compact pilot testlerini L2/Distributed testlerinden ayır.
- [x] Başlangıç klasörlerini, README ve katkı akışını oluştur.

Çıktı: geliştirmeye hazır kaynak yerleşimi ve bu roadmap. Çalışan uygulama veya SDK içermez.

## R1 — Teknik deneyler ve çalışan iskelet

**Bağımlılık:** R0. **Sorumluluk:** Core/SDK, DB, UI, build.

- [x] GitHub namespace'i `github.com/ardaozyildirim/hydrogo` üzerinden ilk `go.mod` oluştur; Go/toolchain sürümünü seç ve sabitle.
- [x] S01: dış Go module tüketici fixture'ı, izinli import sınırı ve seçilmeyen modülün build'e girmemesi.
- [x] S02/S04: SQLite/PostgreSQL driver ve migration deneyi; tek transaction scope ile content/audit/outbox rollback.
- [x] S03: Templ, Quill, HTMX ve gerekli ise Alpine CSP uyumu; iki küçük tema ve upload deneyi. (Yerel, sürümlenmiş asset'ler; nonce'lu CSP fixture'ı; sanitizer, medya ve harici edition build kanıtları tamam.)
- [x] Windows geliştirme/Linux amd64 build; backup ve image encoder araçlarının bulunabilirliği.
- [x] Tek `hydro` girişi, config doğrulama, health/readiness, graceful shutdown, default-deny router, bootstrap/session ve CSRF/audit temeli.
- [x] Go araçları seçildikten sonra CI format/vet/test/build ve iki DB servis matrisini ekle. (CI yeşil; testler ve dialect matrisi çalışıyor.)

**Çıkış:** Temiz ortamda build ve giriş; iki DB rollback; dış tüketici build; CSP deneyi. Bağımlılık sürümleri ve alternatifler `docs/validation/` altında kayıtlı. SDK v0 kalır.

## R2 — Pages ve ilk müşteri teması

**Bağımlılık:** R1. **Sorumluluk:** Pages, auth, media, UI.

İlk dikey dilim push edildi: Pages draft/revision/publish service’i, protected JSON route’ları, public published route, bounded process-local cache ve medya upload ile `pending → ready` metadata akışı hazır. Preview, admin UI ve tam çıkış kanıtları açık.

- [x] UUID/locale/revision, slug ve e-posta normalizasyonunu ilk kalıcı şemadan önce fixture ile sabitle. (Pages contract fixture’ı locale/slug/revision ve normalized route davranışını kapsıyor.)
- [x] Pages oluştur/düzenle/taslak/preview/publish; admin ve JSON aynı service/policy üzerinden çalışsın. (Protected JSON ve CSRF’li admin form aynı ContentService’i kullanıyor.)
- [x] Optimistic locking, kayıt/alan yetkisi ve iki editör conflict davranışı. (Stale editor conflict ve editor/admin mutation policy contract içinde doğrulanıyor.)
- [x] ADR-0014: taslak URL canlıyı değiştirmesin; publish route/generation/audit/outbox atomik olsun. (Draft/published projection ve duplicate route conflict contract testleri geçiyor.)
- [x] Local medya pending/ready, private/public, sanitizer ve immutable revision gövdesi. (Upload metadata service’i ve sanitized revision contract testi hazır.)
- [x] İlk dış Client Edition ve tipli view slotları; Core fork veya internal import olmasın. (examples/client-edition ve import boundary CI kanıtı.)
- [x] Bounded yerel cache, public HTTP politikası ve kalıcı outbox; ilk dilim için zorunlu L2 yok.

**Çıkış:** İki DB'de aynı uçtan uca akış; yetki ve taslak/özel medya sızıntısı testleri; eşzamanlı edit/publish conflict ve crash rollback kanıtı. Kanıt: [R2 doğrulama raporu](validation/R2-pages-media-2026-09-08.md). Tam SEO, sitemap ve geçmiş URL yönetimi R3'tedir.

## R3 — Compact ticari pilot

**Bağımlılık:** R2. **Sorumluluk:** Site Edition, güvenlik, release/işletim.

R3 başlangıç commit’i: disabled user/session revoke, admin kontrollü user disable route’u, encrypted TOTP/recovery code, MFA-zorunlu login, hashed single-use password reset, request/confirm HTTP akışı, admin davet/accept akışı ve ayrı bounded login/MFA limiter temeli hazır. Compact SQLite snapshot/manifest ve DB dışı hash-zincirli silme niyeti günlüğü deneyleri başarılıdır. Kanıt: [R3 MFA/session doğrulama raporu](validation/R3-mfa-session-2026-09-08.md), [SQLite backup manifest deneyi](validation/R3-backup-manifest-2026-09-08.md) ve [dış silme günlüğü deneyi](validation/R3-erasure-log-2026-09-08.md). Üretim notifier adapter'ı, temiz ortam restore, PostgreSQL, silme replay ve Compact kapasite kanıtları açık.

- [x] Navigation, settings, SEO/canonical/Open Graph/sitemap, redirects ve sınırlı form inbox. (Store, SQLite contract, DB-level navigation position uniqueness, public SEO/sitemap, admin mutation, allowlist/rate-limited contact submit, inbox status, retention purge ve transaction içi audit hazır; kişisel veri erişim ve runbook politikası [PERSONAL_DATA_POLICY.md](PERSONAL_DATA_POLICY.md) ile tanımlı.)
- [x] MFA, davet/reset/kurtarma, rol/session iptali ve güvenlik fixture'ları. (TOTP/recovery, single-use reset, transactional invite accept, disabled-user session revoke ve login/MFA limiter contract'ları tamam; production notifier ve çoklu-instance limiter R3 işletim kapısında.)
- [ ] S05-C: Compact `all` ve blue/green geçişinde stale refill, kayıp invalidation, cache bypass/revalidation ve bounded DB yükü. (Local public cache generation guard, generation değişiminde in-flight bypass/revalidation, aynı anahtar için single-flight miss, 64 concurrent lookup → 1 repository call contract'ı ve stale refill reddi hazır; gerçek blue/green trafik ve profil kapasite ölçümü açık.) [Cache doğrulama raporu](validation/R3-cache-generation-2026-09-08.md). L2 kesintisi bu kapının şartı değildir.
- [ ] S06: desteklenecek tek sunucu topolojisinde online migration, eski binary restart, yeni yazımdan sonra rollback ve retained asset testi. (SQLite temiz kurulum/restart/no-op/checksum ve retained media restore contract'ları hazır; online DDL ve yeni yazımdan sonra rollback tatbikatı açık.) [Migration doğrulama raporu](validation/R3-migration-restart-2026-09-08.md)
- [ ] S07: DB+medya restore, session iptali, job/egress karantinası ve eski deployment'ın yazamaması. (Compact SQLite staging restore + kontrol marker'ı hazır; PostgreSQL, gerçek session/job karantinası ve eski deployment testi açık.) Her iki DB restore deneyi kaydedilsin; satılan profil ayrıca sertifikalansın.
- [ ] ADR-0014 için dış silme günlüğü adapter'ı/checkpoint prototipi; backup sonrası silme, crash, eksik günlük ve retention testleri. (Hash-zincirli JSONL, idempotent append ve doğrulanmış checkpoint'ten hata sonrası retry replay hazır; gerçek Pages/media adapter'ı, retention ve dış sağlayıcı tatbikatı açık.)
- [ ] Kaynak limitleri, `max_stale`, session süreleri, RPO/RTO, retention ve rollback penceresini ölç ve sabitle.
- [ ] Runbook, alarmlar, dış yedek/key kurtarma; müşteri teslim/lisans/destek koşullarını satış öncesi karara bağla. (İlk operasyon runbook'u [OPERATIONS_RUNBOOK.md](OPERATIONS_RUNBOOK.md) eklendi; gerçek temiz ortam tatbikatı, alarmlar ve dış key/yedek kanıtı açık.)

**Çıkış:** Bir gerçek müşteri işi, güvenlik/restore ve ölçülmüş Compact kapasite kanıtı. Her runbook temiz ortamda uygulanmış. SDK hâlâ v0; Distributed satış kapsamı değil.

## R4 — News ve SDK doğrulaması

**Bağımlılık:** R3. **Sorumluluk:** News, SDK, DB/release.

- [x] News inceleme/zamanlama/yayın/geri çekme; sabit revision, schedule generation ve yeniden yetki kontrolü. (News migration, domain/repository, SQLite schedule generation contract'ı, worker lease/idempotency, JSON/public route'ları ve CSRF'li protected admin HTML formları hazır.) [R4 doğrulama raporu](validation/R4-news-admin-2026-09-08.md)
- [x] İkinci dış Client Edition, ikinci tema ve farklı özel iş kuralı. (examples/news-edition ayrı module, EditorialTheme, importguard/build CI adımı ve Core'a bağımlı olmayan `EditorialDraft` onay/desk kuralı tamam.) [R4 doğrulama raporu](validation/R4-news-admin-2026-09-08.md)
- [x] İki modülde transaction, hook, event/job lease ve idempotency contract kitleri. (Pages/News transaction-audit/outbox desenleri, bounded queue, SQLite/PostgreSQL uyumlu kalıcı `jobs` lease store, News durable queue adapter'ı, due scheduler discovery, production worker bootstrap ve bounded idempotent `internal/hooks.Dispatcher` hazır.) [Job store doğrulama raporu](validation/R4-job-store-2026-09-08.md)
- [x] Core/modül/schema/config/content/event/job/asset uyumluluk matrisi ve eski müşteri fixture'ları. (Kanıtlanmış epoch/blue-green sınırları [COMPATIBILITY_MATRIX.md](COMPATIBILITY_MATRIX.md) içinde toplandı; migration 008 seviyesinden tam şemaya legacy müşteri upgrade fixture'ı, published projection backfill'i, medya ve auth hash korunumu contract test ile doğrulandı.)
- [x] Kanıtlanan küçük SDK yüzeyini v1 adayı olarak belirle; generator için kalıpları çıkar. (`extension/v1` altında Theme/Module typed contract'ı ve açık candidate version sabitlendi.) [SDK doğrulama raporu](validation/R5-sdk-v1-candidate-2026-09-08.md)

**Çıkış:** İki modül + iki ayrı Edition Core fork olmadan çalışır; SDK v1 adayı incelenir. S08'in SDK kısmı kapanır; generator kısmı R5'te kapanır.

## R5 — Generator ve kararlı v1

**Bağımlılık:** R4. **Sorumluluk:** DX/SDK, veri, release.

- [x] Deterministik CRUD üretici, generated/custom koruması, deny varsayılanı ve incelenecek dialect migration taslağı. (Collection spec generator, gerçek CRUD repository/transport üretimi, secret/destructive deny, Pages/News derlenebilir fixture'ları, published migration immutability ve reviewable dialect migration skeleton hazır.) [Generator doğrulama raporu](validation/R5-generator-2026-09-08.md)
- [x] S08 tamamı: iki kez üretimde sıfır diff, custom hash korunması, published migration değişmezliği. (Pages/News fixture'larında deterministic output, custom preservation, CRUD derleme ve published migration immutability contract'ları geçti.)
- [ ] S09: SQLite ↔ PostgreSQL export/import; kimlik/revision/locale/ilişkiler korunur, secret/session dışarı çıkmaz. (Allowlist tabanlı deterministic SQLite ve geçici schema'lı PostgreSQL round-trip contract'ları, auth/session dışlama ve 256 kayıtlı büyük SQLite fixture'ı hazır; gerçek müşteri benchmark'ı açık.) [R5 doğrulama raporu](validation/R5-portable-export-2026-09-08.md)
- [ ] Standard/PostgreSQL profilini, desteklenen OS/DB/provider sürümlerini ve upgrade yolunu sertifikala.
- [ ] SDK v1, deprecation/destek takvimi, release imzası, lisans envanteri ve yeniden üretilebilir build. (SDK v1 adayı, deterministic Ed25519 manifest/signature contract'ı ve aynı CI runner/toolchain üzerinde gerçek binary reproducibility kontrolü hazır; resmi deprecation takvimi, production key ceremony, provenance attestation ve çapraz OS reproducibility açık.) [Release doğrulama raporu](validation/R5-release-manifest-2026-09-08.md)
- [ ] Standart müşteri işini iki geliştirici/müşteri üzerinde mevcut yöntemle karşılaştırarak süre ölç.

**Çıkış:** Belgeli destek matrisiyle kararlı v1. Ölçülmeyen hız/topoloji için vaat verilmez.

## R6 — Talebe bağlı genişleme

**Bağımlılık:** İlgili v1 sözleşmeleri ve doğrulanmış müşteri ihtiyacı.

- [ ] S05-D: çoklu web/worker, L2 kaybı, fan-out/reconnect, generation uzlaştırması ve bounded fallback.
- [ ] S3 uyumlu medya/CDN, online büyük yedek ve GC pin yarışları; Distributed yük/arıza sertifikasyonu.
- [ ] Talep varsa yeni DB adapter'ı, gelişmiş arama, OIDC/SAML veya özel mobil API; her biri ayrı karar ve test kapısıyla.
- [ ] Commerce, AI ve multi-tenant ayrı kapsam/ADR ile değerlendirilir.

## Kanıt ve ilerleme kaydı

Her çalışma küçük incelenebilir değişiklik olarak teslim edilir. Raporda commit, ortam/toolchain/DB sürümleri, komutlar, beklenen/görülen sonuç, sınırlamalar ve ilgili roadmap maddesi bulunur. `docs/validation/` yalnız gerçek sonuç taşır; planlanan test başarılı gösterilmez. Başarısız deneyde önce alternatif değerlendirilir; kabul edilmiş karar değişiyorsa yeni ADR yazılır.
