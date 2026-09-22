# Payload hızına yaklaşma ve HydroGo geliştirici deneyimi planı

**Durum:** Kısmen uygulanıyor; güncel gerçek durum için
[`IMPLEMENTATION_STATUS.md`](IMPLEMENTATION_STATUS.md) belgesine bakın.  
**Amaç:** HydroGo’yu mimari güvenlik ve modülerlikten ödün vermeden Payload CMS’in “tek tanım → çalışan admin/API” hızına yaklaştırmak.

## Tarama özeti

Mevcut kod ve belgeler birkaç açıdan incelendi: proje generator’ı, module/resource CLI’ları, capability kit registry, Admin Studio manifestleri, frontend adapter’ları, SDK/edition ADR’leri, production roadmap’i ve açık gerçek-ortam kapıları.

Güçlü temel: `hydro new`, `hydro module create`, `hydro resource create`, SQLite/PostgreSQL kontratları, draft/revision/publish, policy katmanı, OpenAPI/TypeScript üretimi, React/HTMX/vanilla adapter’ları ve kit-aware wiring zaten mevcut.

Ana hız açığı: Üretilen çıktıların bir kısmı hâlâ reviewable scaffold niteliğinde; Payload’taki gibi tek bir koleksiyon tanımının admin, API, migration, SDK ve çalışan route’a tamamen otomatik bağlanması gerekiyor. Production roadmap’inde de gerçek müşteri kapasitesi, temiz ortam restore, PostgreSQL/sağlayıcı sertifikasyonu, resmi SDK v1 ve gerçek zamanlı dağıtım gibi kanıt kapıları açık durumda.

## Faz 0 — Ölçülebilir hedef ve referans örnek (ön koşul)

- Standart bir çok dilli haber resource’u tanımla.
- Payload ve HydroGo için aynı “ilk çalışan admin + API + seed + test” senaryosunu ölç.
- Süre, üretilen dosya sayısı, manuel dokunuş sayısı, hata sayısı, bellek ve API performansını kaydet.
- Hız iddialarını ölçüm olmadan kapatma; iki bağımsız müşteri örneğinde doğrula.

**Çıkış kriteri:** 30 dakikalık başlangıç senaryosu ve tekrarlanabilir benchmark raporu.

## Faz 1 — Tek kaynaklı Resource Contract ve CLI

- Alan, relation, validation, index, access, admin, workflow, localization, media ve API ayarlarını tek bir sözleşmede birleştir.
- `hydro project new`, `hydro resource new`, `hydro generate`, `hydro check` komutlarını tek workflow’da birleştir.
- Bir resource’tan migration, model, repository, service, policy, transport, admin manifest, OpenAPI, TypeScript client, frontend adapter, fixture, seed ve test üret.
- Üretilen route/API/admin wiring’i güvenli ve deterministik biçimde otomatik bağla; iş kurallarını tahmin etme.
- `hydro resource check`, `hydro schema diff`, `hydro api diff`, `hydro permissions check` komutlarını ekle.

**Çıkış kriteri:** Yeni resource, elle dosya taşıma veya kayıt ekleme gerektirmeden lokal olarak açılır.

## Faz 2 — Admin Studio parity

- Async relation search, nested arrays/blocks, conditional fields, tabs, rich text ve media picker.
- Bulk edit/publish/delete, saved filters/views, kolon seçimi ve pagination sınırları.
- Alan seviyesinde validation/access hata eşleme.
- Autosave, unsaved-changes, revision diff/restore ve schedule takvimi.
- Erişilebilirlik, keyboard navigation, responsive layout ve büyük listelerde sanallaştırma.

**Çıkış kriteri:** Editör, standart içerik tipleri için özel template yazmadan günlük işlemleri yapabilir.

## Faz 3 — Editorial, preview ve localization

- Güvenli live preview bridge, preview token ve origin doğrulaması.
- Draft/published/changed durumları, yayın onayı ve planlı yayın.
- Locale fallback, locale bazlı durum ve RTL/LTR sunumu.
- Preview, admin ve public rendering için aynı yetkilendirme/publish filtresi.

**Çıkış kriteri:** Haber ve kurumsal site starter’larında editoryal akış uçtan uca test edilir.

## Faz 4 — Kit/plugin ekosistemi

- Versioned kit contract, dependency graph, migration, permissions, health check ve compatibility matrix.
- Official kit’ler: SEO, forms, redirects, search, media/S3/CDN, localization, multi-tenancy, notifications, import/export, commerce.
- Community kit formatı; ancak güvenilmeyen Go kodu runtime’da yüklenmez.
- Her kit için starter, fixture, contract test ve docs.

**Çıkış kriteri:** Yeni capability, Core fork’u olmadan bir komutla seçilip kapatılabilir.

## Faz 5 — Frontend SDK ve starter’lar

- Versioned TypeScript client, React Query/SWR hooks, typed form/table/filter bileşenleri.
- Provider-agnostic design-system adapter’ları: Hydro default, shadcn, headless ve vanilla.
- Starter’lar: news, editorial, corporate, commerce, portal, membership, dashboard, ERP, headless.
- Her starter için seed, E2E, Docker/WSL ve deployment runbook.

## Faz 6 — Production ve upgrade kanıtı

- S3/CDN, backup/restore, observability, alerting ve disaster recovery tatbikatı.
- SQLite/PostgreSQL kapasite ölçümü, gerçek blue/green proxy geçişi ve rollback.
- Migration expand–migrate–contract doğrulaması.
- SDK/runtime/module compatibility matrix, deprecation ve upgrade rehearsal.
- Resmi SDK v1, cross-OS release, signing/key ceremony ve destek politikası.

## “Hazır” tanımı

HydroGo ancak şu üç koşul birlikte sağlandığında Payload hızına yakın ve production-ready kabul edilir:

1. Standart resource 30 dakikada çalışan admin/API/seed/test üretir.
2. Gerçek müşteri akışları ve desteklenen profiller ölçülmüş production kanıtına sahiptir.
3. Core, module, kit, SDK ve müşteri kodu upgrade sırasında compatibility check’ten geçer.

Bu plan, HydroGo’nun Core izolasyonu, varsayılan deny, kullanılmayan modülün sıfır footprint’i ve “önce doğrula, sonra otomatikleştir” ilkeleriyle uyumludur.

## Platform-factory expansion — Payload’ı aşma planı

Bu bölüm, HydroGo’yu yalnızca bir CMS alternatifi değil; CMS, ERP, inşaat,
üyelik, portal ve operasyon yazılımları üreten genel amaçlı bir Workspace/Engine
haline getirme hedefini tanımlar. Her faz ayrı commit, ADR/uyumluluk kaydı ve
ölçülebilir doğrulama ile teslim edilir.

### P0 — Core bloat audit ve sınırların korunması (zorunlu önkoşul)

- `internal/` altındaki her paketi Core primitive, opt-in module, kit, Edition
  veya demo/UI katmanı olarak sınıflandır.
- Core’a import edilen her bağımlılık için boyut, başlatma süresi, binary etkisi,
  Compact profile etkisi ve lisans kaydı çıkar.
- Kullanılmayan modüllerin route, migration, asset, worker, goroutine veya
  bağlantı havuzu oluşturmadığını test et.
- CMS/commerce/construction özel iş mantığını Core’dan Edition/module sınırına
  taşı; Core fork’u oluşmasını engelle.
- Tekrarlanan inline HTML/CSS/JS ve gereksiz runtime bağımlılıklarını typed
  component, embedded asset veya Edition-owned asset sınırına taşı.
- Her capability için `hydro profile explain` ve `hydro profile check` çıktısı
  üret; Compact profil için binary/resource budget kabul kriteri belirle.

**Çıkış kriteri:** Core dependency/import grafiği, capability matrisi ve Compact
  resource budget raporu yayınlanır; gereksiz Core yükleri ayrı commit’lerle
  kaldırılır veya gerekçeli ADR ile tutulur.

### P1 — Versioned domain/resource contract

Blueprint dili `collection`, `global`, `relation`, `value_object`, `computed`,
`workflow`, `event`, `policy`, `job` ve `integration` türlerini destekler.
Blueprint sürümü; schema diff, breaking-change tespiti, migration planı,
rollback metadata ve lock dosyasıyla birlikte yönetilir.

Yeni CLI akışı:

```text
hydro new <edition>
hydro resource new|validate|diff|check-breaking
hydro schema diff
hydro api diff
hydro permissions check
```

### P2 — Zero-setup identity ve policy platformu

`auth: true` yeni bir auth sistemi üretmez; mevcut Identity/Session/RBAC
çekirdeğini composition ile etkinleştirir. Admin, member, service-account,
API-key, OIDC, MFA/WebAuthn, session/device yönetimi ve tenant-aware policy
aynı sözleşmede birleşir. Resource erişimi deny-by-default olur; gerçek policy
`custom/` altında kalır.

### P3 — Güvenli deep querying ve API parity

Query URL’si önce typed filter AST’ye, sonra field/operator allowlist ve query
complexity budget kontrolünden geçer. `equals`, `contains`, `in`, `exists`,
`and`, `or`, `sort`, `limit` gibi operatörler yalnızca tanımlı alanlarda
çalışır. SQL builder prepared statement üretir; tenant predicate, join depth,
page size ve index hint sınırları SQLite/PostgreSQL kontrat testleriyle
sertifikalanır.

### P4 — Globals, validation ve relation parity

Global kaynaklar tekil form/API sözleşmesi üretir; tek kayıt invariant’ı
transaction içinde korunur. Alan validation kuralları (`min`, `max`, regex,
unique, default, immutable, email vb.) HTML, REST ve service katmanında aynı
typed contract’tan çalışır. Relation’lar `has_one`, `has_many` ve
`many_to_many`, async picker, eager loading, cascade/restrict ve N+1 bütçesi
ile genişletilir.

### P5 — Workflow, lifecycle ve event platformu

Draft/review/approved/published gibi state machine’ler Blueprint’ten tanımlanır.
Hook’lar generic ve typed interface olarak üretilir; yan etkiler doğrudan hook
içinde değil transaction → domain event → outbox → worker zincirinde çalışır.
Retry, idempotency, dead-letter, audit ve timeout davranışları zorunlu
kontrat olur.

### P6 — Admin Studio parity

Generated list/form/detail scaffolds; relation search, nested blocks, tabs,
bulk actions, saved filters, column selection, autosave, optimistic locking,
revision diff/restore, schedule takvimi, permission-aware görünürlük ve
erişilebilir keyboard navigation ile genişletilir. Runtime Studio yalnızca
güvenli metadata render eder; iş mantığı ve yetki kararları Go service/policy
katmanında kalır.

### P7 — Profile-aware frontend ve offline saha desteği

Server-rendered Templ/HTMX, headless REST/OpenAPI ve rich application
WebSocket/SSE profilleri birbirinden ayrılır. İnşaat/saha senaryoları için
offline command queue, sync cursor, conflict resolution, attachment retry ve
event replay sözleşmeleri opt-in capability olarak eklenir.

### P8 — Operations, portability ve marketplace

Her resource için export/import, dry-run, validation, media manifest ve schema
version üretilir. Backup/restore, per-tenant recovery, observability, alerting,
SBOM, signed release ve upgrade rehearsal production gate’lerine bağlanır.
Kits versioned dependency graph ile seçilir; güvenilmeyen Go kodu runtime’da
yüklenmez.

### P9 — Golden-path ve ölçümlü Payload karşılaştırması

`hydro new construction-saas` veya `hydro new editorial` komutu çalışan login,
admin, tenant, örnek resource, media, audit, OpenAPI, seed ve test fixture’ı
üretir. Aynı senaryo Payload ile karşılaştırılır: ilk çalışan MVP süresi,
manuel dosya/dokunuş sayısı, hata sayısı, binary boyutu, bellek, p95 latency ve
deployment adımı ölçülür. İki bağımsız Edition’da doğrulanmadan hız iddiası
yayınlanmaz.

### Platform “hazır” kriteri

HydroGo ancak Core bloat raporu temiz, standart resource 30 dakika içinde
çalışan admin/API/seed/test üretiyor, SQLite/PostgreSQL ve tenant isolation
kontratları geçiyor, gerçek pilot kapasitesi ölçülmüş ve Core/module/kit/SDK
upgrade matrisi yeşil ise Payload’ı aşan genel amaçlı platform olarak kabul
edilir.

## 2026-09-16 uygulama notu

Planın SDK/runtime dağıtımına temas eden ilk teslimatları uygulanmıştır:
`hydro project new/check`, lock doğrulaması, `hydro runtime verify`, platforma
göre artifact çözümleme, imza/checksum/URL/gövde sınırları, grant doğrulaması,
atomik kurulum ve cache listeleme/temizleme komutları çalışır durumdadır.
Bu teslimatlar Payload parity iddiasını tek başına kanıtlamaz; ölçüm yöntemi
[`PAYLOAD_BENCHMARK_PROTOCOL.md`](PAYLOAD_BENCHMARK_PROTOCOL.md) içinde, gerçek
faz durumu [`IMPLEMENTATION_STATUS.md`](IMPLEMENTATION_STATUS.md) içindedir.
