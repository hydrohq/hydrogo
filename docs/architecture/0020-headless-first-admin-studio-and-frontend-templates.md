# ADR-0020: Headless-first Admin Studio ve frontend template sınırı

- **Durum:** Önerildi
- **Tarih:** 2026-09-10
- **İlişkili kararlar:** [ADR-0009](0009-site-edition-and-delivery-scope.md), [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0015](0015-versioned-http-api.md), [ADR-0016](0016-structured-page-blocks-and-theme-slots.md), [ADR-0018](0018-filesystem-theme-bundles.md)

## Bağlam

Mevcut integrated sunum, JSON API, server-rendered public görünüm ve tema paketlerini aynı binary içinde taşıyabilir. Bu, küçük Site Edition teslimleri için hızlıdır; ancak frontend geliştiren müşterilerde API ile HTML rota ayrımını belirsizleştirir ve her modülün el yazımı admin ekranı geliştirme maliyetini artırır.

Ürün hedefi iki gereksinimi aynı anda karşılamalıdır:

1. Standart veri odaklı modüller için güvenli admin CRUD yüzeyi tekrar yazılmamalıdır.
2. Müşteri, HydroCore fork etmeden kendi framework'ü ve deployment'ı ile frontend geliştirebilmelidir.

## Önerilen karar

HydroGo'nun varsayılan ürün yönü **headless-first** olur:

- HydroCore; domain servisleri, sürümlü API, medya, auth, yetki, workflow, audit ve commerce altyapısını sağlar.
- **Admin Studio**, modül spec'inden türetilen metadata ile liste, detay, form, ilişki, publish ve audit yüzeylerini üretir.
- Public frontend ayrı bir istemcidir; Core API'sini tüketir. Core public HTML üretmek zorunda değildir.
- Resmî frontend starter/template'leri ayrı repository veya ayrı paket olarak dağıtılır; Core'a gömülü bir iş kuralı taşımaz.
- Mevcut filesystem theme sistemi kaldırılmaz; küçük integrated kurulumlar için opsiyonel `presentation=integrated` adapter'ı olarak kalır. Template, Core API sözleşmesinin yerine geçmez.

Rota sahipliği kesin ayrılır:

```text
/admin/...  Admin Studio ve operatör yüzeyi
/api/v1/... Sürümlü makine API'si
/webhooks/... Sağlayıcı callback'leri
frontend domain'i veya ayrı uygulaması  Müşteri görünümü
```

`/public/...` ad alanı yeni frontend sözleşmesi olarak büyütülmez. Mevcut public HTML/API yüzeyleri, belgeli uyumluluk süresi boyunca korunur veya sürümlü geçiş rehberiyle ayrı adapter'a taşınır.

## Admin Studio sözleşmesi

Bir collection/module spec'i en az aşağıdakileri açıkça tanımlar:

- sabit alan kimliği, türü, required/null/default davranışı ve validasyon;
- ilişki, indeks, locale/revision/export davranışı;
- alan/kayıt yetkisi ve workflow capability'leri;
- admin görünürlüğü, sıralama, filtreleme ve form widget ipucu;
- API exposure ve projection tanımı;
- migration etkisi ve rename/deprecation kaydı.

Spec yalnız standart CRUD metadata'sı üretir. Özel iş akışı, finansal kurallar, provider entegrasyonu ve destructive dönüşümler açık module kodu ve incelenebilir migration ister. Runtime'da spec yükleyerek DDL çalıştırılmaz.

Admin Studio, Core servislerini atlayıp DB'ye doğrudan erişmez. Her mutation aynı application service, yetki, CSRF, audit ve outbox politikasından geçer.

## Frontend template modeli

Resmî template'ler ayrı starter'lar olur:

| Template | Kapsam | Core bağımlılığı |
|---|---|---|
| `site-starter` | SEO, navigation, pages, media, contact form | REST/GraphQL SDK |
| `storefront-starter` | katalog, varyant, cart, checkout, account | Commerce API SDK |
| `editorial-starter` | news listing/detail, locale, preview integration | Content API SDK |

Her starter'ın kendi lockfile'ı, environment örneği, e2e fixture'ı ve Core API compatibility matrix kaydı olur. Temalar yalnız demo veya integrated deployment için asset/layout paketi olarak kalır; müşteri frontend'inin runtime bağımlılığı değildir.

## Geçiş ve uyumluluk

- `/api/v1` ana makine sözleşmesidir; ADR-0015 alias davranışı korunur.
- Mevcut server-rendered rotalar kaldırılmadan önce kullanıcıya görünen linkler frontend adapter URL'lerine taşınır; JSON endpoint ile HTML endpoint aynı rota adı altında karıştırılmaz.
- Admin Studio ilk aşamada Pages, News ve Commerce üzerinde mevcut handler'larla davranış eşitliği kanıtlar. Eski admin rotaları yeni Studio ekranlarına yönlendirme ancak form/CSRF/audit contract'ı korunduktan sonra yapılır.
- Yeni spec alanı migration planına, release metadata'sına ve SQLite/PostgreSQL contract kitine girer. Yayınlanmış migration veya field ID geriye dönük değiştirilmez.

## Kabul kapıları

Bu öneri ancak aşağıdaki deneyler başarılı olursa kabul edilir:

1. Pages ve News için aynı spec'ten deterministik Admin Studio surface'i üretilir; özel workflow kodu korunur.
2. Ayrı bir storefront starter, yalnız `/api/v1` üzerinden login, cart ve checkout happy path'ini e2e testte tamamlar.
3. Integrated tema ve headless edition aynı published projection'ı tüketir; taslak veya private medya sızıntısı oluşmaz.
4. SQLite ve PostgreSQL üzerinde generated admin mutation'ları aynı yetki, conflict, audit ve migration contract testlerini geçer.
5. Compact profilde headless edition public theme asset/route/worker yüklemez.
6. Eski integrated edition yeni Core sürümüyle ve yeni starter eski desteklenen API sürümüyle compatibility fixture'larını geçer.

## Alternatifler

- **Sadece Core içine tema:** hızlı başlangıç sağlar fakat frontend özgürlüğünü ve deployment ayrımını zayıflatır.
- **Sadece headless, starter yok:** mimariyi temiz tutar ama müşteri başlangıç maliyetini yükseltir.
- **Runtime schema builder:** hızlı görünür; migration, yetki, audit, rollback ve DB taşınabilirliği için denetlenemeyen risk oluşturur.

Önerilen hibrit model, otomatik admin deneyimini ve frontend özgürlüğünü aynı anda korur; fakat iki gerçek müşteri akışında doğrulanmadan kararlı v1 sözleşmesi yapılmaz.
