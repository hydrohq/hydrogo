# R94 — Admin extension postMessage contract

Date: 2026-09-19

## Evidence

`extension/v1.AdminMessage` defines version 1 of the host/extension envelope.
It accepts only an allowlisted message type, bounded request ID and bounded
valid JSON payload. `AdminBootstrapPayload` validates the non-secret context
that the host may send after the iframe handshake. Unknown types, attempts to
send cookie-like control messages, malformed IDs, oversized payloads and
newline-bearing locale values are rejected.

Command:

- `go test ./extension/v1 -run TestAdminMessageContract -count=1`
- `go vet ./extension/v1`

Result: passed.

The browser host still needs to enforce exact `targetOrigin`, CSP/frame-ancestors
and sandbox flags before enabling external extensions in production.
