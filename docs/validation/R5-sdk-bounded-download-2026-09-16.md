# R5 SDK bounded artifact download — 2026-09-16

**Durum:** Başarılı (lokal/WSL)

`internal/sdkdist.DownloadAndInstall`, broker’dan alınan URL’den runtime
artifact’ını bounded response body ile geçici dosyaya indirir. Redirect takip
edilmez, HTTP status 200 dışı cevaplar reddedilir, boyut limiti uygulanır ve
sonrasında checksum doğrulamalı atomic install çalışır.

## Kanıt

- Windows: `go test ./internal/sdkdist` başarılı.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` ve `go test ./...` başarılı.
- Local HTTP test sunucusundan doğru artifact indirildi ve kuruldu.
- Redirect ve configured size limit aşımları reddedildi.

## Sınır

Bu client login/token üretmez; URL’nin broker tarafından yetkilendirilmiş
olması çağıranın sorumluluğundadır. Production’da yalnızca HTTPS URL ve
short-lived broker authorization kullanılmalıdır.
