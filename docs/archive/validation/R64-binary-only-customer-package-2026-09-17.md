# R64 — Binary-only customer package (2026-09-17)

## Verification

- Produced customer package using `scripts/build-customer-runtime.ps1` with sample signed license.
- Package strictly contained runtime, license, lock/config, themes, and distribution manifest.
- Verified that no `.go`, `go.mod`, `go.sum`, `internal`, `modules`, `cmd`, `.git`, or private keys were present in the package.
- Removed source `migrations/*.go` copy from Docker final image stage; migration behavior runs via binary embed.
- Builder behaves fail-closed without `HYDRO_TRUSTED_PUBLIC_KEYS` and `HYDRO_CUSTOMER_LICENSE`.

## Note

This test prevents source leaks and improper packaging; it does not eliminate reverse engineering possibilities by authorized customers against running binaries.
