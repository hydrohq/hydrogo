# R5 deterministic generator foundation validation

**Date:** 2026-09-08  
**Roadmap:** R5 / generator and S08 foundation  
**Status:** Spec validation, Pages/News collection fixtures, and compilable CRUD repository/transport generation succeeded.

From a collection spec, `internal/generator` generates a model, CRUD repository, transport route, migration, and contract skeleton, along with a `generator.lock` carrying the spec hash. Output is written only under `generated/`; `custom/` files are left untouched. Secret field names, missing permission/dialect, and destructive migration specs are denied. Runtime automatic DDL or authorization decisions are not generated. The Pages and News fixtures yield the exact same hash and byte output across two generations, and the generated Go package compiles; if the published migration file is manually altered, it is rejected with `ErrPublishedMigration`.

Executed check:

```text
go test ./internal/generator
```

The test validates that the generated file remains byte-for-byte identical on a second run, the custom file hash is preserved, Pages/News fixture hashes, CRUD repository/transport methods, compilability, and migration immutability.
