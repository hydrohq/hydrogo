# R131 — Edition permission catalog and generated RBAC — 2026-09-23

## Scope

Let an Edition register generated resource permission names in the existing
Admin Studio user override surface, without introducing role grants or a
parallel auth implementation. Catalog membership is bounded and explicit;
unknown names are rejected at write time and by the catalog-aware policy API.
Stored grants absent from the active catalog are filtered from resolved
sessions, while their database rows remain unchanged.

The generator emits a deterministic `<Resource>Permissions()` function and an
auth-enabled runtime wiring checklist. The admin user UI displays only the
active catalog, permission identifiers are escaped, and permission updates
have a bounded 256 KiB form body. There is no schema or migration change.

## Verification

- `go test ./... -p 1 -timeout=5m` — passed, including auth, SQLite auth store,
  generator, UI, all generated fixture packages, and contract tests.
- With Redis environment variables unset,
  `go test -tags hydro_no_redis ./internal/auth ./internal/auth/sqlstore ./internal/ui ./internal/generator ./cmd/hydro -count=1` — passed.
- `go vet ./...` — passed.
- `scripts/verify-edition-servers.ps1` — Project and Article SQLite migration,
  health, POST, and GET smoke tests passed.
- `scripts/verify-edition-footprint.ps1` — full matrix passed; the combined
  Compact Workspace measured 21,355,520 bytes and 152 non-standard packages
  on Windows amd64 / Go 1.27.0.
- `hydro resource check --root` accepted Project, Article and Relation Edition
  fixtures at `contract=v0`, verifying 38 generated files apiece.
- `git diff --check` — passed.
- After increasing the bound from 96 KiB to the proven worst-case-safe 256 KiB,
  `go test ./internal/auth -run 'TestAdminPermissionEndpointBoundsFormBody' -count=1` — passed.

## Limits and compatibility

Generated permissions remain deny-by-default for editor/viewer roles. The
Edition must compose all active `<Resource>Permissions()` lists into
`auth.NewPermissionCatalog`, set it on `AuthService` before serving, and use
`catalog.Allows` or `AuthService.Can` in its resource policy. This report does
not certify a production Edition's role matrix, live PostgreSQL RLS, or
customer-specific authorization semantics. Existing Core permissions and
their role defaults are unchanged.
