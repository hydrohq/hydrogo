# Admin Studio yüzey envanteri — 2026-09-10

Bu envanter, [Headless Admin Studio planının Faz 0](../HEADLESS_ADMIN_STUDIO_PLAN.md) için mevcut Pages, News ve Commerce HTTP handler'larının okunmasına dayanır. Amaç, metadata ile üretilebilecek tekrarları özel iş kurallarından ayırmaktır; yeni bir API veya runtime schema sözü değildir.

## Rota sahipliği

| Alan | Admin sahibi | Makine API'si | Mevcut integrated görünüm |
|---|---|---|---|
| Pages | `/admin/pages/{id}` | `/api/pages/*` | `/public/pages/{locale}/{slug}/html` |
| News | `/admin/news/{id}` | `/api/news/*` | `/public/news/{locale}/{slug}/html` |
| Commerce | `/admin/commerce/*` | `/api/commerce/*` | `/shop/*` |

`/public/*` rotaları mevcut integrated adapter uyumluluğu için kalır. Yeni headless istemciler için bunlar genişletilecek rota alanı değildir; hedef sürümlü `/api/v1/*` sözleşmesidir.

## Tekrarlanan Studio yüzeyi

| Yetenek | Pages | News | Commerce | Metadata ile üretim durumu |
|---|---|---|---|---|
| Field label/type/required | Evet | Evet | Evet | `adminstudio.Collection` |
| Text, number, status, textarea widget | Kısmen | Evet | Evet | Ortak form renderer |
| Liste sütunları ve kayıt aksiyonu | İhtiyaç var | İhtiyaç var | Evet | Ortak list renderer |
| CSRF alanı ve HTML escaping | Evet | Evet | Evet | Ortak form renderer |
| Optimistic version alanı | Evet | Evet | Modüle bağlı | Handler/service sorumluluğu |
| Role policy, audit, outbox | Evet | Evet | Evet | Metadata değil; application service |

## Özel extension sınırları

### Pages

- Locale, slug, title ve body standart form alanlarıdır.
- Blok düzenleyici (`hero`, `faq`, `cta`, `pricing_table`, `testimonials`), blok başına payload doğrulaması ve medya asset seçimi özel extension olarak kalır.
- Draft/publish revision ve public projection `ContentService` içinden geçer. Studio yalnız form kabuğunu ortaklaştırabilir; publish işlemini doğrudan DB'ye taşıyamaz.

### News

- Locale, slug, title ve body metadata formuna uygundur. Bu ekran ortak renderer'a geçirilmiştir.
- Schedule, publish ve retract state transition'ları `news.Service` içindeki generation/version sözleşmesini kullanır; ayrı Studio workflow action extension'ı olarak kalır.

### Commerce

- Ürün kataloğu alanları ve ürün listesi ortak metadata ile render edilir.
- Varyant/stok, ödeme sağlayıcısı redirect'i, sipariş/iade ve refund finansal yan etkileri collection CRUD değildir; Commerce service ve provider sınırında kalır.

## Sonraki deney kapısı

Pages editörünün standart alanları ortak forma alınmadan önce blok extension için şu arayüz kanıtlanmalıdır: formdan block payload'a geçiş, asset relation picker, CSRF/version taşıma ve invalid payload hata sunumu. Bu kanıt olmadan blokları genel `relation` widget'ına zorlamak ADR-0016'nın typed block sözleşmesini zayıflatır.
