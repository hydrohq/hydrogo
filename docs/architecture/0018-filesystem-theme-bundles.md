# ADR-0018: Dosya tabanlı tema paketleri ve güvenli slot renderer

- **Durum:** Kabul edildi
- **Tarih:** 2026-09-10
- **İlişkili kararlar:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0016](0016-structured-page-blocks-and-theme-slots.md)

Tema paketleri `themes/<name>/` altında `theme.json`, `layouts/base.html` ve manifestte ilan edilen asset dosyalarından oluşur. Manifest başlangıçta doğrulanır; tema adı, slotlar ve asset yolları allowlist dışına çıkamaz. `..`, mutlak yol ve Windows ayraçları reddedilir.

Core, tema klasöründen Go kodu yüklemez veya çalıştırmaz. `html/template` tabanlı layout yalnızca güvenilir uygulama tarafından üretilmiş `template.HTML` slotlarını alır. Tema seçimi build/edition veya doğrulanmış başlangıç yapılandırması sorumluluğundadır; kullanıcı girdisiyle keyfi tema yolu seçilmez.

Tema yükleme asset varlığını ve layout dosyasını doğrular. Böylece blue/green sürüm geçişinde gerekli asset inventory'si eksikse uygulama başlangıçta başarısız olur; çalışma sırasında sessiz eksik asset üretmez.
