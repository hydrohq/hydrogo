# ADR-0012: İçerik, medya ve veri taşınabilirliği sınırları

- **Durum:** Değiştirildi (kısmen) — slug/301 etkinleşme zamanı için [ADR-0014](0014-publication-routes-and-erasure-recovery.md); diğer hükümler geçerlidir.
- **Tarih:** 2026-09-05
- **İlişkili kararlar:** [ADR-0006](0006-extensible-database-adapters.md), [ADR-0009](0009-site-edition-and-delivery-scope.md), [ADR-0011](0011-consistency-upgrade-and-recovery.md)
- **Uygulama durumu:** Model ilkeleri kabul edildi; tam tablo/Go API tasarımı prototiple doğrulanacak.

## Karar ve alternatifler

İlk modüller tipli model, ilişkisel tablo ve modül repository'leri kullanır. Her şeyi tek EAV/JSON tablosuna taşıyan evrensel içerik motoru kurulmaz. JSON yalnız sürümlü zengin metin/blocks gibi uygun alanlarda kullanılır; filtrelenecek/unique olacak iş alanları için açık kolon/indeks ve dialect migration gerekir. Gelecekteki şema üretici de bu modeli üretir.

Bu seçim bazı modüllerde daha fazla sorgu/migration yazmayı gerektirir; davranış ve indekslerin açık olmasını sağlar. DB'ye özgü JSON/full-text yetenekleri domain sözleşmesi haline gelmez.

## İçerik kimliği ve yayın

- Opaque kimlik uygulama tarafında üretilir; ilk ortak kimlik biçimi UUIDv4'tür. Public URL slug'dan ayrıdır. UUID nesnenin varlığına erişim yetkisi sağlamaz.
- Zaman UTC, persistence/API karşılaştırmalarında mikrosaniye hassasiyetiyle normalize edilir; scheduler yerel saat ve DST yorumunu UI'da açık gösterir.
- Mantıksal içerik kimliği, locale varyantı ve revision kimliği ayrıdır. Düzenleme sürümü optimistic-lock token'ıdır; yayınlanan revision ID'siyle karıştırılmaz.
- Draft kaydı canlı revision'ı değiştirmez. Publish, revision işaretçisi/audit/outbox ile atomiktir. Scheduled job sabit revision'a ve schedule generation'ına bağlıdır; eski iptal edilmiş iş yeni taslağı yayımlayamaz. Çalıştırırken durum ve izin politikası tekrar denetlenir.
- Çeviri varyantları ilişkilidir fakat birbirinden bağımsız yayımlanır. Varsayılan public davranış eksik çeviride 404'tür; fallback ancak açık edition politikası, doğru canonical ve cache anahtarıyla açılır. Taslağa fallback yoktur.
- Route kimliği site + locale + normalize edilmiş path'tir. Yeni slug/301 kaydı publish transaction içinde, reserved path ve döngü kontrolleriyle etkinleşir; taslak kaydı canlı route veya SEO projection'ını değiştirmez. Ayrıntı ADR-0014'tedir. Route normalizasyonu Unicode, case, trailing slash ve yüzde kodlama fixture'larıyla iki DB'de aynı olmalıdır; algoritma prototip kapısında sabitlenir.
- API/UI okuması yayımlanmış ve yetkili projection'dır; DB row'u doğrudan serialize edilmez. Alan bazlı okuma/yazma allowlist'i, public/özel settings ayrımı ve relation filtreleri bulunur.

## Zengin metin

İlk editör adayı Quill'dir; Delta girdi doğrulama ve sanitizer kuralları korunur. Saklanan gövde `format`, `format_version`, `document` zarfı taşır; örneğin `quill-delta` v1. Bu, tek başına bütün editörler arasında kayıpsız dönüşüm garantisi değildir.

Renderer/sanitizer sürümü türetilmiş HTML metadata'sındadır. Editör değişirse eski doküman renderer'ı destek penceresi boyunca kalır; dönüştürücü kayıp olabilecek öğeleri raporlar ve orijinal belge korunur. Format değişmesi release uyumluluk kontrolüne girer. Bilinmeyen format boş HTML'ye çevrilip sessiz veri kaybı yaratmaz.

## Medya

- Media kimliği ile storage key ayrıdır. Revision referansları sabit Media ID kullanır; içerik içine kalıcı sağlayıcı URL'si yazılmaz.
- Local adapter ilk dilimde; S3 uyumlu adapter Standard/Distributed tesliminde uygulanır. S3'e benzeyen endpoint otomatik sertifikasyon değildir: multipart, range, metadata, imza ve hata davranışı test edilir.
- DB ile object store arasında tek transaction varsayılmaz. Upload önce `pending` kayıt ve geçici/quarantine nesne oluşturur. Doğrulama/transform tamamlanınca final immutable key yazılır; ardından DB'de `ready` ve referanslar commit edilir. Hatalı/sahipsiz nesneler bekleme süresiyle temizlenir; olmayan nesne public'e hazır ilan edilmez.
- Public/private sınıfı ve referans yetkisi vardır. Private nesneler public bucket/origin üzerinden kaçırılamaz; signed URL süreli ve erişim kontrolünden sonra üretilir. Hash adı yetkilendirme değildir.
- Taslakta yüklenen medyanın public olması ayrı açık geçiştir. İçerik unpublish edilince daha önce public olmuş dosyanın bütün kopyaları geri alınamaz; medya visibility ve paylaşan diğer içerikler hesaba katılır.
- Variant anahtarı kaynak digest + transform spec + encoder version içerir. Referanslı revision, bekleyen iş ve backup pin varlığında GC silemez. Transform concurrency, byte/piksel ve disk limitleri bounded olur.
- İlk format kapsamı JPEG/PNG/WebP olarak kalır; encoder seçimi ve Windows/Linux build uygunluğu teknik spike ile doğrulanır. SVG, remote URL fetch ve executable dosyalar başlangıçta açılmaz.

## Arama ve export

Arama Core içerik doğruluk kaydı değildir; modülün izinli public projection'ından türetilir. Başlangıç DB sorgusu sınırlı filtre/sıralama sunar. Harici index yeniden kurulabilir; hit'ler yetki ve güncel yayın durumu açısından servis tarafından tekrar denetlenir. İki DB'de aynı relevance sırası vaat edilmez; garanti edilen filtre/pagination semantiği ayrı tanımlanır.

DB bağımsız Hydro Export, sürümlü manifest + tür bazlı NDJSON + referanslı medya checksum'larından oluşur. Kimlikler, locale ilişkileri, revisions, routes ve modül şema sürümleri korunur. Parola hash'leri, session, secret, lisans anahtarı ve job/outbox çalışma durumu varsayılan export'a girmez; kimlik taşıma ayrı güvenli geçiştir.

Export ile üretim restore karıştırılmaz. Import önce boş staging hedefte doğrular, referansları iki geçişte kurar, bilinmeyen modül/formatı raporlar ve ölçüm raporu üretir. Dolu DB'ye otomatik merge veya DB'ler arasında kesintisiz geçiş v1 sözü değildir. İlk public v1 öncesi taşınabilir çıkış yolu round-trip fixture ile kanıtlanır; performanslı toplu taşıma sonraki geliştirme olabilir.

## Etki ve kanıtlar

SQLite ve PostgreSQL kimlik/zaman/locale/slug/unique/revision/ilişki testlerini birlikte geçer. Yeni adapter aynı semantiği sağlamalıdır. Yerel medya Compact için yeterlidir; Distributed ortak object storage gerektirir. Cache revision+locale+visibility sınırını izler, admin/private medya bypass edilir.

Format ve schema değişiklikleri migration metadata'sına girer. Blue/green'de eski reader yeni yazılan content/media formatını okuyamıyorsa online release yapılamaz. Session, asset, job ve backup sürekliliği ADR-0011'e tabidir.

Fixture'lar: iki editör çatışması, yayındaki sayfaya taslak değişikliğinin sızmaması, çeviri eksikliği, schedule iptali, eski slug döngüsü, alan yetkisi, private medya URL tahmini, object yazımı ile DB commit arasında crash, referanslı nesnenin GC'den korunması, iki DB arasında export/import ve desteklenen eski content formatının render edilmesi.
