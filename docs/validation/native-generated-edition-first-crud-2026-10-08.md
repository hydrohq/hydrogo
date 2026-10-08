# Native generated Edition first CRUD and PostgreSQL browser path

**Date:** 2026-10-08  
**Scope:** The pre-composed `examples/project-edition` native generated Admin
runtime; this does not measure a clean scaffold or human composition work.

## First CRUD timing

The same Chromium test was run three times from the E2E workspace. Each
invocation creates a fresh temporary SQLite database, runs the Project Edition
migrations, starts its generated API/Admin server, then creates, lists, reads,
filters, edits, and deletes Blueprint-generated records in Chromium. It also
checks responsive and accessibility behavior. Go build caches were warm.

| Run | End-to-end command time | Browser test time | Result |
| --- | ---: | ---: | --- |
| 1 | 13.730 s | 13.1 s | 1/1 passed |
| 2 | 12.339 s | 11.8 s | 1/1 passed |
| 3 | 12.588 s | 12.0 s | 1/1 passed |
| Median | **12.588 s** | — | 3/3 passed |

Command:

```sh
cd e2e
npm test -- --project=chromium --workers=1 \
  custom/generated-admin-crud.spec.ts
```

Timing wraps the complete npm/Playwright command and includes Chromium startup,
Go server startup/compilation, migration, and browser CRUD. It excludes the
initial `hydro project new`, Blueprint authoring, new policy/composition edits,
and developer review. The existing Edition already wires the generated
aggregate API route, Admin registry, authorization callback, and migrations.
This is therefore a repeatable native runtime smoke-to-CRUD measure, not the
clean Edition first-CRUD time or a PayloadCMS comparison.

## PostgreSQL Chromium run

An isolated PostgreSQL 18 container was used. The same E2E fixture ran the
Edition migration against PostgreSQL, started the generated Edition server,
and completed its Chromium Admin CRUD workflow: **1/1 passed**, browser test
time 11.7 s. The temporary database container was removed after the run.

```sh
HYDRO_E2E_DATABASE_DIALECT=postgres \
HYDRO_E2E_DATABASE_DSN='postgres://hydro_e2e:hydro_e2e@127.0.0.1:55432/hydrogo_e2e?sslmode=disable' \
npm test -- --project=chromium --workers=1 \
  custom/generated-admin-crud.spec.ts
```

This verifies that the real browser route and generated Admin CRUD work with
the PostgreSQL adapter. Other media/relation/editorial browser fixtures remain
separate; their persistence contracts are covered by their existing DB tests.

## Generated rich-text resource on the real Edition migration

The Project Edition now includes an `article` Blueprint with required `title`
and `body:text` (`ui.widget: richtext`). Its generated migration creates the
`articles` table with the portable `TEXT` body column and includes SQLite and
PostgreSQL checksums, additive compatibility metadata, and rollback notes. The
Edition migration command explicitly applies this generated proposal, and its
aggregate API/Admin registration helpers discover the resource from the
Blueprint registry.

The native Chromium flow requests the embedded Quill script, creates an article
with bold formatting, reads its serialized Delta through the generated API,
reloads and edits the record with italic formatting, and deletes it. The same
test passed **1/1 on SQLite** and **1/1 on PostgreSQL 18** using the real
Project Edition migration and API, with a `NOSUPERUSER NOBYPASSRLS` PostgreSQL
role. This closes the gap between synthetic rich-text fixture storage and the
native generated Edition path; the separate fixture remains useful for its
workflow, relation/media and stale-write scenarios.

The latest browser test times were 16.7 s total on SQLite and 16.4 s on
PostgreSQL, including Playwright/Chromium startup, generated Edition migration,
server startup, and the existing CRUD plus rich-text journeys.

The rich-text create and edit pages also pass the WCAG 2.1 A/AA axe scan. The
create page has no horizontal overflow at a 390px viewport. These checks passed
in the SQLite Chromium run (16.6 s total) and the PostgreSQL Chromium run
(16.6 s total).

The real generated API also rejects an empty required `body` with a field-level
`422 validation_failed` response before the browser creates a record. This
negative case passed in both SQLite and PostgreSQL Chromium runs.

The rich-text create page also passes the axe WCAG 2.1 A/AA scan across the
HydroGo, shadcn, Nova, and Aurora theme families in light and dark mode (8/8).
This theme matrix passed as part of the SQLite and PostgreSQL browser runs; the
full acceptance command took 20.1 s and 21.7 s respectively.

The create/edit journey now verifies keyboard-only rich-text formatting
(`Control+B`/`Control+I`) and submits both forms by focusing the submit button
and pressing Enter. The generated API round-trip still confirms the expected
bold/italic Delta after persistence. This focused Chromium run passed 1/1 on
SQLite and 1/1 on PostgreSQL 18 with a `NOSUPERUSER NOBYPASSRLS` role (18.2 s
and 18.4 s browser test time).

On 2026-10-08, the SQLite journey was repeated three times while auditing the
generated edit form: all three passed (19.2 s, 19.3 s, and 29.5 s browser test
time). One earlier attempt had reported low contrast on the edit form before
these repeats. The browser test now asserts that full-page navigation retains
the HydroGo light theme and that the body label/title use the expected dark
foreground colors before running axe. This guards the exact visual state that
was reported; no repeatable product CSS defect was found.

Commands:

```sh
cd e2e
TMPDIR=/var/tmp npm test -- --project=chromium --workers=1 \
  custom/generated-admin-crud.spec.ts
HYDRO_E2E_DATABASE_DIALECT=postgres \
HYDRO_E2E_DATABASE_DSN='postgres://<edition-role>:<password>@127.0.0.1:<port>/<empty-database>?sslmode=disable' \
TMPDIR=/var/tmp npm test -- --project=chromium --workers=1 \
  custom/generated-admin-crud.spec.ts
```

The PostgreSQL run uses an empty disposable database because the example's
reviewed migrations are intentionally not idempotent. The temporary PostgreSQL
container was removed after verification.
