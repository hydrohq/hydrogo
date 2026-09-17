# HydroGo

HydroGo, Core'u çatallamadan müşteri sunucularında çalıştırılabilen modüler bir Go CMS'i ve JSON API platformudur.

Amacı:

> Kullanılmayan altyapı bağımlılıkları eklemeden güvenli, taşınabilir ve işletilebilir bir CMS ile müşterinin verisi ve dağıtımı üzerinde tam kontrol sağlamaktır.

HydroGo “her şeyi yapan bir CMS” değildir; takılıp çıkarılabilen modüllere sahip kararlı bir Core'dur. Devre dışı modüller rota, migration, worker veya zorunlu harici servis oluşturmamalıdır.

## Mevcut durum

Depo çalışan bir teknik temel ve geniş bir sözleşme testi paketi sunar. Pages ve News modülleri, Site Edition akışları, kimlik doğrulama/MFA, medya, cache, yedekleme/geri yükleme, taşınabilir dışa/içe aktarma, release manifestleri ve generator denemeleri uygulanmıştır.

Depo henüz tüm production profilleri için tam sertifikalı ve ticari olarak desteklenen v1 ilan edilmemiştir. R3 pilot, R5 v1 ve R6 Distributed kanıt kapıları roadmap'te ayrı izlenir.

### Yalnızca binary müşteri dağıtımı

Müşteri kurulumları Core veya modül kaynak kodu gönderilmeden teslim edilebilir. Üretici allowlist ile seçilmiş, sembolleri temizlenmiş (isteğe bağlı obfuscation) `hydro` runtime'ı derler; yalnızca binary, `hydro.lic`, `hydrogo.lock`, yapılandırma, temalar ve Client Edition dosyalarını paketler. Paketleyici `internal/`, `modules/`, `cmd/`, `.git/`, `go.mod`, `go.sum`, Go kaynaklarını ve özel anahtarları reddeder. Ayrıntılar için [`PROPRIETARY_NOTICE.md`](PROPRIETARY_NOTICE.md), [`docs/COMMERCIAL_LICENSING.md`](docs/COMMERCIAL_LICENSING.md) ve [`docs/architecture/0143-binary-only-customer-distribution.md`](docs/architecture/0143-binary-only-customer-distribution.md) belgelerine bakın.

Üretici imzası yalnızca `hydro-issuer` ile atılır; müşteri runtime'ı sadece lisans doğrulama/durum yüzeyi sunar. Müşteri paketini `scripts/build-customer-runtime.ps1` veya `.sh` ile oluşturun. `HYDRO_TRUSTED_PUBLIC_KEYS` korumalı üretici derlemesinde eklenir; özel imza anahtarları müşteri paketine girmez. Yetkili bir operatör çalışan binary'yi yine de reverse-engineer edebilir; bu sınır kaynak kodu teslimini ve basit yeniden kullanımı engeller, her türlü binary değişikliğini garanti etmez.

### Halihazırda çözülen başlıklar

- Taslak, revizyon, önizleme, zamanlama ve yayınlama akışlarıyla sayfa ve haber yönetimi
- Editörler arası sessiz veri kaybını önleyen iyimser kilitleme
- HTML ve JSON API yollarında ortak uygulama servisleri ve yetkilendirme kuralları
- Public/private medya yaşam döngüsü ve güvenli yükleme metadatası
- Yetkili editör/admin için sınırlı süreli imzalı private medya URL'leri
- SQLite ve PostgreSQL için ortak davranış sözleşmeleri
- Sınırlı L1 cache, generation invalidation ve single-flight cache miss birleştirme
- MFA, oturum iptali, parola sıfırlama, davet ve CSRF temelleri
- Audit, transactional outbox, job leasing ve idempotency altyapısı
- SQLite snapshot'ları, PostgreSQL dump/restore ve medya manifest doğrulaması
- Hash zincirli harici silme günlüğü ve idempotent silme tekrarı
- Secret ve oturum durumunu dışarıda bırakan allowlist tabanlı taşınabilir aktarım
- Ed25519 imzalı release manifestleri ve güvenilir public-key doğrulaması
- Deterministic generator, Client Edition sınırları ve SDK v1 adayı

## Nasıl çalışır

Web sunucusu ve operasyon komutları tek binary içindeki aynı Core'u kullanır:

```text
hydro serve / migrate / doctor / upgrade check
                         │
                         ▼
              HTTP handler / JSON API
                         │
                         ▼
             Application service + policy
                         │
                         ▼
             Module repository interface
                         │
              ┌──────────┴──────────┐
              ▼                     ▼
           SQLite              PostgreSQL
```

HTML yönetim paneli ve JSON API aynı servis ve politika katmanlarını çağırır. Domain ve application-service kodu SQL lehçesini, cache sürücülerini veya dağıtım proxy ayrıntılarını bilmez.

### Kapasite profilleri ve isteğe bağlı servisler

- `compact`: SQLite ve yerel medya ile ek servis gerektirmeyen geliştirme/küçük kurulum profili.
- `standard`: PostgreSQL, Redis/Valkey ve daha yüksek bağlantı/worker bütçeleri için önerilen profil.
- `distributed`: birden fazla uygulama/worker örneği, Redis/Valkey L2 cache ve S3 uyumlu medya/CDN topolojisi için sözleşme profili; gerçek production sertifikasyonu ayrıca gereklidir.

Redis/Valkey (`HYDRO_REDIS_URL`), S3 uyumlu medya (`HYDRO_MEDIA_STORAGE_DRIVER=s3` ve
`HYDRO_MEDIA_S3_*`), CDN (`HYDRO_MEDIA_CDN_BASE_URL`), SMTP (`HYDRO_SMTP_*`) ve
FFmpeg (`HYDRO_MEDIA_FFMPEG`) yalnızca açıkça yapılandırıldığında kullanılır. Compact
profilinde bu servisler otomatik olarak zorunlu hale gelmez. Sağlayıcı kimlik bilgilerini
YAML veya kaynak koda yazmayın; secret manager ya da korumalı ortam değişkenleri kullanın.

### Modül ve Core sınırları

- `modules/pages`: sayfalar, revizyonlar, taslaklar, önizleme ve yayınlama
- `modules/news`: editoryal inceleme, zamanlama, yayınlama, geri çekme ve worker'lar
- `modules/site`: navigasyon, ayarlar, SEO, yönlendirmeler ve form gelen kutusu
- `internal/auth`: kullanıcılar, oturumlar, MFA, sıfırlama ve davetler
- `internal/media`: yüklemeler, görünürlük, pending/ready durumu ve depolama metadatası
- `internal/cache`: sınırlı L1, TwoTier cache ve invalidation fanout
- `internal/backup`: SQLite/PostgreSQL snapshot, manifest, geri yükleme ve aktivasyon
- `internal/release`: manifest, imza, attestation, tutulan asset'ler ve Blue/Green durumu
- `internal/portable`: allowlist tabanlı veri taşınabilirliği
- `internal/generator`: collection spec'lerinden incelenebilir CRUD ve migration taslakları

## Hızlı başlangıç

Gereksinimler:

- Go 1.27.0
- SQLite için ek servis gerekmez
- PostgreSQL sözleşme testleri için PostgreSQL 18 ve `HYDRO_TEST_POSTGRES_URL` gerekir

Windows PowerShell:

```powershell
go build -o bin/hydro.exe ./cmd/hydro
$env:HYDRO_DATABASE_DSN = "hydro.sqlite"
./bin/hydro.exe migrate
./bin/hydro.exe serve
```

Binary'yi `hydro` adıyla kullanmak için `bin` klasörünü PATH'e ekleyin. PATH
ayarlamadan tüm örneklerde `./bin/hydro.exe` (Linux/macOS'ta `./bin/hydro`)
önekini kullanabilirsiniz. Yapılandırma dosyası vermediğinizde değerler ortam
değişkenlerinden, ardından varsayılanlardan okunur; `--config` ile YAML dosyası
seçilebilir.

Geçerli PowerShell oturumunda PATH'e eklemek için:

```powershell
$env:Path = "$(Resolve-Path .\bin);$env:Path"
hydro version
```

Linux/macOS:

```bash
go build -o bin/hydro ./cmd/hydro
export HYDRO_DATABASE_DSN=hydro.sqlite
./bin/hydro migrate
./bin/hydro serve
```

### Canlı geliştirme

Yerel geliştirmede proxy adresi (`127.0.0.1:8090`) sabit kalırken HydroGo uygulaması içeride yeniden başlatılır. Böylece tarayıcı oturumu korunur; Go, HTML ve Templ değişiklikleri yeniden derlenir, CSS/JS değişiklikleri ise sayfayı yeniler. Derleme hatasında önceki çalışan sürüm ayakta kalır ve tarayıcıda hata ayrıntılarını gösteren kırmızı geliştirme paneli görünür.

```powershell
./bin/hydro.exe run
```

`hydro run` dahili olarak `tools/devserver` proxy'sini başlatır; bu iç Go yolu
doğrudan çalıştırılmamalıdır. `hydro dev` aynı seçenekleri kabul eden geriye
dönük uyumluluk alias'ıdır. Varsayılan proxy `127.0.0.1:8090`, uygulama adresi
`127.0.0.1:8091`'dir.

### SDK ile hızlı proje başlatma

Yeni bir Client Edition iskeleti ve frontend entegrasyon manifesti üretmek için:

```powershell
./bin/hydro.exe project new haber-merkezi --preset=news
Set-Location haber-merkezi
../bin/hydro.exe project check
```

`hydro project check`, varsa `generated/` altındaki tüm resource scaffold'larını
da aynı bağımlılıksız preflight içinde doğrular.

Üretilen projede `frontend/starter.json` ve `frontend/README.md`, Hydro,
shadcn, Material, Carbon, headless, vanilla veya custom sunum katmanlarından birini
seçerek ilerlemenizi sağlar. Collection spec'leri için `hydro resource new`
her kaynak altında tip güvenli `client.ts`, React/SWR/HTMX/vanilla adapter'ları,
`form.schema.json`, OpenAPI ve CRUD yardımcılarını üretir. Bu dosyalar Core'a
çalışma zamanı bağımlılığı eklemez; kimlik doğrulama, politika ve cache
geçersizleştirme Client Edition sınırında kalır. Ayrıntılı akış için
[`docs/SDK_CAPABILITY_MATRIX.md`](docs/SDK_CAPABILITY_MATRIX.md), [`docs/PROJECT_GENERATOR.md`](docs/PROJECT_GENERATOR.md)
ve [`docs/CLI_WORKFLOW.md`](docs/CLI_WORKFLOW.md) belgelerine bakın.

Her üretilen collection kendi deterministic `spec.json` kaynağını taşır;
`hydro resource check --root .` ile tüm kaynakları tek komutla, veritabanına
bağlanmadan doğrulayabilirsiniz. İsterseniz `--spec` vererek tek bir kaynağı
ayrıca kontrol edebilirsiniz.

SDK ad alanında önerilen yazım `hydro project new` komutudur; eski `hydro new`
yazımı geriye dönük uyumluluk için korunur. Generator Core kaynak kodunu
üretilen projeye kopyalamaz; yalnızca Client Edition iskeleti ve incelenebilir
üretim çıktıları oluşturur.

Varsayılan demo veritabanı `data/demo.sqlite` kullanılır. Farklı bir veritabanı için `--database-dsn`; proxy veya dahili uygulama portları için sırasıyla `--address` ve `--app-address` kullanın.

Uç noktalar:

- `/healthz`: yalnızca process canlılık kontrolü; bağımlılıkları doğrulamaz
- `/readyz`: veritabanı ve migration uyumluluğu hazır olma kontrolü
- `/metrics`: kimlik doğrulamalı, durum etiketleri sınırlı operasyon sayaçları
- Sürümlü JSON API yolları `/api/v1/...` altındadır; eski `/api/...` yolları uyumluluk alias'ı olarak korunur ve yeni yolu bildirir
- Admin/API yolları: default-deny routing ve yetkilendirme politikaları

PostgreSQL için `HYDRO_DATABASE_DIALECT=postgres` ayarlayın ve izole bir `HYDRO_DATABASE_DSN` kullanın.

Dağıtım değişikliği açmadan önce Windows'ta yerel binary smoke kontrolünü çalıştırın:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-local.ps1
```

Bu kontrol izole bir binary derler, geçici SQLite veritabanını migrate eder, sunucuyu
geçici loopback portunda başlatır ve `/healthz` ile `/readyz` uç noktalarını doğrular.
Linux CI eşdeğer `scripts/smoke-local.sh` kontrolünü çalıştırır ve ayrıca sunucunun
`SIGTERM` sonrasında temiz kapandığını doğrular.

Production HTTPS ayarları açıkça verilmelidir: HTTPS `HYDRO_PUBLIC_ORIGIN` ile birlikte
`HYDRO_SECURE_COOKIES=true` ve `HYDRO_ENABLE_HSTS=true` (veya YAML'daki eşdeğer
`security.secure_cookies` ve `security.enable_hsts`) değerlerini ayarlayın. HTTPS
canonical origin veya secure cookie olmadan HSTS etkinleştirilmesi reddedilir. Bu
ayarlar yerel HTTP geliştirmesinde varsayılan olarak kapalıdır.

`HYDRO_PUBLIC_ORIGIN` ayarlandığında HydroGo HTTP isteklerini canonical host ve porta
bağlar. Forwarded host header'larına güvenilmez; reverse proxy'nin public `Host`
header'ını koruduğundan emin olun. Origin'i yalnızca yerel geliştirmede boş bırakın.

Release derlemelerine `-ldflags "-X main.buildCommit=COMMIT -X main.buildTime=UTC_TIME"`
ile kimlik metadatası gömülebilir; `hydro version --json` ile incelenebilir.

Runtime ve production binary'si için Node.js/npm zorunlu değildir. Frontend asset
üretimi gerekiyorsa sabitlenmiş geliştirme/release araçları kullanılır ve oluşan
çıktılar binary ile birlikte doğrulanır.

## Sık kullanılan CLI komutları

```text
hydro run [--address ADDR]
hydro new PROJECT [--preset PRESET]  (legacy alias; önerilen: hydro project new)
hydro serve
hydro migrate
hydro version --json
hydro doctor
hydro doctor --json
hydro upgrade check
hydro create-admin --email EMAIL --password-stdin
hydro seed-demo --confirm-demo
hydro validate --config config.yaml
hydro test --package ./modules/news/... --timeout 2m
hydro generate module --spec module.yaml --out modules/catalog
hydro module create --spec module.yaml --out modules/catalog
hydro resource new --name article --field title:text --out .
hydro frontend check --manifest frontend/starter.json
hydro api diff --baseline openapi.json --current openapi.json
hydro jobs dead-letter list --database-dialect sqlite --database-dsn PATH --json
hydro backup sqlite --destination PATH [--media-dir PATH]
hydro backup postgres --destination PATH [--media-dir PATH]
hydro backup prune --root PATH --keep N --dry-run|--confirm [--pinned NAME,...]
hydro restore verify --snapshot PATH
hydro restore stage --snapshot PATH --destination PATH --sessions-revoked --egress-quarantined --old-writes-blocked
hydro restore postgres --snapshot PATH --destination PATH --sessions-revoked --egress-quarantined --old-writes-blocked
hydro restore activate-media --staging PATH --active PATH --rollback PATH
hydro portable export --path PATH
hydro portable import --path PATH --dry-run
hydro portable import --path PATH --confirm
hydro release verify --manifest release.json --public-key release.pub
hydro license verify --license hydro.lic --public-key vendor.pub --project-id PROJECT_ID
hydro license status --license hydro.lic --public-key vendor.pub --json
```

Yerel geliştirmede `hydro run` önerilir; eski otomasyonlar için `hydro dev` alias'ı
aynı davranışı korur.

Taşınabilir içe/dışa aktarma varsayılan olarak 256 MiB ile sınırlıdır; onaylı daha büyük bir müşteri aktarımı gerektiğinde `--max-bytes` kullanın.

`hydro release verify`, manifest içine gömülen public key'e tek başına güvenmez.
Operatör tarafından dağıtılan güvenilir bir Ed25519 public-key dosyası zorunludur.

Tarayıcı kaynaklı durum değiştiren istekler `hydro_csrf` cookie'siyle eşleşen bir CSRF token içermelidir. URL-encoded ve küçük multipart formlar token'ı `_csrf` alanında taşıyabilir. Büyük veya parçalı multipart yüklemeler aynı token'ı `X-CSRF-Token` header'ında göndermelidir; böylece yalnızca CSRF doğrulamak için sınırsız yük gövdesi parse edilmez.

Yedekleme ve geri yükleme bilerek staging-first tasarlanmıştır. SQLite ve PostgreSQL yedekleri yeni, değişmez bir snapshot klasörü oluşturur; PostgreSQL `pg_dump`/`pg_restore` için geçici parola dosyası kullanır ve parolayı process argümanlarına koymaz. Geri yükleme doğrulaması yalnızca hash'leri kontrol eder, staging mevcut hedefin üzerine yazmaz; medya aktivasyonu quarantine işareti ve üç operatör kontrolünün tamamını gerektirir. Canlı snapshot almadan önce uygulama yazmalarını ve arka plan worker'larını duraklatın.

Taşınabilir içe aktarma da açık onay gerektirir: `--dry-run` allowlist'teki, secretsız belgeyi veritabanına yazmadan doğrular; gerçek aktarım `--confirm` ister ve tek transaction içinde çalışır.

Snapshot saklama politikası da açıktır: adayları görmek için `hydro backup prune --dry-run`, silmeyi uygulamak için aynı komutu `--confirm` ile çalıştırın. Pinned snapshot klasör adları hiçbir zaman silinmez.

## Testler ve production kapıları

Günlük geliştirme kontrolleri:

```powershell
go test ./...
go vet ./...
powershell -ExecutionPolicy Bypass -File .\scripts\verify-local.ps1
```

`verify-local.ps1` tek yerel geliştirici kapısıdır: formatı, vet'i, tam test paketini, trimpath derlemesini, Client Edition import sınırlarını ve iki harici edition derlemesini kontrol eder. PostgreSQL gerektirmez. Production kapısı daha katıdır ve gerçek bir PostgreSQL sözleşme veritabanı ister.

`hydro doctor --json`, seçilen profil, veritabanı, yedekleme araçları, medya yazılabilirliği,
transport güvenlik ayarları, binary kimliği ve gömülü şema migration seviyesi için
otomasyona uygun preflight çıktısı üretir.

Dead-letter durumundaki arka plan job'ları payload'ları açığa çıkarmadan incelenebilir:

```text
hydro jobs dead-letter list --database-dialect sqlite --database-dsn PATH --json
hydro jobs dead-letter requeue --id JOB_ID --confirm --database-dialect sqlite --database-dsn PATH
hydro jobs dead-letter purge --older-than 720h --limit 100 --confirm --database-dialect sqlite --database-dsn PATH
```

Requeue ve purge açıkça operatör tarafından başlatılır. Requeue job deneme sayısını ve son hatayı korur; purge yaş ve batch boyutuyla sınırlıdır.

PostgreSQL sözleşme veritabanını ayarladıktan sonra otomatik release kapılarını çalıştırın:

```powershell
$env:HYDRO_TEST_POSTGRES_URL = "postgres://user:password@127.0.0.1:5432/hydro_contract?sslmode=disable"
powershell -ExecutionPolicy Bypass -File .\scripts\verify-production-gates.ps1
```

Linux veya macOS'ta eşdeğer kapı çalıştırıcısı:

```bash
export HYDRO_TEST_POSTGRES_URL='postgres://user:password@127.0.0.1:5432/hydro_contract?sslmode=disable'
bash ./scripts/verify-production-gates.sh
```

Her iki çalıştırıcı formatı, vet'i, tam test paketini, race hassas paketleri, tekrarlanabilir binary derlemelerini ve taşınabilir benchmark'ı kontrol eder. PostgreSQL yoksa sözleşme testlerini sessizce atlamak yerine başarısız olur.

Gerçek production kapıları ayrıca şunları gerektirir:

- Gerçek reverse proxy arkasında iki process ile Blue/Green yeniden başlatma ve rollback
- Temiz SQLite ve PostgreSQL hedeflerine geri yükleme
- Harici yedekleme ve anahtar kurtarma tatbikatları
- Production imza anahtarı oluşturma/teslim töreni
- Temsili müşteri iş yükleriyle RPO/RTO ve kapasite ölçümleri

Bu kanıtlar oluşana kadar depo herhangi bir production profili veya SLA iddiasında bulunmaz.

## Mimari ilkeler

- Müşteri çalışmaları HydroCore fork'u yerine Client Edition ve extension contract'ları üzerinden uygulanır.
- SQLite ve PostgreSQL aynı domain ve application sözleşmelerini uygular.
- Migration, cache, job, asset ve release değişiklikleri açık sözleşme testleriyle ilerler.
- Kullanılmayan servisler Compact profillerinde kapalıdır; Distributed özellikler varsayılan değildir.
- Güvenli varsayılan routing, default-deny yetkilendirme ve veri taşınabilirliği Core sözleşmeleridir.

## Dokümantasyon

- [Roadmap](docs/ROADMAP.md): fazlar, açık kapılar ve kabul ölçütleri
- [HydroGo felsefesi](docs/HYDROGO_PHILOSOPHY.md): ürün ve mimari ilkeler
- [Uygulama planı](docs/implementation_plan.md): kapsam ve hedef mimari
- [ADR dizini](docs/architecture/README.md): bağlayıcı mimari kararlar
- [Production kapı kontrol listesi](docs/validation/PRODUCTION-GATES.md): gerçek ortam kanıt biçimi
- [Açık kapılar ve operatör aksiyonları](docs/OPEN_GATES_AND_OPERATOR_ACTIONS.md): kullanıcı aksiyonları ve production engelleri
- [Operasyon runbook'u](docs/OPERATIONS_RUNBOOK.md): dağıtım, yedekleme, geri yükleme ve alarm akışları
- [Destek matrisi](docs/SUPPORT_MATRIX.md): desteklenen ve sertifikası açık profiller
- [Uyumluluk matrisi](docs/COMPATIBILITY_MATRIX.md): eski/yeni release sınırları
- [Threat model](docs/security/THREAT_MODEL.md): güvenlik varsayımları ve açık riskler
- [Proje durumu](docs/PROJECT_STATUS.md): mevcut özellikler ve gerekli kanıtlar
- [Güvenlik politikası](SECURITY.md): özel zafiyet bildirimi ve açıklama süreci
- [Katkı rehberi](CONTRIBUTING.md) ve [depo kuralları](AGENTS.md)

Bağımlılık güvenlik taraması (`govulncheck`), sabitlenmiş GitHub Actions, haftalık
Dependabot güncellemeleri ve CodeQL Go analizi depo CI'ı üzerinden çalışır. Bu
kontroller tespit ve tedarik zinciri bütünlüğünü güçlendirir; yukarıdaki dağıtıma
özel production tatbikatlarının yerini tutmaz.

## Depo yerleşimi

| Dizin | Sorumluluk |
|---|---|
| `cmd/hydro/` | Uygulama ve operasyon komutlarının giriş noktası |
| `internal/` | Core uygulama ayrıntıları; harici SDK değildir |
| `extension/` | Deneysel v0 müşteri/modül sözleşmeleri |
| `modules/pages/`, `modules/news/` | Modül modelleri, servisleri, repository'leri ve transport katmanı |
| `migrations/` | Lehçe SQL'i ve uyumluluk metadatası |
| `tests/` | Ortak sözleşmeler, entegrasyon testleri ve tüketici fixture'ları |
| `tools/`, `scripts/` | Geliştirme araçları ve production kapı otomasyonu |
| `docs/validation/` | Deney raporları ve doğrulama kapıları |

Compact veya Standard için ticari SLA iddiaları R3/R5 kanıt kapıları kapanana kadar engellidir. Distributed, Redis/Valkey, S3/CDN, OIDC/SAML, gelişmiş arama, commerce, AI ve çok kiracılı özelliklerin provider-neutral ADR sınırları tanımlıdır; yine de gerçek müşteri/sağlayıcı kanıtı ve kendi doğrulama kapıları gerekir. Kalan production işlemleri [`docs/OPEN_GATES_AND_OPERATOR_ACTIONS.md`](docs/OPEN_GATES_AND_OPERATOR_ACTIONS.md) içinde listelenir.

Bu depo özel/proprietary'dir ve açık kaynak lisansı altında yayımlanmamıştır. Kısa teknik dağıtım bildirimi [`PROPRIETARY_NOTICE.md`](PROPRIETARY_NOTICE.md) içindedir; ticari şartlar ayrı bir ürün kararıdır.
