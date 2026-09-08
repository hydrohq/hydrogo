# ADR-0010: SDK olgunluğu, modül yaşam döngüsü ve Edition manifest'i

- **Durum:** Kabul edildi
- **Tarih:** 2026-09-05
- **Değiştirdiği ADR:** [ADR-0008](0008-module-and-extension-contract.md)
- **Uygulama durumu:** SDK v0/v1 adayı, Edition manifest'i, generator ve release provenance sözleşmeleri uygulanıp contract testleriyle doğrulandı. Resmî v1 yayınlama ve production key/provenance kanıtları R5'te açık; güncel durum [roadmap](../ROADMAP.md) içindedir.

## Bağlam

ADR-0008 sınırları doğru koydu ancak geniş bir Go interface'ini iki modül doğrulanmadan zorunlu API gibi sundu. Planın kök `core/` dizini de dışa kapalı uygulama sınırını açık göstermiyordu. Yeni karar önceki extension, generated/custom ve uyumluluk ilkelerini korur; API olgunluğunu ve paketleme davranışını netleştirir.

## SDK ve bağımlılık sınırı

- HydroCore sürümlü Go module olur. Müşteri projesi ayrı Go module ve kendi repository'sidir; Core kaynağı kopyalanmaz. Geliştirme workspace'i bunları birlikte açabilir.
- Core'un `internal/` alanı uygulama ayrıntısıdır. `extension/` müşteri/modül API'sidir. Client Edition yalnız extension ve ilan edilmiş modül API'lerini import eder.
- `internal` kuralı Go'da kaynak ağacı sınırına bağlıdır. Aynı kök altında duran örnek müşteri için tek başına izolasyon sağlamayabilir; CI izinli import listesini ve ayrı dış tüketici fixture'ını da kontrol eder.
- Go paket ayrımı sandbox değildir. Binary'ye derlenen müşteri/modül kodu güvenilen kod kabul edilir; kötü niyetli kodun dosya/ağ/DB erişimini SDK engelleyemez. Güvenilmeyen runtime plugin yüklenmez.
- Domain ve application service SQL/cache dialect bilmez. Modülün altyapı adapter katmanı, platformun transaction'a bağladığı kontrollü DB executor sözleşmesini kullanabilir. Bu yetenek public handler/view SDK'sına sızmaz. “Hiçbir extension SQL kullanamaz” kuralı DB adapter geliştirmesini imkânsızlaştırmamalıdır.

## Olgunluk aşamaları

1. Sınırlar şimdi kabul edilir; paket adları ve Go imzaları deneysel v0'dır.
2. Pages ve News elle geliştirilir; gereksiz arayüzler daraltılır. Kayıt açık composition root üzerinden yapılır; `init()` yan etkileri ve global registry kullanılmaz.
3. Modül kimliği/bağımlılıkları zorunludur. Route, job, admin, migration gibi katkılar ihtiyaca göre küçük sözleşmelerdir; her modülün on metotluk boş implementation yazması gerekmez.
4. İki modül ve iki Client Edition contract testleri geçince yayımlanacak SDK yüzeyi v1 olarak sabitlenir. Generator bundan sonra kararlı olur.

v0 sürümleri de tam sürümle kilitlenir ve değişiklik notu/migration rehberi taşır; sessiz kırılma kabul edilmez. Yayımlanmış v1 sözleşmeleri aynı major içinde korunur. Güvenlik nedeniyle davranış değişmesi gerekirse etki ve yükseltme yolu açıkça duyurulur.

## Extension biçimleri ve hook davranışı

Tercih sırası: config/tema tokenları → tipli validation/hook → view/component slot → özel service/route → tam görünüm override.

- Validation hook yazmayı reddedebilir veya izin verilen girdi alanlarını normalize edebilir. Son yetki, veri doğrulama ve durum kontrolü bundan sonra servis tarafından tekrar uygulanır; hook çekirdek politikasını gevşetmez.
- Transaction içi hook kısa ve deterministik olur; ağ/e-posta çağrısı yapmaz. Hatası transaction'ı geri alır. Retry olabileceği için dış yan etki yasaktır.
- Commit sonrası event kalıcı outbox ile en az bir kez iletilir. Event şeması sürümlü, tüketici idempotent olur; bir tüketicinin hatası diğerinin teslim durumunu silmez.
- Hook sırası açık priority ve sabit kimlikle deterministiktir; döngü/çift kayıt başlangıç hatasıdır. Timeout/cancellation sözleşmesi uygulanır; bu, güvenilmeyen Go kodunu zorla durdurma garantisi değildir.
- View slotları tipli ve derleme sırasında bağlanır. Dinamik registry ancak iki temada ihtiyaç kanıtlanırsa kullanılır. Client view doğrudan yetki veya yayın filtresini atlayamaz; güvenli view model alır.

## Modül yaşam döngüsü

| İşlem | Davranış |
|---|---|
| Build'e ekleme | Bağımlılık çözümlemesi, DB capability, lisans/sürüm kontrolü; yalnız seçilen modüller import/embed edilir |
| Etkinleştirme | İncelenmiş release içinde migration planı uygulanır; ardından yeni binary hazır olur. HTTP başlangıcı kendiliğinden DDL çalıştırmaz |
| Çalıştırma | Yalnız etkin modüllerin route, asset, job ve kaynakları açılır |
| Devre dışı bırakma | Bağımlı modüller kontrol edilir; yeni işler durur; eski worker'lar drain edilir; sıradaki işler park edilir; yeni build modülü içermez |
| Yeniden etkinleştirme | Saklanan schema/migration ledger denetlenir; desteklenen yükseltme yoluyla yeni release hazırlanır |
| Kaldırma | Otomatik tablo/dosya silme yok. Veri ve ilişkiler için ayrı purge planı, yedek, yetki ve audit gerekir |
| Downgrade | Yalnız mevcut schema, event/job ve content formatlarıyla uyumlu eski release'e dönüş; genel otomatik down migration yok |

Runtime'da admin tıklamasıyla keyfi Go modülü yüklenmez. Modül seçimi release oluşturur. Geçici operasyonel kapatma yeni üretimi durdurabilir; build'deki byte'ları kaldırmaz. Kullanılmayan modül yükü vaadi, modülün import/embed edilmediği Edition build'i üzerinden doğrulanır. Saklanan eski müşteri verisi “aktif modül kaynağı” değildir.

Platform orchestrator pasif modülün ledger/veri kaydını görebilir; pasif modülün migration'ını otomatik çalıştırmaz. Bağımlı aktif modül pasif modülün verisine ihtiyaç duyuyorsa kapatma reddedilir.

## Manifest ve release

Üç ayrı belge vardır; isimler uygulama öncesi taslaktır:

- `edition.yaml`: Edition kimliği, seçilen modüller, sunum özellikleri ve içerik tanımları. Sır içermez.
- `edition.lock`: Çözümlenmiş tam Core/modül/toolchain/generator sürümleri ve içerik hash'leri. Go module graph ile tutarlı olmalıdır. İlk resmî modüller Core ile birlikte sürümlenebilir; bağımsız sürümleme zorunlu değildir.
- `release.json`: Gerçekte build edilen bağımlılıklar, kaynak revizyonları, asset inventory, binary digest, DB/migration uyumluluğu, job/event/content formatları, config schema ve build provenance. Paket imzası bununla binary'yi bağlar; imzalayan anahtara güven kurulumda tesis edilir.

Deployment profili, DB bağlantısı, domain ve secret referansları ortam config'idir. Secret değeri manifest/lock içine girmez. Öncelik: açık CLI bayrağı → environment/secret referansı → config dosyası → belgeli varsayılan. Çelişkili secret kaynakları ve bilinmeyen config anahtarları hata verir. Driver dahil değilse config başka DB'yi sessizce seçemez.

Go'nun çözdüğü gerçek bağımlılık grafiği lock beklentisini karşılamıyorsa build durur. “Core 1.x + modül 2.x” etiketi tek başına uyumluluk kanıtı değildir. Release matrisi SDK, schema, config, content, event/job ve asset uyumluluğunu birlikte sınar.

## Sunum ve üretici

Site Edition varsayılanı `integrated`: admin + public Templ site + public JSON API. `headless`: admin + API, public Templ site kapalı. `hybrid` ayrı çalışma motoru değildir; integrated API'sini dış istemcilerle kullanma biçimidir. Eski belgedeki üç mod arasındaki belirsizlik böyle kaldırılır.

`generated/` tamamen yeniden üretilebilir; `custom/` üretici tarafından yazılmaz. Generator sürümü ve spec hash'i kaydedilir. Yeniden üretim önce diff sunar; mevcut yayımlanmış migration'ı değiştirmez. Güvenlik kararı ve destructive dönüşüm tahmin edilmez.

## Alternatifler, etkiler ve doğrulama

Geniş interface'i hemen v1 yapmak erken soyutlama; runtime Go plugin ise güven/dağıtım maliyetidir. Seçilen build composition müşteriye özel yeniden build gerektirir, fakat runtime ve destek yüzeyini daraltır.

SQLite/PostgreSQL ve gelecek adapter'lar modül contract kitine tabidir. Pasif modül cache namespace/worker açmaz. Compact minimum build kaynak testinden; Distributed bounded pool/worker testinden geçer. Blue/green'de eski/yeni modül seti, job payload'ı ve asset retention birlikte değerlendirilir.

Kanıtlar: dış tüketici internal import edemez; iki Client Edition Core kopyalamadan build olur; modül kapatıldığında route/asset/job/migration kaydı oluşmaz; bağımlılık döngüsü reddedilir; aynı spec ikinci üretimde diff oluşturmaz; custom dosya hash'leri korunur; eski desteklenen client yeni Core contract testini geçer.

## Kaynak

[Go internal package sınırı](https://go.dev/doc/go1.4#internalpackages). Go davranışı kaynakla doğrulandı; bu belgede seçilen paketleme ve SDK yaşam döngüsü HydroGo tasarım kararıdır.
