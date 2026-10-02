# Frontend nasıl geliştirilir (insan rehberi)

Bu projede aslında **iki ayrı "frontend" sistemi** var, birbirine
karıştırma:

| | Site görünümü (tema) | Admin paneli |
|---|---|---|
| Kimin görür | Site ziyaretçileri | Senin ekibin/editörler |
| Nerede | `themes/<tema-adı>/` | `modules/designsystem/`, `internal/ui/` |
| Yazım dili | Düz HTML | Templ (özel bir şablon dili) |
| Stil | Tailwind, tema başına derleniyor | Tailwind, panel paketi başına derleniyor |
| Değişiklik ne zaman görünür | Küçük bir derleme komutu gerekir | `hydro run` çalışırken otomatik |

## Site görünümünü değiştirmek (tema)

`themes/corporate/` gibi bir klasörün içinde:
```
layouts/base.html        ← sayfanın genel iskeleti (header/footer vb.)
blocks/hero.html          ← ana sayfa gibi yerlerde kullanılan "blok"lar
blocks/cta.html
blocks/faq.html
...
static/theme.source.css  ← stil DEĞİŞİKLİKLERİNİ buraya yap
static/theme.css         ← bu otomatik üretiliyor, elle dokunma
```

HTML dosyalarını (blocks/, layouts/) direkt düzenleyebilirsin, ekstra bir
adım gerekmiyor. **Renk/stil değişikliği** yapacaksan `theme.source.css`'i
düzenle, sonra bir ajana "tema CSS'ini derle" de — küçük bir script
çalıştırıp senin için derliyor.

**Önemli bir detay:** CSS'i her derledikten sonra, dosyanın "önbellek
kırma" numarasının (`theme.css?v=abc12345` gibi bir şey) üç ayrı yerde
güncellenmesi gerekiyor — bunu unutursan tarayıcılar eski CSS'i
önbellekten göstermeye devam edebilir. Bir ajana yaptırıyorsan bu adımı
atlamaması gerektiğini hatırlat, kendi script'i zaten sana söylüyor.

## Admin panelini değiştirmek

Admin panelinin görsel bileşenleri "Templ" denen özel bir dilde yazılıyor
(`.templ` uzantılı dosyalar). **`hydro run` çalışırken bu dosyaları
kaydettiğinde otomatik olarak derlenip panelde anında görünüyor** —
ekstra bir komut çalıştırman gerekmiyor, sadece dosyayı kaydet, tarayıcıyı
yenile.

`_templ.go` diye biten dosyalara **asla elle dokunma** — bunlar otomatik
üretiliyor, senin elle yaptığın değişiklik bir sonraki derlemede silinir.

## Değişikliği doğrulamak

En güvenilir yöntem: `hydro run`'ı çalıştır, değiştirdiğin sayfayı
tarayıcıda gerçekten aç ve gör. Otomatik kontroller (lint, derleme) sadece
"bozuk değil" der, "güzel görünüyor" demez — görsel değişiklikleri mutlaka
gözle kontrol et.

## Next.js/React seviyesinde kaliteli arayüzler üretmek

HydroGo ile aynı görsel kaliteyi hedefleyebilirsin. Tarayıcıya ulaşan HTML ve
CSS; tipografi, yerleşim, responsive davranış ve animasyon için aynı imkânları
sunar. Next.js/React ekosistemindeki hazır bileşenler geliştirmeyi hızlandırır;
HydroGo'da bu kolaylığı mevcut Templ bileşenleri, Tailwind ve ortak tasarım
kuralları üzerinden oluşturuyoruz.

### 1. Ekranın görsel yönünü belirle

Kod yazmaya başlamadan önce hedef kullanıcıyı, sayfanın ana işini ve birkaç
somut görsel referansı belirle. Yazı boyutları, içerik genişliği, boşluklar,
renkler, köşe yarıçapları ve gölgeler için tutarlı bir ölçek kullan. Bunları
her ekranda yeniden seçmek yerine tema tokenlarında tanımla.

Projenin [görsel yönünü](../../../DESIGN.md) ve etkilenen temanın mevcut
kurallarını oku. Belirli bir yüzey için kullanıcının açık tercihini uygula;
örneğin Docs'un mavi paleti bütün müşteri temalarının mavi olması gerektiği
anlamına gelmez. Ortak renk ve durum anlamlarını
[semantic token sözleşmesi](../../architecture/0053-semantic-ui-token-contract.md)
üzerinden koru.

### 2. Mevcut bileşenlerden bir ekran kur

Önce `internal/ui/components/` ve `modules/designsystem/` içindeki parçaları
incele. Typed Button, Input, Card ve Form gibi temel bileşenleri kullan;
erişilebilirlik, escaping ve form davranışını her sayfada tekrar kurma.
Tailwind bu bileşenlerin ve temanın görsel dilini yazmayı kolaylaştırır.
Değiştirdiğin yüzeye ait derleme adımlarını yukarıdaki bölümlerden takip et.

Bir menü, dialog, sekme, tablo veya boş durum tasarlarken mevcut karşılığını
ara. Eksikse doğru bileşen sınırında geliştir. Aynı görünümün sayfalar arasında
kopyalanması yerine ortak bileşeni iyileştir; farklı sayfaların içerik ve
yerleşim ihtiyaçlarını da koru. Müşteriye özel tasarım ve davranış Client
Edition'ın `custom/` alanında yaşar. `generated/` dosyalarını ve `_templ.go`
çıktılarını elle değiştirme.

### 3. Görünümle birlikte bütün durumları tasarla

Bir formun yalnızca başarılı hâli yeterli değildir. Yüklenme, boş veri,
doğrulama hatası, başarı ve devre dışı durumlarını da göster. Klavye odağı,
anlaşılır etiketler ve hata mesajları tasarımın parçasıdır. İngilizce ve
Türkçe metinleri, özellikle uzun etiketleri, gerçek içerikle kontrol et.

Menü, dialog ve küçük animasyonlarda CSS, tarayıcının yerel kontrolleri ve
gerektiğinde küçük JavaScript bileşenleri kullanabilirsin. Sunucudan içerik
güncellemede mevcut HTMX allowlist ve mutation sözleşmesine uy. Hareketi
amacına göre ekle ve `prefers-reduced-motion` tercihine saygı göster.

### 4. AI'ya uygulanabilir bir tasarım tarifi ver

"Modern ve premium bir sayfa yap" yerine referansları, mevcut bileşenleri,
gerçek metinleri ve kabul ölçütlerini birlikte ver. Örneğin:

> Bu ekranı mevcut tema tokenları ve typed Templ bileşenleriyle geliştir.
> Hedef kullanıcı editör; ana işlem içerik yayınlamak. Eklediğim görsel
> referansın tipografi ve boşluk düzenini takip et. Mevcut bileşenleri kullan,
> eksikleri önce belirle. Gerçek İngilizce ve Türkçe içerikle yüklenme, boş,
> hata ve başarı durumlarını tamamla. 360, 768 ve 1440 piksel genişliklerde
> kontrol et; ekran görüntülerindeki taşma, hizalama ve okunabilirlik
> sorunlarını düzelt.

İlk tasarımı tarayıcıda incele, somut sorunları belirt ve tekrar düzelt.
Görsel kalite, bu kısa inceleme döngülerinde tutarlı hâle gelir.

### 5. Teslimden önce görsel kontrol yap

- Mobil, tablet ve masaüstünde içerik sırasını, satır uzunluklarını ve yatay
  taşmayı kontrol et.
- Klavyeyle bütün etkileşimlere ulaş; odağın görünür olduğunu doğrula.
- Form, tablo ve içerik bloklarını gerçek veri, boş veri ve uzun metinle dene.
- Ekranın desteklediği dilleri ve açık/koyu temaları kontrol et.
- Animasyonların azaltılmış hareket tercihinde uygun davranmasını doğrula.
- CSS derlemesini, etkilenen asset kontrollerini ve zorunlu repo kontrollerini
  tamamla. Test geçmesi görsel incelemenin yerine geçmez.

### Ortak kaliteyi geliştirmek için sonraki adımlar

Mevcut bileşenleri bir **bileşen vitrini** içinde; varyantları, durumları,
responsive örnekleri ve kullanım kodlarıyla göstermek geliştirmeyi
kolaylaştırır. Bunun üzerine giriş, yönetim listesi, düzenleme formu, içerik
detayı ve landing page gibi az sayıda tamamlanmış örnek ekran kurabiliriz.
Bu vitrin ve ekran seti burada önerilen geliştirme adımlarıdır; bu rehber
onların tamamlanmış veya bütün Edition'larda kullanılabilir olduğunu ilan
etmez.

Çok yoğun istemci durumu gerektiren bir görsel editör veya canvas uygulaması
için Edition'a ait ayrı bir React/Next.js frontend değerlendirilebilir.
Core ve Compact profile genel bir frontend runtime eklemek yerine, API ve
asset sınırlarını Edition kapsamında tasarla. Başsız frontend yönünü ele
alan [ADR-0020](../../architecture/0020-headless-first-admin-studio-and-frontend-templates.md)
henüz **Proposed** durumundadır; hazır bir Next.js starter veya doğrulanmış
entegrasyon garantisi olarak yorumlama.

İlgili sözleşmeler:
[typed server UI](../../architecture/0209-typed-server-ui-contract.md),
[tasarım sistemi sağlayıcı sınırı](../../architecture/0023-provider-agnostic-design-systems.md),
[bileşenler ve hafif etkileşimler](../../architecture/0266-design-system-blocks-and-interactions.md),
[Tailwind yazım kuralları](../../architecture/0340-tailwind-styling-migration.md).
