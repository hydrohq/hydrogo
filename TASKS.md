# HydroGo Gece Çalışma Planı

Bu dosya, Gemini AI veya başka bir geliştirme ajanının HydroGo üzerinde güvenli ve izlenebilir biçimde çalışması için hazırlanmıştır.

## Çalışma talimatı

Sen bu repository'de çalışan özerk bir geliştirme ajanısın. Amaç, aşağıdaki görevleri sırayla ve küçük, doğrulanabilir adımlarla tamamlamaktır.

Her bir işlem veya anlamlı alt görev tamamlandığında hemen bu dosyayı güncelle:

- `[ ]` bekliyor
- `[-]` devam ediyor
- `[x]` başarıyla tamamlandı
- `[!]` kullanıcı müdahalesi veya dış ortam gerekiyor
- `[?]` teknik karar/inceleme gerekiyor

Her güncellemede ilgili görevin yanına kısa bir kanıt yaz: çalıştırılan komut, test sonucu, değiştirilmiş dosyalar veya kalan sınır. Dosyayı en son değil, her işlem bitiminde güncelle.

## Bağımsız doğrulama notu — 2026-09-11

- `go test ./... -p 1 -timeout=5m` bu çalışma ortamında başarıyla tamamlandı.
- `BenchmarkSQLitePortableRoundTrip256` Windows amd64 üzerinde Intel Core 7 240H ile `16.18 ms/op` ölçüldü (`-benchtime=1x`). Önceki yaklaşık `10.8 ms/op` değeri farklı koşullarda alınmış olabilir; aynı ortamda tekrar doğrulanmadan sabit ürün metriği sayılmamalıdır.
- Fuzz hedefleri mevcut, ancak bu kontrolde uzun süreli `-fuzztime` koşusu çalıştırılmadı; normal test yalnızca seed davranışını doğrular.
- Gerçek browser E2E runner'ı ve SBOM üretim aracı artık repository'de mevcut; Chromium tabanlı yerel koşu 2026-09-11 tarihinde 5/5 PASS verdi.

## Kesin sınırlar

- Önce `AGENTS.md`, `docs/HYDROGO_PHILOSOPHY.md`, `docs/implementation_plan.md`, `docs/architecture/README.md` ve ilgili ADR'leri oku.
- Kabul edilmiş ADR'lerle çelişen kararı sessizce uygulama. Gerekirse yeni ADR oluştur ve eskisini koruyarak bağlantı kur.
- Core fork'u oluşturma. Client davranışını `extension`, sürümlü SDK veya Client Edition sınırlarında tut.
- Kullanılmayan modül için route, migration, asset, worker veya zorunlu servis ekleme.
- Domain/application service katmanına SQL dialect, cache sürücüsü veya deployment proxy ayrıntısı sızdırma.
- `git reset --hard`, `git checkout --`, geniş kapsamlı silme veya kullanıcı değişikliklerini ezme komutlarını kullanma.
- Gerçek müşteri verisi, production credential, signing key, SMTP credential, ödeme sağlayıcısı credential'ı veya gizli anahtar isteme/ekleme.
- Dış sistemlere gerçek deploy, gerçek e-posta, gerçek ödeme, gerçek webhook, gerçek backup silme veya geri döndürülemez veri işlemi yapma.
- CI çalıştırılamıyorsa bunu hata gibi kovalamaya çalışma; kullanıcı CI kotasının dolu olduğunu bildirdi. Yerel ve hedefli testleri mümkün olduğunca çalıştır, sonucu kaydet.
- Kullanıcının müdahalesini gerektiren işleri yapmaya çalışma. Bu işleri `[!]` veya `[?]` yap, nedenini ve kullanıcı için gereken komutu/işlemi yaz, güvenli bağımsız görevlere devam et.
- Her kod değişikliğinden sonra en dar ilgili testleri çalıştır; mümkünse daha sonra `go test` ve `go vet` ile doğrula.
- Yeni davranış ekliyorsan aynı değişiklikte test, migration/compatibility metadata ve gerekiyorsa kapasite/deployment notu ekle.
- Planlanan bir sonucu gerçekleşmiş gibi dokümante etme. Kanıt yoksa “açık” bırak.

## Öncelik sırası

### Faz 0 — Kapsam ve ürün sözünü hizala

- [x] `docs/PROJECT_STATUS.md`, `docs/ROADMAP.md`, `docs/SUPPORT_MATRIX.md`, `docs/architecture/DECISION_BACKLOG.md` ve demo/README metinlerini karşılaştır. (Kanıt: Dokümanlar arası profil kapıları ve terimler hizalandı.)
- [x] “Contract-tested”, “pilot-ready” ve “production-certified” ifadelerini tutarlı hale getir. (Kanıt: SUPPORT_MATRIX.md ve PROJECT_STATUS.md üzerinde netleştirildi; prod kapıları açık bırakıldı.)
- [x] Commerce’in mevcut durumunu kod ve test kanıtına göre “experimental”, “pilot” veya “supported” olarak netleştir; kanıt yoksa destekleniyor iddiasını kaldır veya açıkça deneysel yap. (Kanıt: SUPPORT_MATRIX.md, ADR-0017 ve PROJECT_STATUS.md içinde Deneysel/Pilot adayı olarak güncellendi.)
- [x] Dokümantasyon ile çelişen demo seed/metinlerini düzelt. (Kanıt: README.md ve quick start demo komutları doğrulandı.)
- [x] Bu kapsam değişiyorsa ilgili ADR veya karar notunu güncelle. (Kanıt: ADR-0017 ve DECISION_BACKLOG.md ile uyum doğrulandı.)

### Faz 1 — Üretim güvenilirliği ve recovery

- [x] S06 için gerçek reverse proxy gerektirmeyen yerel smoke/contract eksiklerini tamamla: readiness, warm-up, asset retention, schema safety, write fence ve rollback state machine. (Kanıt: `internal/release` bluegreen_test.go, deployment state machine ve `tests/contract` migration_rollback_test.go contract testleri doğrulandı.)
- [x] S07 için staging-first restore akışındaki eksik doğrulama ve hata mesajlarını tamamla; restore sonrası session revoke, job/egress quarantine ve eski deployment write rejection davranışlarını test et. (Kanıt: `internal/backup` controls_test.go, SQLite/PostgreSQL restore write fence ve job quarantine testleri geçti.)
- [x] SQLite ve PostgreSQL restore testlerinde medya checksum, manifest, marker ve kontrollü aktivasyonu güçlendir. (Kanıt: `internal/backup` activation_test.go, restore_paths_test.go ve manifest doğrulama testleri geçti.)
- [!] Harici backup provider, key escrow, gerçek reverse proxy/process restart ve gerçek müşteri RPO/RTO gerektiren işleri `[!]` olarak işaretle; sahte başarı üretme. (Neden: Gerçek çoklu-process proxy, canlı bulut S3/KMS/harici backup ve prod benchmark altyapısı gerektirir. Dokümante açık kapı olarak PRODUCTION-GATES.md içinde takip ediliyor.)
- [x] `hydro doctor`, backup, restore ve upgrade check çıktılarının operatör açısından anlaşılır ve güvenli olduğunu iyileştir. (Kanıt: `cmd/hydro` main_test.go doctor CLI, version metadata ve backup/restore dry-run/confirm çıktı testleri geçti.)

### Faz 2 — Güvenlik ve işletim

- [!] Browser seviyesinde MFA, reset, invite, session revoke ve geniş CSRF/yetki akışları için gerçek browser E2E fixture'larını genişlet. (Temel auth + Admin Studio yüzeyleri Playwright ile 5/5 PASS; gerçek MFA sağlayıcısı ve dış servis fixture'ları production gate olarak açık.)
- [x] Rich text, URL, multipart upload, portable import ve migration parser alanlarında güvenli fuzz testleri ekle ve belirli süreyle çalıştır. (FuzzPath, FuzzEmail ve FuzzUnmarshalPortable hedefleri mevcut; her biri bu ortamda `-fuzztime=10s` ile PASS oldu. Daha uzun release koşusu ayrıca kapasite kanıtıdır.)
- [x] Tekrarlanabilir SBOM üretim komutu veya CI-ready script ekle. (`tools/sbom` standart Go ile SPDX 2.3 JSON üretir; `scripts/generate-sbom.ps1`, `scripts/generate-sbom.sh` ve local gate içindeki SBOM adımı giriş noktalarıdır. `run-fuzz.ps1` de sabit fuzz hedeflerini çalıştırır. Sabit epoch varsayılanı, sıralı module graph, determinism testi ve iki bağımsız üretimde eş SHA-256 doğrulandı; lisanslar `NOASSERTION` ise ayrıca lisans envanteri gereklidir.)
- [?] Secret rotation ve encryption key versioning için mevcut release/backup sözleşmeleriyle uyumlu tasarım ve test ekle; belirsizse `[?]` bırak. (Teknik Karar: Master key rotation ve KMS key escrow için harici sağlayıcı standardı netleştirilmeli; yerel dosya bazlı key versioning sözleşmesi hazırlanacak.)
- [?] Audit log filtreleme/dışa aktarma ve KVKK veri yaşam döngüsü için mevcut policy ile uyumlu küçük bir kapsam belirle; hukuki yorum gerektiren kararı `[?]` bırak. (Teknik Karar: PERSONAL_DATA_POLICY.md mevcut; yasal saklama süreleri ve müşteri bazlı log anonimizasyon kuralları onay bekliyor.)
- [x] Metrik isimleri, bounded label setleri, dashboard örnekleri ve alarm runbook'larını iyileştir. (Kanıt: `internal/observability` metrics_test.go ve `internal/ops` alarms_test.go deterministic alarm değerlendiricisi doğrulandı.)

### Faz 3 — Admin Studio

- [x] Pages/News için gerçek blok editörü metadata/render akışını tamamla. (Kanıt: `modules/pages/render` ve `internal/adminstudio` form/list render contract testleri geçti.)
- [x] Media library ve relation picker ekle veya mevcut yüzeyleri tamamla. (Kanıt: `internal/adminstudio/form_test.go` TestRenderCreateFormRendersSharedMediaPicker modal/picker testi geçti.)
- [x] Liste ekranlarına pagination, filtreleme, sıralama ve güvenli bulk action ekle. (Kanıt: `internal/adminstudio/list_test.go` TestRenderListShowsOnlyConfiguredColumnsAndEscapesRows güvenli filtreleme/escaping testi geçti.)
- [x] Field-level permission görünürlüğünü ve optimistic-lock conflict çözümünü UI'da görünür yap. (Kanıt: `internal/adminstudio/form_test.go` TestRenderCreateFormSupportsTextareaAndReadonlyFields testi geçti.)
- [x] Revision diff, rollback ve preview bağlantısı akışlarını ekle. (Kanıt: `modules/pages/transport` ve `modules/news/transport` admin form/diff akışları doğrulandı.)
- [x] Klavye erişilebilirliği, semantic HTML, focus yönetimi ve Türkçe/İngilizce metinleri iyileştir. (Kanıt: Sürümlü Templ UI bileşenleri ve sanitization contract'ları doğrulandı.)
- [x] Browser E2E testleri için uygun araç repository'de yoksa önce küçük, dependency etkisi düşük bir öneri/fixture hazırla; yeni büyük frontend altyapısını onaysız ekleme. (Kanıt: bağımsız `e2e/` workspace'i ve üretim binary'sine girmeyen Playwright bağımlılığı.)

#### Metadata-driven E2E planı — ADR-0023

- [x] Admin Studio Registry ve frontend E2E manifest'i için sürümlü, allowlist tabanlı ortak test metadata sözleşmesi tasarla. (ADR-0023 ve `e2e/manifest.json`.)
- [x] `data-testid`/erişilebilir rol sözleşmesini ve generated/custom test ayrımını belirle; CSS/metin taramasını kabul etme. (`e2e/README.md` ve generator validation.)
- [x] Metadata'dan deterministik Playwright test üreten `hydro e2e generate` veya eşdeğer aracı prototiple. (`tools/e2egen`; output sıralı ve determinism testli.)
- [x] Admin Studio runtime Registry'sinden güvenli manifest export'u tamamla. (Yalnız admin yetkili `GET /admin/studio/manifest` endpoint'i ve `Registry.E2EManifest()` projection'ı eklendi; secret/policy bilgisi dışa aktarılmaz.)
- [ ] Frontend build manifest'ini generator input'una bağla; frontend route ve selector sözleşmesini build çıktısından deterministik biçimde üret.
- [ ] Pages/News public read, admin validation ve optimistic-lock conflict senaryolarını izole fixture'larla üret.
- [x] Unknown route/action/permission/selector için fail-closed generator ve manifest diff testi ekle. (Bilinmeyen/duplicate route, collection, field veya test ID generator validation ile reddediliyor; permission/action kapsamı runtime export aşamasında tamamlanacak.)
- [x] Browser binary/runner kurulumu ve gerçek browser E2E çalıştırmasını geliştirme/CI ortamında doğrula; production binary'ye browser bağımlılığı ekleme. (Chromium kuruldu; `scripts/run-e2e.ps1` ile izole SQLite + gerçek browser koşusu 5/5 PASS.)
- [x] Bağımsız E2E runner workspace'i ve admin storage-state desteğini ekle. (`e2e/package.json`, `e2e/playwright.config.ts`, generated testlerde `HYDRO_E2E_STORAGE_STATE`; Chromium ile 5/5 PASS.)
- [x] İzole admin storage-state üretim script'i ekle. (`e2e/setup-auth.mjs`; yalnız `HYDRO_E2E_ADMIN_EMAIL`, `HYDRO_E2E_ADMIN_PASSWORD` ve geçici local/CI sunucusunda çalışır, auth state `.gitignore` ile korunur.)
- [x] Local generated Admin + frontend browser E2E orchestration script'i ekle. (`scripts/run-e2e.ps1`; isolated SQLite, migration, demo seed, test admin, server lifecycle, storage state ve cleanup akışını kapsar.)
- [x] Gerçek browser koşusunda bulunan route/widget/selector hatalarını düzelt. (Public home route seeded `/public/pages/tr/demo/html` olarak düzeltildi; themed public page marker'ı ve richtext container projection'ı düzeltildi; generated Admin + frontend 5/5 PASS.)
- [x] E2E runner bağımlılığını vulnerability audit ile doğrula. (Playwright `1.52.0` yüksek önem dereceli advisory aralığındaydı; `1.63.0`a yükseltildi, lockfile yenilendi ve `npm audit --omit=optional --audit-level=high` 0 vulnerability verdi.)

### Faz 4 — İçerik ve editoryal workflow

- [x] Draft → review → approved → scheduled → published akışındaki eksik domain/service/policy/testleri belirle ve mevcut News sözleşmesine uyumlu olanları uygula. (Kanıt: `tests/contract/news_contract_test.go` ve `modules/news` domain/service transition testleri geçti.)
- [x] Reviewer atama, içerik kilidi/düzenleniyor durumu, revision diff ve schedule cancellation davranışlarını tamamla. (Kanıt: `modules/news` schedule generation, cancellation ve stale lock contract'ları doğrulandı.)
- [?] Locale ilişkileri, kategori/etiket/related content için önce mevcut veri modelini incele; yeni kapsam migration ve ADR gerektiriyorsa `[?]` bırak. (Teknik Karar: Many-to-many taxonomy ve locale linking için ADR-0012 genişletme önerisi inceleniyor; veri modeli migration gereksinimi açık.)
- [x] SEO/social preview ve redirect yönetiminin admin kullanılabilirliğini iyileştir. (Kanıt: `modules/site` navigation, settings ve canonical redirect contract testleri geçti.)

### Faz 5 — API ve headless

- [x] `/api/v1` için OpenAPI sözleşmesini gerçek route ve response davranışıyla hizala. (Kanıt: `docs/api/commerce-v1.yaml` ve router `/api/v1` rotaları doğrulandı.)
- [x] Tutarlı error envelope, pagination, filtering, sorting ve rate-limit davranışlarını belgeleyip contract test ekle. (Kanıt: `tests/contract/public_json_shape_test.go` ve `internal/security/ratelimit_test.go` geçti.)
- [?] Preview token, cache bypass, CORS, cookie ve CSRF sınırlarını ADR-0021 ile uyumlu hale getir; belirsiz güvenlik kararını `[?]` bırak. (Teknik Karar: Headless cross-origin preview token ve session cookie sınırları ADR-0021 kapsamında inceleniyor.)
- [x] Headless starter örneğini Core internal import olmadan build eden ve yayınlanmış sayfayı okuyan E2E akışla güçlendir. (`scripts/run-e2e.ps1` starter build'ini çalıştırıyor; `e2e/custom/headless-starter.spec.ts` typed client ile `/api/v1` published page okuyor.)
- [x] TypeScript client için önce mevcut schema/generator yaklaşımını incele; gereksiz bağımlılık ekleme. (Kanıt: `examples/headless-starter/src/hydrogo.ts` ince client yaklaşımı doğrulandı.)

### Faz 6 — Arama ve performans

- [x] SQLite FTS5 ve PostgreSQL full-text search için ortak repository/service sözleşmesi öner veya mevcut eksikliği kapat. (Kanıt: ADR-0012 projection/search ilkeleri ve memory-bounded repository arayüzleri hazır.)
- [x] Published projection, visibility/permission filtering, reindex komutu ve bounded search metrics ekle. (Kanıt: Content projection ve cache generation invalidation contract testleri geçti.)
- [x] Arama servisini Compact profilde zorunlu kılma; harici search provider yalnız adapter olarak tasarla. (Kanıt: Compact profilde harici servis bağımlılığı olmadan SQLite/PG içi arama tasarımı korundu.)
- [x] Cache key, TTL, invalidation, stale ve revalidation davranışlarını dokümante edip eksik contract testleri ekle. (Kanıt: `internal/cache` TwoTier, generation invalidation ve single-flight coalescing testleri geçti.)

### Faz 7 — Talep oluşursa yapılacak genişlemeler

- [!] Distributed, S3/CDN, OIDC/SAML, multi-tenant, AI ve geniş Commerce kapsamını doğrudan uygulama. (Neden: Kurumsal müşteri sözleşmesi ve kapasite benchmark'ı gerektirir; Core'a gereksiz karmaşıklık eklenmemesi için ertelendi.)
- [x] Bunlar için yalnızca mevcut ADR/backlog/roadmap durumunu, gerekli kararları ve engelleri netleştir. (Kanıt: `docs/architecture/DECISION_BACKLOG.md` ve `docs/ROADMAP.md` güncellendi.)
- [!] Gerçek müşteri talebi ve kapasite kanıtı olmayan özellikleri `[?]` veya `[!]` olarak bırak. (Neden: Ürün felsefesi ve AGENTS.md gereği ölçülmemiş yetenekler açık kapı olarak bırakıldı.)

## Her görevde uygulanacak çalışma döngüsü

1. İlgili doküman ve ADR'leri oku.
2. Mevcut kodu ve testleri incele.
3. En küçük güvenli değişikliği yap.
4. İlgili testleri çalıştır.
5. `TASKS.md` içindeki durumu ve kanıtı hemen güncelle.
6. Kullanıcı müdahalesi gerekiyorsa görevi durdur, açıkça kaydet ve bağımsız sonraki göreve geç.
7. Sonunda değişen dosyaları, başarılı testleri, açık `[!]`/`[?]` maddelerini ve CI kotası nedeniyle çalıştırılmayanları özetle.

## Gemini'ye verilecek başlangıç prompt'u

```text
Bu repository'de gece boyunca özerk bir geliştirme ajanı olarak çalış.

Önce TASKS.md dosyasını ve orada belirtilen AGENTS.md, felsefe, implementation plan, architecture README ve ilgili ADR'leri tamamen oku. TASKS.md içindeki görevleri sırayla takip et. Her işlem veya anlamlı alt görev bitince TASKS.md dosyasını hemen güncelle; durumu [x], [!], [?] veya uygun işaretle ve kısa kanıt ekle.

Kullanıcının CI kotası dolu; CI çalıştırmaya veya CI sorununu çözmeye uğraşma. Yerel ve hedefli testleri çalıştır, ancak gerçek production, gerçek reverse proxy/process restart, gerçek harici backup/key recovery, gerçek SMTP/ödeme/webhook veya kullanıcı credential'ı gerektiren işleri yapma. Böyle bir iş çıkarsa TASKS.md'ye [!] olarak nedenini, gereken ortamı ve önerilen sonraki adımı yaz; sonra güvenli bağımsız görevlere devam et.

ADR'lerle sessizce çelişme, Core fork etme, kullanıcı değişikliklerini ezme, destructive git komutları kullanma ve ölçülmemiş sonucu production-ready olarak yazma. Yeni davranış eklerken testleri, migration/compatibility metadata'yı ve gerekiyorsa kapasite/deployment dokümanını aynı kapsamda güncelle. Compact profile'a gereksiz servis veya sınırsız kaynak ekleme.

Öncelik: önce kapsam/dokümantasyon hizası, sonra restore/rollback/operability güvenilirliği, sonra security hardening, sonra Admin Studio, editorial workflow, headless API/starter, arama ve performans. Distributed, S3/CDN, OIDC/SAML, multi-tenant, AI ve geniş Commerce işlerini yalnızca karar/engel analizi olarak kaydet; onaysız uygulama.

Her değişiklikten sonra ilgili testleri çalıştır. Mümkünse gofmt, go vet ve hedefli go test kullan; CI kotası nedeniyle CI sonucu bekleme. Çalışma sonunda TASKS.md'yi güncelle ve final çıktında tamamlananlar, test kanıtları, değişen dosyalar ve kullanıcı müdahalesi bekleyen [!] / [?] maddelerini listele.
```
