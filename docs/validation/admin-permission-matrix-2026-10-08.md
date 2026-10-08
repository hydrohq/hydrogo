# Admin izin ve rol denetimi

**Tarih:** 2026-10-08  
**Sonuç:** Auth rol × izin × Admin route matrisi ve mutation ret testleri geçti  
**Kapsam:** Edition rol varsayılanları, kayıtlı izinler, kullanıcı yönetimi
mutasyonları ve yöneticinin kendi erişimini değiştirme girişimi.

Doğrulama komutu:

```sh
TMPDIR=/var/tmp/hydro-go-tests go test ./internal/auth ./cmd/hydro -count=1
```

Testler şu davranışları doğrular:

- `/admin/control`, security, operations, users, permissions, site, developer,
  resource usage ve account GET route'ları aktif/inaktif Admin, Editor, Viewer
  için; bilinen her izin override'ının varsayılan/allow/deny etkisiyle çapraz
  test edilir. Site, operations ve users-management route'larının role ek
  koşulları da kontrol edilir.
- Admin/editor/viewer varsayılan izinleri ve etkin kullanıcı izin override'ları.
- Edition kataloğunda olmayan izinlerin yetki vermemesi ve izin yazımından
  filtrelenmesi.
- Editor ve viewer rollerinin rol, izin ve hesap kapatma mutasyonlarından
  reddedilmesi.
- `admin.users.manage` izni açıkça reddedilmiş yöneticinin kullanıcı yönetimi
  mutasyonlarından reddedilmesi.
- Yöneticinin kendi rolünü veya kendi izinlerini değiştirme isteğinin `409`
  olması ve izin deposuna yazma yapılmaması.
- Tek Edition adminiyle rol ve permission form akışının gerçek Chromium
  testinde çalışması; editör deny/allow override ve rol değişiklikleri.
- Editor ve viewer oturumlarının geçerli CSRF tokenıyla başka kullanıcı için
  rol değiştirme, izin override'ı yazma ve hesabı kapatma POST'larından `403`
  alması.
- Editor ve viewer'ın rol, permission, disable, invitation ve password-reset
  user-management mutation endpoint'lerinden `403` alması.
- Editor ve viewer'ın geçerli CSRF tokenıyla IP ban ekleme/kaldırma ve backup
  POST'larından `403` alması; geçerli CSRF admin/user management akışlarının
  kendi izin kurallarına göre çalışması.

İlgili Chromium doğrulama komutu:

```sh
cd e2e
npx playwright test custom/admin-management.spec.ts --project chromium --workers=1 \
  --grep 'editor and viewer sessions' --reporter=line
```

Bu, tek tek davranış kanıtıdır. Her route × rol × CSRF durumu × override
değerini kapsayan çapraz matris Go route-policy testinde üretilmiştir. Browser
testi CSRF tokenlı mutation ve CSRF'siz form retlerini ayrıca doğrular.
PostgreSQL auth sözleşme testi önceki doğrulama aşamasında geçmiştir. Bu aşama
çalışma zamanı veya veritabanı davranışını değiştirmez.

Generated Admin resource policy contract also verifies that a collection
denied by `CanReadResource` is omitted from the resource index and that its
list/create/record routes return `404` without calling the API. A valid-CSRF
generated create whose resource API denies the write returns a user-facing
`403` and never redirects as if it succeeded.

ADR-0361 adds optional Edition-provided `ResourceAccess` capabilities for
basic generated CRUD. Handler tests prove that denied create/update/delete
actions disappear, direct routes return `403` before a mutation reaches the
resource API, and update can be allowed independently from create/delete.
Draft workflow and other editorial mutations keep their dedicated callbacks.
The generated block Chromium journey also verifies read-only list state,
valid-CSRF create/update/delete denials, and unchanged persisted data; the full
generated editorial suite passed 7/7 on SQLite and PostgreSQL 18. Database
behavior and migrations are unchanged.

```sh
TMPDIR=/var/tmp go test ./internal/adminstudio -run 'TestGeneratedAdmin(ResourceReadPolicy|SurfacesResourceAPIDenial|ResourceCapabilities)' -count=1
TMPDIR=/var/tmp go test ./internal/adminstudio -count=1
go test ./extension/v1/adminstudio ./examples/project-edition/custom -count=1
cd tests/external-edition && go test ./... -count=1
```
