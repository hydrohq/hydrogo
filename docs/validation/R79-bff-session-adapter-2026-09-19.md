# R79 — BFF server-side session adapter (2026-09-19)

`internal/security/bffsession.Store` now supplies the local/Compact BFF
reference boundary used by `internal/security/bffproxy`.

Verified with `go test ./internal/security/bffsession`:

- Core cookies remain server-side while the browser receives only an opaque,
  `HttpOnly` cookie;
- CSRF tokens are hashed at rest and compared in constant time;
- missing, invalid, revoked and expired sessions fail closed;
- session creation is bounded by a configurable maximum entry count;
- cookie names are validated against the HTTP token grammar before startup;
- cookie expiry, `Secure` and SameSite attributes are explicit.

This does not certify a shared multi-instance session backend, login/token
exchange, key rotation, or staging cross-origin browser E2E. Those remain
deployment and production gates.
