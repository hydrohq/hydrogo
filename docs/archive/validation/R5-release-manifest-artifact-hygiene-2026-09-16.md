# R5 — Release manifest artifact hygiene

The runtime distribution chain deduplicates artifact paths during signed manifest verification and requires each entry to provide safe relative paths, non-negative sizes, and valid SHA-256 metadata. Duplicate paths, absolute paths, traversal, or corrupt checksums are not accepted.

Windows verification:

```text
go test ./internal/release ./internal/sdkdist ./cmd/hydro
PASS
```

WSL Ubuntu (`testuser`) verification:

```text
go test ./internal/release ./internal/sdkdist ./cmd/hydro
exit code 0
```

This check verifies signatures; trusted key management and external release pipelines remain production gates.
