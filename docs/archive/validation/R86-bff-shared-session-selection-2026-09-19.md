# R86 — BFF shared session selection boundary

Date: 2026-09-19  
Scope: `bffsession.SessionStore` and `bffproxy.AuthHandler`

## Evidence

The BFF login/logout exchange now depends on the common
`bffsession.SessionStore` interface instead of the process-local memory store
concrete type. Both the bounded `Store` and encrypted Redis/Valkey `RedisStore`
implement the same `Create`, `CoreCookie`, `Validate`, and `Revoke` boundary;
request contexts are passed through so shared adapters can enforce bounded I/O.
This makes shared sessions selectable without changing browser-facing routes.

Command:

```text
go test ./internal/security/bffsession ./internal/security/bffproxy
```

Result: passed.

Provider TLS/ACL, key rotation, persistence/failover, capacity/eviction and
staging multi-instance browser E2E remain external release gates.
