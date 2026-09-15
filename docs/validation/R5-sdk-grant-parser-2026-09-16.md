# R5 SDK grant parser — 2026-09-16

**Durum:** Başarılı (lokal/WSL)

`internal/sdkdist.ParseGrant`, broker response gövdesini 64 KiB ile sınırlar,
JSON içindeki trailing veriyi reddeder ve grant’i entitlement doğrulamasına hazır
hale getirir. Token saklanmaz veya log’lanmaz; `ValidateGrant` sonrasında
`DownloadGrantAndInstall` çağrılmalıdır.

## Kanıt

- Windows: `go test ./internal/sdkdist` başarılı.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` başarılı.
- WSL’de değişiklik sonrası tam `go test ./...` çalıştırıldı; paketler test
  edilmeye devam ederken CLI gözlem süresi doldu, bu nedenle bu rapor yalnızca
  tamamlanan `internal/sdkdist` kanıtını kapatır. Önceki tam suite kanıtı
  `R5-sdk-grant-download-integration-2026-09-16.md` içinde korunur.
- Oversized ve trailing JSON grant response’ları reddedildi.

## Sınır

Bu parser OAuth/device flow, token issuance veya revocation yapmaz.
