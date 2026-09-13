# HydroGo Professionalization Backlog

Bu belge, mevcut mimari ve doğrulama çıktıları yeniden taranarak hazırlanmış
üretimleşme backlog'udur. Yerel testlerin geçmesi, dış üretim kapılarının
tamamlandığı anlamına gelmez. Her madde kanıt, test ve gerekiyorsa ADR ile
kapatılmalıdır.

## P0 — Production öncesi zorunlu

### Dağıtım ve geri dönüş

- [ ] Staging ve production için tekrarlanabilir OCI/Docker image üret.
- [ ] Config/secrets ayrımını tamamla; secret'ların log, dump ve SBOM'a
  sızmadığını doğrula.
- [ ] Reverse proxy arkasında iki prosesli blue/green deploy tatbikatı yap.
- [ ] Readiness, warmup, asset retention ve schema compatibility kapılarını
  deploy pipeline'ına bağla.
- [ ] Yeni yazma sonrası rollback ve eski binary restart testini gerçek
  PostgreSQL ile çalıştır.
- [ ] Graceful shutdown, connection draining ve worker kapanışını ölç.
- [ ] Otomatik rollback, başarısız migration durdurma ve insan onayı akışlarını
  runbook'a ekle.
- [ ] Desteklenen OS, Go, PostgreSQL, Redis/Valkey ve FFmpeg sürümlerini
  SUPPORT_MATRIX.md ile sabitle.

### PostgreSQL ve veri güvenliği

- [ ] Desteklenen PostgreSQL sürümünde contract suite'i gerçek DSN ile CI'da
  zorunlu çalıştır.
- [ ] Eski production kopyası üzerinde migration/rollback testi yap.
- [ ] Şifreli off-server backup al ve temiz hedefe restore et.
- [ ] Restore sonrası session revoke, job quarantine, egress fence ve erasure
  replay kanıtı üret.
- [ ] RPO/RTO, retention, backup encryption ve key recovery prosedürlerini
  müşteri verisiyle ölç.
- [ ] Pool boyutları, transaction timeout, lock timeout ve statement timeout
  için profile bazlı varsayılanları doğrula.
- [ ] Büyük tablo/index migration'larında `CREATE INDEX CONCURRENTLY` ve
  expand-migrate-contract akışını belge/test et.

### Güvenlik ve kimlik

- [ ] MFA, invitation, reset, recovery, session revoke ve CSRF akışlarını gerçek
  browser E2E ile kapsa.
- [ ] Rate-limit Redis kaybı, failover, clock skew, key-space ve bypass
  senaryolarını test et.
- [ ] Upload abuse, SSRF, path traversal, decompression bomb ve oversized body
  testleri ekle.
- [ ] Secret rotation, signing-key rotation ve emergency revoke prosedürü yaz.
- [ ] Threat model ve dış penetrasyon testi tamamla.
- [ ] Security headers, CSP report-only → enforce geçişi ve trusted-origin
  modelini production proxy ile doğrula.
- [ ] PII maskeleme, audit erişim yetkisi ve log retention politikasını uygula.

## P1 — Güvenilirlik ve işletilebilirlik

### Gözlemlenebilirlik

- [ ] Request/actor/session trace korelasyonu ekle; token, parola ve kişisel
  verileri maskele.
- [ ] Latency/error/throughput, DB pool, cache, queue, worker, disk ve media
  metriklerini standardize et.
- [ ] Prometheus/OpenTelemetry export ve dashboard şablonları oluştur.
- [ ] Backup, disk, migration, SMTP, dead-letter, cache outage ve yüksek hata
  oranı alarmlarını gerçek notification kanalıyla test et.
- [ ] SLO/SLI ve hata bütçesi tanımla.

### Performans ve kapasite

- [ ] Compact, Standard ve Distributed profilleri gerçek müşteri benzeri veriyle
  benchmark et.
- [ ] Load, stress, spike ve 24 saatlik soak testleri ekle.
- [ ] Cache stampede, stale refill, invalidation kaybı ve Redis partition
  senaryolarını çoklu instance ile çalıştır.
- [ ] DB sorgu planlarını, indeks kullanımını ve N+1 ihtimallerini ölç.
- [ ] Büyük medya, paralel upload, FFmpeg timeout/kill ve disk kotası testleri
  ekle.
- [ ] CPU/RSS/FD/connection/queue sınırlarını ölçüp deploy limitlerine bağla.

### Veri yaşam döngüsü

- [ ] Kalıcı telemetry/time-series şemasını ürün kararı ve ADR ile tanımla.
- [ ] Retention, anonimleştirme, export, silme ve KVKK/GDPR rıza akışlarını
  uygula.
- [ ] Backup sonrası silinen verilerin erasure log/checkpoint ile uzlaştırılmasını
  gerçek provider üzerinde test et.
- [ ] Tenant/organizasyon izolasyonu gerekiyorsa ayrı ADR, model ve kapasite
  sınırları çıkar; erken multi-tenant ekleme.

## P1 — Frontend ürünleşmesi

- [ ] Tek design-system token seti (type, spacing, color, radius, elevation,
  motion) tanımla.
- [ ] Button, form, field error, table, pagination, modal, toast, tabs, empty,
  loading ve error bileşenlerini ortaklaştır.
- [ ] Admin, publisher ve public temalarını aynı erişilebilirlik sözleşmesine
  bağla.
- [ ] Responsive desktop/tablet/mobile viewport E2E ve visual regression ekle.
- [ ] Keyboard navigation, focus management, reduced motion, contrast ve WCAG
  kontrollerini CI'a al.
- [ ] HTMX fragment yaşam döngüsünde document-level listener'ları idempotent
  bind/cleanup yap; listener çoğalması için browser memory testi ekle.
- [ ] Asset cache-buster sürümlerini build manifestinden tekil fingerprint ile
  üret; elle tarih yazılan sürümleri kaldır.
- [ ] Frontend build manifestini generator input'una bağla; route/selector
  sözleşmesini deterministik üret.
- [ ] i18n, uzun metin, RTL ve tarih/sayı/para formatı testlerini ekle.
- [ ] Kritik checkout, auth, preview, editor ve media picker akışlarını
  cross-origin ve refresh/back-forward senaryolarıyla doğrula.
- [ ] Quill/HTMX gibi vendor asset'leri sürüm/hash/license manifestiyle yönet.

## P1 — Core ve modül sınırları

- [ ] Her route'u public/authorized/admin olarak sınıflandırıp default-deny testi
  üret.
- [ ] Pages/News public read, admin validation ve optimistic-lock fixture'larını
  izolasyonlu hale getir; TASKS.md'deki stale maddeleri kanıtla kapat.
- [ ] Modül inactive iken route, migration, asset, worker ve zorunlu kaynak
  üretmediğini tüm modüller için doğrula.
- [ ] Core'a modül-spesifik iş kuralı sızmasını importguard ve architecture test
  ile engelle.
- [ ] Capability contract'larında versiyonlama, deprecation ve compatibility
  testleri ekle.
- [ ] SQL/cache/queue ayrıntılarını domain/application katmanından uzak tutan
  boundary testlerini genişlet.
- [ ] Public API'nin draft/deleted/scheduled içerik sızdırmadığını her modülde
  test et.
- [ ] Concurrent edit, idempotent job, outbox replay ve provider timeout
  belirsizliği testlerini tamamla.

## P2 — Geliştirici deneyimi ve kalite

- [ ] `hydro doctor` çıktısını actionable hale getir; eksik DB/cache/media,
  writable path ve config sorunlarını güvenli biçimde raporla.
- [ ] WSL/Docker ile tek komut local bootstrap, seed data ve temizleme akışı ekle.
- [ ] `hydro new` tüm preset'leri üretip migrate/serve edebilen temiz makine
  testine bağla.
- [ ] Pre-commit/pre-push format, vet, guard ve hızlı test kancaları ekle.
- [ ] Fuzz testleri: parser, sanitizer, path, config, token, migration ve webhook.
- [ ] Dependency update botu, license inventory ve düzenli vulnerability triage
  süreci oluştur.
- [ ] Test fixture'larında `sync.Mutex`/`sync.WaitGroup` değer kopyalarını vet ile
  engelle; race testini her Linux runner'da zorunlu tut.
- [ ] API, extension, module authoring ve troubleshooting dokümantasyonunu
  örneklerle tamamla.

## P2 — Ürün kabiliyetleri

- [ ] Taksonomi için çoklu dil, kategori/etiket ve related-content ADR'si çıkar.
- [ ] Gelişmiş arama yalnızca ölçülmüş ihtiyaca göre eklenmeli; indeks ve arama
  provider sözleşmesini önce tanımla.
- [ ] Minimal production preset'leri (`compact-site`, `editorial`, `commerce`)
  oluştur; gereksiz servisleri kapalı tut.
- [ ] Commerce sağlayıcılarının sandbox callback/signature/idempotency testlerini
  gerçek hesaplarla doğrula.
- [ ] S3/CDN ve durable queue ancak kapasite ve müşteri ihtiyacı kanıtlandıktan
  sonra eklenmeli.
- [ ] SDK v1, deprecation window, LTS ve SLA sürelerini ticari olarak kararlaştır.
- [ ] OIDC/SAML, mobile API, AI ve geniş multi-tenant kapsamını ayrı ADR ve
  müşteri sözleşmesi olmadan Core'a ekleme.

## Çıkarılacak veya sadeleştirilecekler

- [ ] Kullanılmayan vendor asset, tema varyantı, demo endpoint ve örnek verileri
  release image'ından çıkar.
- [ ] Elle yönetilen asset tarih/cache sürümlerini kaldır.
- [ ] Kullanıcıya teknik hata, SQL/cache sürücüsü, stack trace veya iç kapasite
  ayrıntısı gösteren mesajları Information Hierarchy'ye göre sadeleştir.
- [ ] Kullanılmayan feature flag, ölü route, duplicate helper ve erişilemeyen
  migration'ları periyodik olarak temizle.
- [ ] Gerçek talep ve kapasite kanıtı olmayan Distributed, AI, OIDC/SAML, S3/CDN
  ve geniş Commerce kodunu Core'a erken dahil etme.

## Release kabul kriteri

- [ ] Tüm P0 maddeleri kanıtlı ve runbook'a bağlı.
- [ ] P1 güvenilirlik, güvenlik ve frontend maddeleri için ölçülebilir kabul
  kriteri var.
- [ ] Yerel/CI: format, vet, full test, race, fuzz smoke, guards, SBOM,
  vulnerability scan ve reproducible build yeşil.
- [ ] Gerçek PostgreSQL restore, proxy rollback, backup/key recovery ve kapasite
  raporları ekli.
- [ ] Desteklenen sürümler, SLA, lisans, veri yaşam döngüsü ve incident runbook'u
  müşteri teslim paketinde mevcut.

