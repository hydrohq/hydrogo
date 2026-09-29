# hydrogo — Kapsamlı Denetim, Tur 2-8

Bu dosya bir denetim planıdır. Her turda **bir alan** işlenir, sonra durulur.
Tur 1 (`modules/members`) tamamlandı ve `main`'e birleştirildi. Buradan devam.

## Nasıl çalışılacak

1. Her tur için ayrı bir dal aç: `git checkout -b audit-<tur-no> main`
2. O dalda çalış, commit at.
3. `go build ./...` + `go test ./...` yeşilse ve tur bittiyse: dalı `main`'e
   fast-forward et, sonra dalı sil.
4. Bir sonraki tura geç.

Kural: **turları tek oturumda arka arkaya yapma.** Her tur bitince dur ve
rapor ver. Bu, bu depoda daha önce yaşanan iki hatanın (uydurma doğrulama
çıktısı, işlevsiz test) sebebidir.

---

# ORTAK KURALLAR — her turda geçerli

Bunlar istek değil, zorunluluktur.

1. **Test önce.** Önce başarısız testi yaz, ÇALIŞTIR, ÇIKTISINI GÖSTER.
   Sonra düzelt. Asla önce düzelt, sonra test yazma.

2. **Testin gerçek olduğunu kanıtla.** Düzeltme geçtikten sonra üretim
   kodunu geçici olarak geri al, testi tekrar çalıştır, ve **KIRILDIĞINI
   GÖSTER**. Sonra geri koy. Düzeltilmemiş kodda geçen test kanıt değildir.
   Hiç kırıldığını görmediğin test kanıt değildir.

3. **Gerçek katmanı test et.** Hata SQL'de, dosya sisteminde veya HTTP
   middleware'deyse, test o gerçek uygulamadan geçmelidir. Test, test edilen
   davranışın kendisini içermeyen bir bellek içi test double'ına karşı
   çalışıyorsa hiçbir şey kanıtlamaz. Double kullanırsan, production ile
   aynı davrandığını kanıtla.

4. Bir `.go` kaynak dosyasını okuyup `strings.Contains` yapan test **test
   değildir**, grep'tir. Öyle yazma.

5. Bir uygulamayı değiştirdiğinde, düzeltmenin ötesinde değişen davranışı
   raporla: artık başarısız olan başarı durumları, artık kırpan sınırlar,
   artık farklı hata dönen yollar. Bunları sonra öğrenme.

6. Başarısız testle kanıtlayamadığın bir hatayı **teslim etme**. Kanıtsız
   olarak raporla ve seni neyin durdurduğunu yaz. Asla çıktı uydurma, asla
   gözlemmediğin bir "test failed" metni yazma.

7. Bir testin geçmesinin veya kırılmasının sebebi test double'ı ise, double'ı
   düzelt, yeniden doğrula ve raporda belirt.

8. Düzeltme ve testi **aynı commit'te** gönder. Sadece düzeltmeyi asla
   commit'leme.

9. Commit'ten önce çalıştır ve çıktıyı göster:
   `go build ./...`
   `go test ./... -count=1`
   `go vet ./<paket>/...`

10. Commit mesajı düz, insan diliyle. `Co-Authored-By` trailer'ı **yok**.

11. Raporunda şunu ayrıca belirt: commit'lerde `Co-Authored-By` var mı yok mu
    (`git log -1 --format='%(trailers)'`).

---

# TUR 2 — `internal/auth` (26 dosya / 5881 satır)

```fish
git checkout -b audit-2 main
```

```
Audit internal/auth ONLY. Do not read anything outside internal/auth except
for interface definitions you need to satisfy.

[ORTAK KURALLAR]

Bu pakette daha önce düzeltilmiş ve commitlenmiş olanlar - tekrar denetleme:
üç form handler'ının request body boyut sınırları. Geri kalan her şey
hiç incelenmedi.

Bu paket recovery.go, mfa.go, invite.go, password.go, mobile.go ve oturum
kodunu içeriyor. Bunlar kimlik bilgisi yolları ve daha önce hiç denetlenmedi.
Buradan başla.

1. Parola sıfırlama (recovery.go): token tek kullanımlık mı - kullanıldığı
   anda mı tüketiliyor, yoksa kullanımdan sonra mı? Süresi var mı ve süre
   KONTROLÜ okuma yolunda mı, sadece oluşturma anında mı? Saklanma biçimi
   hash mi plaintext mi? Karşılaştırma sabit zamanlı mı? Token hangi
   hesaba bağlı - A için üretilen token B'nin parolasını sıfırlayabilir mi?

2. MFA (mfa.go): kod kaba kuvvetle kırılabilir mi? Deneme sayacı kullanıcı
   başına mı global mi, ve istekten gelen değere güvenilerek mi uygulanıyor
   sunucu tarafında mı? Kullanılmış kod tekrar oynatılabilir mi? Kurtarma
   yedek kodu tek kullanımlık mı?

3. Oturum (service.go ve komşuları): oturum id'si nereden geliyor - yeterli
   baytla crypto/rand mı, yoksa zaman veya sayaçtan türetiliyor mu? Neyi
   hash'liyor, neyi ham saklıyor? Süre her okumada kontrol ediliyor mu, ve
   süresi dolmuş oturum reddediliyor mu, yoksa sadece gizleniyor mu?

4. Giriş ve hız sınırlama: sınırlayıcı saldırganın kontrol ettiği bir şeyle
   mi anahtarlanıyor? Anahtar kullanıcı tarafından verilen e-posta ise,
   saldırgan onlar için başarısız giriş deneyerek keyfi hesapları
   kilitleyebilir mi, ve e-postayı değiştirerek sınırı atlatabilir mi?

5. Davet (invite.go): davet tekrar oynatılabilir mi, süresi dolduktan sonra
   kullanılabilir mi, veya zaten var olan bir hesabı ele geçirmek için
   kullanılabilir mi?

6. Bir token, hash veya sır için sabit zamanlı yardımcı yerine `==` ile
   karşılaştırma yapılan her yer. Tüm paketi özellikle bunun için ara.
   Her bulgu bir bulgudur - dosya, satır ve ifadenin kendisini raporla.

7. Parola hash'i: her çağrı yerinde kullanılan algoritmayı ve maliyet
   parametresini raporla. Daha zayıf hash'leyen ya da parolayı `==` ile
   karşılaştıran yolu işaretle.

Öncelik: 1, 2 ve 6. Her sır karşılaştırması için bulduğun yol, mekanizması
ve ifadesiyle birlikte bir tablo sun.
```

Bitince: `git checkout main && git merge --ff-only audit-2 && git branch -d audit-2`

---

# TUR 3 — `internal/media` (27 dosya / 4897 satır)

```fish
git checkout -b audit-3 main
```

```
Audit internal/media ONLY.

[ORTAK KURALLAR]

ready ve adminVisibility handler'larının request body boyut sınırları zaten
düzeltildi ve commitlendi. Multipart yüklemelerin yapılandırılmış bir
multipart limiti ile sınırlı olduğu bildirildi - bu iddiayı KENDİN doğrula,
kabul etme, ve gerçek yapılandırılmış değerin ne olduğunu raporla.

1. Path traversal: saklanan dosya adının veya yolun dosya sistemi
   çağrısına - okuma, yazma, silme, yeniden adlandırma, stat - ulaştığı her
   yerde, çözümlenen yolun depolama kökünün içinde kaldığını kanıtla. Gerçek
   uygulamayı testteki bir yardımcıya değil incele. Sonra zor varyantı
   kontrol et: kökün dışına işaret eden bir SYMLINK olan saklanan ad. Symlink
   çözmeyen yol doğrulaması aynı primitifle atlatılabilir.

2. Dosya adı üretimi: saklanan ad sunucuda mı üretiliyor, yoksa kullanıcı
   girdisinden mi türetiliyor? Türetiliyorsa hangi temizleme uygulanıyor ve
   hedef dosya sistemi için doğru mu - baştaki noktalar, ayrılmış adlar,
   unicode normalizasyonu ve tam olarak ".." olan bir ad dahil.

3. MIME doğrulama: beyan edilen Content-Type'a mı güveniliyor, yoksa içerik
   mi koklanıyor? Beyan edilen türe güveniliyorsa, dosyaların SERVED
   EDİLDİĞİ biçimi kontrol et - inline mı attachment mı, ve
   Content-Disposition ile X-Content-Type-Options başlıkları. Güvenilen tür +
   inline servis + saklanmış .html veya .svg = depolanan XSS.

4. Boyut ve adet sınırlama: sınır gövde belleğe veya geçici dosyaya
   okunmadan ÖNCE mi uygulanıyor, yoksa sonradan mı kontrol ediliyor? Oku
   sonra reddet, bellek veya disk DoS'udur. Tek multipart isteğindeki dosya
   SAYISI için bir sınır var mı, ve toplam açılmış boyut için var mı?

5. Silme: kayıt id'si ile depolama anahtarı ayrı ayrı çözülüyorsa, bir
   çağrı başka bir tenant'ın dosyasını silebilir mi? İkisini de izle.

6. Bu paketteki her tar veya zip açma işlemi için - ../ girdileri, mutlak
   yollar ve hedefin dışına çıkan symlink girdileri.
```

Bitince: `git checkout main && git merge --ff-only audit-3 && git branch -d audit-3`

---

# TUR 4 — `extension/v1` (23 dosya / 2459 satır)

```fish
git checkout -b audit-4 main
```

```
Audit extension/v1 ONLY.

[ORTAK KURALLAR]

Bu, dışarıdan erişilebilir API yüzeyi. Her isteği tamamen saldırgan
kontrollü, her alanı düşman kabul et. Daha önce hiç denetlenmedi.

1. Kimlik doğrulama: HER rotayı say. Geçerli bir kimlik bilgisi olmadan
   erişilebilen handler'ları listele. Router'ın global bir middleware
   uyguladığını varsayma - rota kaydını oku ve rota başına kanıtla.

2. Yetkilendirme: her rota için "kimliği doğrulandı"nın ötesinde ne kontrol
   ediliyor? Rota -> gereken izin -> gerçekte ne doğruladığı şeklinde bir
   tablo kur. Kontrolü komşularından zayıf olan her rotayı işaretle.

3. Mass assignment: istek gövdesini doğrudan bir domain veya kalıcılık
   struct'ına çözen her handler'ı bul. Her biri için istemcinin set etmemesi
   gereken hangi alanları set edebildiğini listele - id, tenant_id, role,
   status, version, created_at, owner_id, doğrulanmış bayraklar. JSON
   gövdesinden role veya tenant_id set edebilen bir istemci yetki
   yükseltmedir.

4. Girdi doğrulama: her string alan, her query parametresi, her yol
   segmenti. Sınırsız uzunlukları, doğrulanmamış enum'ları ve string olarak
   alınıp ayrıştırılmadan kullanılan sayısal alanları raporla.

5. Yanıt sızıntısı: doğrudan JSON'a serileştirilen ve iç alanlar, parola
   hash'leri, token hash'leri, sır değerleri veya arayanın görmemesi gereken
   tenant kimlikleri içeren struct'lar. SELECT * tarzı bir projeksiyonun
   yanıta ulaşıp ulaşmadığını kontrol et.

6. İdempotensi ve tekrar: aynı gövdeyle tekrar edilen bir isteğin ikinci bir
   etki yaratmasına yol açan bir uç nokta - çift ücret, mükerrer kayıt,
   mükerrer bildirim.

Mass assignment için test mekaniktir ve her vaka için yapmaya değer: role veya
tenant_id'yi ayrıcalıklı bir değere ayarla, sonra saklanan kaydın değerinin
DEĞİŞMEDİĞİNİ doğrula. Bu test düzeltmeden önce kırılmalı.
```

Bitince: `git checkout main && git merge --ff-only audit-4 && git branch -d audit-4`

---

# TUR 5 — `internal/backup` (10 dosya / 2036 satır)

```fish
git checkout -b audit-5 main
```

```
Audit internal/backup ONLY.

[ORTAK KURALLAR]

Bu kod kullanıcı verisi yazar ve üzerine yazar. Repodaki en yıkıcı yüzey.
Buradaki bir hata veri kaybettirir, sadece sızdırmaz.

1. Arşiv açma: HER tar veya zip açma işleminde girdi adının temizlendikten
   SONRA doğrulandığını kontrol et - "../" dizilerini, mutlak yolları ve
   hedefin dışına çıkan symlink ya da hardlink girdilerini reddet. Yazmadan
   önce çözümlenen yolun kökün içinde olduğunun teyit edildiğini, yoksa
   sadece birleştirildiğini raporla. Kontrol etmeden birleştirme hatadır.

2. Geri yükleme: geri yükleme ne kabul ediyor - yol mu, isim mi, id mi,
   manifest mi? Her birini kaynağına kadar izle. Geri yükleme kendi hedef
   dizininin dışına yazabilir mi? Keyfi bir dosyanın üzerine yazabilir mi?
   Olumlu ve olumsuz vaka ile test et.

3. Yıkıcı işlemler: drop, truncate, sil veya üzerine yazan HER fonksiyonu
   listele. Her biri için hedefin önce doğrulandığını, bir onay adımı
   olup olmadığını ve o onayın atlatılabilir olup olmadığını raporla -
   istekten gelen bir bayrakla, doğrudan servis çağrısıyla, ya da onayı
   atlayan ikinci bir yolla.

4. Kimlik bilgileri: bir yedek arşivi veritabanı kimlik bilgileri, API
   anahtarları veya oturum malzemesi içeriyor mu? İçeriyorsa şifreli mi?
   Geri yükleme tarafında arşiv nasıl çözülüyor - anahtar yapılandırmadan mı
   geliyor, ve KDF tuzlanmış ve tekrarlı mı?

5. Eşzamanlılık: aynı hedefe aynı anda iki geri yükleme, ya da bir geri
   yükleme ve bir yedekleme. Kilit var mı, ve hata yolunda bırakılıyor mu?

6. Yedekleme yönünde yol güvenliği: yedekleme symlink'leri veri dizininin
   dışına izliyor mu, ve özenle hazırlanmış bir dosya adı arşivin dışındaki
   dosyaları içermesine yol açabilir mi?

Her yıkıcı fonksiyon için korumasıyla, ya da "KORUMASIZ" ibaresiyle bir
tablo sun.
```

Bitince: `git checkout main && git merge --ff-only audit-5 && git branch -d audit-5`

---

# TUR 6 — `internal/adminstudio` (26 dosya / 2795 satır)

```fish
git checkout -b audit-6 main
```

```
Audit internal/adminstudio, EXCEPT the sqlstore subpackage.

[ORTAK KURALLAR]

sqlstore zaten denetlendi ve düzeltildi. Buradaki hatalarından biri
safeTarget'ın ters eğik çizgiyle başlayan yolu kabul etmesiydi. Aynı sınıf
hatanın bu paketin başka yerlerinde de var olduğunu varsay.

1. Studio'daki HER handler için tablo kur: rota -> gereken izin -> handler'ın
   gerçekte ne kontrol ettiği. Hiç kontrol etmeyen, ya da komşularından daha
   zayıf kontrol eden her handler'ı işaretle. İzin kontrolsüz erişilebilen
   bir studio handler'ı birincil hedeftir.

2. Bir URL'ye veya HTML özniteliğine ulaşan girdi. Her birini HTTP
   handler'ından çıkış noktasına kadar izle. href, src, action, formaction
   veya yönlendirme hedefine doğrulamasız ulaşan her şeyi raporla. Her biri
   için yalnızca belirgin olanı değil, ters eğik çizgi varyantını ve //host
   varyantını da kontrol et.

3. Dosya ve dizin yönetimi: her silme, yeniden adlandırma ve yazma. Yol
   istekten mi alınıyor, yoksa saklanan bir id'den kök içinde yeniden mi
   çözülüyor? Çözümlenen yolun symlink çözüldükten sonra kökün içinde
   olduğu teyit ediliyor mu?

4. Templ işleme (admin_ux.templ ve ham HTML yardımcıları): Templ'in varsayılan
   kaçışını atlayan her yeri bul ve kullanıcı kontrollü verinin oraya
   ulaşıp ulaşmadığını raporla.

5. CSRF: her durum değiştiren studio handler'ı CSRF token'ını doğruluyor mu,
   ve kontrol rota'ya mı uygulanıyor, yoksa yalnızca bazı handler'ların
   içinde mi?
```

Bitince: `git checkout main && git merge --ff-only audit-6 && git branch -d audit-6`

---

# TUR 7 — `internal/database` (11 dosya / 1614 satır)

```fish
git checkout -b audit-7 main
```

```
Audit internal/database ONLY.

[ORTAK KURALLAR]

Bu altyapı. Buradaki bir bulgunun etki alanı, yaprak bir pakettekinden geniştir.
Neyin erişilebilir olduğu konusunda kesin ol.

1. Sorgu kurulumu: SQL'in parametre bağlama yerine birleştirilerek kurulduğu
   her yeri bul. Her biri için birleştirilen değeri kaynağına kadar izle ve
   bir isteğin ulaşıp ulaşamadığını raporla. Tablo ve sütun adları
   parametrelenemeyeceği için, bunların bir izin listesiyle doğrulandığını
   kontrol et ve doğrulanmayanları raporla.

2. Migration güvenliği: migration dosyalarını bul. Her biri için boş olmayan
   bir tabloda uzun bir kilit olmadan çalışabilir mi, ve geri alma yolu var
   mı? Geri alma yolu olmayan yıkıcı işlemleri işaretle: DROP COLUMN,
   DROP TABLE, mevcut tabloya varsayılan değersiz eklenen NOT NULL, ve daraltan
   ya da hassasiyet kaybettiren her tür değişikliği.

3. Transaction'lar: transaction'a sarılmamış çok deyimli yazmaları ve tüm
   adımlar başarılı olmadan commit eden transaction'ları bul. Ayrıca
   transaction içinde rollback yapmadan dönen hata yollarını bul.

4. N+1: her iterasyonda sorgu gönderen döngüyü bul. Çağrı yerini ve çağrı
   derinliğini, döngü sınırının kullanıcı girdisinden gelip gelmediğini ve
   sınırsız olup olmadığını raporla.

5. Kaynak yaşam döngüsü: bağlantı, rows veya hazırlanmış ifadenin hata
   yolunda kapatılmadığı her yeri bul. Bir kaynak edinmiş her fonksiyondaki
   her erken dönüşü kontrol et.

6. Bağlantı havuzu ayarları: MaxOpenConns, MaxIdleConns ve ConnMaxLifetime
   değerlerini ve hiç ayarlanıp ayarlanmadıklarını raporla. Çok kiracılı bir
   sunucuda ayarlanmamış havuz kendi başına bir bulgudur.
```

Bitince: `git checkout main && git merge --ff-only audit-7 && git branch -d audit-7`

---

# TUR 8 — `internal/mailer` + `internal/observability` (24 dosya / 3345 satır)

```fish
git checkout -b audit-8 main
```

```
Audit internal/mailer AND internal/observability.

[ORTAK KURALLAR]

1. Başlık enjeksiyonu: bir posta başlığına konulan her kullanıcı kontrollü
   değeri bul - To, From, Reply-To, Subject, CC, BCC ve her özel başlık. CR ve
   LF'in reddedilip reddedilmediğini kontrol et. Başlık değerindeki bir
   satır sonu başlık eklemeye izin verir ve uygun bir kontrol ile alıcıyı
   değiştirmeye ya da BCC eklemeye yol açar. Her çıkış noktasını ve
   süzülüp süzülmediğini raporla.

2. Şablon enjeksiyonu: her posta gövdesinin nasıl kurulduğunu bul. Bir
   kullanıcı değerinin, işlemeden ÖNCE şablon kaynak dizesine birleştirildiği
   her yeri raporla - şablon VERİSİ olarak geçirilmesi değil. Kaynağa
   birleştirme bulgudur; veri olarak geçirmek değildir. Her vaka için
   saldırganın işlenmiş çıktıda neyi kontrol ettiğini açıkla.

3. Loglar: tüm pakette bir parolanın, token'ın, oturum id'sinin, API
   anahtarının, çerezin ya da tam kart numarasının loglanmasını ara - bir
   hata mesajının içinde olsa ve bir hata mesajının yukarıda loglanması
   söz konusu olsa bile. Redaksiyon yardımcılarının var olup olmadığını ve
   her çağrı yerinin onları kullandığını raporla.

4. Telemetri: bir e-posta adresi, kullanıcı id'si, sorgu dizesinde token
   içeren tam bir URL ya da istek gövdesi taşıyan her metrik, span veya log
   alanı. Metriklerde yüksek kardinalite ve PII'nin ikisi de bulgudur.

5. Panic kurtarma: bu pakette bir middleware panic'i yakalıyor mu? Yakalıyorsa
   kurtarma istek gövdesini, sorgu dizesini veya başlıkları logluyor mu? Bu
   isteği loga yazan kurtarılmış bir panic, her çöküşte bir veri sızıntısıdır.

6. İstemciye dönen hata mesajları: iç bir yol, bir SQL parçası ya da sürücü
   hata dizesi içerenler.
```

Bitince: `git checkout main && git merge --ff-only audit-8 && git branch -d audit-8`

---

# Her tur bittiğinde

```fish
go build ./... && go test ./... -count=1
git log -1 --format='%an  %s'     # yazar sen olmalısın
git log -1 --format='%(trailers)' # boş olmalı
git status --short                # temiz olmalı
```

# Bir turda takılırsan

Dur ve engeli açıkla. Varsayımı doğrulamadan test yazıp geçme - bu depoda
iki kez işe yaramaz test üretildi ve ikisi de "regresyon testi eklendi"
diye kayda geçti. Kanıtsız test, testten iyidir.
