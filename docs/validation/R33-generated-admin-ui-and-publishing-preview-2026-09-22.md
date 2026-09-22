# R33 — Generated admin UI and publishing preview — 2026-09-22

## Scope

This validation covers Blueprint UI metadata, generated Templ CRUD scaffolds,
module-manifest navigation metadata, typed Sidebar components, the optional
Editor.js block editor and the bounded HTMX live-preview boundary.

## Evidence

- `go test ./internal/generator` passed.
- `go test ./internal/ui` passed.
- `go test ./internal/ui/components` passed.
- `go test ./internal/modules` passed.
- Full `go test ./...` passed after this record was added.
- Generated list/form artifacts are deterministic, use typed components, carry
  CSRF through `ui.Form`, and contain no HTML comments.
- Preview requests are bounded to 256 KiB, return `private, no-store`, and do
  not expose renderer error details.

## Limitations

This is a code and contract validation, not browser certification. A Client
Edition must provide the Editor.js asset, protected preview route, tenant and
permission policy, sanitized block renderer, CSP nonce, and browser E2E test
before enabling the publishing editor in production.
