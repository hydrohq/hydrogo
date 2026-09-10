# ADR-0016: Yapılandırılmış sayfa blokları ve tema slotları

- **Durum:** Kabul edildi
- **Tarih:** 2026-09-10
- **İlişkili kararlar:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0012](0012-content-media-and-portability.md), [ADR-0015](0015-versioned-http-api.md)
- **Uygulama durumu:** Blok sözleşmesi, SQLite/PostgreSQL migration'ları, store round-trip doğrulaması, admin görsel editörü ve güvenli renderer uygulandı.

## Karar

Pages, rich text gövdesine ek olarak sıralı ve sürümlü `Block` listesi taşıyabilir. Blok payload'ları JSON olarak saklanır; ancak çalışma zamanı yalnızca ilan edilmiş blok tiplerini kabul eder. Sözleşme `hero`, `faq`, `cta`, `pricing_table` ve `testimonials` tiplerini kapsar.

- En fazla 64 blok ve blok başına 64 KiB payload kabul edilir.
- Payload JSON object olmalıdır ve tip bazlı zorunlu alanlardan geçmelidir.
- URL alanlarında `javascript:` ve `data:` şemaları reddedilir.
- Draft, immutable revision ve published projection aynı blok listesini transaction içinde taşır.
- Yeni blok tipi migration gerektirmez; fakat sürümlü sözleşme, renderer ve export/import testi gerektirir.
- Bilinmeyen blok tipi fail-closed reddedilir; boş içerikle sessizce kaybolmaz.

Tema sistemi mevcut `Theme` sözleşmesini kırmaz. Blok destekleyen temalar opsiyonel `BlockTheme` sözleşmesini uygular. Tema seçimi build/edition composition ile yapılır; runtime'da keyfi Go kodu veya güvenilmeyen tema yüklenmez.

## Veri ve uyumluluk

`pages.blocks`, `pages.published_blocks` ve `page_revisions.blocks` alanları SQLite'ta bounded TEXT, PostgreSQL'te JSONB olarak saklanır. Domain ve service katmanı SQL dialect bilmez. Eski kayıtlar boş blok listesiyle okunur. Format değişikliği yeni content format/version ve expand-migrate-contract geçişi gerektirir.

Public ve headless çıktılar aynı published projection ve yetki politikasını kullanır. Blok renderer'ı doğrudan DB kaydı değil, güvenli doğrulanmış model alır. Media ID'leri yalnız media service üzerinden çözülür; payload içine kalıcı sağlayıcı URL'si yazılmaz.

## Sonraki kapılar

Admin block editor, her blok için alan bazlı yetki ve sanitizer davranışı; HTML renderer; JSON/OpenAPI örnekleri; iki dialect migration contract'ı; export/import; tema asset inventory ve blue/green retained asset testiyle tamamlanır.
