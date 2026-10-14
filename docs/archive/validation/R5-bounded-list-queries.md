# R5 validation: bounded list queries

## Local evidence

- `go test ./modules/news/repository/sqlstore ./modules/pages/repository/sqlstore`
- The news admin list and public page list fixtures run against the migrated
  SQLite schema and assert exactly one bounded query for each result.

## Open gate

PostgreSQL `EXPLAIN (ANALYZE, BUFFERS)` and customer-sized latency/N+1
measurement still require the supported production-like database and data set.
