# R5 — SDK runtime cache lifecycle

## Scope

`ListCache` lists only regular artifact files under `root/<version>`. `PruneCache` takes versions to preserve; it produces a dry-run plan or removes non-preserved files. Symlinks and non-regular file entries are rejected.

## Verification

```text
go test ./internal/sdkdist
ok github.com/ardaozyildirim/hydrogo/internal/sdkdist
```

Tests verify that dry-run does not touch files, apply mode removes older versions, retains current versions, and rejects symlinks.

CLI surface is also verified:

```text
hydro runtime cache list --root PATH
hydro runtime cache prune --root PATH --keep v2.0.0 --dry-run
hydro runtime cache prune --root PATH --keep v2.0.0 --confirm
```

Target and full suite also passed on WSL Ubuntu:

```text
go test ./cmd/hydro ./internal/sdkdist
go test ./... -p 1 -timeout=5m
exit code 0
```
