# R5 SDK v1 candidate surface validation

**Date:** 2026-09-08  
**Roadmap:** R5 SDK v1 candidate  
**Status:** Candidate surface pinned; official v1 certification and deprecation schedule remain open.

The small surface actually used by the Pages and News modules and two external Client Editions is pinned under `extension/v1`: `Theme`, `Module`, `ModuleRoute`, typed view data, and API version constant. The experimental v0 API in the `extension` root is preserved; the v1 candidate does not depend on internal packages.

```text
go test ./extension/v1 ./...
go vet ./...
```

For the official v1, the generator fixture, PostgreSQL/SQLite certification, release signature, and deprecation/support schedule must be completed in R5 gates.
