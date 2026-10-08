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
storage for story drafts, locales, and generated Admin article CRUD. The six
real Chromium scenarios passed 6/6 with that storage enabled, including
formatted body and relation/media round trips, stale edit conflict, localized
publish, review/publish, and draft discard. Article and editorial tables are
synthetic test-fixture schema; this does not execute the production Project
Edition article migration. The default in-memory Chromium suite passed 6/6 too.

The generated rich-text fixture now mounts its article handler on the public
`/api/v1/article` routes as well as passing it to the Admin renderer. This lets
the media journey verify its saved `author_id` and `cover_id` through an actual
browser-visible API request. The picker modal passes axe WCAG 2.1 A/AA, and the
media item can be selected with Enter. The full six-scenario Chromium suite
passed again on SQLite and PostgreSQL 18 (6/6 each, `NOSUPERUSER
NOBYPASSRLS`). This remains synthetic fixture storage, not a native Edition
migration or production media policy certification.

The separate native Project Edition browser run now covers generated article
rich-text create, Delta API readback, edit, and delete using its real generated
SQLite/PostgreSQL migration and API. That path passed 1/1 in both database
modes; see [native generated Edition validation](native-generated-edition-first-crud-2026-10-08.md).

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
