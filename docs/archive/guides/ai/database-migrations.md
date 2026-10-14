# Database migrations: adding one correctly, dialect pitfalls (AI guide)

Audience: an agent adding a new migration or writing dialect-aware SQL in a
repository/store file. The second half of this guide documents a real bug
class found and fixed across 20+ files in this codebase — read it before
writing any hand-rolled SQL query, not just when adding a migration.

## Anatomy of a migration

`migrations/<id>/`:
```
migration.yaml     # metadata: module, class, transactional, checksums, compatibility notes
sqlite.sql
postgres.sql
mysql.sql
```
All three dialect SQL files are required even if the schema is
conceptually identical — syntax differs enough (see table below) that
there's rarely a dialect-neutral SQL that works unmodified across all
three.

`migration.yaml` carries a **SHA-256 checksum per dialect SQL file**
(`checksum_sha256.sqlite/.postgres/.mysql`). `tools/migrationguard`
validates these match the actual file content and is a **mandatory CI
gate** (per this repo's `AGENTS.md` — "Automated CI guards such as
moduleguard, migrationguard, and importguard are always mandatory and
cannot be skipped"). After writing or editing a dialect SQL file:
```sh
sha256sum migrations/<id>/mysql.sql   # or shasum -a 256 on macOS, Get-FileHash on Windows
```
Put the resulting hex digest into `migration.yaml`. Run
`go run ./tools/migrationguard -root migrations` to confirm before
considering the migration done.

Migrations are applied in directory-name order and are **immutable once
applied** (`AGENTS.md`: "create a new ADR, mark the previous decision as
Superseded" is the pattern for architecture docs; the same immutability
principle applies here — don't edit an already-applied migration's SQL,
write a new one).

## Running migrations

```sh
hydro migrate --config config.yaml
# or, in a Core checkout dev loop:
hydro migrate --database-dsn <dsn>
```
Idempotent — safe to re-run; already-applied migrations are skipped via
the ledger. `hydro run` (local dev) already runs this automatically on
every start against its dev DSN.

## The dialect-SQL bug class (read this before hand-writing any query)

A systemic bug was found this session across ~20 files: code that builds
SQL as a Go string literal defaulted to **Postgres** placeholder syntax
(`$1, $2, ...`), added a special case for **SQLite** (`?`), and simply
**never added a MySQL branch** — MySQL silently fell through to the
Postgres literal and crashed at runtime with errors like `Error 1327:
Undeclared variable: $2`. This was not caught by `go build`/`go vet`/unit
tests because it's a runtime-dialect issue, not a compile-time one, and
the SQLite-only test suite never exercised the MySQL path.

**If you write a query with inline dialect branching, it needs a branch
for all three dialects actually in use, not two.** The pattern to use
(from `internal/auth/sqlstore/user_store.go`'s already-correct `bind()`
helper):
```go
func (s *Store) bind(position int) string {
    if s.dialect == database.Postgres {
        return fmt.Sprintf("$%d", position)
    }
    return "?"   // SQLite AND MySQL both use plain ? — this is the part that was repeatedly missed
}
```
The bug was always the inverse: defaulting to `$N` and special-casing only
SQLite, instead of defaulting to `?` and special-casing only Postgres (or
explicitly branching all three). Grep for the bad pattern before trusting
existing code in an unfamiliar file:
```sh
grep -rn '\$1' --include=*.go . | grep -v _test.go
```
then check whether the file has a `database.MySQL` branch at all.

### Other dialect-specific syntax differences that bit us

- **Upsert**: SQLite/Postgres use `ON CONFLICT(col) DO UPDATE SET ...`;
  MySQL needs `ON DUPLICATE KEY UPDATE col = VALUES(col)`. Before writing
  the MySQL branch, confirm the conflict column actually has a
  `PRIMARY KEY`/`UNIQUE` constraint in `mysql.sql` — `ON DUPLICATE KEY
  UPDATE` silently does nothing useful without one.
- **Numbered placeholders**: SQLite supports `?1`, `?2` (explicit
  positional reuse, useful when the same value appears twice in one
  query); MySQL does **not** support this syntax at all, only plain `?`
  repeated once per occurrence. A helper that does
  `if sqlite { "?1" } else { "$1" }` silently sends MySQL invalid numbered
  `?N` syntax too if MySQL falls into the "else" branch — check for this
  specifically, it's an easy miss.
- **Timestamps**: the MySQL driver (`internal/database/driver_mysql.go`)
  sets `ParseTime=true, Loc=time.UTC` on the DSN — this is required for
  MySQL `DATETIME`/`TIMESTAMP` columns to round-trip as Go `time.Time`
  values instead of raw byte strings. Code that formats a time value as an
  RFC3339Nano string specifically for SQLite (which stores timestamps as
  `TEXT`) must NOT reuse that same formatted string for MySQL — MySQL
  should receive a native `time.Time` (same as the Postgres path), not a
  string. A `?`/`?`/`?` three-way branch that reuses the SQLite string
  formatting for MySQL too is a second, subtler version of the same bug
  class.

### Verifying a dialect fix

Don't trust that a fix compiles and passes SQLite-backed tests as proof it
works for MySQL — the entire bug class above did exactly that. If a real
MySQL instance is reachable, run the specific query path against it
directly. If not, at minimum trace through the exact SQL string and args
that would be sent for each dialect by hand, and check `migrations/<id>/mysql.sql`
for the actual constraint/column types involved (nullable vs not, type
affinity) rather than assuming they match SQLite's.
