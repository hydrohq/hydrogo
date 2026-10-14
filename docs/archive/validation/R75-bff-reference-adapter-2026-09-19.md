# R75 — BFF reference adapter (2026-09-19)

## Scope

The server-side BFF boundary now has a reusable `internal/security/bffproxy`
handler on top of the request policy primitive. It is intended for a separate
web frontend deployment and does not change Core's browser session model.

## Verified behavior

- only versioned `/api/v1` requests are forwarded;
- the server-side session resolver supplies the Core cookie;
- browser cookies, origin and forwarding headers are not reused as Core
  credentials;
- unsafe methods fail closed without a deployment CSRF validator;
- Core `Set-Cookie` response headers are not exposed to the browser;
- request and response bodies are bounded (1 MiB defaults, configurable);
- upstream failures return a bounded BFF error response.

```text
go test ./internal/security/bffproxy ./internal/security/bffpolicy
```

## Remaining external gate

This is a reference adapter, not a complete login/session store or production
reverse proxy. A frontend deployment must supply session persistence, login and
logout exchange, CSRF token issuance, TLS/proxy configuration, and staging
browser E2E evidence.
