# R5 SDK proje doğrulama komutu — 2026-09-16

**Durum:** Başarılı (lokal/WSL)

`hydro project check`, public SDK workflow’una dependency-free proje doğrulama adımı olarak eklendi. Mevcut `hydro validate` mantığını kullanır; veritabanını açmadan config, profil ve modül invariants’larını kontrol eder.

## WSL kanıtı

```bash
cd /mnt/c/dev/cms
go test ./...
go run ./cmd/hydro project check --config /tmp/hydro-project-check/config.yaml --json
```

Gerçek sonuç:

```json
{"valid":true,"profile":"compact","database":"sqlite","modules":["core","pages"],"cache_entries":256,"queue_entries":10000}
```

## Sınır

Bu komut migration uygulamaz, runtime başlatmaz ve private artifact erişimi sağlamaz. Amaç, `hydro dev` veya `hydro migrate` öncesinde hataları erken ve güvenli biçimde göstermektir.
