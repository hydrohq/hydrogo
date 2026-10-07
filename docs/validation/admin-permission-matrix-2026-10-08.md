# Admin izin ve rol denetimi

**Tarih:** 2026-10-08  
**Sonuç:** Hedefli auth testleri geçti  
**Kapsam:** Edition rol varsayılanları, kayıtlı izinler, kullanıcı yönetimi
mutasyonları ve yöneticinin kendi erişimini değiştirme girişimi.

Doğrulama komutu:

```sh
TMPDIR=/var/tmp/hydro-go-tests go test ./internal/auth ./cmd/hydro -count=1
```

Testler şu davranışları doğrular:

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

İlgili Chromium doğrulama komutu:

```sh
cd e2e
npx playwright test custom/admin-management.spec.ts --project chromium --workers=1 \
  --grep 'editor and viewer sessions' --reporter=line
```

Bu, tek tek davranış kanıtıdır. Her route × rol × CSRF durumu × override
değerini kapsayan çapraz matris henüz üretilmiş değildir; planın Faz 4 kabul
koşulu açık kalır. PostgreSQL auth sözleşme testi önceki doğrulama aşamasında
geçmiştir. Bu aşama çalışma zamanı veya veritabanı davranışını değiştirmez.
