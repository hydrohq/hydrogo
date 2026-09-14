# HydroGo Prototype Velocity Plan

Amaç: Production sözleşmelerini koruyan ilk çalışan modülü Next.js benzeri
çeviklikle üretebilmek.

## İlkeler

- Generator yalnızca tekrar kanıtlanmış kalıpları üretir; iş kuralları açık kod olarak kalır.
- Server-rendered varsayılan korunur; gerektiğinde interaktif island ve typed API client kullanılır.
- Her üretilen parça erişilebilirlik, yetkilendirme, migration ve test iskeletiyle gelir.
- Core müşteri davranışlarıyla kirletilmez; preset ve Client Edition sınırları kullanılır.

## Aşamalar

### V0 — Tek komutla geliştirme ortamı

- `hydro dev`: SQLite, migration, seed, mail preview, Redis/MinIO ve browser açılışını orkestre eder.
- Dosya izleme, template/CSS hızlı yenileme, temiz hata ve port çakışması yönetimi.
- `.env.example`, `hydro doctor --fix`, disposable test verisi ve WSL/Windows eşdeğerliği.

### V1 — Schema-driven üretim

- `hydro generate resource|crud|page|api|admin|form` komutları.
- Tek şemadan model, migration, repository, service, policy, form, table, API, OpenAPI, TypeScript client, fixture ve contract test üretimi.
- Üretilen kodda TODO sınırları, örnek authorization ve review checklist’i.
- `hydro validate generator --spec SPEC --root OUTPUT` lock hash’ini ve beklenen
  generated artifact setini yazmadan doğrular; upgrade/pre-deploy CI adımı olarak
  kullanılabilir.

### V2 — Ürünlaştırılmış UI sistemi

- shadcn uyumlu light/dark tema ve mevcut tema sisteminin birlikte çalışması.
- Form, data table, filter, modal, drawer, toast, wizard, upload, empty/error state ve responsive layout preset’leri.
- Component showcase; RTL, uzun metin, klavye, reduced-motion ve yüksek kontrast örnekleri.
- Visual regression baseline ve tasarım token lint’i.

### V3 — Hazır ürün preset’leri

- news-site, ecommerce, corporate, membership, dashboard ve erp başlangıçları.
- Her preset navigation, yetkiler, seed, demo ekranları, API ve deployment notlarıyla gelir.
- Preset’ler generator smoke ve upgrade testlerinden geçer.

### V4 — Etkileşim ve veri katmanı hızlandırma

- HTMX/HTML action sonuçları, optimistic UI ve uzun işlerde progress bileşenleri.
- Typed API client, pagination/filter/sort standardı, cache ve invalidation üretim şablonları.
- Arama, import/export ve bulk action için bounded ortak bileşenler.

### V5 — Ekip ve kalite ölçeği

- Module contract test generator, API compatibility check ve seçili mutation testleri.
- ADR/template/doc generator ve drift kontrolü.
- `time-to-first-route`, `time-to-first-crud`, build süresi ve defect escape ölçümü.
- SDK v1, semver, deprecation, upgrade assistant ve backwards-compatibility testleri.

## Kabul ölçütleri

- Temiz projede ilk public sayfa ve admin CRUD 15 dakikadan kısa sürede üretilebilmeli.
- Üretilen CRUD SQLite/PostgreSQL contract testlerinden geçmeli.
- Yetkisiz erişim, CSRF, validation, migration rollback ve erişilebilirlik testleri otomatik gelmeli.
- `hydro dev` tek komutla local bağımlılıkları başlatabilmeli.

İşletmeye özel muhasebe/ERP kuralları, ödeme seçimi, müşteri UX kararları ve
production credentials generator tarafından uydurulmaz; Client Edition veya
ayrı ADR/ürün kararlarıdır.
