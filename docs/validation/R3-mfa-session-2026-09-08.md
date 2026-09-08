# R3 kimlik ve MFA başlangıç doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R3 güvenlik başlangıcı  
**Durum:** Başlangıç kapısı başarılı; ticari pilot güvenlik kapısı açık.

## Kanıtlanan davranışlar

- Kullanıcı disable edildiğinde bütün DB-backed session kayıtları iptal edilir.
- Disabled kullanıcı session’ı tekrar doğrulanamaz.
- TOTP secret’i AES-GCM ile master key üzerinden şifrelenerek saklanır.
- Recovery code’lar yalnız SHA-256 hash olarak saklanır ve atomic update ile tek kullanımlıdır.
- SQLite ve PostgreSQL auth contract testleri MFA enroll/enable/verify ve recovery code tekrar kullanımını çalıştırır.
- Admin kontrollü user disable HTTP route’u ve audit olayı eklenmiştir.

## Çalıştırılan kontroller

```text
go test ./...
go vet ./...
go build ./cmd/hydro
```

MFA master key değeri `HYDRO_MFA_MASTER_KEY` üzerinden secret kaynağından alınır; config veya manifest içine düz değer yazılmaz.

## Açık kapsam

MFA recovery operasyonları, rate limit/concurrency bütçesi, güvenlik bildirimi, invite/reset akışları ve temiz ortamda restore/blue-green senaryoları R3’ün sonraki kapılarıdır.
