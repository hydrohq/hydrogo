# ADR-0011: Cache tutarlılığı, işlem sınırı, yükseltme ve kurtarma

- **Durum:** Kabul edildi
- **Tarih:** 2026-09-05
- **Kısmen değiştirdiği kararlar:** [ADR-0003](0003-cache-architecture.md) invalidation/arıza garantisi; [ADR-0005](0005-zero-downtime-deployment.md) warm-up/geçiş doğrulaması.
- **Kapsam:** Önceki kararların L0–L3 katmanları, profil seçimi, session/asset kuralları, proxy ve expand–migrate–contract yaklaşımı korunur. Aşağıdaki maddeler çelişen genel ifadelerin yerine geçer.
- **Uygulama durumu:** Migration, cache, restore, blue/green ve job/lease hükümlerinin önemli bölümü uygulanıp contract testleriyle doğrulandı. Gerçek proxy/process, multi-instance ve müşteri restore tatbikatları açık; güncel durum [roadmap](../ROADMAP.md) içindedir.

## Bağlam ve bulgular

Worker ayrı proses ise kendi L1'ini temizlemek web L1'lerini temizlemez. Release sürümü içeren cache key, aynı release sırasında değişen yayın verisini güncel tutmaz. Daha yeni her şemayı reddeden eski binary expand migration sonrasında yeniden başlayamaz. Bir transaction'dan sonra gönderilen e-postanın ağ cevabı kaybolursa yalnız idempotency alanı çift göndermeyi engellemez. PostgreSQL restore dosya dizini rename işlemine indirgenemez.

## İşlem ve job sözleşmesi

- İçerik, revision, yayın işaretçisi, audit ve outbox aynı DB transaction'ında yazılır. Modül repository ve Core audit/outbox portları aynı transaction scope'undan oluşturulur; ayrı pool bağlantısına kaçamaz. Commit başarısızsa hiçbiri görünür olmaz.
- Transaction retry yalnız tekrar çalıştırılması güvenli DB işi üzerinde, sınırlı deneme ve deadline ile yapılır. HTTP sonucu commit tamamlanmadan başarılı sayılmaz.
- Job kuyruğu başlangıçta aynı DB'dedir: `pending/leased/succeeded/dead/parked`, attempt, next-attempt, lease-owner, lease-until, fencing/claim token ve payload-version taşır. Tam Go alan isimleri prototipte belirlenir.
- Claim atomiktir. Bitirme/yenileme yalnız güncel claim token sahibinden kabul edilir. DB tarafındaki zaman ve adapter testleri lease hesabını tutarlı kılar. Süresi biten eski worker dış servise hâlâ ulaşabilir; fencing DB güncellemesini korur, harici yan etkiyi tek başına korumaz.
- Tüketici başına teslim kaydı ve idempotency scope'u vardır. Aggregate revision sırası eski olayın yeni durumu geri almasını engeller. Bilinmeyen payload sürümü karantinaya alınır; uyumsuz worker tarafından çalıştırılmaz.
- Teslim **en az bir kez** olur. Sağlayıcının idempotency anahtarı varsa kullanılır. Yoksa örneğin e-posta kabul edildikten sonra cevap kaybolduğunda tekrar gönderim ihtimali açıkça kalır; uzlaştırma/operatör kararı gerekir. Genel exactly-once vaadi verilmez.
- Compact polling/backoff ile sınırlı worker kullanır. PostgreSQL claim/locking optimizasyonları adapter içindedir. Yeni DB aynı crash/lease/retry contract kitini geçer.

## Cache ve yayın görünürlüğü

1. Immutable revision gövdesi cache edilebilir. Hangi revision'ın güncel olduğu ayrı mutable lookup'tır; yalnız key'e revision yazmak bu lookup'ın eskimesini çözmez.
2. Yayın transaction'ı içerik ve koleksiyon generation'larını artırır; outbox olayı bu sürümleri taşır. L1 girdisi yüklendiği generation ile etiketlenir. Geç tamamlanan eski okuma yeni generation altında yazılamaz.
3. Tek `all` prosesinde commit sonrası yerel invalidation yapılır; outbox retry yolu kalır. Web/worker ayrıldığında **her web prosesine** bildirim/fan-out gerekir. Tek kuyruk mesajını tek worker'ın alması fan-out değildir.
4. Compact blue/green geçişinde mutable public response/lookup cache'i geçici bypass edilir. Green readinesstan sonra warm-up yapılabilir; trafik geçmeden generation yeniden doğrulanır veya ısıtılan mutable cache temizlenir. Slot geçişi boyunca yayın değişmesi eski ısınmış cevabı sabitlemez.
5. Distributed profilde L2 invalidation yanında her instance için kopma tespiti ve generation uzlaştırması bulunur. Pub/sub tek başına kayıpsız doğruluk kaydı değildir. Kopma sırasında mutable L1 bypass/flush edilir; yeniden bağlantıda senkronizasyon tamamlanmadan tekrar kullanılmaz.
6. Normal public içerik için bounded eventual consistency kabul edilir. Politika `max_stale` süresini tanımlar; L1, L2, edge ve browser yaşları bu toplam bütçeyi aşamaz. TTL katman geçişinde yeniden başlayıp eski verinin ömrünü sınırsız uzatamaz. Sayısal bütçe trafik ölçümü ve içerik ihtiyacıyla pilot öncesi atanır.
7. Admin kaydetme sonucu DB'den yeni revision'ı gösterir. Public yayının tamamlanması ayrı durumdur: `committed`, `propagating`, `propagated` veya `failed`. CDN purge onayı kullanıcının tarayıcısındaki kopyanın silindiği anlamına gelmez; private/no-store preview üzerinden anlık doğrulama sağlanır.
8. Taslak, özel medya, session ve yetkiyle kişiselleşmiş cevap shared cache'e girmez. Public olmuş içeriğin bütün kullanıcı kopyalarını geri alma garantisi verilmez. Hızlı geri çekme gerektiren route, cache politikası ve purge süresiyle ayrıca sertifikalanır.

## Cache arızası ve aşırı yük

Cache kaybı asıl veriyi kaybettirmez; her arızada hizmetin kesintisiz kalacağı anlamına gelmez. L2 arızasında bounded DB fallback, singleflight, concurrency admission, circuit breaker ve retry bütçesi vardır. DB kapasitesi dolduğunda kontrollü 429/503, sınırsız DB kuyruğuna tercih edilir. Yalnız public, izinli ve `max_stale` sınırındaki veri stale sunulabilir; gizli içerik için bu yol yoktur.

Cache key allowlist query normalization, edition/site, locale, representation (HTML/fragment/JSON), izinli host, publication generation ve serialization sürümünü kapsar. Authorization/Cookie taşıyan kişiselleşmiş cevaplar bypass edilir. Host ve `X-Forwarded-*` yalnız doğrulanmış domain ve güvenilen proxy'den kabul edilir. `Vary` tek başına cache sağlayıcısının doğru davranacağını garanti etmez; edge contract testi gerekir. Etkin olmayan modül key, invalidation worker veya tag envanteri oluşturmaz.

## Release uyumluluk metadata'sı

Binary yalnız tek “schema version” sayısına bakmaz. Core ve etkin modüller için okunabilir/yazılabilir schema epoch aralığı, gerekli migration ID'leri, config/content/job/event formatları ve uyumlu release seti ilan eder. Ledger checksum'ları ve gerekli migration varlığı birlikte doğrulanır; yalnız geniş bir sayı aralığı yeterli değildir.

Migration metadata'sı her dialect için şunları içerir:

- Mantıksal kimlik, modül, bağımlılıklar ve yayımlanmış checksum
- `online/backfill/maintenance` sınıfı ve transactional/nontransactional yürütme biçimi
- Önkoşullar, lock/statement timeout, disk tahmini ve smoke/sonkoşul kontrolleri
- Eski/yeni binary okuma/yazma uyumluluğu ve job/content sürüm etkisi
- Başarısızlık/yeniden başlatma prosedürü, backfill checkpoint'i ve rollback kısıtı

Online sınıfı SQL ifadesinin adına göre seçilmez. Aynı migration PostgreSQL'de online, SQLite'ta maintenance olabilir. Uygulama başlangıcı eksik/uyumsuz schema durumunda readiness vermez; deploy orchestrator migration'ı ayrı çalıştırır. Üretimde “contract test” müşteri verisine yazan test takımı değildir: contract testleri izole DB kopyasında/CI'da, üretimde yalnız güvenli readiness ve smoke kontrolleri çalışır.

## Blue/green ve rollback

ADR-0005 akışı korunur; preflight tüm format uyumluluklarını ve iki slot için kaynak payını kontrol eder. Online migration sonrası eski binary de mevcut ledger ile yeniden başlayabilmelidir. Yeni worker başlamadan eski lease'ler drain/expiry politikasıyla devredilir; eski web'in ürettiği event yeni worker tarafından okunabilir olmalıdır.

Rollback yalnız binary'yi geri çevirir; yeni release'in yazdığı veriyi geri almaz. Yeni yazımlar eski binary tarafından okunabiliyorsa geri dönüş mümkündür. Restore ise yedekten sonraki yazımları kaybettirebilen ayrı kurtarma işlemidir; sessiz otomatik rollback gibi sunulmaz.

Yeni release yalnız kendi embedded asset'lerini taşırsa eski HTML kırılabilir. Deploy, imzalı asset inventory'den hash'li dosyaları ortak release asset dizinine çıkarır veya önceki paketlerin asset'lerini sunan katmanı korur. GC minimum rollback penceresi ve izinli HTML/CDN cache ömrü dolmadan bunları silemez. HTTP servis uygulaması tek binary kalır; retained asset'ler deployment verisidir.

## Backup ve restore

Backup (aynı DB'ye felaket kurtarma), export (taşınabilir mantıksal veri) ve binary rollback ayrı komut/sözleşmelerdir.

- Compact ilk uygulaması yazmaları/worker'ları kontrollü durdurup tutarlı DB snapshot ve immutable medya manifest'i alabilir. Beklenen duraklama/byte bütçesi preflight'ta hesaplanır; süre sınırı aşılırsa güvenli iptal edilir. Bütün büyük medya kopyası boyunca “kısa kilit” vaadi verilmez.
- Standard/Distributed için DB snapshot'ındaki referanslı immutable medya envanteri ile GC retention/pin koordine edilir. Pin oluşturma ve silme kararı aynı koordinasyon sınırını kullanır; bütün kopyalama boyunca yazmaları durdurmak gerekmez. Bu yol doğrulanmadan online büyük yedek satılmaz.
- Local/S3 nesne listelemesi DB snapshot'ının yerine geçmez. DB, medya manifest'i, checksum, key referansları, sürümler ve yedek durumu aynı backup kimliğine bağlanır.
- SQLite kapalı hedefte staging dosya/dizin üzerinden restore edilebilir; platforma özel atomik devreye alma ve WAL/SHM temizliği doğrulanır.
- PostgreSQL ayrı boş DB/instance'a kendi restore yöntemiyle döner. Referanslar/smoke doğrulandıktan sonra uygulama yazmaları durdurulur ve bağlantı hedefi kontrollü değiştirilir. DB restore “medya dizinini rename et” işlemi değildir.
- PITR desteği ancak DB geçmişiyle uyumlu medya retention ve key kurtarma modeliyle sunulur. Yalnız DB PITR olması bütün ürün için aynı RPO'yu garanti etmez.
- Restore edilmiş outbox/job'lar eski e-posta/webhook'ları yeniden gönderebilir. Restore önce izole, worker ve dış egress kapalı açılır; yan etkiler uzlaştırıldıktan sonra açıkça etkinleştirilir.
- Restore edilmiş session'lar varsayılan iptal edilir; imza/preview/API secret sürümleri ve eski aktif deployment'ın kapatılması runbook'ta yer alır. Secret kurtarma ayrı güvenli kanaldadır, backup'ın yanında açık anahtar tutulmaz.

## Alternatifler ve doğrulama

Senkron DB+cache çift yazımı seçilmez: tek atomik transaction değildir. Cache'i tamamen kaldırmak doğruluğu sadeleştirir fakat yük hedefini karşılamayabilir; ihtiyaç olmayan girdilerde Noop/bypass geçerlidir. Kuyruk için harici broker ilk şart değildir. Genel down migration veri dönüşümünün güvenli tersini garanti etmez.

Zorunlu kanıtlar: commit sonrası crash, event tekrar/sıra değişimi, lease expiry, provider timeout belirsizliği, ayrı iki web L1'i, kayıp invalidation, stale refill yarışı, L2 kesintisinde bounded DB yükü, migration sırasında eski binary restart, yeni yazımlardan sonra rollback, önceki asset URL'leri, iki DB'de gerçek restore ve restore sonrası yan etki karantinası. Başarı ölçüleri [hazırlık incelemesinde](../PRE_IMPLEMENTATION_REVIEW.md) izlenir.

## Kaynaklar

- [SQLite isolation](https://www.sqlite.org/isolation.html): Eşzamanlı bağlantılara rağmen tek writer sınırı.
- [PostgreSQL CREATE INDEX](https://www.postgresql.org/docs/current/sql-createindex.html): Concurrent index ve transaction/lock kısıtları; dialect başına migration yürütme gereksinimi.
- [Transactional outbox](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/transactional-outbox.html): Atomik DB+event kaydı ve tekrar teslim olasılığı.

Bu kaynaklar mekanizma kısıtlarını destekler; HydroGo'nun tercih ettiği akış ve kabul ölçütleri ürün tasarımıdır.
