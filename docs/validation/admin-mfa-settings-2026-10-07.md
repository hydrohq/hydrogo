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
ve MFA grubu **5/5** geçti. Chromium gerçek Edition admin oturumu açtı; setup,
recovery code gösterimi, TOTP etkinleştirme, yanlış kodla kapatma reddi ve
doğru kodla kapatmayı doğruladı. 390px mobil viewport'ta axe-core WCAG 2.1
A/AA taraması ve yatay taşma kontrolü geçti. CSP/no-store başlıkları ve
onboarding görünürlüğü de browser testinde doğrulandı.

MFA mobil ekran görüntüsü `e2e/artifacts/admin-mfa-mobile.png` altında Playwright
artifact'ı olarak üretildi ve görsel olarak incelendi. Artifact git'e eklenmez.

## Sınırlar

Bu çalışma yeni cross-tenant Super Admin veya yeni auth route/şeması eklemez.
TOTP/recovery API'si mevcut Edition auth sözleşmesinin üstünde kullanılır.
Faz 4'te diğer admin mutation kombinasyonlarının tamamı ve Faz 1/2/3/5 açık
kabul maddeleri ana ürün planında takip edilmeye devam eder.
