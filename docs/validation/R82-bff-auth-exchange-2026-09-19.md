# R82 — BFF login/logout exchange (2026-09-19)

`internal/security/bffproxy.AuthHandler` now covers the local server-side auth
exchange on top of `bffsession.Store`.

Verified with `go test ./internal/security/bffproxy`:

- browser `Cookie` and `Authorization` headers are rejected before Core login;
- bounded form credentials are sent only to Core `/auth/login`;
- Core's session cookie is captured server-side and replaced with an opaque BFF
  cookie plus a one-time CSRF bootstrap token;
- logout validates CSRF, calls Core `/auth/logout`, revokes the BFF session and
  emits a deletion cookie;
- missing Core session cookies fail closed without exposing upstream details.

This is a reference exchange, not production SSO/provider certification or a
shared multi-instance session service. TLS, key rotation, login abuse policy,
staging browser E2E and external operational drills remain open.
