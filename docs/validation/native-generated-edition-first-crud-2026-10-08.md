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
