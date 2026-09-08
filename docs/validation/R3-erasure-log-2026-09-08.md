# R3 dış silme niyeti günlüğü doğrulaması

**Tarih:** 2026-09-08  
**Durum:** Compact dosya adapter prototipi başarılı; sağlayıcı/anahtar kurtarma tatbikatı açık.

`internal/erasure` append-only JSONL adapter'ı silme niyetlerini DB dışında saklar. Her kayıt edition, sabit hedef, işlem türü, sıra ve önceki hash bağlantısını taşır. Dosya yazımı `Sync` ile tamamlanır; aynı işlem kimliği idempotenttir ve farklı hedefle yeniden kullanılamaz. `Verify` sıra boşluğu, hash zinciri, bozuk JSON ve kayıt alanı hatalarını reddeder.

`EntriesAfter` checkpoint sonrasındaki niyetleri doğrulanmış zincirden yeniden oynatılmak üzere döndürür; içerik ve medya temizleme işlemlerinin idempotent uygulaması üst katmanın sorumluluğundadır.

Çalıştırılan kontrol:

```text
go test ./internal/erasure
```

Bu prototip henüz Pages/media silme işlemini bu günlüğe bağlamaz; dış yedek sağlayıcısı, anahtar kurtarma, retention ve restore replay S07 kapsamındaki sonraki kapılardır.
