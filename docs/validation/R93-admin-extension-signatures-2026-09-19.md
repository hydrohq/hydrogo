# R93 — Signed Admin Studio extension manifests

Date: 2026-09-19

## Evidence

Admin extension manifests can now be signed with Ed25519 in a release
environment and verified only against operator-configured public keys. The
verification covers the canonical JSON representation of the validated
manifest, rejects unknown key IDs, malformed signatures and tampered fields,
and supports an explicit key-rotation map. Public keys supplied by the
extension artifact are never trusted automatically.

Command:

- `go test ./extension/v1 -run TestSignedAdminExtensionManifest -count=1`
- `go vet ./extension/v1`

Result: passed.

Remaining production gates are key storage/ceremony, rotation rollout,
CSP/iframe isolation, postMessage protocol and extension lifecycle rollback.
