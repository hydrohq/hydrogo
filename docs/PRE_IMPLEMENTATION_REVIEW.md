# HydroGo — Uygulama öncesi mimari inceleme ve kanıt planı

**Tarih:** 2026-09-05  
**Kapsam:** Manifesto, ana plan, ADR-0001–0014 ve karar havuzu.  
**Gerçek durum (2026-09-09):** Depo çalışan Go projesi, CLI, SQLite/PostgreSQL adapter'ları, Pages/News modülleri, SDK sözleşmeleri ve contract testleri içerir. Açık riskler ve gerçek ortam kanıtları [roadmap](ROADMAP.md) ile [üretim kapısı kontrol listesinde](validation/PRODUCTION-GATES.md) izlenir.

## Sonuç

HydroGo için temel yön yeterince belirgindir: müşteri başına ayrı deployment, modüler Go uygulaması, Core fork etmeden Client Edition, ilk sertifikalı DB hedefleri SQLite/PostgreSQL ve ihtiyaçla açılan altyapı katmanları. Ana risk bundan sonra yeni özellik fikri eksikliği değil, planın henüz çalıştırılmamış olmasıdır.

Bu inceleme mimari temel kararlarını ve tespit edilen çelişkilerin karşılıklarını kaydeder. Takip incelemesindeki route/yayın ve restore silme geçmişi açıkları ADR-0014 ile karara bağlanmıştır; uygulama kanıtı bekler. **Uygulama tamamlanmış değildir; Aşama 0'ın dokümantasyon çıktısı hazırlanmıştır.** Sonraki kapı sınırlı teknik deneyler ve ilk çalışan dilimdir. Her sorunun sonsuza kadar kapanmasını beklemek yerine hangi belirsizliğin hangi teslimi engellediği aşağıda belirtilmiştir.

## Tespit edilen sorunlar ve karşılıkları

| ID | Eski planın sorunu | Etki | Güncel karar |
|---|---|---|---|
| F01 | Haber ilk ürün, Site Edition başka yerde öneri | Kapsam ve geliştirme sırası ayrışıyordu | Pages ilk dilim, News ikinci modül; ADR-0009 |
| F02 | Ayrı deployment backlog'da hâlâ açık | Kabul edilmiş karar yeniden tartışılıyordu | ADR-0001 geçerli; backlog düzeltildi |
| F03 | Geniş Module interface kararlı SDK gibi görünüyordu | İki örnekten önce API kilitleniyordu | Sınırlar kabul; Go API v0; iki modül/iki Edition sonrası v1; ADR-0010 |
| F04 | Kök core/ ve aynı ağaçtaki client/ izolasyonu belirsiz | Müşteri iç uygulamaya bağlanabilirdi | Dış Go module, internal + CI allowlist; ADR-0010 |
| F05 | Pasif modülün binary'ye dahil olma durumu açıklanmamıştı | Sıfır yük vaadi ölçülemiyordu | Seçili import/embed ile Edition build; lifecycle/parked jobs; ADR-0010 |
| F06 | Yeni bütün schema sürümlerini reddetme | Expand sonrası eski slot restart edemezdi | Core/modül schema ve format uyumluluk matrisi; ADR-0011 |
| F07 | Worker kendi L1'ini temizliyordu | Diğer web'ler eski içerik sunabilirdi | Her instance'a invalidation + generation uzlaştırması; ADR-0011 |
| F08 | Warm-up sırasında yayın değişmesi ele alınmamıştı | Green eski veriyi trafik geçince sunabilirdi | Cache bypass/revalidation/flush ve stale bütçesi; ADR-0011 |
| F09 | Cache kaybı kesinti yaratmaz gibi mutlak ifade | DB fallback origin'i aşırı yükleyebilirdi | Bounded fallback ve kontrollü 429/503; ADR-0011 |
| F10 | Idempotency alanı çift e-postayı kesin engeller deniyordu | Harici servis belirsizliği gizleniyordu | En az bir kez teslim, provider idempotency, uzlaştırma; ADR-0011 |
| F11 | Repository/audit/outbox aynı transaction'a nasıl bağlanacak belirsizdi | Sözde atomiklik, pratikte ayrı connection riski | Ortak transaction scope; ADR-0011 |
| F12 | PostgreSQL restore dizin rename'e indirgenmişti | Kurtarma prosedürü uygulanamazdı | Ayrı DB restore + kontrollü bağlantı geçişi; ADR-0011 |
| F13 | Büyük medya kopyası boyunca kısa backup kilidi varsayılmıştı | Büyüdükçe yazma kesintisi uzardı | Bütçeli Compact snapshot; büyük profilde snapshot+GC pin; ADR-0011 |
| F14 | Tek binary eski embedded asset'leri sunmayabilirdi | Cache'li sayfalarda eksik CSS/JS | Retained asset inventory ve GC penceresi; ADR-0011 |
| F15 | Restore eski session ve işleri canlandırabilirdi | Eski erişim/çift dış etki | Session iptali, job/egress karantinası; ADR-0011 |
| F16 | Quill formatı ve locale/revision modeli erken örtük bağımlılıktı | Editör/depolama değişikliği zorlaşırdı | Versioned content zarfı, ayrı revision/locale; ADR-0012 |
| F17 | Object write ve DB commit tek işlem gibi ele alınabilirdi | Kayıp/öksüz/private medya riski | Pending/ready + quarantine + GC/backup pin; ADR-0012 |
| F18 | SQLite audit için DB rolü seviyesinde değişmezlik varsayılmıştı | Güvenlik sözü platformu aşıyordu | Append-only API, profile göre sınır, gerekirse dış hedef; ADR-0013 |
| F19 | S3 medya hem planlı hem ertelenmiş/yasak görünüyordu | Distributed kapsamı tutarsızdı | Local pilot; S3 Distributed önkoşulu; SQLite remote dosya farklı konu |
| F20 | Üretimde DB contract testi ifadesi vardı | Müşteri verisinde test mutasyonu riski | Contract CI/izole kopyada; prod güvenli smoke/readiness |

Eski ADR-0008 korunup ADR-0010 ile değiştirildi. ADR-0003 ve 0005'in değişen hükümleri üstlerinde işaretlendi; kalan katman/topoloji kararları geçerlidir. Çelişkiler yeni kararlara bağlanmadan sessizce kapatılmadı.

## Karar, deney ve teslim ayrımı

| Kapı | Şimdi ne durumda? | Kapanması için gereken |
|---|---|---|
| Mimari temel | Dokümante edildi | Uygulama ilerledikçe ADR/plan tutarlılığının korunması |
| Walking skeleton'a geçiş | Deneyler planlandı, başlamadı | S01–S04'ün temel sonuçları, module path/toolchain seçimi; ilk kod bu kanıtı üretir |
| İlk Pages dilimi | Başlamadı | İki DB contract, auth/policy, draft/public ayrımı, yerel medya |
| Ticari pilot | Başlamadı | S05-C, S06–S07, ADR-0014 dış silme günlüğü deneyi, MFA, restore, ölçülü kapasite ve runbook |
| Kararlı SDK/v1 | Başlamadı | İki modül + iki Edition, upgrade matrisi, generator/export ve destek politikası |
| Distributed | Mimari hedef | Çoklu web/worker/L2/object store arıza, yük ve tutarlılık testleri |

“Kodlamadan önce her şeyden emin olmak” için gereken çıktı bir garanti belgesi değildir: test edilebilir sınırlar ve başarısız deney karşısında değiştirilebilecek uygulama ayrıntılarıdır. Go interface imzaları, ORM/query aracı veya slot mekanizması kanıt üretilmeden kararlı API sayılmaz.

## Sınırlı teknik deneyler

Her deney en küçük bağımsız fixture ile yapılır; üretim koduna otomatik taşınmaz. Sonuç raporu kod revizyonu, ortam/sürümler, kullanılan veri, beklenen/görülen davranış, başarısızlık ve seçilen alternatifi içerir. Aşağıdaki testler **çalıştırılmadı**.

| ID | Deney / somut başarı ölçütü | Başarısızsa alternatif | Sahip rol / kapı |
|---|---|---|---|
| S01 | Dış Client Edition Core internal import edemez; izinli extension ile derlenir; seçilmeyen modül route/embed/worker'a girmez | Paket ağacını ve build composition'ı daralt | SDK / skeleton |
| S02 | Aynı service iki DB'de content+audit+outbox yazarken ortada hata üret; üçü de rollback olur. İki editor version yarışında tek başarı olur | Transaction scope ve repository sözleşmesini düzelt; sürücü seçimini yeniden değerlendir | DB / ilk dilim |
| S03 | İki küçük Templ teması aynı view model'le build olur; Quill/HTMX giriş-kayıt-upload akışı CSP altında çalışır | Registry'yi küçült; uyumsuz client bileşenini değiştir; format reader'ı koru | UI / ilk dilim |
| S04 | SQLite ve PostgreSQL driver/migration fixture'ı Windows geliştirme ve Linux amd64 release'de çalışır; seçilen backup/encoder araçları bulunur | Destek matrisini açık daralt veya driver/araç değiştir; karar kaydet | Build/DB / skeleton |
| S05-C | Compact all ve blue/green'de yayın değiştir, invalidation düşür ve geç eski refill yarat; gizli veri sızmasın, toplam stale bütçesi ve DB yükü sınırlı kalsın. Slot geçişinde bypass/revalidation doğrulansın | Mutable cache'i bypass et veya yerel koordinasyonu düzelt | Cache / pilot |
| S05-D | Çoklu web+worker ve L2'de fan-out, reconnect/generation uzlaştırması, kayıp invalidation ve bounded DB fallback kanıtlansın | Mutable cache'i bypass et; dağıtık koordinasyonu düzelt | Cache / L2 kullanan profil; Distributed önkoşulu |
| S06 | Her DB'de online expand sonrası eski binary yeniden başlasın; yeni yazımdan sonra rollback okusun; önceki asset'ler çalışsın | Değişikliği maintenance sınıfına al veya sürümlere böl | Release / pilot |
| S07 | DB+medya yedeğini boş hedefe restore et; hash/referanslar doğru, eski session'lar kapalı, dış işler karantinada | Backup/restore akışını düzelt; RPO/RTO iddiasını yayımlama | İşletim / pilot |
| S08 | Pages ve News + iki Client Edition Core fork olmadan çalışsın; generator iki kez aynı çıktı versin ve custom hash'leri değişmesin | Genel interface'i küçült; generator kapsamını daralt | SDK / v1 |
| S09 | SQLite export → PostgreSQL import ve tersinde mantıksal kimlik/revision/locale/ilişkiler aynı; özel secret/session dışarı çıkmasın | Format/spec'i düzelt; bilinmeyen modülü açıkça reddet | Veri / v1 |

Takip düzeltmeleri: SEO/redirects pilot kapsamındadır; ilk Pages dilimi route/publish atomikliğini kapsar. ADR-0014 dış silme günlüğü deneyi ve ayrıntılı iş sırası [Roadmap](ROADMAP.md) R3'te izlenir.

Tam DB/Go/OS/kütüphane sürümleri burada hayalî biçimde seçilmedi. S04 sonucu destek matrisini, S03 UI bağımlılıklarını sabitler. Go module namespace'i kullanıcının kontrolündeki repository/domain üzerinden ilk `go.mod` öncesinde seçilir; marka adı henüz teknik import yolu zorunluluğu değildir.

## İlk pilot öncesi sayısallaştırılacaklar

- RPO, RTO, izinli yazma duraklaması ve rollback retention penceresi.
- İçerik türü başına toplam `max_stale`, publish propagation alarmı ve CDN/browser policy'si.
- HTTP body/in-flight/deadline, DB pool, queue, worker, image transform ve password hash concurrency limitleri.
- Session idle/absolute süreleri, reset/davet/preview/MFA deneme süre/sayı politikaları.
- Yedek, log/audit, form verisi, revision ve medya retention/GC eşikleri.
- Kabul edilen p95/p99/hata oranı; sıcak/soğuk cache, yazma trafiği, veri hacmi ve donanımı belirtilmiş benchmark.

Bu sayılar tahminle pazarlama vaadi yapılmaz. Pilot fixture'ında ölçülüp config defaults/support matrix'e girer. Go memory limit tek başına prosesin tüm native/cgo/OS bellek kullanımını sınırladığı varsayılmaz; RSS/peak de ölçülür.

## Geliştirme hızının kanıtı

Örnek iş: hazır Site Edition üzerinde hizmetler collection'ı eklemek, beş alan ve bir ilişki tanımlamak, custom validation yazmak, bir liste/detay görünümü bağlamak ve staging'e teslim etmek.

Aynı kabul ölçütünü mevcut yöntemle ve HydroGo ile ölç. Şablonun ilk tasarımı, eğitim/ilk kullanım, tekrar uygulama, hata düzeltme, güvenlik/test ve deployment sürelerini ayrı kaydet. Sonra başka geliştirici ve ikinci müşteri üzerinde tekrarla. 1–2 saat hedefi bu tür standart işe aittir; özgün iş akışları veya tasarım için genel garanti değildir.

CLI komut adları deneysel olabilir; kullanıcının görmek istediği akış sabittir: Edition başlat → hazır modül seç → standart içerik tanımla → özel davranışı ekle → yerelde gör → doğrula → build/deploy → gerektiğinde upgrade/restore. Komut sayısı değil, toplam güvenli teslim süresi başarıdır.

## Ticari kararlardan kalanlar

Mimari bunları destekler; kesin ticari politika ayrıca seçilmelidir:

- Öneri: managed hizmet birincil paket, self-hosted seçenek; lisans süresi public siteyi kapatmaz, güncelleme/destek hakkını etkiler. Offline çalışma koşulu ve enforcement modeli henüz kabul edilmiş ürün sözleşmesi değildir.
- Kaynak kod teslimi, yeniden dağıtım hakkı, bağımlılık lisansları ve marka kullanım sınırı netleştirilmelidir. “Kontrol bizde” bütün üçüncü taraf kodunun telifinin bize geçtiği anlamına gelmez.
- SDK aynı-major kuralı kabul edildi; desteklenen major sayısı, deprecation/LTS süreleri ve kritik güvenlik güncellemesi SLA'sı satış öncesi karara bağlanmalıdır.
- HydroGo/HydroCore adları için ticari marka/domain uygunluğu bu incelemede doğrulanmadı; satış materyali öncesi ayrı araştırma gerekir.

Fiyat veya hukuki lisans koşulu bu incelemeyle belirlenmemiştir. Buradaki kontrol listesi ticari karar girdisidir.

## Bu incelemenin doğrulama sınırı

Yapılan iş: belgelerin karşılaştırılması, çelişkilerin kaydı, yeni ADR'lerin yazılması, eski ADR durumlarının bağlanması, plan/backlog tutarlılığı ve yerel Markdown bağlantılarının kontrolü. Go/DB/işletim/güvenlik test sonucu yoktur. Test kapılarının boş kalması hata gizlemek değil, tamamlanma durumunu doğru tutmaktır.

Teknik mekanizma iddiaları için kullanılan birincil kaynaklar ilgili ADR'lerin Kaynaklar bölümlerindedir: Go internal sınırı; SQLite concurrency; PostgreSQL DDL; transactional outbox; OWASP session/reset/SSRF rehberleri. Kaynaklar HydroGo'nun uygulandığına ilişkin kanıt değildir.
