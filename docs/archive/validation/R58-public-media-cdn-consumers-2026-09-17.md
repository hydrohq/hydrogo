# R58 — Public media CDN consumers

Date: 2026-09-17

## Scope

The optional `Media.PublicURL` provider is now consumed by the News and
Commerce server-rendered image components, in addition to Pages. When a
provider-backed public URL is unavailable, each component keeps the local
`/media/<key>` URL and does not make a private or failed asset public.

## Evidence

- `go test ./modules/news/transport ./modules/commerce/transport ./cmd/hydro` — PASS
- News and Commerce transport tests cover configured CDN URLs, HTML escaping,
  and safe local fallback when the provider returns an error.
- `cmd/hydro/main.go` passes the configured media service to both handlers only
  within the normal module composition boundary.

## Remaining production gate

This proves application wiring and fallback behavior only. Real CDN IAM,
cache/purge, retention, failover, and origin access policy remain operator
validation items in `docs/OPEN_GATES_AND_OPERATOR_ACTIONS.md`.
