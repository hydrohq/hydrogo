# Admin Studio hedefli doğrulama — 2026-10-09

## Sonuç

Admin Studio, Blueprint generator, Edition composition ve güvenlik paketlerinin
testleri geçti. İlk tam `go test ./...` denemesi Admin dışı paketlerde loopback
listener yasağına takıldı. Listener gerektirmeyen HTTP contract'leri in-memory
RoundTripper'a, HTTP/FCGI server yaşam döngüsü testleri `net.Pipe` listener'a
taşındı. Son tam depo koşusu ve `go vet ./...` geçti.

## Geçen doğrulamalar

Doğru test geçici dizini (`/tmp/hydrogo-admin-tests`) ve repo dışı Go cache
kullanılarak:

```sh
GOCACHE="$PWD/.cache/admin-test" TMPDIR=/tmp/hydrogo-admin-tests GOPROXY=off \
  go test ./internal/adminstudio ./internal/collections/... ./internal/mailer/... \
  ./internal/media/... ./internal/push/... ./internal/trash/... ./internal/webhooks/... \
  ./internal/ui ./tests/contract ./examples/article-edition/custom \
  ./examples/project-edition/custom ./internal/generator ./internal/auth \
  -count=1 -p 1 -timeout=5m
```

Bu komutta paketi başarısız yapan tek test `internal/auth` içindeki loopback
bağlantılı Social OAuth testiydi. Admin/role/permission testleri ayrıca dar
kapsamda geçti. Bu ayrımın ardından:

```sh
GOCACHE="$PWD/.cache/admin-test" TMPDIR=/tmp/hydrogo-admin-tests GOPROXY=off \
  go test ./internal/auth -run 'TestPermissionCatalog|TestAdminUserManagementMutationDeniesExplicitPermissionOverride|TestAdminCannotChangeOwnRoleOrPermissions|TestAdminUserMutationEndpointsRejectEditorAndViewer|TestAdminPermissionEndpoint' \
  -count=1 -p 1 -timeout=3m

GOCACHE="$PWD/.cache/admin-test" TMPDIR=/tmp/hydrogo-admin-tests GOPROXY=off \
  go test ./cmd/hydro -run 'Test(Admin|Resource|ProjectCheck|RunResource|RunInteractiveResource|ResourceWizard|InteractiveResource|EveryRegisteredAdminControlRoute|RunPermissionsCheck)' \
  -count=1 -p 1 -timeout=5m
```

Her iki odaklı test komutu geçti. İlgili package seti için `go vet` ve
`git diff --check` de geçti. İki Edition composition testi, Generated Admin
HTTP testleri ve SMTP testleri artık listener açmadan çalışıyor; generator
testlerinde repo dışı `TMPDIR` kullanımı, geçici modülün ana repo modülünü
yanlışlıkla miras almasını engelliyor.

## Kalan çalışma ortamı kapıları

- Admin responsive Playwright testi artifact output'u yazılabilir dizine
  yönlendirilerek başlatıldı. Go setup ve SQLite migration tamamlandı, fakat
  fixture `127.0.0.1` test sunucusunu açarken `listen EPERM` aldı. Yeni browser
  screenshot, axe sonucu veya touch-target ölçümü yoktur.
- `HYDRO_TEST_POSTGRES_URL` bu çalıştırmada tanımlı değildi; Docker/PostgreSQL
  dış kapısı tekrar doğrulanmadı. Önceki PostgreSQL kanıtları planın ilgili
  kayıtlarında duruyor; bu tur için yeni canlı PostgreSQL sonucu sayılmaz.

## Bu turdaki tekrar doğrulama

İlk tam suite sırasında görülen listener hatalarının ardından AI, OAuth,
Sentry, link checker, reverse proxy, SDK broker/download, BFF ve public site
transport testleri in-memory HTTP transport kullanıyor. HTTP server graceful
shutdown ve FastCGI testleri gerçek socket açmadan `net.Pipe` kullanan listener
ile aynı framing ve drain davranışını çalıştırıyor. Üretim HTTP transportu ve
dinleme davranışı değişmedi; yalnız server'ın listener alan iç yordamı test
edilebilir hale geldi.

```sh
GOPROXY=off GOCACHE="$PWD/.cache/admin-test" GOTMPDIR=/tmp/hydro-go-home TMPDIR=/tmp \
  go test ./... -count=1 -p 1 -timeout=8m
GOPROXY=off GOCACHE="$PWD/.cache/admin-test" GOTMPDIR=/tmp/hydro-go-home TMPDIR=/tmp \
  go vet ./...
```

Her iki komut geçti. Playwright ve PostgreSQL kapıları yukarıda açık kalır.
