# R5 SDK artifact grant sözleşmesi — 2026-09-16

**Durum:** Başarılı (lokal/WSL)

`internal/sdkdist.ArtifactGrant` ve `ValidateGrant`, broker ile CLI arasındaki
kısa ömürlü yetki verisini tanımlar. Grant proje ID’si, Core/SDK sürümleri,
artifact URL’si, issued/expiry zamanları ve access token’ı bağlar. Token
persist edilmez veya log’lanmaz; expiry en fazla 24 saat, zaman toleransı 5
dakikadır ve URL ayrıca download policy’den geçer.

## Kanıt

- Windows: `go test ./internal/sdkdist` başarılı.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` ve `go test ./...` başarılı.
- Yanlış proje/sürüm, expired, gelecekte geçerli olacak, uzun ömürlü ve unsafe
  URL grant’leri reddedildi.
- `RedactToken` kısa veya tam token sızıntısını önlüyor.

## Sınır

Bu bir broker veya OAuth uygulaması değildir. Token üretimi, kullanıcı oturumu,
revocation ve entitlement kontrolü dış serviste kalır; CLI yalnızca broker’ın
ürettiği grant’i bu sözleşmeyle doğrular.
