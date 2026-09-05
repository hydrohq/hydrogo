# HydroGo (HydroCore) — Birleştirilmiş Uygulama Planı ve Üretim Mimarisi

> **Belge durumu:** Uygulamaya esas ana plan  
> **Proje kod adı:** `HydroGo` / `HydroCore`  
> **CLI:** `hydro` (kısa takma ad isteğe bağlı olarak `hy`)  
> **İlk sürüm hedefi:** SQLite ve PostgreSQL üzerinde aynı iş sözleşmelerini destekleyen, yeni veritabanlarının sertifikalı adaptörlerle eklenebildiği yeniden markalanabilir CMS ve JSON API  
> **Dağıtım modeli:** Uygulama varlıkları gömülü tek çalıştırılabilir dosya; veritabanı, medya, yapılandırma ve sırlar harici kalıcı veri dizininde

> **Gerçekleşme durumu (2026-09-05):** Dokümantasyon aşaması. SDK, uygulama, CLI ve testler henüz uygulanmadı. Bu belgedeki davranışlar hedef ve kabul ölçütüdür. [Hazırlık incelemesi](PRE_IMPLEMENTATION_REVIEW.md) kanıt bekleyen konuları izler.

## 1. Amaç ve kapsam

HydroGo; kurumsal siteler, haber portalları ve mobil uygulama backend'leri için Go ile geliştirilen, çalışma ve üretim derlemesi sırasında Node.js/npm gerektirmeyen, HTML yönetim paneli ile JSON API'nin aynı iş mantığını kullandığı modüler bir CMS ürün fabrikasıdır. Aynı Core sözleşmeleri Compact, Standard ve Distributed kapasite profillerinde kullanılır.

Mimari kararların ana kaynağı [`docs/architecture`](architecture/README.md) dizinidir. Yeni geliştirmeler ilgili ADR'lerle uyumlu olmalıdır.

İlk sürüm yalnızca şu sonucu kanıtlamaya odaklanır:

- Yönetici güvenli biçimde giriş yapabilir.
- Yetkili editör sayfa oluşturabilir, taslak revizyonunu önizleyebilir ve yayımlayabilir. News ikinci modülde inceleme ve zamanlama davranışını kanıtlar.
- HTML yönetim paneli ile JSON API aynı doğrulama ve yetki kurallarını uygular.
- İki editörün eşzamanlı değişiklikleri sessizce birbirini ezmez.
- Veritabanı ve medya birlikte yedeklenip temiz bir kuruluma geri yüklenebilir.
- Müşteri görünüm özelleştirmeleri derleme veya başlangıç sırasında doğrulanır.

İlk ticari ürün Site Edition'dır; kapsam ve pilot/v1/Distributed ayrımı [ADR-0009](architecture/0009-site-edition-and-delivery-scope.md) içindedir. Form Builder, e-ticaret ve yapay zekâ sonraki kapsamlardır. SQLite hızlı yerel geliştirme sunar; PostgreSQL contract testleri ilk Pages diliminden başlar ve kararlı v1 öncesinde tamamlanır. MySQL ve diğer motorlar gerçek talep üzerine adapter sertifikasyonuyla eklenir.

## 2. Temel mimari kararlar

### 2.1 Katmanlar ve bağımlılık yönü

Akış aşağıdaki gibidir:

```text
HTTP Handler (Templ/JSON)
        ↓
Application Service
        ↓
Module-owned Repository Interface
        ↓
SQLite / PostgreSQL / Future Certified Adapter
```

- HTML ve JSON handler'ları doğrudan veritabanına erişmez.
- Doğrulama, yetki, durum geçişleri, işlem sınırları ve audit üretimi uygulama servislerinde yaşar.
- Haberle ilgili model, servis ve repository arayüzleri `modules/news` altında bulunur.
- HydroCore ortak altyapıyı `internal/` altında uygular; sürümlü müşteri/modül yüzeyi `extension/` ve ilan edilmiş modül API'leridir. Medya domain'i ile storage altyapısı ayrıdır. Client Edition ayrı Go module olarak Core'u kullanır; ayrıntı [ADR-0010](architecture/0010-sdk-lifecycle-and-edition-manifest.md) içindedir.
- Servis metotları `context.Context` alır; veritabanı ve dış işlem zaman aşımını taşır.
- Paketler global değişkenlere dayanmaz; bağımlılıklar uygulama başlangıcında açıkça bağlanır.

### 2.2 Önce çalışan modül, sonra jeneratör

İlk Pages modülü, ardından iş akışlı News modülü uçtan uca elle geliştirilir. İki farklı Client Edition ile SDK v0 doğrulanır; ardından v1 yüzeyi ve kod üretici sabitlenir. Tek örneğe özgü ayrıntılar kalıcı jeneratör sözleşmesine dönüşmez. Standart CRUD şeması build girdisidir; çalışma anında otomatik DDL uygulanmaz.

### 2.3 Tek binary sözleşmesi

Tek çalıştırılabilir dosya hedefi için web sunucusu ve işletim komutları aynı binary altında alt komut olarak sunulur:

```text
hydro serve
hydro migrate
hydro backup
hydro restore
hydro admin reset-password
hydro generate module
hydro doctor
```

Templ çıktıları, statik CSS ve sabit istemci kütüphaneleri `embed.FS` ile binary içine alınır. Veritabanı, yüklenen medya, yapılandırma, şifreleme anahtarları ve çalışma logları gömülmez; harici kalıcı kaynaklarda tutulur. PostgreSQL harici servis olabilir. Önceki HTML'nin ihtiyaç duyduğu hash'li asset'ler deploy sırasında imzalı inventory'den ortak asset dizinine çıkarılıp retention süresince korunabilir. Uygulama binary'si, bakım araçlarını içerse de ilk kararlı SDK doğrulanmadan generator tamamlanmış sayılmaz.

## 3. Klasör yapısı

Bu bir hedef yerleşimdir; mevcut depoda yalnız belgeler vardır. Modül içindeki her sorumluluk için hemen ayrı paket açılmaz; ihtiyaç oluşunca ayrılır. `client/` altındaki eski aynı-module örneği yerine dış Client Edition kullanılır.

```text
C:\dev\cms
├── cmd/
│   └── hydro/                 # Tek giriş noktası ve alt komutlar
├── internal/
│   ├── audit/                 # Uygulama düzeyinde append-only audit
│   ├── auth/                  # Kullanıcı, parola, session ve yetki altyapısı
│   ├── config/                # Tipli yapılandırma ve başlangıç doğrulaması
│   ├── database/              # SQLite/PostgreSQL ve genişletilebilir adaptörler
│   ├── cache/                 # Noop, bounded memory ve Redis/Valkey cache
│   ├── jobs/                  # Kalıcı işler, retry ve e-posta worker'ı
│   ├── mailer/                # E-posta sağlayıcı arayüzü
│   ├── media/                 # Dosya doğrulama ve saklama
│   ├── observability/         # Log, metric ve health/readiness
│   ├── router/                # Güvenli router ve middleware zinciri
│   └── ui/                    # İç layout/Templ uygulaması
├── extension/                 # Deneysel v0; kanıt sonrası sürümlü SDK v1
├── modules/
│   ├── pages/                 # İlk uçtan uca içerik modülü
│   └── news/
│       ├── domain/            # Article ve durum kuralları
│       ├── service/           # Use case'ler ve yetki
│       ├── repository/        # Arayüz ve sertifikalı dialect uygulamaları
│       ├── httpadmin/         # Templ/HTMX handler'ları
│       ├── httpapi/           # JSON handler'ları
│       └── ui/                # Haber Templ bileşenleri
├── examples/
│   └── client-edition/        # Ayrı Go module ile dış tüketici fixture'ı
├── migrations/                # Dialect bazlı, sıralı ve değişmez migration'lar
├── docs/
│   └── architecture/          # ADR kayıtları ve karar indeksi
├── static/
│   ├── vendor/                # Sabitlenmiş htmx, Alpine CSP, Quill dosyaları
│   └── css/                   # Önceden üretilmiş Tailwind çıktısı
├── tests/
│   ├── integration/
│   └── fixtures/
├── tools/                     # Geliştirme araçlarının sürüm tanımları
├── Makefile
├── go.mod
├── go.sum
└── config.env.example
```

`schema/user.yaml` kullanılmayacaktır. Çalışma zamanı şeması SQL migration'larıyla yönetilir. Jeneratörün ileride alacağı YAML tanımı ayrı bir `module-specs/` dizininde tutulur ve veritabanı şemasıyla karıştırılmaz.

Gerçek müşteri projesi ayrı repository'de `edition.yaml`, `edition.lock`, `custom/`, `generated/` ve kendi build girişine sahiptir. İmzalanan release metadata'sı `release.json` ile üretilir. Dosya formatları ADR-0010 doğrulama aşamasında sabitlenir.

## 4. Güvenlik mimarisi

Kimlik yaşam döngüsü [ADR-0013](architecture/0013-identity-and-security-boundaries.md), aktörler/güven sınırları ve saldırı testleri [tehdit modeli](security/THREAT_MODEL.md) içindedir. Güvenilen modül kodu aynı proses içinde sandbox değildir. Ticari pilotta internetten erişilen admin/publisher hesapları MFA kapısını geçmelidir.

### 4.1 Varsayılan erişim reddi

- Rotalar açıkça public veya yetki gerektiren rota olarak kaydedilir; sınıflandırılmamış rota başlangıçta hata üretir.
- Yetkilendirme yalnızca role dayanmaz. `CanEdit`, `CanSubmit`, `CanPublish`, `CanDelete` gibi kayıt ve durum bazlı politikalar kullanılır.
- API ve HTML akışlarında aynı servis metotları çağrılır.
- Kimlik doğrulama başarısızlığı ile yetki reddi doğru `401/403` davranışını verir; hassas kaydın varlığını açığa çıkarmamak gereken yerlerde `404` kullanılır.

### 4.2 Oturum ve parola güvenliği

- Parolalar Argon2id ile, sürümlenmiş parametrelerle saklanır; parametreler güvenlik testi ve sunucu kapasitesine göre belirlenir.
- Session kimlikleri kriptografik rastgelelikte üretilir; veritabanında ham değer yerine hash saklanır.
- Çerezler `HttpOnly`, `Secure` ve `SameSite=Lax` olur; üretimde HTTPS zorunludur.
- Girişte session kimliği yenilenir. Mutlak oturum süresi ile boşta kalma süresi ayrı tanımlanır.
- Parola değişikliği ve acil CLI sıfırlaması kullanıcının bütün aktif oturumlarını iptal eder.
- Giriş uç noktasında kullanıcı ve IP düzeyinde hız sınırlama ile artan bekleme uygulanır.
- Hassas formlarda yeniden kimlik doğrulama gerekebilir.
- İlk yönetici rastgele başlangıç parolasıyla otomatik oluşturulmaz; açık bootstrap komutuyla oluşturulur.

### 4.3 CSRF, CSP ve istemci tarafı güvenliği

- Durum değiştiren HTML isteklerinde bağlanmış CSRF token doğrulanır.
- `script-src 'self'` ile nonce/hash gerektirmeyen katı CSP uygulanır; satır içi script ve event handler kullanılmaz.
- Alpine.js CSP dağıtımı kullanılır.
- HTMX için `allowEval: false` ve `allowScriptTags: false` ayarlanır.
- `frame-ancestors 'none'`, `base-uri 'self'`, `object-src 'none'`, uygun `Referrer-Policy`, `X-Content-Type-Options` ve `Permissions-Policy` gönderilir.
- Güvenlik başlıkları ve CSP ihlalleri otomatik test edilir.

### 4.4 Zengin metin güvenliği

- İlk editör adayı Quill'dir. Delta JSON `format`, `format_version`, `document` zarfında saklanır; temiz HTML renderer/sanitizer sürümü taşıyan türetilmiş çıktıdır. Eski formatın okunması ve dönüşümde kayıp raporu [ADR-0012](architecture/0012-content-media-and-portability.md) ile tanımlıdır.
- Sunucu yalnızca tanımlı Delta operasyonlarını kabul eder; istemciden gelen HTML güvenilir sayılmaz.
- HTML sunucuda izin listesiyle temizlenir. Etiketlerin yanında öznitelik ve URL protokolleri de sınırlandırılır.
- `javascript:`, tehlikeli `data:` türleri, event handler'lar, `style`, `iframe`, `script` ve bilinmeyen öznitelikler reddedilir.
- Harici görsel politikası açıkça belirlenir; ilk sürümde yalnızca yerel medya URL'lerine izin verilir.
- Sanitizer sürümü içerikle birlikte saklanır. Politika değiştiğinde Delta'dan toplu HTML yeniden üretme komutu sağlanır.
- `templ.Raw` yalnızca sanitizer tarafından üretilmiş türlendirilmiş güvenli içerik üzerinde kullanılabilir.

### 4.5 Medya yükleme

- Dosya uzantısı, bildirilen MIME ve dosya imzası birlikte kontrol edilir.
- İzin verilen formatlar ilk sürümde JPEG, PNG ve WebP ile sınırlandırılır; SVG kabul edilmez.
- Dosya ve açılmış görüntü için byte, piksel ve boyut sınırı uygulanır.
- Görseller decode edilip yeniden encode edilerek tehlikeli ek veri ve metadata temizlenir.
- İstemci dosya adı saklama anahtarı olarak kullanılmaz; rastgele/icerik-hash tabanlı ad kullanılır.
- Medya web kökünün dışında saklanır ve kontrollü handler üzerinden doğru `Content-Type` ile sunulur.
- Sahipsiz medya temizliği referans ve bekleme süresi üzerinden güvenli bir iş olarak çalışır.
- Private/public visibility, pending→ready upload ve crash/GC davranışı ADR-0012'ye tabidir. Draft medya sırf hash'li URL aldı diye public sayılmaz; backup pin ve revision referansları GC'den korunur.

## 5. Veritabanı, cache ve tutarlılık

### 5.1 Veritabanı profilleri

Domain ve application service kodu SQL dialect'i bilmez. İlk sürümde SQLite ve PostgreSQL aynı modül repository sözleşmelerini uygular; sorgular, migration'lar ve transaction ayrıntıları sürücüye aittir. Yeni motorlar capability ve contract testleriyle sertifikalı adaptör olarak eklenir. Ayrıntılı karar [ADR-0006](architecture/0006-extensible-database-adapters.md) içinde tanımlıdır.

#### SQLite

İlk sürüm tek uygulama instance'ı ve yerel kalıcı NVMe/SSD üzerinde çalışır. NFS ve ağ dosya sistemleri desteklenmez.

Her yeni bağlantıda aşağıdaki ayarlar uygulanır ve sonuçları başlangıçta doğrulanır:

```sql
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;
```

`synchronous=NORMAL` performans ile güvenlik dengesi olarak kabul edilir; güç kaybında son onaylanmış işlemlerin kaybolabileceği işletim kararına kaydedilir. Finansal/sipariş profilinde `FULL` veya PostgreSQL kullanılır.

- Connection pool boyutu SQLite'ın tek yazarlı yapısına göre sınırlandırılır.
- Yazma transaction'ları kısa tutulur; ağ, e-posta veya dosya dönüştürme işlemleri transaction içinde yapılmaz.
- Migration'lar sürümlü, sıralı ve yayınlandıktan sonra değişmezdir.
- Uygulama Core/modül bazında gerekli migration/checksum ve ilan edilmiş schema okuma/yazma uyumluluğunu doğrular. Uyumlu expand migration sonrası eski binary başlayabilir; destek aralığı dışındaki şema reddedilir. Ayrıntı [ADR-0011](architecture/0011-consistency-upgrade-and-recovery.md) içindedir.
- Migration öncesi otomatik yedek ve geri dönüş prosedürü tanımlanır.
- `PRAGMA integrity_check` ve `foreign_key_check` bakım/geri yükleme doğrulamasında çalıştırılır.

#### PostgreSQL

PostgreSQL çoklu instance, yüksek yazma eşzamanlılığı ve scale-out kurulumların önerilen sürücüsüdür. Connection pool, isolation seviyesi, serialization/deadlock retry ve online migration davranışı açıkça yapılandırılır.

#### Gelecekteki adaptörler

MySQL ve diğer veritabanları yalnız connection driver eklenerek desteklenmiş sayılmaz. Modül sorguları, migration, error mapping, transaction/locking, backup/restore ve ortak contract testleri tamamlandığında sertifikalı adaptör olur.

### 5.2 İşlem sınırları ve kalıcı işler

Bir içerik değişikliği, yeni revizyon, geçerli durum geçişi, audit olayı ve gerekiyorsa kalıcı iş kaydını tek transaction içinde yazar. Repository, audit ve outbox aynı transaction scope'undan bağlanır; bağımsız pool çağrılarına ayrılmaz. E-posta ve ileride açılacak webhook gibi dış etkiler commit sonrasında kalıcı iş kuyruğundan işlenir.

- İşler benzersiz idempotency anahtarı taşır.
- Hatalarda sınırlı exponential backoff ve dead-letter durumu kullanılır.
- Worker yeniden başladığında yarım kalan işler güvenli biçimde devam eder.
- Tekrarlanan komutlarda DB içi yayın değişikliği idempotent korunur. Dış etkiler en az bir kez teslim edilir; sağlayıcı idempotency sunmadığında e-posta timeout'u belirsiz sonuç/tekrar riski taşır. Genel exactly-once garantisi verilmez. Lease, fencing, tüketici durumu ve payload uyumluluğu ADR-0011'de tanımlıdır.

### 5.3 Eşzamanlı düzenleme

`articles` tablosunda artan `version` alanı bulunur. Güncelleme şu koşulla yapılır:

```sql
UPDATE articles
SET ..., version = version + 1
WHERE id = ? AND version = ?;
```

Sıfır satır değişirse servis `conflict` sonucu döndürür. Yönetim paneli kullanıcıya kayıt yeniden yükleme ve değişiklikleri karşılaştırma imkânı verir; sessiz son-yazan-kazan davranışı kullanılmaz.

### 5.4 Çok katmanlı cache

Cache; request-scope memoization, bounded L1 memory, isteğe bağlı Redis/Valkey uyumlu L2 ve HTTP/CDN L3 katmanlarından oluşur. Compact profilde harici cache servisi gerekmez; Distributed profilde dağıtık cache zorunludur.

- Cache yalnız yeniden üretilebilir veri taşır; arızada bounded DB fallback ve backpressure uygulanır. Kapasite aşımında kontrollü 429/503 olabilir; asıl veri kaybolmaması kesintisizlik garantisi değildir.
- TTL, jitter, versioned key, tag invalidation, stampede koruması ve boyut sınırı zorunludur.
- Publish transaction'ı cache/CDN invalidation için outbox olayı yazar.
- Admin, preview, session, audit, para ve stok doğruluk kayıtları public/shared cache'e girmez.
- Hit/miss, latency, eviction, byte kullanımı ve invalidation failure ölçülür.

Ayrıntılı katman sözleşmesi [ADR-0003](architecture/0003-cache-architecture.md), güncel fan-out/generation/stale bütçesi ve arıza kuralları [ADR-0011](architecture/0011-consistency-upgrade-and-recovery.md) içindedir. Worker'ın yerel L1 temizliği diğer web proseslerinin L1'ini temizlemez; ayrık topoloji bu testi geçmeden sertifikalanmaz.

## 6. Haber alan modeli

Bu bölüm ikinci modül News'in davranışıdır; ilk Pages dilimi taslak/yayın/revision üzerine kurulur. Ortak içerik kimliği, locale, route ve medya sınırları [ADR-0012](architecture/0012-content-media-and-portability.md) içindedir. Genel Pages modeline News'in bütün iş akışı yüklenmez.

### 6.1 Durumlar

```text
draft → in_review → scheduled → published → archived
   ↑         ↓           ↓           ↓
   └─────────┴───────────┴───────────┘ (yetkili geri çekme)
```

- `preview` kalıcı durum değildir; yetkili kullanıcının belirli revizyonu görüntüleme işlemidir.
- Her geçişin kaynak durumu, hedef durumu ve gerekli yetkisi açık bir tabloda tanımlanır.
- Zamanlanmış yayın UTC olarak saklanır; arayüzde kullanıcı saat dilimine çevrilir.
- Yayındaki içerik düzenlenirken yeni taslak revizyon oluşturulur; canlı sürüm kaydetme anında değişmez.
- Önizleme URL'si kısa ömürlü, imzalı ve revizyona bağlıdır; arama motorlarınca indekslenmez.

### 6.2 Kimlik, slug ve SEO

- Kayıt kimliği URL'den bağımsız sabit bir ID'dir.
- Slug benzersizlik ve normalizasyon kuralları tanımlanır.
- Taslak slug değişikliği canlı yolu etkilemez. Publish sırasında önceki aktif yol gerekiyorsa 301 yönlendirmesine eklenir; path conflict ve döngü kontrolleri aynı transaction içinde uygulanır. [ADR-0014](architecture/0014-publication-routes-and-erasure-recovery.md) bağlayıcıdır.
- Canonical URL, meta başlık/açıklama, Open Graph alanları, sitemap ve geçmiş URL yönetimi Pilot Site Edition kabul ölçütüdür (Aşama 3). İlk Pages dilimi route kimliği ve taslak/publish atomikliğini kanıtlar.
- Silme ilk aşamada soft-delete olur; kalıcı silme ayrı yetki, bekleme süresi ve audit gerektirir.

## 7. API sözleşmesi

- API `/api/v1` altında sürümlenir.
- Liste uçları cursor tabanlı sayfalama ve üst sınır kullanır.
- Hatalar tek bir makinece okunabilir biçimde döner: `code`, `message`, `fields`, `request_id`.
- Tarihler RFC 3339 ve UTC olarak gönderilir.
- İstek gövdelerinde bilinmeyen alanlar reddedilir; boyut sınırı uygulanır.
- Yalnız izin verilen alanlar yazılabilir; toplu model binding ile rol, yayın durumu veya sahiplik değiştirilemez.
- Public API yalnız yayımlanmış güncel revizyonu döndürür. Taslak ve zamanlanmış içerik yetkisiz çağrıda açığa çıkmaz.
- ETag/`If-Match` veya açık sürüm alanı ile API güncellemelerinde de optimistic locking uygulanır.
- OpenAPI sözleşmesi kodla birlikte sürümlenir ve sözleşme testlerinde doğrulanır.
- Mobil istemci kimlik doğrulaması yönetim paneli session'ından ayrı bir ADR ile belirlenmeden public olmayan mobil API yayımlanmaz.

## 8. View Registry ve müşteri özelleştirmesi

Sabit sözleşme tipli view/component slotudur; runtime registry aşağıdaki davranışlara gerçekten ihtiyaç olduğu iki temada kanıtlanırsa kullanılır. İlk implementation derleme zamanı composition ile başlayabilir. SDK olgunluğu [ADR-0010](architecture/0010-sdk-lifecycle-and-edition-manifest.md) ile yönetilir.

- Çekirdek ve modüller kararlı görünüm anahtarlarını açıkça kaydeder.
- Müşteri override'ı bilinmeyen anahtar, eksik parametre veya uyumsuz sözleşme kullanırsa derleme ya da başlangıç başarısız olur.
- Templ bileşenleri derleme zamanı Go kodudur; çalışma zamanında keyfi `.templ` dosyası yüklenmez.
- Override'lar aynı binary içine derlenir. Bir müşteri tasarımını değiştirmek yeni build gerektirir.
- Registry çakışmaları deterministik değildir diye sessizce çözülmez; çift kayıt hata üretir.
- Haber kartı ve detay sayfası override'ı snapshot/golden görsel veya HTML yapısı testiyle doğrulanır.

## 9. Yedekleme ve geri yükleme

Güncel protokol [ADR-0011](architecture/0011-consistency-upgrade-and-recovery.md) içindedir. Backup, mantıksal export ve binary rollback ayrı işlemlerdir.

### 9.1 Tutarlı yedek

`hydro backup` driver'ın güvenilir snapshot mekanizmasını kullanır. SQLite Online Backup API veya doğrulanmış `VACUUM INTO`; PostgreSQL sürüme uygun backup aracı veya sertifikalı sağlayıcı snapshot/PITR entegrasyonu ile çalışır. Adapter araç/sürüm gereksinimleri `hydro doctor` tarafından doğrulanır. Tek uygulama binary'si harici DB backup aracının gereksiz olduğu anlamına gelmez.

- İlk Compact yöntemi uygulama/worker yazmalarını kontrollü durdurabilir. Byte ve süre bütçesi önceden ölçülür; bütün medya kopyası boyunca kısa duraklama garantisi verilmez.
- Büyük/dağıtık yöntemde DB snapshot'ının referanslı immutable medya manifest'i GC pin/retention ile korunur. GC ile pin yarışı engellenir; büyük kopya boyunca bütün yazmalar durdurulmaz. Bu yöntem profil testi geçmeden sunulmaz.
- Manifest backup kimliği, Core/modül/schema sürümleri, dosya/nesne envanteri ve SHA-256 doğrulamalarını içerir. Eksik medya tamamlanmış yedek kabul edilmez.
- Arşiv belgeli güçlü şifreleme kullanır; anahtar aynı yerde açık saklanmaz. Secret/key kurtarma ayrı kontrollü kanaldadır.
- Geçici çıktı doğrulanmadan tamamlanmış olarak işaretlenmez. Başarısızlık kilitleri/pinleri güvenli biçimde çözer, korunması gereken nesneleri erken silmez.
- Sunucu dışı kopya, saklama, son başarı/yaş alarmı ve düzenli restore tatbikatı zorunludur.

### 9.2 Geri yükleme

`hydro restore` çalışan uygulamanın hedefini doğrudan ezmez; boş/staging hedef kullanır. Önce manifest, sürüm, DB bütünlüğü/referansları ve medya checksum'ları doğrulanır.

- SQLite kapalı staging veri dizininde hazırlanıp platforma uygun biçimde devreye alınır; WAL/SHM ve açık prosesler ayrıca kontrol edilir.
- PostgreSQL ayrı DB/instance'a restore edilir. Smoke doğrulaması ve kontrollü yazma durdurma sonrasında bağlantı hedefi değiştirilir; dosya dizini rename işlemi PostgreSQL restore değildir.
- İlk açılışta worker ve dış egress kapalıdır. Eski outbox/e-posta/webhook etkileri uzlaştırılır; session'lar varsayılan iptal edilir. Silme geçmişi ADR-0014 uyarınca restore'dan bağımsız dış günlükten alınır; güncel ve eksiksiz olduğu doğrulanmadan trafik açılmaz. Secret/key sürümleri değerlendirilir.
- Eski deployment yeniden yazamayacak şekilde kapatılır. Binary rollback ile yedekten restore karıştırılmaz; restore yedekten sonraki yazımları kaybettirebilir.

RPO/RTO, kabul edilen yazma duraklaması, retention ve anahtar kurtarma hedefleri ilk pilot öncesi sayılandırılır. DB PITR ve medya geçmişi birlikte doğrulanmadan bütün ürün için PITR garantisi verilmez. DB bağımsız export/import modeli ADR-0012'ye tabidir.

## 10. Audit, gizlilik ve veri yaşam döngüsü

- Giriş, başarısız giriş, parola/session değişikliği, içerik oluşturma/güncelleme/durum geçişi/silme, yetki değişikliği, yedekleme ve geri yükleme audit edilir.
- Audit olayı aktör, eylem, hedef, zaman, request ID ve güvenli özet içerir; parola, session tokenı veya hassas tam gövde içermez.
- Audit API'si append-only'dir. PostgreSQL runtime rolü UPDATE/DELETE yapamaz; migration/bakım rolü ayrıdır. SQLite dosyasına tam erişim sahibi karşısında aynı DB rol garantisi yoktur; OS izolasyonu ve gerekirse dış audit hedefi kullanılır. Saklama/dışa aktarma politikası vardır; ayrıntı ADR-0013'tedir.
- KVKK için hangi kişisel verinin neden işlendiği, saklama süresi, silme/anonimleştirme ve veri dışa aktarma süreçleri tanımlanır.
- Rıza çubuğu yalnız gerçekten rıza gerektiren istemci depolaması/izleme varsa gösterilir; zorunlu session çerezi ile analitik rıza ayrılır.

## 11. Gözlemlenebilirlik ve işletim

- JSON biçimli yapılandırılmış loglarda request ID bulunur; sırlar ve kişisel içerik maskelenir.
- `/healthz` yalnız prosesin yaşadığını, `/readyz` veritabanı ve migration uygunluğunu gösterir.
- HTTP sunucusunda read-header, read, write ve idle timeout; gövde boyutu ve kontrollü graceful shutdown bulunur.
- Temel metrikler: istek süresi/hata oranı, DB pool/lock/deadlock/busy durumu, cache hit/miss/eviction, kuyruk derinliği, başarısız işler, disk kullanımı, yedek yaşı ve son yedek sonucu.
- Disk doluluk eşiği, sertifika süresi, başarısız yedek ve migration hatası için alarm tanımlanır.
- Build sürümü, commit ve şema uyumluluğu `hydro version` ve erişimi korunan diagnostics çıktısında bulunur; public health/readiness ayrıntılı iç bilgi döndürmez.
- Sırlar environment veya secret file üzerinden alınır; loglanmaz ve `config.env.example` içine gerçek değer konmaz.

## 12. Geliştirici deneyimi ve tedarik zinciri

- Üretim binary'sini derlemek için Node.js/npm gerekmez. Tailwind CSS çıktısı sabitlenmiş standalone araçla geliştirme veya release hazırlığı sırasında üretilip repoya alınır.
- Go, Templ, Air, Tailwind standalone ve diğer araç sürümleri sabitlenir.
- Windows için PowerShell komutları veya çapraz platform Go tabanlı görev aracı sağlanır; `make` zorunlu tek yol olmaz.
- Üçüncü taraf JS dosyaları sürüm ve SHA-256 ile sabitlenir; lisansları dağıtıma eklenir.
- Release build yeniden üretilebilirlik, `go test`, `go vet`, format, migration ve gömülü varlık kontrollerinden geçer.
- `go:generate` çıktılarının güncel olup olmadığı CI'da doğrulanır.

## 13. Test stratejisi

Testler katman davranışını ve kritik riskleri kanıtlar:

- **Birim:** Durum geçişleri, yetki politikaları, slug ve sanitizer kuralları.
- **Repository entegrasyonu:** SQLite ve PostgreSQL üzerinde ortak contract testleri; migration, foreign key, transaction, deadlock/retry ve optimistic locking. Gelecekteki adaptör aynı test kitini kullanır.
- **HTTP entegrasyonu:** CSRF/CSP, auth, HTML/API davranış eşitliği, doğru hata kodları ve taslak sızıntısı olmaması.
- **Güvenlik:** Zararlı HTML/URL örnekleri, dosya imzası uyuşmazlığı, aşırı büyük görsel, session fixation ve rate limit.
- **Cache:** TTL/jitter, invalidation, stampede, bounded memory, L2 kesintisinde DB fallback ve gizli içeriğin cache'lenmemesi.
- **Yedek/geri yükleme:** Her veritabanı sürücüsünde eşzamanlı yazma altında tutarlı snapshot, manifest bozulması, eksik medya ve temiz kuruluma gerçek restore.
- **Kapasite profilleri:** Compact kaynak bütçesi ve Distributed concurrency/topoloji testi.
- **Migration:** Desteklenen eski üretim şemasından geçiş, uyumlu expand sonrası eski binary restart, aralık dışı schema reddi ve her dialect için online/backfill/maintenance davranışı.
- **Eşzamanlılık:** İki editör aynı sürümü kaydettiğinde birinin açık conflict alması.
- **View Registry:** Bilinmeyen override ve çift kaydın fail-fast davranışı.
- **Kurtarma/dağıtım:** Yeni yazımdan sonra eski binary rollback; retained asset'ler; restore edilmiş session iptali ve job/egress karantinası; gerçek müşteri DB'sinde yazan contract testleri çalıştırılmaması.

Testler yalnız uygulamayı tekrar eden yüzeysel kontrollerden oluşmaz; her biri tanımlı bir riski doğrular.

## 14. Aşamalı yol haritası ve çıkış ölçütleri

Her aşama uygulama/test kanıtıyla kapanır. Kabul edilmiş ADR uygulamanın tamamlandığı anlamına gelmez. Ayrıntılı deneyler [hazırlık incelemesinde](PRE_IMPLEMENTATION_REVIEW.md) izlenir.

İş bazında ilerleme ve kanıt kapıları [Roadmap](ROADMAP.md) içinde izlenir; R0–R6 bu aşamalarla eşleşir.

### Aşama 0 — Mimari temel ve risk kaydı

- İlk ürün/tenancy, schema/custom sınırı, SDK yaşam döngüsü, cache/transaction, migration, içerik/medya ve auth kararları ADR-0009–0013 ile netleştirilir.
- Tehdit modeli, kapsam dışı özellikler ve kararların doğrulama yöntemi kaydedilir.
- Açık soruların hangi teslimi engellediği belirtilir; bütün geleceğin detaylarını kapatma şartı yoktur.

**Çıkış:** Temel kararlar tutarlı; kalan belirsizliklerin sahibi, deney ve kapısı belli. Bu belge güncellemesi tasarım kısmını sağlar, teknik doğrulama aşağıdaki aşamada bekler.

### Aşama 1 — Teknik deneyler ve walking skeleton

- Dış Client Edition import sınırı, Go module yolu, toolchain/driver/migration/UI bağımlılıkları değerlendirilir ve sürümlenir.
- `go mod`, tek `hydro` giriş noktası, tipli config, manifest preflight, health/readiness ve graceful shutdown.
- SQLite ve PostgreSQL bağlantısı, transaction-scope, migration ledger/uyumluluk deneyi.
- Default-deny router, session/bootstrap, CSRF/CSP ve temel audit.
- Compact kaynak bütçeleri, bounded L1 ve public HTTP cache; ek queue/cache servisi zorunlu değil.
- Windows geliştirme ve Linux amd64 build akışı; Templ/editör/HTMX CSP uyumu.

**Çıkış:** Temiz ortamda build/giriş yapılır; iki DB'de transaction rollback ve gerekli migration kontrolleri geçer. Somut teknoloji seçimleri deney sonuçlarıyla kaydedilir.

### Aşama 2 — Elle Pages ve ilk Client Edition

- Pages oluşturma/düzenleme, draft/revision/publish, optimistic locking, locale/route temeli.
- Admin, public HTML ve JSON aynı service/policy üzerinden çalışır.
- Sürümlü rich-text, sanitizer ve private/public medya akışı.
- Local storage, sınırlı cache invalidation ve kalıcı outbox.
- Tipli view slotlarıyla gerçek müşteri teması; Core fork veya internal import yok.

**Çıkış:** İki editör çakışması, taslak/özel medya sızıntısı ve SQLite/PostgreSQL ortak davranış testleri geçer. SDK hâlâ v0'dır.

### Aşama 3 — Pilot Site Edition ve işletim

- Navigation, settings, SEO/redirects, sınırlı form inbox, revisions ve public API tamamlanır.
- MFA, davet/reset/kurtarma, role/session iptali; güvenlik fixture'ları.
- Backup/restore, harici kopya, key kurtarma; restore job/egress karantinası ve dış silme günlüğü/checkpoint deneyi (ADR-0014).
- Ölçülmüş RPO/RTO, kaynak/soak testleri, log/metrik/alarmlar, veri saklama ve runbook.
- Sertifikalanacak tek sunucu profilinde online migration, blue/green, eski asset ve rollback deneyi.

**Çıkış:** Pilotun gerçek müşteri akışı, güvenlik ve işletim kapıları geçer. Henüz doğrulanmamış Distributed desteği veya kararlı SDK sözü verilmez.

### Aşama 4 — News, ikinci Edition ve SDK doğrulaması

- News: draft/in-review/scheduled/published/archived, kayıt/alan yetkisi, schedule iptali, revizyon preview.
- İkinci tema ve özel iş kuralı; iki modülde aynı transaction/extension sınırı.
- İki DB için job lease, tekrar/sıra değişimi, migration ve cache contract kitleri.
- Core/modül/config/content/event/job/asset release uyumluluk matrisi; geriye uyumluluk fixture'ları.

**Çıkış:** İki farklı modül ve iki markalı dış Client Edition geçer; SDK v1'e açılacak küçük yüzey belirlenir. Gelecekteki DB sertifikasyon kiti yayıma hazırdır.

### Aşama 5 — Generator ve kararlı v1

- Sürümlü collection spec'ten Go/Templ CRUD ve incelenecek dialect migration taslağı.
- Deterministik üretim, diff, generated/custom koruması; published migration checksum'ları değişmez.
- Yetki varsayılanı deny; özel iş kuralı ve destructive değişiklik tahmin edilmez.
- Gerçek standart müşteri işinde süre ölçümü; tekrar üretim ve özel kural değişikliği dahil.
- SDK v1, SQLite/PostgreSQL sertifikasyonu, taşınabilir export/import fixture'ı, desteklenen topoloji/OS/DB sürüm matrisi.
- Lisans/destek/deprecation takvimi ve sürüm yükseltme rehberi yayımlanmadan önce kesinleşir.

**Çıkış:** Belgeli kapsamıyla satılabilir v1; eksik profile ait ölçümler başarı gibi gösterilmez.

### Aşama 6 — Talep ile genişleme

- PostgreSQL üzerinde çoklu web/worker, L2, S3 uyumlu object storage ve CDN içeren Distributed profil; fan-out/reconnect, aşırı yük ve arıza testleri.
- Gerçek talebe göre MySQL veya yeni sertifikalı adapter; bulk veri taşıma.
- Form Builder, gelişmiş arama, OIDC/SAML ve private mobil API ayrı kapsam/tehdit incelemeleriyle.
- Commerce para/stok/transaction/ledger ADR'leriyle; AI ise yetki, gizlilik, audit, maliyet ve insan kontrolüyle ayrı tasarlanır.

**Çıkış:** Her yeni özellik ve profil kendi kanıtıyla destek matrisine eklenir.

## 15. Üretime çıkış kontrol listesi

- [ ] Tüm rotalar public/yetkili olarak sınıflandırılmış ve default-deny testi geçiyor.
- [ ] Session yenileme, süre aşımı, iptal ve rate limit testleri geçiyor.
- [ ] Ticari pilot için admin/publisher MFA, davet/reset ve kurtarma akışları doğrulanmış.
- [ ] HTML sanitizer ve medya kötüye kullanım testleri geçiyor.
- [ ] SQLite PRAGMA'ları ve PostgreSQL pool/isolation ayarları başlangıçta doğrulanıyor.
- [ ] SQLite ve PostgreSQL ortak contract ve migration testlerini geçiyor.
- [ ] Cache sınırlı, gözlemlenebilir ve L2 kesintisinde güvenli fallback yapıyor.
- [ ] Release ile desteklendiği ilan edilen her profil tanımlı referans yükte benchmark/soak testlerini geçiyor; Distributed ayrıca sertifikalanıyor.
- [ ] Online migration içeren release blue/green ile public kesinti oluşturmuyor.
- [ ] Eşzamanlı düzenleme sessiz veri kaybı üretmiyor.
- [ ] Public API taslak, silinmiş veya zamanlanmış içeriği sızdırmıyor.
- [ ] Migration eski üretim kopyası üzerinde denenmiş.
- [ ] Yedek sunucu dışında saklanıyor ve temiz ortama restore edilmiş.
- [ ] Restore sonrası session iptali ve job/egress karantinası doğrulanmış; geçmiş silme işlemleri uzlaştırılmış.
- [ ] RPO/RTO, saklama ve anahtar kurtarma prosedürü yazılmış.
- [ ] Loglarda parola, token ve hassas içerik bulunmadığı doğrulanmış.
- [ ] HTTP timeout, graceful shutdown, disk ve yedek alarmları çalışıyor.
- [ ] Binary, JS/CSS varlıkları, lisanslar ve sürüm bilgisi eksiksiz.
- [ ] Release imzası/manifest'i doğrulanmış; schema/content/job/config uyumluluğu ve önceki asset retention'ı test edilmiş.
- [ ] Runbook: deploy, rollback, backup, restore, parola sıfırlama ve disk dolması senaryolarını kapsıyor.

## 16. Ertelenmiş kararlar

Aşağıdaki konuların destek ve erteleme sınırı açıktır. Yeni yetenekler açılmadan önce ilgili ADR güncellenir veya yeni karar yazılır:

- SQLite dosyasını NFS veya object storage üzerinde çalıştırma desteklenmez. Medya için S3 uyumlu adapter ADR-0012 ile planlıdır; Distributed sertifikasyonu öncesi uygulanır.
- Çok kiracılı veri modeli.
- Mobil özel API token/OAuth modeli.
- E-ticaret ödeme, stok ve sipariş tutarlılığı.
- Sertifikalı veritabanları arasında kesintisiz canlı veri taşıma.
- Yapay zekâ araçlarının yazma/yayımlama yetkisi.

Bu planın başarısı dosya ve katmanların oluşturulmasıyla değil; güvenli yayınlama akışının, çakışmasız düzenlemenin ve doğrulanmış geri yüklemenin uçtan uca kanıtlanmasıyla ölçülür.
