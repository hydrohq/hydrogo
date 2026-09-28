# R129 — PostgreSQL physical schema diff

`hydro db diff` now inspects PostgreSQL tables, views, columns, indexes, and
primary, unique, check, and foreign-key constraints against the selected
migration bundle. It builds that reference in a random scratch schema on the
existing connection, then restores the configured `search_path` and drops the
scratch schema. The configured role therefore needs `CREATE` and `DROP SCHEMA`
privileges. Application rows and objects in the active schema are not modified.

The PostgreSQL integration tests run the complete embedded migration bundle,
confirm it matches the migrated test database except for deliberate drift, and
exercise unexpected table/column/index/constraint detection, a mismatched
column type, and missing index/foreign-key detection. The tests also confirm
that no scratch schema remains.

Verification on PostgreSQL 18.6:

```text
HYDRO_TEST_POSTGRES_URL=postgres://... go test -p 1 ./internal/database -run 'TestPostgresSchemaDiff' -count=1 -v
PASS
```

SQLite remains covered by `TestSQLiteSchemaDiffReportsPhysicalDriftWithoutMutatingLiveDatabase`
and `TestSQLiteSchemaDiffFindsDefinitionChanges`. MySQL catalog inspection is
not claimed because MySQL is not yet a certified adapter (ADR-0006).
