# ADR-0023: Metadata-driven Admin ve Frontend E2E

**Durum:** Önerildi

## Bağlam

HydroGo'nun Admin Studio yüzeyi collection, field, action ve widget metadata'sı
ile tanımlanıyor. Headless frontend ise sürümlü API client üzerinden çalışıyor.
Bu yüzeylerden bağımsız, elle yazılmış E2E testleri zamanla panel ve frontend
değişikliklerinden kopabilir. Ham HTML taraması ise kırılgan selector'lar,
yetki bypass'ları ve yanlış pozitifler üretir.

## Karar önerisi

E2E test senaryoları iki açık metadata kaynağından üretilecek:

1. Admin Studio `Registry`: collection, field, widget, readonly/required bilgisi,
   action ve route sahipliği.
2. Frontend E2E manifest'i: route, görünür sayfa amacı, erişim seviyesi,
   kararlı `data-testid` değerleri ve beklenen API projection'ı.

`hydro e2e generate` veya eşdeğer build aracı bu metadata'dan incelenebilir,
deterministik Playwright testleri üretir. Üretim çıktısı ayrı `generated/`
alanında tutulur; müşteri/özel akışlar `custom/` altında bulunur ve yeniden
üretimde korunur.

Üretici yalnızca güvenli ve tanımlı senaryoları oluşturur:

- public sayfa açılışı ve yayınlanmış içerik görünürlüğü;
- admin login/session ve yetkili sayfa açılışı;
- metadata'dan required/readonly/widget doğrulamaları;
- form validation ve optimistic-lock conflict akışı;
- Pages/News publish, schedule ve preview gibi açıkça ilan edilmiş action'lar.

Yazma senaryoları yalnızca izole seed veritabanında, idempotent fixture'larla
çalışır. Gerçek production, gerçek ödeme, gerçek e-posta veya dış webhook
çağrısı E2E kapsamına alınmaz. Bilinmeyen route, selector, action veya permission
fail-closed biçimde üretimi durdurur.

## Sınırlar

- HTML/CSS sınıfı veya metin içeriği otomatik selector olarak kullanılmaz;
  uygulama kararlı `data-testid` veya erişilebilir rol/etiket sözleşmesi sağlar.
- Admin registry doğrudan DB erişimi veya yetki kararı taşımaz.
- Frontend manifest'i API contract'ını genişletmez; yalnız test niyetini ve
  gözlemlenecek yüzeyi tarif eder.
- Browser runner ve browser binary'si Compact production binary'sine eklenmez;
  yalnız geliştirme/CI test aracı olur.
- Generated testler bir güvenlik kanıtının yerine geçmez; HTTP/service contract
  testleri korunur.

## Kabul ölçütleri

- Aynı registry ve manifest iki kez üretildiğinde sıfır diff oluşur.
- Bilinmeyen veya yetkisiz metadata üretimi fail-closed reddedilir.
- Pages ve News için generated happy path ile validation/conflict path çalışır.
- Headless starter, Core internal import etmeden generated public read testini
  çalıştırabilir.
- Gerçek browser çalıştırması ayrı raporlanır; browser kurulumu yoksa test
  “atlandı” değil, açık ortam kapısı olarak raporlanır.

## Sonuç

Bu yaklaşım Admin Studio ve headless frontend'in gerçek sözleşmelerinden test
üretir, fakat rastgele HTML keşfi veya sınırsız otomasyonla sisteme kırılganlık
eklemez. Uygulama öncesi selector, manifest formatı, runner sürümü ve fixture
izolasyonu ayrıca doğrulanmalıdır.
