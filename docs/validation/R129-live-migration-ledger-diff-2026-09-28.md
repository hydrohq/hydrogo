# R129 — Live migration ledger diff

`hydro db diff` is a read-only one-shot inspection using the configured
database and active module set. It reports unapplied embedded migrations as
JSON or concise CLI output. `database.PendingMigrations` rejects checksum
mismatches and unknown applied migration IDs.

The SQLite command integration test confirms a fresh database reports pending
migrations and does not create `schema_migrations`. The command uses the
existing SQLite/PostgreSQL migration ledger contract and adds no schema,
worker, connection pool, cache, or persistent file.

Verification:

```text
go test ./...
PASS
```

This implementation does not inspect physical tables or columns, so issue
#129 remains open pending a certified cross-dialect catalog contract.
