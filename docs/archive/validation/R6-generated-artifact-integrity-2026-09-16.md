# R6 generated artifact integrity validation

Generated collections now record SHA-256 checksums for every emitted artifact
in `generated/generator.lock`. Compatibility checks verify these checksums after
the regular-file and symlink checks, so editing a generated client, schema,
OpenAPI document, or migration is detected before project validation succeeds.

Validation:

```powershell
go test ./cmd/hydro -run 'TestProjectCheckValidatesGeneratedResourcesWhenPresent'
go test ./internal/generator -run 'TestCheckCompatibility'
```

Existing lockfiles without an artifact map remain accepted to preserve the
documented compatibility path; regenerating them upgrades the lock format.
