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
- Password reset token’ları purpose, hash, expiry ve tek kullanım taşır; reset aynı transaction’da parola değişimi ve session revoke ile uygulanır.
- Login ve MFA denemeleri ayrı, bounded process-local limiter'larla sınırlanır; limiter kapasitesi dolduğunda yeni anahtarlar eski kayıtları tahliye eder.
- Password reset request/confirm HTTP sözleşmesi eklendi; istek endpoint'i bilinmeyen hesaplarda da `202 Accepted` döner, token yalnız enjekte edilen notifier'a verilir ve confirm tek kullanımlı token ile session iptalini transaction içinde çalıştırır.

## Çalıştırılan kontroller

```text
go test ./...
go vet ./...
go build ./cmd/hydro
```

MFA master key değeri `HYDRO_MFA_MASTER_KEY` üzerinden secret kaynağından alınır; config veya manifest içine düz değer yazılmaz.

## Açık kapsam

Limiter'ın çoklu instance sözleşmesi, üretim notifier/invite adapter'ı, güvenlik bildirimi ve temiz ortamda restore/blue-green senaryoları R3’ün sonraki kapılarıdır.
