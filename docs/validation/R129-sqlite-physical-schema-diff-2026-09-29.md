# R129 — SQLite physical schema comparison

`hydro db diff` compares the SQLite live catalog to an in-memory database
built from the selected migration bundle. It reports missing objects, SQL
definition mismatches, and unexpected tables, indexes, views, or triggers.
The configured database is read-only; the test verifies a missing migration
ledger is not created by inspection.

Verification:

```text
go test ./internal/database ./cmd/hydro
PASS
```

No production migration metadata is required because the reference schema
comes from the binary's immutable migration bundle. The one-shot command adds
temporary memory and CPU proportional to the selected schema and SQLite
catalog, and no steady-state resource cost. PostgreSQL physical catalog
comparison and the MySQL adapter remain open; this report makes no drift
claim for them.
