# R1 Güvenlik, Kimlik Doğrulama ve CSRF Doğrulaması

**Tarih:** 2026-09-06  
**Roadmap maddesi:** R1 Çalışma Zamanı Güvenlik Temeli  
**Durum:** Düzeltme turunda; yerel testler başarılı, CI ve auth akışının tamamı henüz kapanmadı.

## Doğrulanan Davranışlar

1. **Argon2id Parola Hashing:**
   - OWASP uyumlu parametrelerle (`memory=64MB`, `iterations=1`, `parallelism=4`, `saltLength=16`, `keyLength=32`) parola hashing ve sabit zamanlı (`subtle.ConstantTimeCompare`) doğrulama uygulandı.
    - Zamanlama saldırılarına karşı geçersiz kullanıcı sorgularında gerçek kullanıcı hash'iyle aynı maliyetli sahte doğrulama işletilir.

2. **Opaque DB-Backed Session Yönetimi:**
   - 32 byte kriptografik rastgele token üretimi (`crypto/rand`).
   - Veritabanında yalnızca token'ın SHA-256 özeti (`token_hash`) saklanır; veritabanı sızıntısında açık oturum token'ları ele geçirilemez.
   - Oturum çerezi `HttpOnly`, `SameSite=Lax`, `Secure` bayraklarıyla ayarlanır.
   - SQLite ve PostgreSQL üzerinde oturum oluşturma, doğrulama, son görülme güncelleme (`touch`), silme ve süresi dolanları temizleme sözleşmeleri `tests/contract/auth_contract_test.go` ile test edildi.

3. **CSRF Koruması:**
   - Durum değiştiren HTTP metotlarında (`POST`, `PUT`, `PATCH`, `DELETE`) `X-CSRF-Token` başlığı veya `_csrf` form parametresi zorunlu kılındı.
   - Geçersiz veya eksik CSRF token içeren istekler 403 Forbidden ile engellendi.

4. **Sıkı Güvenlik Başlıkları & Dinamik CSP Nonce:**
   - Her HTTP isteği için kriptografik CSP nonce üretilerek bağlama (`context`) enjekte edildi.
   - `Content-Security-Policy: default-src 'self'; script-src 'self' 'nonce-...'; style-src 'self' 'nonce-...'; img-src 'self' data: blob:; object-src 'none'; frame-ancestors 'none'; form-action 'self'` başlığı uygulandı.
   - `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin` başlıkları zorunlu kılındı.

5. **CLI Bootstrap:**
    - `hydro create-admin --email ... --password-stdin` komutu ile ilk yönetici hesabı oluşturma CLI seviyesinde doğrulandı; parola komut satırına yazılmaz.

## Çalıştırılan Komutlar ve Test Çıktısı

```text
go test -v ./internal/auth/...
go test -v ./internal/security/...
go test -v ./internal/server/...
go test -v ./tests/contract/...
go test -v ./cmd/hydro/...
```
Tüm testler başarıyla (PASS) sonuçlandı.
