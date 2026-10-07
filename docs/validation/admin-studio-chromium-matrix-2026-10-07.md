# Admin Studio ve müşteri akışları Chromium matrisi

**Tarih:** 2026-10-07  
**Sonuç:** 63/63 Chromium testi geçti  
**Kapsam:** Admin ve editör deneyimi, generated/dynamic CRUD, MFA, erişilebilirlik,
responsive davranış ve public sayfalar.

## Çalıştırma ortamı

Playwright paketi, seed edilmiş izole SQLite Edition ve geçerli tek kullanımlık
admin storage state ile çalıştırıldı. Generated Admin smoke testleri
`HYDRO_E2E_BASE_URL` ve `HYDRO_E2E_STORAGE_STATE` değişkenlerini kullanır;
kimlik doğrulaması olmayan rastgele bir `localhost:8090` sunucusu tam Admin
matrisi için geçerli fixture değildir. `HYDRO_E2E_ADMIN_EMAIL` ve
`HYDRO_E2E_ADMIN_PASSWORD` yalnızca izole yerel test hesabı içindir.

İzole sunucunun hazırlığı:

```sh
go build -trimpath -buildvcs=false -o "$TMPDIR/hydro" ./cmd/hydro
export HYDRO_MEDIA_DIR="$TMPDIR/media"
export HYDRO_ENABLE_DEMO_FRONTEND=true
"$TMPDIR/hydro" migrate --database-dialect sqlite --database-dsn "$TMPDIR/edition.sqlite"
"$TMPDIR/hydro" seed-demo --confirm-demo --database-dialect sqlite --database-dsn "$TMPDIR/edition.sqlite"
printf '%s\n' "$HYDRO_E2E_ADMIN_PASSWORD" | "$TMPDIR/hydro" create-admin --email "$HYDRO_E2E_ADMIN_EMAIL" --password-stdin --database-dialect sqlite --database-dsn "$TMPDIR/edition.sqlite"
"$TMPDIR/hydro" serve --dev --database-dialect sqlite --database-dsn "$TMPDIR/edition.sqlite" --address 127.0.0.1:PORT
```

Sunucu çalışırken, geçici `auth-state.json` üretildikten sonra:

```sh
cd e2e
export HYDRO_E2E_BASE_URL=http://127.0.0.1:PORT
export HYDRO_E2E_STORAGE_STATE="$TMPDIR/auth-state.json"
npm run setup-auth
npm run generate
npx playwright test --project=chromium --workers=1 --reporter=line
```

## Sonuç

- İlk doğrudan koşu fixture hazırlanmadan yapıldığı için generated Admin smoke
  testleri login sayfasına yönlendi. Bu, ürün hatası değil test ortamı
  eksiğiydi. Testler açıkça uygun admin session gerektirir.
- Doğru isolated server ve `setup-auth.mjs` ile tam Chromium matrisi **63/63**
  geçti. Kapsamda 15 viewport/responsive kontrolü, keyboard/focus ve adlandırma,
  admin role/CSRF, davet/parola/MFA, schema-driven dynamic collection CRUD,
  generated CRUD, rich-text/relation/media, taslak/yayın, kritik müşteri
  yolculukları ve manifestten üretilmiş public/Admin smoke testleri bulunur.
- İlk tam koşuda generated Admin axe taraması tek bir kontrast uyarısı verdi;
  izole tekil tekrar ve fixture-auth ile yapılan tam tekrar geçti. E2E helper'ı
  bundan sonra axe node failure summary'sini de assertion çıktısına ekler.
  Bu görünüm HydroGo temasıyla test edildi; bütün tema varyantları ve bütün
  üretilen sayfa türleri için WCAG kabulü hâlâ açık.

Admin ve MFA grubu ayrıca `custom/admin-management.spec.ts`,
`custom/admin-password-reset.spec.ts` ve `custom/admin-mfa-settings.spec.ts`
ile 5/5 geçti. Suite aynı gerçek login/challenge ve mutasyon akışlarını
bağımsız tekrarlar.
