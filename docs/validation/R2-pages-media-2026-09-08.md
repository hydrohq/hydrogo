# R2 Pages ve medya çıkış doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R2 Pages ve ilk müşteri teması  
**Durum:** Başarılı; R1 S03 Quill/HTMX deneyi ayrı açık kapıdır.

## Kanıtlanan davranışlar

- SQLite ve PostgreSQL migration contract’ları aynı Pages draft/revision/publish akışını çalıştırır.
- Draft güncellemesi published projection’ı değiştirmez; publish generation, audit ve outbox aynı transaction içinde ilerler.
- Aynı locale/slug için iki draft oluşturulabilir; yalnız ilk başarılı publish route’u kazanır ve kaybeden conflict döner.
- Stale editor version ile yapılan update conflict döner.
- Sanitizer, revision gövdesine yazılmadan önce uygulanır.
- Media upload metadata’sı `pending → ready` geçişine sahiptir; ready olmayan asset servis edilmez.
- Private media oturumsuz istekte 401, public ready media başarılı cevap verir.
- Injected outbox hatasında page, revision ve audit kayıtları birlikte rollback olur.

## Çalıştırılan kontroller

```text
go test ./...
go vet ./...
go build ./cmd/hydro
cd examples/client-edition && go build ./...
```

GitHub CI’de SQLite ve PostgreSQL contract servisleriyle aynı kontroller çalıştırıldı. Migration metadata checksum’ları ve fixture izolasyonu doğrulandı.

## Açık sınırlamalar

- R3 kapsamındaki SEO, restore, blue/green, MFA ve ticari Compact kapasite kanıtları bu raporun kapsamında değildir.
- R1 S03 için gerçek Quill/HTMX asset ve CSP deneyinin tamamlanması gerekir.
