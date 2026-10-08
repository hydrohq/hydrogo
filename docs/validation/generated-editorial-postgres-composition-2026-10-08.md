# Generated editorial PostgreSQL composition validation

**Date:** 2026-10-08  
**Scope:** Generated Admin HTTP handler and Edition-owned editorial callbacks.

## Result

`TestEditorialAdminPostgresComposition` runs the same composed handler journey
as the SQLite composition test against an isolated PostgreSQL schema, using a
non-superuser role with `NOBYPASSRLS`. It saves a working draft, closes and
reopens the database connection, and confirms the draft survived while the
published record stayed unchanged. It then verifies stale draft/workflow/locale
writes return `409`, transitions draft → review → published, atomically promotes
the draft and removes it, restores a prior revision, and saves/copies a
localized value without changing the default locale.

The test creates only synthetic tables in its isolated schema and drops that
schema on cleanup. It does not execute product migrations.

The generated rich-text Playwright fixture also supports opt-in PostgreSQL
storage for story drafts and locales. The six real Chromium scenarios passed
6/6 with that storage enabled, including localized publish, review/publish, and
draft discard. Article rich-text/relation/media records in that fixture remain
in-memory; this is not the production Project Edition migration/browser path.
The default in-memory Chromium suite passed 6/6 as well.

SQLite and PostgreSQL full repository Go suites both passed after these tests
were added. The temporary PostgreSQL container was stopped, and the fixture's
shutdown removed its isolated schema.

## Commands

```sh
TMPDIR=/var/tmp/hydro-go-tests go test ./... -count=1 -p 1 -timeout=8m
HYDRO_TEST_POSTGRES_URL="$TEST_DATABASE_URL" TMPDIR=/var/tmp/hydro-go-tests \
  go test ./... -count=1 -p 1 -timeout=8m
cd e2e && HYDRO_TEST_POSTGRES_URL="$TEST_DATABASE_URL" TMPDIR=/var/tmp \
  npm test -- --project=chromium --workers=1 \
  --grep "generated rich-text Edition Admin"
```

The PostgreSQL run used a disposable PostgreSQL 18 container and an isolated
`NOSUPERUSER NOBYPASSRLS` contract role. The container was removed after the
run.
