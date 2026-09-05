# ADR-0008: Core, modül ve müşteri extension sınırı

- **Durum:** Değiştirildi
- **Tarih:** 2026-09-05
- **Yerine geçen ADR:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md)

> Tarihsel karar: Aşağıdaki geniş interface kararlı SDK değildir. Güncel paketleme, v0/v1 olgunluğu, modül yaşam döngüsü ve sunum tanımları ADR-0010'dadır. Bu metin karar geçmişi için korunur.

## Bağlam

HydroGo'nun geliştirme hızını artırması ve müşteri projelerinin Core güncellemelerini alabilmesi için hangi kodun kararlı sözleşme, hangi kodun iç uygulama ayrıntısı olduğu kesin olmalıdır.

## Karar

Kod üç görünürlük seviyesine ayrılır:

```text
internal/     HydroGo iç uygulaması; müşteri için uyumluluk garantisi yok
extension/    Sürümlü müşteri ve modül SDK'sı
client/       Müşteriye özel edition kodu
```

Client Edition yalnız `extension` sözleşmelerini ve açık modül API'lerini kullanabilir. CI yasak internal import'larını reddeder.

## Modül sözleşmesi

Bir modül en az şunları ilan eder:

```go
type Module interface {
    ID() string
    Version() Version
    Requires() Requirements
    Capabilities() []Capability
    Routes() []Route
    Permissions() []Permission
    Migrations(database string) []Migration
    Admin() AdminContribution
    Jobs() []JobDefinition
    CachePolicies() []CachePolicy
}
```

- Modül bağımlılık grafiği döngüsüzdür.
- Etkin olmayan modül hiçbir route, migration, asset veya worker yüklemez.
- Modülü devre dışı bırakmak veriyi otomatik silmez.
- Modülün kaldırılması açık veri saklama/silme planı gerektirir.
- Gerekli DB/cache/storage capability başlangıçta doğrulanır.

## Extension biçimleri

Tercih sırası:

1. Yapılandırma ve tema tokenları
2. Sürümlü hook/event sözleşmeleri
3. Component/view slot composition
4. Müşteriye özel application service veya route
5. Tam görünüm override'ı

HydroCore fork'u desteklenen extension biçimi değildir.

## Kod üretimi

- `generated/` tamamen yeniden üretilebilir ve elle değiştirilmez.
- `custom/` insan kodudur ve generator tarafından yazılmaz.
- Generator standart CRUD, migration taslağı, admin formu, API ve test iskeleti üretir.
- İş kuralı, yetki kararı ve destructive migration otomatik tahmin edilmez.

## Sunum modları

Edition şu çalışma modlarından birini açıkça seçer:

- `integrated`: Admin + Templ public site
- `headless`: Admin + API
- `hybrid`: Templ ve API birlikte

İlk ticari Site Edition için `integrated` varsayılandır.

## Uyumluluk

- Aynı major sürümde yayımlanmış extension sözleşmeleri korunur.
- Deneysel API ayrı namespace ve açık etiket taşır.
- Kırılma yeni major, deprecation dönemi ve migration rehberi gerektirir.
- Desteklenen Client Edition contract testleri geçmeden Core release yayımlanmaz.
