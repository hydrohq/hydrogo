# R5 SDK platform hedefleri — 2026-09-16

**Durum:** Başarılı (lokal/WSL)

`internal/sdkdist` paketi, broker ve installer’ların paylaşacağı platform sözleşmesini ekledi. Linux/Windows/macOS ve amd64/arm64 hedefleri normalize edilir; Windows için `.exe` uzantısı eklenir; bilinmeyen platform, mimari veya path içeren ürün adı fail-closed reddedilir.

## Kanıt

- Windows: `go test ./internal/sdkdist` başarılı.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` ve `go test ./...` başarılı.
- Testler Linux/Windows/macOS adlandırmasını ve unsafe/unsupported hedef reddini doğrular.

## Örnek adlar

```text
hydro-linux-amd64
hydro-linux-arm64
hydro-windows-amd64.exe
hydro-darwin-arm64
```

## Sınır

Bu paket artifact indirme, imzalama veya installer yayınlamaz; yalnızca tüm dağıtım bileşenlerinin kullanacağı canonical target/name sözleşmesini sağlar.
