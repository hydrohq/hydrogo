# HydroGo uygulama roadmap'i

**Tarih:** 2026-09-08 · **Durum:** R0 ve R1 tamamlandı; R2 ilk dikey dilimde.

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
- [ ] S03: Templ, Quill, HTMX ve gerekli ise Alpine CSP uyumu; iki küçük tema ve upload deneyi. (Auth/audit ve upload akışı hazır; gerçek Quill/HTMX deneyi açık.)
- [x] Windows geliştirme/Linux amd64 build; backup ve image encoder araçlarının bulunabilirliği.
- [x] Tek `hydro` girişi, config doğrulama, health/readiness, graceful shutdown, default-deny router, bootstrap/session ve CSRF/audit temeli.
- [x] Go araçları seçildikten sonra CI format/vet/test/build ve iki DB servis matrisini ekle. (CI yeşil; testler ve dialect matrisi çalışıyor.)

**Çıkış:** Temiz ortamda build ve giriş; iki DB rollback; dış tüketici build; CSP deneyi. Bağımlılık sürümleri ve alternatifler `docs/validation/` altında kayıtlı. SDK v0 kalır.

## R2 — Pages ve ilk müşteri teması

**Bağımlılık:** R1. **Sorumluluk:** Pages, auth, media, UI.

İlk dikey dilim push edildi: Pages draft/revision/publish service’i, protected JSON route’ları, public published route, bounded process-local cache ve medya upload ile `pending → ready` metadata akışı hazır. Preview, admin UI ve tam çıkış kanıtları açık.

- [ ] UUID/locale/revision, slug ve e-posta normalizasyonunu ilk kalıcı şemadan önce fixture ile sabitle.
- [ ] Pages oluştur/düzenle/taslak/preview/publish; admin ve JSON aynı service/policy üzerinden çalışsın.
- [ ] Optimistic locking, kayıt/alan yetkisi ve iki editör conflict davranışı.
- [ ] ADR-0014: taslak URL canlıyı değiştirmesin; publish route/generation/audit/outbox atomik olsun.
- [ ] Local medya pending/ready, private/public, sanitizer ve immutable revision gövdesi.
- [ ] İlk dış Client Edition ve tipli view slotları; Core fork veya internal import olmasın.
- [ ] Bounded yerel cache, public HTTP politikası ve kalıcı outbox; ilk dilim için zorunlu L2 yok.

**Çıkış:** İki DB'de aynı uçtan uca akış; yetki ve taslak/özel medya sızıntısı testleri; eşzamanlı edit/publish conflict ve crash rollback kanıtı. Tam SEO, sitemap ve geçmiş URL yönetimi R3'tedir.

## R3 — Compact ticari pilot

**Bağımlılık:** R2. **Sorumluluk:** Site Edition, güvenlik, release/işletim.

- [ ] Navigation, settings, SEO/canonical/Open Graph/sitemap, redirects ve sınırlı form inbox.
- [ ] MFA, davet/reset/kurtarma, rol/session iptali ve güvenlik fixture'ları.
- [ ] S05-C: Compact `all` ve blue/green geçişinde stale refill, kayıp invalidation, cache bypass/revalidation ve bounded DB yükü. L2 kesintisi bu kapının şartı değildir.
- [ ] S06: desteklenecek tek sunucu topolojisinde online migration, eski binary restart, yeni yazımdan sonra rollback ve retained asset testi.
- [ ] S07: DB+medya restore, session iptali, job/egress karantinası ve eski deployment'ın yazamaması. Her iki DB restore deneyi kaydedilsin; satılan profil ayrıca sertifikalansın.
- [ ] ADR-0014 için dış silme günlüğü adapter'ı/checkpoint prototipi; backup sonrası silme, crash, eksik günlük ve retention testleri.
- [ ] Kaynak limitleri, `max_stale`, session süreleri, RPO/RTO, retention ve rollback penceresini ölç ve sabitle.
- [ ] Runbook, alarmlar, dış yedek/key kurtarma; müşteri teslim/lisans/destek koşullarını satış öncesi karara bağla.

**Çıkış:** Bir gerçek müşteri işi, güvenlik/restore ve ölçülmüş Compact kapasite kanıtı. Her runbook temiz ortamda uygulanmış. SDK hâlâ v0; Distributed satış kapsamı değil.

## R4 — News ve SDK doğrulaması

**Bağımlılık:** R3. **Sorumluluk:** News, SDK, DB/release.

- [ ] News inceleme/zamanlama/yayın/geri çekme; sabit revision, schedule generation ve yeniden yetki kontrolü.
- [ ] İkinci dış Client Edition, ikinci tema ve farklı özel iş kuralı.
- [ ] İki modülde transaction, hook, event/job lease ve idempotency contract kitleri.
- [ ] Core/modül/schema/config/content/event/job/asset uyumluluk matrisi ve eski müşteri fixture'ları.
- [ ] Kanıtlanan küçük SDK yüzeyini v1 adayı olarak belirle; generator için kalıpları çıkar.

**Çıkış:** İki modül + iki ayrı Edition Core fork olmadan çalışır; SDK v1 adayı incelenir. S08'in SDK kısmı kapanır; generator kısmı R5'te kapanır.

## R5 — Generator ve kararlı v1

**Bağımlılık:** R4. **Sorumluluk:** DX/SDK, veri, release.

- [ ] Deterministik CRUD üretici, generated/custom koruması, deny varsayılanı ve incelenecek dialect migration taslağı.
- [ ] S08 tamamı: iki kez üretimde sıfır diff, custom hash korunması, published migration değişmezliği.
- [ ] S09: SQLite ↔ PostgreSQL export/import; kimlik/revision/locale/ilişkiler korunur, secret/session dışarı çıkmaz.
- [ ] Standard/PostgreSQL profilini, desteklenen OS/DB/provider sürümlerini ve upgrade yolunu sertifikala.
- [ ] SDK v1, deprecation/destek takvimi, release imzası, lisans envanteri ve yeniden üretilebilir build.
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
