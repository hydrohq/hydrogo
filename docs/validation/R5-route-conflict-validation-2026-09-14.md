# R5 route conflict validation — 2026-09-14

Related decision: [ADR-0089](../architecture/0089-route-conflict-validation.md)

## Scope

The composition root validates every route pattern before handing routes to
`http.ServeMux`. The check includes compatibility aliases generated for legacy
`/api/...` paths, so an accidental collision fails startup with a stable error
instead of a `ServeMux` panic.

## Evidence

- `internal/router/router_test.go` covers duplicate literal patterns and
  duplicate generated API-v1 aliases.
- `cmd/hydro/main_test.go` covers the optional-module route assembly contract:
  inactive `site` and `media` modules do not add their admin routes, while the
  `site` route appears when enabled. The media module remains the sole owner of
  `GET /admin/media`.
- `scripts/verify-local.ps1` runs the router and composition-root tests as part
  of the full local verification hook.

## Result

The local verification hook completed successfully for commit `9f18d96`.
Production reverse-proxy routing, blue/green traffic switching, and customer
deployment topology remain external gates and are not implied by this test.
