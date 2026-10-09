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

## Limits

These are in-process route tests. They do not prove rendered menu behavior,
browser CRUD, PostgreSQL deployment, or customer Edition configuration.
