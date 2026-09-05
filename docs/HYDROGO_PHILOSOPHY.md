# Hydro Manifesto

> **Su gibi uyumlu. Kaya gibi güvenilir. Kontrol daima bizde.**

Bir projenin teknik mimarisi onun iskeletiyse, felsefesi onun ruhudur. Kod zamanla refactor edilir, kütüphaneler ve altyapılar değişir; HydroGo'nun temel karar ilkeleri ise ürünün saflığını, gücünü ve müşteriye verdiği sözü korur.

HydroGo farklı ölçeklere ve sektörlere uyum sağlar. Bu uyum biçimsizlik değildir: güvenlik, veri mülkiyeti, açık sözleşmeler ve işletilebilirlik değişmeyen özellikleridir.

## 1. Su Gibi Ol — Akışkanlık ve Uyum

> Sistem kaba direnmez; ihtiyaç kadar biçim alır.

HydroGo kurumsal bir sitenin hafifliğine, haber platformunun okuma trafiğine, mobil uygulamanın API ihtiyacına ve işlem odaklı sistemlerin ciddiyetine ayrı profillerle uyum sağlar.

Uyum, her özelliği her kuruluma taşımak değildir. Etkin olmayan modül route, migration, asset, worker veya altyapı yükü oluşturmaz.

**Tavizsiz kural:** Kullanılmayan hiçbir yetenek müşterinin kaynaklarına veya işletim yüküne eklenemez.

## 2. Mülkiyet ve Bağımsızlık — Sovereignty

> Kontrol edemediğin altyapının yönünü de kontrol edemezsin.

HydroGo başka bir CMS'in forku değildir. Ürünün kaynak yönü, build zinciri, dağıtım biçimi ve veri taşınabilirliği bizim kontrolümüzdedir. Müşteri kendi verisini, yedeğini ve çalışma ortamını kontrol edebilir; platformdan çıkış yolu her zaman bulunur.

Bağımsızlık her şeyi yeniden yazmak değildir. Güvenilir bağımlılıklar bilinçli seçilir, sürümleri ve lisansları sabitlenir, değiştirilebilecekleri sınırların arkasında tutulur.

**Tavizsiz kural:** Veri, build ve deployment kontrolünü tek bir dış sağlayıcının insafına bırakmayız.

## 3. Önce Doğrula, Sonra Otomatikleştir — Pragmatizm

> Çalıştırmadığın şeyi soyutlama; görmediğin şeyi otomatikleştirme.

Önce gerçek bir müşteri akışı elle ve uçtan uca geliştirilir. Servis sınırı, veri modeli, yetki, admin deneyimi ve işletim davranışı sahada doğrulanır. CLI ve şema üreticisi yalnız tekrarı kanıtlanmış kalıpları otomatikleştirir.

Bir generator iş mantığını tahmin etmez. Standart CRUD ve sunum iskeletini üretir; müşteriye özgü kurallar açık extension noktalarında insan tarafından yazılır.

**Tavizsiz kural:** İki farklı gerçek kullanımda doğrulanmayan kalıp Core sözleşmesine dönüşmez.

## 4. Tasarımla Güvenlik — Secure by Design & Default Deny

> Açık bırakılan kapı hata değil, önlenmesi gereken bir tasarım kusurudur.

Güvenlik geliştiricinin her seferinde doğru ayrıntıyı hatırlamasına bırakılmaz. Route sınıflandırması, kayıt bazlı yetki, güvenli tipler, otomatik escaping, sanitizer, CSRF ve güvenli session varsayılan olarak uygulanır.

Templ güvenli çıktı üretmeyi kolaylaştırır; tek başına XSS garantisi değildir. `templ.Raw`, URL, zengin metin ve üçüncü taraf istemci kodu ayrıca güvenlik sözleşmelerine tabidir.

**Tavizsiz kural:** Güvenli yol varsayılan, görünür ve en kolay geliştirme yolu olmalıdır.

## 5. Kaya Gibi Sade, Işık Gibi Hızlı — Minimalist Power

> Mükemmellik az satırda değil, az sürpriz ve az gereksiz yükte aranır.

Her katman, paket ve bağımlılık varlığını gerçek bir sorumlulukla hak eder. Performans iddiaları tanımlı donanım, trafik, veri ve cache koşullarında ölçülür. Küçük ve dağıtık kurulumlar aynı Core sözleşmelerini kullanır; yalnız kapasite ve altyapı profili değişir.

HTTP cache, yerel bellek, dağıtık cache, veritabanı ve CDN birbirinin rakibi değildir. İhtiyaç duyulan katman etkinleşir.

**Tavizsiz kural:** Karmaşıklık ancak ölçülmüş bir ihtiyacı çözüyor ve toplam sistemi sadeleştiriyorsa eklenebilir.

## 6. Çekirdek Kararlı, Müşteri Özgür — Backwards Compatibility

> Gövde büyürken dallar hazırlıksız kırılmaz.

HydroCore ile Client Edition arasındaki sınır açık ve sürümlüdür. Müşteri geliştirmesi Core fork'u olarak yapılmaz. Tema tokenları, modül sözleşmeleri, view slotları ve application service extension noktaları üzerinden ilerler.

Aynı major sürüm içindeki yayımlanmış sözleşmeler korunur. Zorunlu kırılma yeni major sürüm, deprecation süresi, migration aracı ve açık yükseltme yolu gerektirir. Kesintisiz deployment, eski ve yeni sürümün geçici birlikte çalışmasını sağlayan expand–migrate–contract disipliniyle korunur.

**Tavizsiz kural:** Core değişikliği, desteklenen müşteri edition'larında contract testleri geçmeden yayımlanamaz.

## Mühendislik karar filtresi

Yeni bir teknoloji, katman veya özellik önerildiğinde şu sorular sorulur:

1. Müşterinin veri ve deployment kontrolünü artırıyor mu?
2. Compact profilde kullanılmadığında tamamen kapanabiliyor mu?
3. Sertifikalı veritabanlarındaki davranışı ve yeni adaptörlere etkisi tanımlı mı?
4. Cache invalidation ve arıza davranışı belli mi?
5. Eski ve yeni sürümün birlikte çalışmasını etkiliyor mu?
6. Müşteri Core fork etmeden ihtiyacını çözebiliyor mu?
7. Gerçek kullanım veya benchmark ile doğrulandı mı?

Bu sorulardan biri mimari yönü değiştiriyorsa karar ADR olarak kaydedilir.

## Kısa manifesto

Biz su gibi uyum sağlar, kullanılmayan yükü taşımayız.  
Biz verimizi, build'imizi ve deployment yönümüzü kontrol ederiz.  
Biz önce gerçek kullanımda doğrular, sonra otomatikleştiririz.  
Biz güvenliği hatırlanacak görev değil, sistemin varsayılanı yaparız.  
Biz performansı ölçer, karmaşıklığı gerekçelendiririz.  
Biz Core'u kararlı tutar, müşteriyi özgür bırakırız.

---

**HydroGo:** Her ölçeğe uyum sağlayan; güvenlik, veri mülkiyeti ve açık sözleşmeler karşısında biçim değiştirmeyen CMS platformu.
