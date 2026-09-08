# R3 dış silme niyeti günlüğü doğrulaması

**Tarih:** 2026-09-08  
**Durum:** Compact dosya adapter prototipi, checkpoint replay ve SQLite Pages/media SQL adapter'ı başarılı; sağlayıcı/anahtar kurtarma tatbikatı açık.

`internal/erasure` append-only JSONL adapter'ı silme niyetlerini DB dışında saklar. Her kayıt edition, sabit hedef, işlem türü, sıra ve önceki hash bağlantısını taşır. Dosya yazımı `Sync` ile tamamlanır; aynı işlem kimliği idempotenttir ve farklı hedefle yeniden kullanılamaz. `Verify` sıra boşluğu, hash zinciri, bozuk JSON ve kayıt alanı hatalarını reddeder.

`EntriesAfter` checkpoint sonrasındaki niyetleri doğrulanmış zincirden yeniden oynatılmak üzere döndürür; içerik ve medya temizleme işlemlerinin idempotent uygulaması üst katmanın sorumluluğundadır.

`Replay`, zinciri doğruladıktan sonra niyetleri sırayla handler'a verir. Handler hatasında checkpoint ilerletilmez; aynı başlangıç checkpoint'iyle yeniden çalıştırma contract testinde doğrulanır. `SQLAdapter`, `erase.page` için revision ve page kayıtlarını, `erase.media` için medya metadata'sını transaction içinde idempotent siler; aynı log checkpoint'i ikinci kez uygulandığında hata vermez.

Çalıştırılan kontrol:

```text
go test ./internal/erasure
```

Kalan S07 işleri dış yedek sağlayıcısı, anahtar kurtarma, retention, storage garbage-collection ve temiz ortam restore replay tatbikatıdır.
