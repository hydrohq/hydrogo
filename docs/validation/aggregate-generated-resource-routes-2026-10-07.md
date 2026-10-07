# Blueprint aggregate API route registration doğrulaması

**Tarih:** 2026-10-07  
**Durum:** SQLite ve PostgreSQL 18.6 Project Edition akışları geçti.  
**Karar:** [ADR-0355](../architecture/0355-aggregate-generated-resource-route-registration.md)

## Doğrulanan akış

- Bir Go modülü içindeki Blueprint seti deterministik
  `generated.RegisterGeneratedResources` üretir. Helper her resource için
  Edition policy callback'ini, tenant kaynakları için resolver zorunluluğunu
  kontrol eder; eksik dependency varken hiçbir route kaydetmez.
- `examples/project-edition` iki generated resource'u tek aggregate API
  çağrısıyla bağlar. SQLite integration testi API create/read, Admin liste
  görünümü ve eksik resource policy'si halinde tüm route'ların kapalı kalmasını
  doğrular.
- Bağımsız `tests/external-edition` Go modülüne Blueprint ile tenant-scoped
  `article` üretilir. Generated migration SQLite'a uygulanır; API create,
  tenant içi read, tenantlar arası 404, generated Admin create/update/delete,
  CSRF reddi, viewer deny ve tenant başına ayrı Admin listesi aynı SQLite
  veritabanıyla gerçek HTTP testinde doğrulanır. Eksik tenant resolver
  başlangıçta reddedilir ve route kaydı oluşmaz.
- Aynı bağımsız Edition, ayrı bir Chromium E2E server'ında SQLite ile açılır;
  browser Admin üzerinden kayıt create/edit/delete, viewer deny, cross-tenant
  API 404 ve diğer tenant listesindeki kayıt yokluğu doğrulanır.
- `hydro project check` aggregate helper'ı module path ve Blueprint setiyle
  karşılaştırır; generator testi helper'ın elle değiştirilmesini reddeder.
- Göreli `--out tests/external-edition` yolu CLI'nin scaffold `gofmt` adımında
  aynı dizinin iki kez eklenmesine neden oluyordu. CLI doğrulama kökünü mutlak
  yola çevirir; ek regresyon testi göreli kökte gofmt ve `go test ./...`
  çalıştırır. Göreli `--out` ile tam scaffold doğrulaması da geçti.

## Sonuçlar

- `go test ./... -count=1 -p 1 -timeout=5m` geçti.
- `go vet ./...`, actionguard, moduleguard, migrationguard (110 migration) ve
  Client Edition importguard'ları geçti. Yeni bağımsız
  `tests/external-edition` fixture'ı da `importguard` kontrolünden geçti.
- `go test ./internal/generator ./examples/project-edition/... -count=1`,
  external Edition `go test -mod=mod ./... -count=1`, göreli scaffold CLI
  regresyonu ve `hydro project check` stale-helper testi geçti.
- Generated Admin Chromium CRUD **1/1** geçti.
- Bağımsız Edition Chromium Admin CRUD ve tenant/role akışı **1/1** geçti.
- İzole PostgreSQL 18.6 cluster'ında
  `HYDRO_TEST_POSTGRES_URL=... go test ./examples/project-edition/custom
  -run '^TestPostgresEditionGeneratedAdminCRUD$' -count=1` **1/1 geçti**.
  Aynı aggregate helper'la API create ve generated Admin list readback yapıldı.
- Aggregate helper mevcut resource SQL transport'unu route eder; yeni query,
  schema, transaction veya migration davranışı getirmez.

## Komutlar

```sh
go test ./internal/generator ./examples/project-edition/... -count=1
go test ./cmd/hydro -run TestGeneratedScaffoldVerificationResolvesRelativeRoot -count=1
```

Bağımsız Edition paket testi `tests/external-edition` dizininden
`go test -mod=mod ./... -count=1` ile çalışır. Tam generated Admin Chromium
CRUD testi de **1/1 geçti**:

```sh
cd e2e
npx playwright test custom/generated-admin-crud.spec.ts --project=chromium --workers=1 --reporter=line
```

## Sınırlar

Helper açıkça çağrılınca bütün aggregate Blueprint resource'ları API'ye
bağlanır; kısmi Edition doğrudan seçili `Register<Name>Route` fonksiyonlarını
kullanır. Auth, policy, CSRF, tenant scope ve migration kararı Edition'a
aittir. Generator DDL çalıştırmaz. `hydro project new` hâlâ çalıştırılabilir Go
Edition/server üretmediği için temiz scaffold'tan ilk browser CRUD süresi açık
kalmıştır. `HYDRO_TEST_POSTGRES_URL` bu koşuda tanımlı değildi; bu değişiklik
SQL/schema davranışını değiştirmedi ve mevcut SQLite/PostgreSQL resource
contract'leri korunur.
