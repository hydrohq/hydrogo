# R92 — Runtime Admin Studio extension contract

Date: 2026-09-19

## Evidence

`extension/v1.AdminExtensionManifest` and
`internal/adminstudio.ExtensionRegistry` provide a validated, deterministic
runtime manifest boundary for customer-owned Admin Studio surfaces. Validation
rejects non-HTTPS entrypoints, credential-bearing URLs, unsafe paths, duplicate
surfaces/events, malformed identifiers and unbounded values. The registry stores
metadata only; it does not load Go plugins or execute customer code in Core.
Backend customization is represented as named event subscriptions for the
existing external webhook boundary.

Commands executed:

- `go test ./extension/v1 ./internal/adminstudio -count=1`
- `go vet ./extension/v1 ./internal/adminstudio`

Result: passed.

## Remaining production gates

Before accepting customer manifests in a production tenant, add signature
verification/key rotation, CSP and iframe sandbox policy, a versioned
postMessage protocol, lifecycle/rollback handling and webhook provider
certification. The contract intentionally does not support runtime Go plugin
loading or direct database access.
