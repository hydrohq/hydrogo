# R5 SDK artifact çözümleme — 2026-09-16

**Durum:** Başarılı (lokal/WSL)

`internal/sdkdist.ResolveRuntimeArtifact`, imzalı release manifestinden hedef
OS/mimariye ait canonical runtime artifact’ı tekil olarak seçer. Eksik, çift,
unsafe path veya boş checksum metadata’sı fail-closed reddedilir. İmza ve dosya
bütünlüğü doğrulaması sırasıyla `release.Verify` ve `release.VerifyArtifacts`
tarafından yapılır.

## Kanıt

- Windows: `go test ./internal/sdkdist` başarılı.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` ve `go test ./...` başarılı.
- Linux/Windows artifact seçimi ve hatalı metadata/missing artifact reddi test edildi.

## Sınır

Bu katman ağ erişimi, login, entitlement veya dosya indirme yapmaz; broker ve
installer’ın deterministic target resolution sözleşmesidir.
