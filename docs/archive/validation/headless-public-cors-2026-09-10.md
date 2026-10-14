# Headless public CORS validation — 2026-09-10

Related decision: [ADR-0021](../architecture/0021-headless-browser-session-boundary.md). This record is not evidence of credentialed browser session support; it only validates the published read allowlist behavior.

## Automated evidence

- `go test ./internal/config ./internal/security ./internal/server ./cmd/hydro`
- `go test ./...`

`internal/security/cors_test.go` covers the following boundaries:

- allowlisted origin gets exact `Access-Control-Allow-Origin` and `Vary: Origin` in published read `GET` response such as `/api/v1/news`;
- `Access-Control-Allow-Credentials` is never produced;
- unknown origin, `POST` and `OPTIONS` requests do not receive an allow header;
- wildcard, path, user-info, and non-HTTP(S) config origins are rejected.

## Local seed smoke

The current binary was run on the seeded SQLite demo data with `HYDRO_PUBLIC_API_ORIGINS=http://localhost:3000`.

| Request | Result |
|---|---|
| `GET /api/v1/news?locale=tr`, `Origin: http://localhost:3000` | `200`; two published news; `Access-Control-Allow-Origin: http://localhost:3000`, `Vary: Origin`; no credential header |
| `GET /api/v1/commerce/cart`, same origin | `401 unauthorized`; no CORS allow/credential header |

This boundary does not open CORS access to Admin, auth, private media, webhooks, operations, or mutation endpoints.
