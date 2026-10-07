# Admin Studio MFA self-service doğrulaması

**Tarih:** 2026-10-07  
**Durum:** Tamamlandı  
**Kapsam:** Tek Edition yöneticisi ve hesap sahibinin MFA kurulum, doğrulama ve kapatma akışı.

## Kullanıcı akışı

- `/admin/account` MFA durumunu gösterir; master key yapılandırılmamışsa
  kurulum eylemi sunulmaz.
- Kurulum TOTP secret, authenticator URI ve tek kullanımlık recovery code'ları
  gösterir. Kullanıcı kodları sakladığını onaylamadan MFA'yı açamaz.
- MFA etkinleştirme güncel TOTP kodu ister. Kapatma ise güncel TOTP veya
  recovery code ister.
- Etkin faktör varken yeni enrollment 409 ile reddedilir; faktörü sessizce
  değiştiremez. Yeniden enrollment recovery code hashlerini döndürür.
- Status, enrollment ve hesap ekranının satır içi kodları `private, no-store`
  yanıt davranışıyla taşınır. CSP inline asset hashleri mevcut strict CSP'ye
  birleştirilir.
- UI typed Templ ve Admin Studio'nun ortak `studio-card`, `studio-form`,
  `studio-form-field` ve `button` stillerini kullanır. Tarayıcı görüntüsü
  kontrol edildi; başlık/açıklama/kart/aksiyon aynı Admin görsel diliyle
  render edilir.

## Doğrulama

Başarılı komut ve senaryolar:

```sh
go test ./modules/designsystem ./internal/auth ./internal/auth/sqlstore ./internal/ui ./cmd/hydro -count=1
go test ./tests/contract -run '^TestSQLiteAuthContract$' -count=1
go test ./tests/contract -run '^TestPostgresAuthContract$' -count=1
HYDRO_TEST_POSTGRES_URL=<izole PostgreSQL 18 test URL'si> go test -p 1 ./... -count=1 -timeout=5m
HYDRO_TEST_POSTGRES_URL=<izole PostgreSQL 18 test URL'si> go test ./tests/contract -run '^TestPostgres' -count=1 -p 1
go vet ./modules/designsystem ./internal/auth ./internal/auth/sqlstore ./internal/ui ./cmd/hydro
go run ./tools/actionguard
go run ./tools/importguard -root examples/client-edition
go run ./tools/importguard -root examples/news-edition
go run ./tools/moduleguard -root modules -allow modules/api -dependencies modules/dependencies.json
go run ./tools/migrationguard -root migrations
go run ./tools/importguard -root examples/construction-edition
```

Tam PostgreSQL koşusu PostgreSQL 18.6 üzerinde, `NOBYPASSRLS` test rolüyle ve
temiz izole veritabanında tamamlandı. Dedicated PostgreSQL auth contract grubu
ayrı temiz veritabanında tekrar geçti. Auth contract MFA'yı açma, yanlış kod,
TOTP ile kapatma, recovery code ile kapatma, aktif faktörde yeniden enrollment
reddi ve kod rotasyonunu kapsar. Migration eklenmedi.

```sh
cd e2e
npx playwright test custom/admin-mfa-settings.spec.ts --project=chromium --workers=1 --reporter=line
npx playwright test custom/admin-management.spec.ts custom/admin-password-reset.spec.ts custom/admin-mfa-settings.spec.ts --project=chromium --workers=1 --reporter=line
```

Tekil MFA browser testi **1/1**, birleşik admin yönetim, davet, parola reset
ve MFA grubu **5/5** geçti. Genişletilmiş Chromium akışı TOTP
etkinleştirmesinden sonra çıkış yapıp tekrar giriş yapar, paroladan sonra MFA
challenge ister ve kod doğrulanana kadar `/auth/me` oturumunun oluşmadığını
kanıtlar. Doğru TOTP ile girişten sonra 15 Edition yönetim rotası açılır;
güvenlik IP engeli eklenip kaldırılır. Hesap MFA ekranında yanlış kodla
kapatma reddi ve doğru kodla kapatma da doğrulanır. 390px mobil viewport'ta
axe-core WCAG 2.1 A/AA taraması ve yatay taşma kontrolü geçti. MFA hesap
ekranının CSP/no-store başlıkları ve onboarding görünürlüğü de browser testinde
doğrulanır.

15 yönetim rotası boyunca her render edilmiş satır içi scriptin SHA-256 değeri
yanıtın `Content-Security-Policy` başlığındaki allowlist ile birebir
karşılaştırılır; Playwright ayrıca CSP console ihlallerini toplar. News ve media
Admin script hashleri makine kaynaklı kod için kataloglanır, News kataloğu
yalnızca modül açıkken eklenir. Control sayfası CSRF değerini script gövdesine
gömmez; cookie'den okur, böylece script hash'i kullanıcı/session verisine bağlı
kalmaz. MFA E2E 1/1 ve birleşik admin yönetim/davet/parola reset/MFA grubu 5/5
geçti. `go test ./... -count=1 -p 1 -timeout=5m`, `go vet ./...` ve action,
module, migration ve Edition import guard'ları da geçti.

MFA mobil ekran görüntüsü `e2e/artifacts/admin-mfa-mobile.png` altında Playwright
artifact'ı olarak üretildi ve görsel olarak incelendi. Artifact git'e eklenmez.

## Sınırlar

Bu çalışma yeni cross-tenant Super Admin veya yeni auth route/şeması eklemez.
TOTP/recovery API'si mevcut Edition auth sözleşmesinin üstünde kullanılır.
MFA ile doğrulanmış oturumda izin, yedek, kullanıcı ve içerik mutasyonlarının
tümleşik kombinasyon matrisi ile diğer Faz 1/2/3/5 açık kabul maddeleri ana
ürün planında takip edilmeye devam eder. Bu test 15 yönetim rotasının CSP
script hashlerini kapsar; sayfa tasarımlarının tüm tema varyantları ve tüm
admin mutasyon kombinasyonları için tam kabul kanıtı sağlamaz.
