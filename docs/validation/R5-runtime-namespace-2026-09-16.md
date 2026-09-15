# R5 runtime namespace doğrulaması — 2026-09-16

**Durum:** Başarılı (lokal/WSL)

Release bütünlük doğrulaması, SDK kullanıcıları için `hydro runtime verify`
namespace’i üzerinden de erişilebilir. Komut, mevcut `hydro release verify`
uygulamasını tekrar kullanır; iki farklı doğrulama mantığı oluşmaz.

## Kanıt

- `go test ./cmd/hydro ./internal/release` başarılı.
- WSL Ubuntu/Go 1.27 üzerinde `go test ./...` başarılı.
- CLI testi, geçerli manifest/artifact’ı kabul eden ve değiştirilmiş artifact’ı
  reddeden `runtime verify` alias’ını doğrular.

## Kullanım

```bash
hydro runtime verify --manifest release.json --public-key release.pub --root ./runtime
```

## Sınır

Bu yalnızca yerel doğrulamadır; indirme, login, entitlement veya artifact broker
sağlamaz.
