# R5 SDK project namespace verification — 2026-09-16

**Status:** Successful (local/WSL)  
**Scope:** `hydro project new` namespace for public SDK ergonomics

## Changes

Added `hydro project new`. The existing `hydro new` syntax is retained for backward compatibility. Both commands use the same secure project generator; they do not produce separate copies of Core or runtime.

## WSL evidence

Environment: Ubuntu WSL2, Go 1.27.0 linux/amd64, user `testuser`.

```bash
cd /path/to/project
go test ./cmd/hydro
go run ./cmd/hydro project new sdk-demo --preset compact-site --no-demo --root /tmp/hydro-sdk-alias-check
test -f /tmp/hydro-sdk-alias-check/sdk-demo/hydrogo.lock
```

Expected/actual result:

```text
ok github.com/ardaozyildirim/hydrogo/cmd/hydro
project-created path=/tmp/hydro-sdk-alias-check/sdk-demo preset=compact-site core=licensed-runtime
sdk-alias-ok
```

Following the change, `go test ./...` was also run on WSL; all Go packages succeeded.

## Scope & Boundaries

This change only provides CLI namespace ergonomics. Private runtime downloading, `hydro login`, artifact broker, cross-platform installer, and signed releases are not yet implemented; these are subsequent phases detailed in [HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md](../HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md).
