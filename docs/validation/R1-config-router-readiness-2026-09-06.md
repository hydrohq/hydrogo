# R1 config, router ve readiness doğrulaması

**Tarih:** 2026-09-06  
**Roadmap maddesi:** R1 çalışma zamanı temeli  
**Durum:** Kısmi başarı; bootstrap/session, CSRF ve tam audit güvenlik dilimi bekliyor.

## Uygulanan davranış

- `hydro serve` ve `hydro migrate` aynı tipli runtime config'ini kullanır.
- Config önceliği non-secret alanlarda CLI → environment → YAML → varsayılan biçimindedir. YAML bilinmeyen alanı başlangıç hatasıdır.
- Database DSN secret'ı YAML, environment ve CLI kaynaklarından yalnız birinde tanımlanabilir; çoğaltılmış secret kaynakları açık hata üretir.
- Desteklenmeyen dialect, geçersiz listen address ve eksik DSN başlangıçta reddedilir.
- SQLite PRAGMA'ları kullanıcı DSN biçimine bağlı kalmadan `database.Open` tarafından her bağlantıda zorunlu uygulanır. İlk pool sınırları SQLite için 1/1, PostgreSQL için 10 açık/5 idle bağlantıdır; gerçek profil varsayılanları benchmark bekler.
- HTTP başlangıcı migration çalıştırmaz. `hydro migrate` ayrı komuttur.
- `/healthz` yalnız proses yaşamını bildirir. `/readyz` iki saniyelik bütçede DB ping ve migration ledger checksum kontrolü yapar; hata ayrıntısını public yanıta sızdırmadan 503 döner.
- Router her rotayı `Public` veya `Protected` olarak sınıflandırmayı zorunlu kılar. Sınıflandırılmamış rota ve authorizer'sız protected rota oluşturma aşamasında reddedilir.
- HTTP read-header/read/write/idle timeout, header byte sınırı ve kontrollü shutdown yapılandırıldı.

## Yerel kanıt

Windows amd64 ve Go 1.27.0 üzerinde `go test ./...`, `go vet ./...`, CLI build ve SQLite `hydro migrate` smoke çalışması başarılı oldu. Testler config önceliğini, unknown-key ve secret conflict reddini, router default-deny davranışını, readiness hata gizliliğini ve DB contract'ını kapsar.

## Sınırlar

Mevcut protected-route authorizer yalnız router sözleşmesini kanıtlayan test bağımlılığıdır; gerçek session değildir. Bootstrap admin, Argon2id, DB-backed opaque session, CSRF, güvenlik başlıkları ve append-only audit API'si sonraki R1 güvenlik diliminde uygulanacaktır. Profile göre pool ölçümü ve config schema sürümleme de açık kalır.
