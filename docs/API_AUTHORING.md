# HydroGo API ve Modül Geliştirme Rehberi

Bu rehber, Core'u çatallamadan bir Client Edition, modül veya extension
geliştirmek için doğrulanmış sınırları özetler. Uygulama kodu SQL/cache/queue
sürücülerini doğrudan bilmemeli; bu ayrıntılar repository ve adapter
sözleşmelerinde kalır.

## Client Edition ve extension

`extension/v1` içindeki sözleşmeler aday-stabil API'dir. Yeni yetenekler önce
versioned bir contract'a eklenir; mevcut method imzaları sessizce değiştirilmez.
Breaking değişiklik için yeni major sürüm, deprecation alias'ı ve migration notu
gerekir. Edition'a özel davranış `custom/` veya ilgili module alanında tutulur;
HydroCore müşteri için fork edilmez.

## Route ekleme

Her route açık bir erişim sınıfıyla kaydedilir:

```go
router.Route{
    Pattern: "GET /public/example/{slug}",
    Public:  true,
    Handler: http.HandlerFunc(handler),
}
```

Public, protected ve admin route'ları birbirine karıştırmayın. Yeni route için
aynı pakette erişim sınıfı, yetkisiz istek, yanlış method ve güvenli hata mesajı
testleri ekleyin. Public projection hiçbir zaman draft, silinmiş veya scheduled
içeriği döndürmemelidir.

## Handler ve servis sınırı

Handler; kimlik, CSRF, body limiti, input doğrulama ve HTTP durum kodunu yönetir.
Domain/service katmanı iş kuralını ve transaction sınırını yönetir. Repository
SQL lehçesini soyutlar. Hata gövdelerinde driver, SQL, stack trace veya secret
detayı gösterilmez; ayrıntı log/audit bağlamında request ID ile ilişkilendirilir.

Yeni yazma akışlarında optimistic lock, idempotency ve retry belirsizliği için
test yazın. Dış provider çağrıları bounded timeout/backoff kullanmalı ve tekrar
oynatılabilir sonuç üretmelidir.

## Migration ve inactive module

Migration'lar SQLite ve PostgreSQL davranışlarını ayrı doğrulamalı; rollback ve
compatibility notu içermelidir. Inactive module route, migration, asset, worker
ve zorunlu kaynak üretmemelidir. Module açılış/kapanışını `cmd/hydro` preset smoke
testine ekleyin.

## Doğrulama döngüsü

```powershell
go test ./...
go test -race ./...
pwsh ./scripts/fuzz-smoke.ps1
pwsh ./scripts/verify-local.ps1
```

Kod değişikliğinden önce `git diff --check` ve `gofmt` çalıştırın. Release
öncesinde `hydro doctor`, SBOM ve vulnerability scan çıktısını saklayın.

## Sorun giderme

- `/readyz` başarısızsa önce `hydro doctor` ve migration durumunu kontrol edin.
- Upload reddedilirse boyut, MIME, decoded pixel limiti ve storage dizini
  izinlerini doğrulayın; geçici dosyalar kalıcı storage'a taşınmamalıdır.
- `429` yanıtında `Retry-After` değerine uyun; Redis limiter hatasında bypass
  yerine fail-closed davranış beklenir.
- Generator çıktısı beklenmiyorsa `hydro new --frontend-manifest` JSON
  sözleşmesini, benzersiz route adı/path ve `test_id` alanlarını doğrulayın.

Production PostgreSQL/Redis failover, backup-restore, blue/green rollback,
kapasite ve dış penetrasyon doğrulamaları bu yerel rehberin kapsamı dışındadır;
ilgili açık gate'ler `PROFESSIONALIZATION_BACKLOG.md` içinde tutulur.
