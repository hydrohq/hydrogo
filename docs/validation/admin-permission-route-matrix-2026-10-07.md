# Admin izinlerinin modül route doğrulaması

**Tarih:** 2026-10-07  
**Durum:** İki aktif permission-aware modül route'u doğrulandı  
**Kapsam:** `admin.members.manage` liste route'u ve `admin.site.manage` ayar mutation'ı.

`GET /admin/members` gerçek kayıtlı route handler'ı üzerinden şu durumlarda
denetlendi: editor varsayılanı, editor explicit allow, viewer explicit allow,
admin explicit deny ve inactive user explicit allow. İzin verilen iki durumda
route 200 döndürdü; kalan üç durumda 403 döndürdü.

`POST /admin/site/settings` SQLite transactor ve recording store ile denetlendi:
editor varsayılanı/deny, viewer allow, admin deny, inactive allow ve admin
varsayılanı. İzin verilen POST'lar 204 döndürüp bir ayarı yazdı; reddedilenler
403 döndürüp hiç yazmadı.

## Komut

```sh
go test ./modules/members/transport ./modules/site/transport -count=1
```

Sonuç: iki paket geçti. Testler route-level role default, explicit allow/deny,
inactive-user deny ve mutation yan etkisini kapsar. Diğer izinler, diğer route
ve mutation'lar için tam permission × role matrisi Faz 4'te açık kalır.
