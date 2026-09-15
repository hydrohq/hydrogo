# Payload hızına yaklaşma ve HydroGo geliştirici deneyimi planı

**Durum:** Planlandı; uygulama başlamadı  
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
