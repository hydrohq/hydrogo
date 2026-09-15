# R5 SDK grant → download → install entegrasyonu — 2026-09-16

**Durum:** Başarılı (lokal/WSL)

`DownloadGrantAndInstall`, broker grant’ini proje/Core/SDK sürümleri ve expiry
ile doğrulamadan HTTP isteği başlatmaz. Başarılı doğrulama sonrasında bounded
download, checksum doğrulaması ve atomic cache install zinciri çalışır.

## Kanıt

- Windows: `go test ./internal/sdkdist` başarılı.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` ve `go test ./...` başarılı.
- Local broker benzeri HTTP sunucusundan geçerli grant ile kurulum başarılı.
- Project mismatch grant, ağ isteği başlamadan reddedildi.

## Sınır

OAuth/device login, grant issuance, token revocation ve entitlement servisi
uygulanmış değildir. Bu fonksiyon broker’ın ürettiği grant için güvenli istemci
giriş noktasıdır.
