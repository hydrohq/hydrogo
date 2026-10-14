# R97 — Signed Admin Extension runtime host

Date: 2026-09-19

## Evidence

The runtime now loads bounded signed JSON manifests from an operator directory,
verifies them against an external Ed25519 key map, exposes validated metadata
behind admin authorization, and renders an isolated iframe shell with a
per-origin CSP and non-secret bootstrap envelope. Empty configuration leaves
the feature disabled.

Commands:

- `go test ./internal/adminstudio ./extension/v1 -count=1`
- `go test ./cmd/hydro ./internal/config -count=1`
- `go vet ./internal/adminstudio ./extension/v1 ./cmd/hydro`

Result: passed.

This is repository-level contract evidence. Trusted-key rotation, revocation,
browser/CSP staging E2E, external hosting availability, and operator rollout
drills remain open; this does not certify a production marketplace.
