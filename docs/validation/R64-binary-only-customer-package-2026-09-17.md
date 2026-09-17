# R64 — Binary-only customer package (2026-09-17)

## Doğrulama

- `scripts/build-customer-runtime.ps1` ile örnek imzalı lisans kullanılarak müşteri paketi üretildi.
- Paket yalnızca runtime, lisans, lock/config, tema ve dağıtım manifestini içerdi.
- Paket içinde `.go`, `go.mod`, `go.sum`, `internal`, `modules`, `cmd`, `.git` veya private key bulunmadığı kontrol edildi.
- Docker final image aşamasından kaynak `migrations/*.go` kopyası kaldırıldı; migration davranışı binary içindeki embed ile çalışır.
- `HYDRO_TRUSTED_PUBLIC_KEYS` ve `HYDRO_CUSTOMER_LICENSE` olmadan builder fail-closed davranır.

## Not

Bu test kaynak sızıntısını ve yanlış paketlemeyi önler; yetkili müşterinin çalışan binary üzerinde yapabileceği reverse engineering’i ortadan kaldırmaz.
