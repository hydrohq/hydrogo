# Dynamic Collections route role dispatch — 2026-10-09

## Scope

ADR-0287 defines Dynamic Collections as an opt-in, admin-only schema manager.
No distinct customer-configurable permission is declared for that surface.
This verification improves the existing role-denial test so it executes each
registered route through the actual protected router.

## Evidence

- Every registered route is dispatched using its declared HTTP method and a
  concrete `/admin/collections/...` or `/api/v1/collections/...` path.
- Active Editor and Viewer identities are authenticated by the fixture and
  rejected by the route authorization with HTTP 403.
- `TestCollectionMutationRoutesEnforceCSRFAndAdminRole` continues to derive
  all nine registered POST/PUT/DELETE mutations from the route table and
  rejects invalid CSRF, Editor, and Viewer requests.
- Targeted route/CSRF tests passed:
  `go test ./internal/collections/transport -run 'TestCollectionRoutesRequireAdmin|TestCollectionMutationRoutesEnforceCSRFAndAdminRole' -count=1 -p 1 -timeout=4m`.
- Full module tests passed:
  `go test ./internal/collections/... -count=1 -p 1 -timeout=5m`.
- `go vet ./internal/collections/transport` and `git diff --check` passed.

## Live browser role boundary

The clean `--with-collections` Edition Chromium journey now keeps a real
generated collection present while authenticating Editor and Viewer sessions.
Both roles can use the permitted Pages shell but do not see the Collections
sidebar or search command. Direct schema/record routes, schema-builder asset,
and Collections API reads return `403`. With valid CSRF, both roles are also
denied schema creation, record creation, and schema deletion through the HTML
and API routes. An Admin reads the schema and 51 existing records afterward,
confirming no role-denied record was stored and the schema was not deleted.
Afterward, Admin deletes the schema as normal cleanup.

`npx playwright test custom/dynamic-collections.spec.ts --project=chromium
--workers=1 --reporter=line --trace=off` passed **1/1** on SQLite and passed
**1/1** on PostgreSQL 18.6 with a `NOSUPERUSER NOBYPASSRLS` application role.
The PostgreSQL database and DSN were supplied through
`HYDRO_E2E_DATABASE_DIALECT=postgres` and `HYDRO_E2E_DATABASE_DSN`; the E2E
fixture creates the PostgreSQL Edition config before migration and serving.
Both runs use the local `--with-collections` fixture with seeded
Admin/Editor/Viewer identities.

## Limits

These browser runs certify the exercised role boundary on local SQLite and
PostgreSQL Edition fixtures. They do not certify customer Edition
deployment/configuration, nor do they complete the full role × route ×
mutation × CSRF matrix.
