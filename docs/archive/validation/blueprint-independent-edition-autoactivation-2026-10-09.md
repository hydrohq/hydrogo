# Blueprint activation in a second independent Edition — 2026-10-09

## Scope and result

A disposable copy of `tests/external-edition` was used to verify that a newly
generated Blueprint resource joins an Edition whose API and Admin composition
already call the aggregate helpers once. The updated CLI command was:

```sh
hydro resource create --name activity_log --admin-label 'Activity log' \
  --field title:text --field details:text --out .
```

The generated `admin_registry.json`, `admin_registration.go`, and
`route_registration.go` all discovered `activity_log`; neither the Admin nor
API aggregate composition call needed another resource ID or route entry.
`hydro project check --config config.yaml`, `hydro resource check --root .`,
the full nested Edition `go test ./... -count=1 -p 1 -timeout=5m`, `go vet
./...`, and the Edition importguard passed.

The CLI flags path previously left `title_field` empty even when the resource
had a `title` field, making the row title plain text instead of its record
link. `resource create` now selects a string/text field named `title`, or the
first string/text field in the entered order when no such name exists. The
Admin registry remains deterministic because the choice is written explicitly
to the generated spec before fields are sorted. CLI regression tests cover the
named-title and fallback cases; `go test ./cmd/hydro -count=1`,
`go vet ./cmd/hydro`, and `go test ./internal/generator -count=1` passed.

For runtime verification, only the disposable Edition test server was given an
explicit call to apply the reviewed `activity_log` migration. Its existing
aggregate API/Admin composition remained unchanged. Chromium opened the new
collection from generated navigation, submitted a record, and read it back
through `/api/v1/activity_log/{id}`: **1/1 passed on SQLite** and **1/1 on
PostgreSQL 18**. PostgreSQL used a `NOSUPERUSER NOBYPASSRLS` application role
that owned the disposable database.

## Limits

This repeats activation and first CRUD in an already composed independent
Edition; it is not the full clean-scaffold-to-runtime journey. The fixture
requires an Edition-owned, reviewed migration application step. Policy,
authentication, tenant behavior, and CSRF remain Edition-owned. The disposable
PostgreSQL server was removed after the run; the acceptance copy and SQLite
database remain under `/tmp`, outside the repository.
