# R5 — Schema/migration compatibility diff

`hydro schema diff` compares migration checksum ledgers between two signed release manifests. Adding new migrations is reported as compatible; deleting an existing migration or altering its checksum fails closed. Manifest signatures and trusted public keys must be verified before comparison occurs.

Windows verification:

```text
go test ./cmd/hydro ./internal/release
PASS
```

WSL Ubuntu verification:

```text
go test ./cmd/hydro ./internal/release
PASS
```

The current command executed was `go test ./cmd/hydro ./internal/release ./internal/sdkdist`, returning exit code 0 under user `testuser` on WSL.

Raw SQL or database dialect details are never exposed in user output.
