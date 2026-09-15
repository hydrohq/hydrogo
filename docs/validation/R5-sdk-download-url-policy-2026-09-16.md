# R5 SDK artifact download URL politikası — 2026-09-16

**Durum:** Başarılı (lokal/WSL)

`internal/sdkdist.ValidateDownloadURL`, broker’dan gelen artifact URL’sini
HTTPS ve host zorunluluğu, credential/fragment reddi ve scheme allowlist’i ile
kontrol eder. HTTP yalnızca açıkça etkinleştirilen localhost geliştirme
senaryosunda kabul edilir.

## Kanıt

- Windows: `go test ./internal/sdkdist` başarılı.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` ve `go test ./...` başarılı.
- Uzak HTTP, `file:` URL’si, credential, fragment ve malformed URL reddedildi.
- Açık localhost HTTP geliştirme istisnası kabul edildi.

## Sınır

Bu politika HTTP isteği yapmaz, token üretmez ve redirect takip etmez. İndirme
client’ı ve broker authentication sonraki fazlardır.
