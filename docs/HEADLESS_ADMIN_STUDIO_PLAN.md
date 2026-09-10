# Headless-first Admin Studio ve frontend starter planı

**Durum:** Uygulama deneyleri sürüyor; mimari yön için [ADR-0020](architecture/0020-headless-first-admin-studio-and-frontend-templates.md) hâlâ öneri durumundadır ve ancak deney kapılarından sonra kabul edilecektir.

## Hedef deneyim

Bir müşteri şu iki yoldan birini seçebilmelidir:

```text
hydro edition create --presentation=headless
  → Admin Studio + API
  → müşteri kendi Next.js/Nuxt/Astro frontend'ini bağlar

hydro frontend create storefront
  → ayrı frontend starter repository
  → API URL ve public key/config girilir
```

Küçük kurulumda `presentation=integrated` seçeneği server-rendered template ile devam eder; bu seçenek varsayılan frontend geliştirme modeli olmaz.

## Faz 0 — Kullanım araştırması ve sınır doğrulama

**Amaç:** Generator'ın otomatikleştireceği tekrarı gerçek akıştan ayırmak.

- Pages, News ve Commerce'in mevcut admin handler/formlarını alan, ilişki, workflow ve özel kural olarak envanterle.
- İki referans iş seç: kurumsal site + varyantlı storefront.
- Kullanıcı yolculuğunu ölç: collection ekleme, alan ekleme, publish, katalog yönetimi, frontendte liste/detail/cart.
- Her URL'yi admin, API veya frontend sahibi olarak sınıflandır; API/HTML karışan rotaları kayda al.

**Çıkış:** Tekrarlanan CRUD alanları ile custom kod gerektiren kurallar yazılı kanıtla ayrılmış olur. Bu faz `module-spec` v1'i dondurmaz. İlk envanter: [Admin Studio yüzey envanteri](validation/admin-studio-inventory-2026-09-10.md).

## Faz 1 — Module spec v0 ve metadata compiler

**Amaç:** Mevcut generator'ı Admin Studio metadata'sına genişletmek.

- `module-specs/` altında sürümlü collection spec tanımı tasarla.
- Field ID, validation, relation, permission, API projection, admin widget ve migration intent'ini kapsa.
- Deterministik compiler çıktısı üret: Go typed metadata + Admin Studio manifest + incelenebilir migration taslağı.
- `generated/` ve `custom/` ayrımını ADR-0010 kurallarıyla koru.
- Runtime DDL, implicit rename/drop veya açık izin olmadan public API exposure üretme.

**Kabul:** Aynı spec iki kez üretildiğinde sıfır diff; custom dosyalar değişmedi; invalid/destructive spec fail-closed reddedildi.

## Faz 2 — Admin Studio çekirdeği

**Amaç:** Metadata ile tutarlı ve güvenli yönetim yüzeyi üretmek.

- Ortak layout, navigation, list/detail/form ekranları, pagination, filtre, relation picker ve validation error UI.
- Mutation'lar yalnız application service üzerinden geçer; DB query veya yetki kısayolu yoktur.
- CSRF, CSP, optimistic-lock conflict, audit request ID ve field-level permission görünürlüğünü ortaklaştır.
- Pages ve News'i ilk iki doğrulama modülü olarak Studio'ya geçir; özel publish/schedule davranışlarını custom panel extension olarak tut.

**Kabul:** Eski ve Studio admin akışları SQLite/PostgreSQL HTTP contract testlerinde aynı hata kodu, audit ve state transition sonucunu verir.

## Faz 3 — API client ve frontend contract

**Amaç:** Frontend'in Core'a bağlı ama Core UI'ından bağımsız gelişmesi.

- OpenAPI/GraphQL schema'dan typed TypeScript client üret veya sürümlü ince client yaz.
- Public read, preview, auth/session, cart ve checkout için net `/api/v1` sözleşmelerini belgeleyip fixture ekle.
- CORS, cookie domain, CSRF ve preview token kararlarını ayrı threat-model ekiyle doğrula.
- Public route sahipliğini frontend uygulamasına ver; Core sadece API/admin/webhook yollarını sunar.

**Kabul:** Başka repository'deki fixture frontend, Core internal import etmeden yalnız SDK ile build ve e2e test yapar.

## Faz 4 — Resmî frontend starter'lar

**Amaç:** Headless başlangıç maliyetini azaltmak.

1. `site-starter`: page liste/detail, SEO, navigation, locale, media ve form.
2. `storefront-starter`: ürün, varyant, authenticated cart, checkout ve provider redirect handling.
3. `editorial-starter`: news liste/detail ve yayınlanmış içerik cache/ISR stratejisi.

Starter'lar ayrı deployment'tır; Core binary'sine asset veya Node runtime eklemez. Her starter Core sürüm matrisi, `.env.example`, seed fixture ve Playwright/e2e happy path taşır.

**Kabul:** Sıfırdan seed edilmiş ortamda bir geliştirici starter'ı dokümante akışla ayağa kaldırır; login-cart-checkout ve publish-to-rendered-page e2e kanıtı geçer.

## Faz 5 — Integrated template'i adapter olarak konumlandırma

**Amaç:** Mevcut tema yatırımını korurken ana sunum modelini sadeleştirmek.

- Tema paketlerini `presentation=integrated` seçeneğiyle sınırla.
- Public HTML route'larının API eşleriyle URL çakışmasını kaldır; eski yollar için deprecation/redirect planı yaz.
- Tema asset manifest, retained asset ve blue/green contract'larını koru.
- Headless build'de tema route, asset ve render bağımlılıklarının hiç yüklenmediğini test et.

**Kabul:** Aynı content projection, integrated template ve ayrı starter tarafından farklı UI ile doğru, yetkili ve cache-safe gösterilir.

## Faz 6 — Geçiş, destek ve release

- Mevcut edition'lar için integrated → headless migration rehberi ve uyumluluk tablosu.
- API deprecation pencereleri, starter semver politikası ve edition/starter lockfile uyumluluğu.
- SQLite/PostgreSQL migration, rollback, blue/green asset ve API contract testleri.
- İki gerçek müşteri işiyle süre ve hata oranı ölçümü; otomasyonun net faydası doğrulanmadan yeni spec yüzeyini v1 yapmama.

## Uygulama sırası

Önce Faz 0–2 ile **admin otomasyonunu**, sonra Faz 3–4 ile **headless frontend deneyimini** doğrula. Faz 5 ancak starter'lar gerçek kullanımla doğrulandıktan sonra yapılır. Böylece mevcut integrated müşteriler kırılmadan yeni varsayılan deneyim kanıtlanır.

## Açık kararlar

- Typed TypeScript SDK kod üretimi mi, minimal fetch client mı?
- Frontend auth için same-site cookie mi, BFF mi, token tabanlı ayrı mobile API mi?
- Preview akışı hangi signed-token ve cache-bypass sözleşmesiyle çalışacak?
- Starter'ların release/version compatibility matrix'i Core release manifest'ine nasıl bağlanacak?

Bu sorular uygulamaya başlamayı değil, Faz 3'ün dış API ve güvenlik yüzeyini dondurmayı engeller.
