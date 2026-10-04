# R5 SDK proje namespace doğrulaması — 2026-09-16

**Durum:** Başarılı (lokal/WSL)  
**Kapsam:** Public SDK ergonomisi için `hydro project new` namespace’i

## Değişiklik

`hydro project new` eklendi. Mevcut `hydro new` yazımı geriye dönük uyumluluk için korunuyor. İki komut aynı güvenli project generator’ını kullanıyor; yeni bir Core veya runtime kopyası üretmiyor.

## WSL kanıtı

Ortam: Ubuntu WSL2, Go 1.27.0 linux/amd64, kullanıcı `testuser`.

```bash
cd /path/to/project
go test ./cmd/hydro
go run ./cmd/hydro project new sdk-demo --preset compact-site --no-demo --root /tmp/hydro-sdk-alias-check
test -f /tmp/hydro-sdk-alias-check/sdk-demo/hydrogo.lock
```

Beklenen/gerçek sonuç:

```text
ok github.com/ardaozyildirim/hydrogo/cmd/hydro
project-created path=/tmp/hydro-sdk-alias-check/sdk-demo preset=compact-site core=licensed-runtime
sdk-alias-ok
```

Değişiklik sonrasında WSL’de ayrıca `go test ./...` çalıştırıldı; tüm Go
paketleri başarılı oldu.

## Sınırlar

Bu değişiklik yalnızca CLI namespace ergonomisini sağlar. Private runtime indirme, `hydro login`, artifact broker, cross-platform installer ve imzalı release henüz uygulanmış değildir; bunlar [HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md](../HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md) içindeki sonraki fazlardır.
