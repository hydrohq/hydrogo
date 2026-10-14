# R114 — Generated auth route guard

Date: 2026-09-22

## Scope

`auth: true` generated resources now require an Edition-provided request
authentication checker during route registration and reject anonymous requests
with `401 Unauthorized` before policy/transport execution.

## Evidence

- `go test ./internal/generator -run TestLoadBlueprintMapsAuthCompositionFlag -count=1` — PASS.
- Generated auth runtime contains the checker parameter, nil registration guard,
  and bounded `http.StatusUnauthorized` response.
- Non-auth Article and Project generated Edition tests remain green and keep the
  source-compatible registration signature.
- ADR index drift and generator formatting checks pass.

## Boundary result

The existing Identity/Session/RBAC implementation remains the source of truth;
Core only enforces that an Edition has explicitly connected it. No credentials,
sessions, grants, or provider-specific code are generated.
