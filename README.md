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

- Page and news authoring with draft, revision, preview, scheduling, and publishing flows
- Optimistic locking to prevent silent data loss between editors
- Shared application services and authorization rules for HTML and JSON API paths
- Public/private media lifecycle and safe upload metadata handling
- Protected private-media delivery with bounded presigned URLs for authorized editors/admins
- Common SQLite and PostgreSQL behavioral contracts
- Bounded L1 cache, generation invalidation, and single-flight miss coalescing
- MFA, session revocation, password reset, invitations, and CSRF foundations
- Audit, transactional outbox, job leasing, and idempotency infrastructure
- SQLite snapshots, PostgreSQL dump/restore, and media manifest verification
- Hash-chained external erasure log and idempotent erasure replay
- Allowlist-based portable export/import excluding secrets and session state
- Ed25519-signed release manifests and trusted public-key verification
- Deterministic generation, Client Edition boundaries, and an SDK v1 candidate

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

### Modül ve Core sınırları

- `modules/pages`: pages, revisions, drafts, previews, and publishing
- `modules/news`: review, scheduling, publishing, withdrawal, and workers
- `modules/site`: navigation, settings, SEO, redirects, and form inbox
- `internal/auth`: users, sessions, MFA, reset, and invitations
- `internal/media`: uploads, visibility, pending/ready state, and storage metadata
- `internal/cache`: bounded L1, TwoTier cache, and invalidation fanout
- `internal/backup`: SQLite/PostgreSQL snapshots, manifests, restore, and activation
- `internal/release`: manifests, signatures, attestations, retained assets, and Blue/Green state
- `internal/portable`: allowlist-based data portability
- `internal/generator`: reviewable CRUD and migration drafts from collection specs

## Hızlı başlangıç

Gereksinimler:

- Go 1.27.0
- No extra service is required for SQLite
- PostgreSQL 18 and `HYDRO_TEST_POSTGRES_URL` are required for PostgreSQL contract tests

Windows PowerShell:

```powershell
go build -o bin/hydro.exe ./cmd/hydro
$env:HYDRO_DATABASE_DSN = "hydro.sqlite"
./bin/hydro.exe migrate
./bin/hydro.exe serve
```

### Canlı geliştirme

Yerel geliştirmede proxy adresi (`127.0.0.1:8090`) sabit kalırken HydroGo uygulaması içeride yeniden başlatılır. Böylece tarayıcı oturumu korunur; Go, HTML ve Templ değişiklikleri yeniden derlenir, CSS/JS değişiklikleri ise sayfayı yeniler. Derleme hatasında önceki çalışan sürüm ayakta kalır ve tarayıcıda hata ayrıntılarını gösteren kırmızı geliştirme paneli görünür.

```powershell
./bin/hydro.exe run
```

### SDK ile hızlı proje başlatma

Yeni bir Client Edition iskeleti ve frontend entegrasyon manifesti üretmek için:

```powershell
hydro project new haber-merkezi --preset=news
cd haber-merkezi
hydro project check
```

`hydro project check`, varsa `generated/` altındaki tüm resource scaffold'larını
da aynı dependency-free preflight içinde doğrular.

Üretilen projede `frontend/starter.json` ve `frontend/README.md`, Hydro,
shadcn, Material, Carbon, headless veya vanilla sunum katmanlarından birini
seçerek ilerlemenizi sağlar. Collection spec'leri için `hydro resource new`
her kaynak altında tip güvenli `client.ts`, React/SWR/HTMX/vanilla adapter'ları,
`form.schema.json`, OpenAPI ve CRUD yardımcılarını üretir. Bu dosyalar Core'a
çalışma zamanı bağımlılığı eklemez; kimlik doğrulama, politika ve cache
geçersizleştirme Client Edition sınırında kalır. Ayrıntılı akış için
[`docs/SDK.md`](docs/SDK.md), [`docs/PROJECT_GENERATOR.md`](docs/PROJECT_GENERATOR.md)
ve [`docs/CLI_WORKFLOW.md`](docs/CLI_WORKFLOW.md) belgelerine bakın.

Her generated collection kendi deterministic `spec.json` kaynağını taşır;
`hydro resource check --root .` ile tüm kaynakları tek komutla, veritabanına
bağlanmadan doğrulayabilirsiniz. İsterseniz `--spec` vererek tek bir kaynağı
ayrıca kontrol edebilirsiniz.

Varsayılan demo veritabanı `data/demo.sqlite` kullanılır. Farklı bir veritabanı için `--database-dsn`; proxy veya dahili uygulama portları için sırasıyla `--address` ve `--app-address` kullanın.

Uç noktalar:

- `/healthz`: process liveness only
- `/readyz`: database and migration compatibility readiness
- `/metrics`: authenticated operational counters with bounded status labels
- Versioned JSON API routes use `/api/v1/...`; existing `/api/...` paths remain compatibility aliases and advertise their successor.
- Admin/API routes: default-deny routing and authorization policies

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

## Operasyon komutları

```text
hydro run [--address ADDR]
hydro serve
hydro migrate
hydro version --json
hydro doctor
hydro doctor --json
hydro upgrade check
hydro create-admin --email EMAIL --password-stdin
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

`hydro release verify` does not trust the public key embedded in the manifest by itself. It requires a trusted Ed25519 public-key file distributed by the operator.

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

`hydro doctor --json` provides automation-friendly preflight output for the selected profile, database, backup tools, media writability, transport security settings, binary identity, and embedded schema migration.

Dead-lettered background jobs can be reviewed without exposing their payloads:

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

- Blue/Green restart and rollback with two real processes and a reverse proxy
- Restore into clean SQLite and PostgreSQL targets
- External backup and key-recovery drills
- A production signing-key ceremony
- RPO/RTO and capacity measurements using representative customer workloads

Bu kanıtlar oluşana kadar depo herhangi bir production profili veya SLA iddiasında bulunmaz.

## Mimari ilkeler

- Customer work is implemented through Client Editions and extension contracts, not HydroCore forks.
- SQLite and PostgreSQL implement the same domain and application contracts.
- Migration, cache, job, asset, and release changes move forward through explicit contract tests.
- Unused services are disabled in Compact profiles; Distributed features are not defaults.
- Secure-by-default routing, default-deny authorization, and data portability are core contracts.

## Dokümantasyon

- [Roadmap](docs/ROADMAP.md): phases, open gates, and acceptance criteria
- [HydroGo philosophy](docs/HYDROGO_PHILOSOPHY.md): product and architectural principles
- [Implementation plan](docs/implementation_plan.md): scope and target architecture
- [ADR index](docs/architecture/README.md): binding architectural decisions
- [Production gate checklist](docs/validation/PRODUCTION-GATES.md): evidence format for real environments
- [Open gates and operator actions](docs/OPEN_GATES_AND_OPERATOR_ACTIONS.md): user actions, unverified tests, and production blockers
- [Operations runbook](docs/OPERATIONS_RUNBOOK.md): deploy, backup, restore, and alarm flows
- [Support matrix](docs/SUPPORT_MATRIX.md): supported and uncertified profiles
- [Compatibility matrix](docs/COMPATIBILITY_MATRIX.md): old/new release boundaries
- [Threat model](docs/security/THREAT_MODEL.md): security assumptions and open risks
- [Project status](docs/PROJECT_STATUS.md): what HydroGo currently provides and the evidence still required
- [Security policy](SECURITY.md): private vulnerability reporting and disclosure guidance
- [Contribution guide](CONTRIBUTING.md) and [repository rules](AGENTS.md)

Bağımlılık güvenlik taraması (`govulncheck`), sabitlenmiş GitHub Actions, haftalık
Dependabot güncellemeleri ve CodeQL Go analizi depo CI'ı üzerinden çalışır. Bu
kontroller tespit ve tedarik zinciri bütünlüğünü güçlendirir; yukarıdaki dağıtıma
özel production tatbikatlarının yerini tutmaz.

## Depo yerleşimi

| Directory | Responsibility |
|---|---|
| `cmd/hydro/` | Application and operational command entry point |
| `internal/` | Core implementation details; not the external SDK |
| `extension/` | Experimental v0 customer/module contracts |
| `modules/pages/`, `modules/news/` | Module models, services, repositories, and transport |
| `migrations/` | Dialect SQL and compatibility metadata |
| `tests/` | Shared contracts, integration tests, and consumer fixtures |
| `tools/`, `scripts/` | Development tools and production-gate automation |
| `docs/validation/` | Experiment reports and verification gates |

Compact veya Standard için ticari SLA iddiaları R3/R5 kanıt kapıları kapanana kadar engellidir. Distributed, Redis/Valkey, S3/CDN, OIDC/SAML, gelişmiş arama, commerce, AI ve çok kiracılı özelliklerin provider-neutral ADR sınırları tanımlıdır; yine de gerçek müşteri/sağlayıcı kanıtı ve kendi doğrulama kapıları gerekir. Kalan production işlemleri [`docs/OPEN_GATES_AND_OPERATOR_ACTIONS.md`](docs/OPEN_GATES_AND_OPERATOR_ACTIONS.md) içinde listelenir.

Bu depo özel/proprietary'dir ve açık kaynak lisansı altında yayımlanmamıştır. Kısa teknik dağıtım bildirimi [`PROPRIETARY_NOTICE.md`](PROPRIETARY_NOTICE.md) içindedir; ticari şartlar ayrı bir ürün kararıdır.
