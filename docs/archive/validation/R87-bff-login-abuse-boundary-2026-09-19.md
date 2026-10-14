# R87 — BFF login abuse boundary

Date: 2026-09-19  
Scope: `bffproxy.AuthHandler.Login`

## Evidence

`AuthHandler` now accepts an injected `LoginLimiter`. Before contacting Core it
uses a normalized email and host-only remote address key, returns a generic
429 with `Retry-After: 60` when the limiter rejects, and does not make an
upstream request. The limiter interface permits the bounded local limiter or a
fail-closed Redis/Valkey implementation to be selected by deployment.

Command:

```text
go test ./internal/security/bffproxy -run TestAuthHandlerLoginLimiterBlocksBeforeCore
```

Result: passed.

The production choice of limits, trusted proxy address handling, shared-store
availability policy, and staging abuse drill remain deployment gates.
