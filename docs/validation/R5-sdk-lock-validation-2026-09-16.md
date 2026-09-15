# R5 SDK lockfile doğrulaması — 2026-09-16

**Durum:** Başarılı (lokal/WSL)

Project generator’a bounded `hydrogo.lock` yükleme/doğrulama eklendi. `hydro project check`, lockfile mevcutsa sürüm 1, proje kimliği, edition/preset, Core dağıtımı, generator sürümü, lisans zorunluluğu ve duplicate/boş feature değerlerini kontrol eder. Bu işlem ağ veya veritabanı bağlantısı gerektirmez.

## Kanıt

- Windows: `go test ./internal/projectgen ./cmd/hydro` başarılı.
- WSL Ubuntu/Go 1.27: `go test ./...` başarılı.
- Üretilmiş geçerli lockfile kabul edildi.
- Boş Core sürümüne sahip değiştirilmiş lockfile reddedildi.

## Sınır

Bu kontrol yalnızca lockfile bütünlüğü ve yerel uyumluluk metadata’sıdır. Runtime entitlement, broker erişimi ve binary imzası ayrıca doğrulanır.
