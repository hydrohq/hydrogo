# HydroGo uyumluluk matrisi

Bu tablo mevcut R3/R4 sözleşmelerinin kanıtlanmış sınırını gösterir. “Uyumlu” ifadesi ilgili contract testleri ve migration checksum kontrolüyle doğrulanmış davranışı anlatır; desteklenmeyen kombinasyonlar satış taahhüdü değildir.

| Alan | Epoch / kimlik | Eski → yeni deploy | Yeni → eski deploy | Kanıt |
|---|---|---|---|---|
| Core runtime/config | config schema mevcut sürüm; bilinmeyen alan deny | Yeni binary eski config'i okur; yeni zorunlu alan yoksa default doğrulanır | Eski binary yalnız kendi bilmediği route/module yoksa çalışır | config/router testleri |
| Module manifest | `extension.APIStatus`, edition import boundary | Yeni edition eski Core slotuna yalnız public extension sözleşmesiyle bağlanır | Eski edition yeni News/Site modülü yüklemez | Client/News importguard CI |
| Schema/migration | `schema_migrations` SHA-256; 16 migration | Yeni binary önce migration uygular, sonra checksum compatibility kontrol eder | Eski binary yeni alanları kullanmadan mevcut page verisini okuyup güncelleyebilir | migration restart/checksum + rollback compatibility contract |
| Pages/content | revision + optimistic `version`; published projection | Yeni revision eski published projection ile birlikte okunabilir | Eski binary yeni draft revision'ı görmez, published alanları korur | Pages SQLite/PostgreSQL contract |
| Site/SEO/forms | Site Edition migration 009–011; form retention | Yeni route'lar eski binary'de yoksa trafik almaz; publish projection korunur | Eski binary yeni navigation/form tablolarına yazmaz | Site contract + runbook |
| News/events | revision + `schedule_generation`; event id action/version içerir | Yeni scheduler eski draft'ı generation kontrolüyle güvenle atlar | Eski binary scheduled News tablosunu kullanmaz; News route'u açılmadıysa etkilenmez | News schedule/worker contract |
| Jobs | migration 013; lease token + expiry | Yeni worker eski queued row'ları lease ile devralabilir | Eski in-process worker SQL job tablosunu görmez | SQL job store SQLite contract |
| Assets/media | immutable storage key, pending → ready | Yeni binary eski ready asset'leri private/public metadata ile sunar | Eski binary retained asset'i silmez | media upload/backup contract |

## Açık doğrulamalar

- `tests/contract/testdata/legacy_customer.json` ile migration 008 seviyesinden tam şemaya yükseltme; yayınlanmış sayfa projection backfill'i, medya ve auth hash korunumu doğrulandı.
- PostgreSQL SQL job lease için servis matrisi henüz çalıştırılmadı.
- PostgreSQL online DDL lock ölçümü, rollback sonrası retained asset ve gerçek blue/green trafik tatbikatı S06/S07 kapılarında açık; SQLite expand/read ve iki dialect eski binary rollback okuma-yazma contract'ı doğrulandı.
