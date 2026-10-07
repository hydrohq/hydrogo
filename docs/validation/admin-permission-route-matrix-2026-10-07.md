# Admin izinlerinin modül route doğrulaması

**Tarih:** 2026-10-07  
**Durum:** Üye/site yönetimi ile sayfa/haber içerik route'ları doğrulandı  
**Kapsam:** `admin.members.manage`, `admin.site.manage`, `admin.content.read` ve `admin.content.write`.

`GET /admin/members` gerçek kayıtlı route handler'ı üzerinden şu durumlarda
denetlendi: editor varsayılanı, editor explicit allow, viewer explicit allow,
admin explicit deny ve inactive user explicit allow. İzin verilen iki durumda
route 200 döndürdü; kalan üç durumda 403 döndürdü.

`POST /admin/site/settings` SQLite transactor ve recording store ile denetlendi:
editor varsayılanı/deny, viewer allow, admin deny, inactive allow ve admin
varsayılanı. İzin verilen POST'lar 204 döndürüp bir ayarı yazdı; reddedilenler
403 döndürüp hiç yazmadı.

Sayfa/haber mutasyonları `admin.content.write`; içerik detay API'si ve Admin
liste ekranları `admin.content.read` matrisiyle çağrıldı. Haber taxonomy
kataloğu create/delete formu içerdiği için `admin.content.write` kapsamındadır.
Editor, viewer, allow/deny
override ve inactive user kombinasyonlarında izinli isteklerin permission
kontrolünü geçtiği, reddedilenlerin handler başında 403 aldığı doğrulandı.

`GET /api/pages/{id}`, sayfa preview, `GET /api/news/{id}` ve Admin
list/detail ekranları `admin.content.read` ile korunuyor. Admin detaylarında
write izni yoksa sayfa güvenli preview iframe'i, haber sanitize edilmiş rich
text görünümü ve salt okunur metadata ile render ediliyor; mutation formu yok.
API detail handler'ları explicit
read deny durumunda draft JSON döndürebiliyordu. Ayrıca salt okunur Admin
liste ekranları gereksiz yere write izni istiyordu. Testler varsayılan reader
erişimini, explicit deny ve inactive user için 403'ü doğruluyor.

## Komut

```sh
go test ./modules/pages/transport ./modules/news/transport ./modules/members/transport -count=1
go test ./modules/site/transport -run '^TestAdminSiteSettingUsesSiteManageOverrides$' -count=1
go test ./tests/contract -run '^TestCriticalUserJourneysContract$' -count=1
```

Sonuç: ilgili route paketleri ve kritik kullanıcı yolculuğu contract testi
geçti. Testler route-level role defaults, explicit
allow/deny, inactive-user deny, content read/write ayrımı ve site mutation yan
etkisini kapsar. Her izinli content route'u handler yetkisini geçer; tam
authorization + persistence uçtan uca matrisi ve diğer permission consumers
Faz 4'te açık kalır.

Viewer'ın sayfa listesine erişmesi, import/düzenleme aksiyonlarını görmemesi ve
`content.read` ile salt-okunur detay ve API'yi görebilmesi için Chromium rol
testi eklendi.
Bu restricted sandbox'ta test setup `execFileSync("go", ...)` adımında
`spawnSync go EPERM` ile durdu; bu browser testi burada geçiş kanıtı değildir.
Read-only sayfa/haber renderer'larının HTTP/component testleri ilgili Go
transport paketlerinde geçti.

`go test ./... -count=1 -p 1 -timeout=5m` de denendi. Bu çalışma alanının
restricted network sandbox'ı `httptest.NewServer` kullanan testlerde loopback
port açmayı `operation not permitted` ile engelledi; bu nedenle repository'nin
tam Go suite sonucu burada doğrulanamadı. Tam suite'in yeniden çalıştırılması
network/socket izinli CI ortamında açık kalır.
