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
