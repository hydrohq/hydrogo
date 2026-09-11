# R4 legacy customer upgrade validation

**Date:** 2026-09-08  
**Status:** Succeeded

`tests/contract/testdata/legacy_customer.json` represents a realistic customer snapshot created at migration level `202609080008`. The contract test first installs this legacy schema and writes the page, revision, ready media, and user hash; it then applies the full migration set and verifies the checksum contract via `CheckCompatibility`.

During the upgrade, migration `202609080014_published_projection_backfill` moves pages that were published before projection columns were added to the new public projection. The test proves that the old page can be read via the public route, and that the media storage key/status values and password hash have not changed.

Executed checks:

```text
go test ./tests/contract -run Legacy -count=1
go test ./...
go vet ./...
```

This fixture runs on SQLite; PostgreSQL migration and portable round-trip contracts are executed in separate contract tests.
