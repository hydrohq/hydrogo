# Açık Kapılar ve Operatör Aksiyonları

**Durum:** Kod deposundaki güvenli yerel doğrulamalar tamamlanabilir; bu belge production,
dış servis, hukuk, ticari karar veya gerçek müşteri ortamı gerektiren işleri listeler.
**Güncelleme:** 2026-09-12

Bu liste “yapılmadı” ile “kullanıcı/operatör ortamı olmadan yapılamaz” ayrımını korur.
Bir madde için kanıt yoksa production-ready kabul edilmez. CI kotası tükenmişse CI sonucu
üretilmiş gibi gösterilmez.

## 1. Kullanıcının veya operatörün yapması gerekenler

### P0 — Lisans ve ticari dağıtım

- [ ] Resmi release binary’sini vendor public key ile build et. `internal/licensing.TrustedPublicKeyBase64`
  değeri güvenli release ortamında `-ldflags` ile enjekte edilmelidir. Private signing key
  müşteri makinesine, repository’ye veya binary üretim dışındaki ortamlara kopyalanmamalıdır.
- [ ] Lisans üretim anahtar çifti için ayrı, erişimi kısıtlı bir vendor/key-management ortamı kur.
  Keygen ve issue komutları aynı runtime binary’sinde bulunduğu için CLI erişimi dağıtım paketinde
  ayrıca sınırlandırılmalıdır.
- [ ] `LICENSE` ve müşteri yeniden dağıtım koşullarını hukukçuya incelet; kullanım hakkı,
  güncelleme süresi, destek/SLA, lisans iptali, kaynak kodu ve tersine mühendislik hükümlerini
  netleştir.
- [ ] Müşteri projesi için `project_id`, edition, feature listesi ve geçerlilik süresiyle lisans üret.
  `hydro.lic` ile `hydrogo.lock` aynı proje kimliği/edition/feature sözleşmesini taşımalıdır.

### P0 — Production güvenliği ve erişim

- [ ] Production’da HTTPS termination, `HYDRO_SECURE_COOKIES=true`, public origin ve HSTS ayarlarını
  birlikte yapılandır; gerçek domain ve sertifika ile doğrula.
- [ ] İlk admin hesabını güçlü parola ve MFA ile oluştur. İnternete açık publisher/admin hesaplarında
  MFA zorunluluğunu işletim prosedürüne bağla.
- [ ] Gerçek SMTP sağlayıcısını bağla ve password reset, invitation, MFA/recovery e-postalarının
  teslim/başarısızlık/yeniden deneme akışını doğrula.
- [ ] Production secret, MFA master key, session signing/config secret ve ödeme anahtarlarını secret
  manager veya izinleri kısıtlı ortam değişkenlerinde tut; repository’ye yazma.

### P0 — Veritabanı, yedek ve geri dönüş

- [ ] Desteklenen gerçek PostgreSQL sürümünde contract suite’i `HYDRO_TEST_POSTGRES_URL` ile çalıştır.
  PostgreSQL yoksa testin atlandığı bir sonuç certification kanıtı değildir.
- [ ] Gerçek müşteri benzeri veriyle SQLite/PostgreSQL backup, restore, migration ve rollback tatbikatı
  yap; süre, RPO, RTO, disk kullanımı ve veri kaybı sonuçlarını kayıt altına al.
- [ ] Gerçek reverse proxy arkasında iki prosesli blue/green restart, trafik geçişi ve rollback deneyi
  yap. Eski binary’nin yeni şemayla ve yeni binary’nin geri alınmış şemayla davranışını ölç.
- [ ] Harici backup/S3 uyumlu storage ve KMS/key escrow kullanılıyorsa erişim kaybı, anahtar kurtarma,
  snapshot bütünlüğü ve temiz hedefe restore tatbikatını yap.
- [ ] Backup retention, şifreleme, silme ve erasure replay politikalarını müşteri veri yaşam döngüsüyle
  onayla. Harici deletion/erasure checkpoint olmadan silme tamamlandı sayılmaz.

### P0 — Medya ve dış entegrasyonlar

- [ ] Production encoder ortamına `ffmpeg` kur ve `HYDRO_MEDIA_FFMPEG` ile açıkça yapılandır.
  Büyük video/fotoğraf dosyalarında WebP/WebM dönüşümünü gerçek dosyalarla doğrula.
- [ ] Encoder yokken upload’ın pending/failed durumuna düştüğünü, RAM limitlerinin aşılmadığını ve
  tekrar deneme/backpressure davranışını test et. Kalıcı job/queue ihtiyacı varsa ayrı kapasite kararı ver.
- [ ] Stripe/iyzico/PayTR sandbox callback ve signature doğrulamasını gerçek sandbox hesaplarıyla çalıştır.
  Canlı ödeme anahtarları, webhook URL’leri, idempotency, refund ve timeout davranışı production öncesi
  ayrıca doğrulanmalıdır.
- [ ] S3/CDN veya uzak medya sağlayıcısı kullanılacaksa provider sözleşmesi, erişim/yenileme, cache
  invalidation, hata fallback’i ve GC/erasure davranışını gerçek ortamda ölç.

## 2. Çalıştırılması gereken ama bu çalışma kapsamında kanıtlanmamış testler

- [x] Chromium Playwright E2E: izole SQLite sunucusu, migration, seed, admin storage state ve
  `scripts/run-e2e.ps1` ile 14/14 PASS. Aynı çalışma Windows CI `browser-e2e` job'ında zorunludur;
  discovery sonucu değil, gerçek browser sonucu kaydedilmiştir.
- [ ] Her üretilen frontend için gerçek responsive E2E: masaüstü/tablet/mobil viewport, route root,
  yatay taşma, kırık görsel, boş/error state, haber detay, ürün kartı/varyant, sepet/checkout ve
  üyelik login/logout akışları.
- [ ] Admin browser E2E: MFA, password reset, invitation, session revoke, CSRF, yetki sınırı,
  media upload/derivative ve dark-mode/first-load layout davranışları.
- [ ] Gerçek browser cross-origin session mutasyonları: cart, checkout, account ve preview için
  BFF/same-site proxy kararı ve ADR-0021 acceptance testleri.
- [ ] Temiz makinede generator testi: `hydro new` ile tüm preset’leri üret, runtime-loadable tema,
  config, lock, seed ve E2E manifestini doğrula; `--git` davranışında remote/push olmadığını kontrol et.
- [ ] Generated scaffold’ın lisanslı resmi binary ile gerçekten migrate/serve edilebildiğini doğrula.
  Generator yalnızca Core’suz bir Client Edition scaffold üretir; tek başına Core binary veya
  tam deploy edilmiş haber/e-ticaret uygulaması üretmez.
- [ ] Race, fuzz ve uzun süreli yük testlerini temiz ortamda yeniden çalıştır; özellikle auth/session,
  media derivative, rate limit/Redis, queue lease, cache invalidation ve metrics sayaçlarını ölç.
- [ ] Gerçek müşteri kapasite ölçümü: Compact/Standard/Distributed profillerinde RSS, CPU, DB pool,
  queue depth, cache hit/miss, disk, latency ve hata oranı sınırlarını donanım bazında kaydet.

## 3. Teknik karar veya hukuki onay bekleyen konular

- [ ] **Ertelenmiş ürün ergonomisi — minimal kurulum preset'leri:** `compact-site`, `editorial`,
  `commerce` ve `headless` gibi açık modül setleriyle örnek yapılandırma/preset sun. Mevcut
  `config.example.yaml` demonstrasyon amacıyla tüm modülleri etkinleştirir; bu madde uygulanana
  kadar gerçek kurulumda yalnızca gerekli modüller seçilmelidir. Her preset için kapalı modüllerin
  route, migration, asset ve worker yüklemediği SQLite/PostgreSQL contract testi gerekir.
- [ ] Kalıcı ürün analitiği: günlük/time-series tablo, retention, anonimleştirme, export ve KVKK/GDPR
  açık rıza metni kararı. Şu an process-local telemetri kalıcı tarihsel rapor değildir.
- [ ] Kullanıcı eylemi–hata korelasyonu: actor/session trace modeli, kişisel veri maskeleme, saklama
  süresi ve erişim yetkisi için ayrı karar/ADR gerekir.
- [ ] Taksonomi: çoklu dil, many-to-many kategori/etiket ve related content veri modeli için ADR-0012
  genişletmesi ve SQLite/PostgreSQL migration kararı gerekir.
- [ ] Headless browser session: credentialed CORS yerine BFF/same-site proxy sınırı ve preview token
  politikası ADR-0021 kapsamında kesinleştirilmelidir.
- [ ] OIDC/SAML, mobile API, multi-tenant, AI ve geniş Commerce kapsamı ancak gerçek müşteri ihtiyacı,
  threat model, kapasite ve ayrı ADR ile açılmalıdır.
- [ ] Redis/Valkey, durable queue, S3/CDN, Distributed profil ve canlı ödeme “her zaman açık” bağımlılık
  yapılmamalı; Compact profilde yalnızca açıkça seçilen servisler kaynak tüketmelidir.
- [ ] SDK v1, deprecation/support window, LTS ve SLA süreleri ticari olarak seçilmeden stable v1 iddiası
  yapılmamalıdır.

## 4. Bu tarihte yerel olarak doğrulananlar

- [x] `go test ./... -p 1 -timeout=5m`
- [x] `go vet ./...`
- [x] `git diff --check` (mevcut CRLF dönüşüm uyarısı dışında hata yok)
- [x] Lisans imza, tamper, expiry/grace, project/edition/feature kontrolleri
- [x] Self-advertised public key’in trust kaynağı olarak reddedilmesi
- [x] Runtime’da lisans ve `hydrogo.lock` varlığı/JSON/project/edition/feature eşleşmesi
- [x] `hydro new` preset/path traversal/existing directory ve Core `internal/` kopyalanmaması
- [x] Üretilen minimal tema ve config parse doğrulaması

Bu bölüm yalnızca mevcut yerel kanıtı ifade eder; production certification, gerçek müşteri verisi,
dış servis veya canlı ödeme başarısı anlamına gelmez.

## 5. İlgili kaynaklar

- [Production Gates](validation/PRODUCTION-GATES.md)
- [Operations Runbook](OPERATIONS_RUNBOOK.md)
- [Threat Model](security/THREAT_MODEL.md)
- [Personal Data Policy](PERSONAL_DATA_POLICY.md)
- [Commercial Licensing](COMMERCIAL_LICENSING.md)
- [Media Derivatives and E2E](validation/media-derivatives-and-e2e.md)
- [Roadmap](ROADMAP.md)
- Admin Studio: `Geliştirici > Kaynak tüketimi` (anlık process, DB pool, cache ve job görünümü)
