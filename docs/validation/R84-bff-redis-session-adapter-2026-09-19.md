# R84 — BFF Redis/Valkey shared session adapter

Date: 2026-09-19  
Scope: `internal/security/bffsession.RedisStore`

## Evidence

The adapter encrypts the Core cookie and CSRF hash with AES-GCM before writing
to Redis/Valkey, binds ciphertext to the opaque browser session ID, validates
configuration (including a 32-byte key and bounded TTL), and applies bounded
operation contexts. Missing keys, timeouts, Redis errors, expiry, and corrupt
records fail closed. The lifecycle test runs against `HYDRO_TEST_REDIS_URL`
when configured and is skipped otherwise; no external Redis service is claimed
by this repository-only result.

Command:

```text
go test ./internal/security/bffsession
```

Result: passed (conditional external lifecycle test skipped when
`HYDRO_TEST_REDIS_URL` is not set).

## Open release gates

An operator still needs to certify provider TLS and ACLs, secret key ceremony
and rotation, persistence/restore, replication and failover, eviction and
capacity/noisy-neighbor limits, and multi-instance browser E2E in staging.
