# ADR-0153: BFF shared session backend

- Status: Accepted (Redis/Valkey adapter; provider certification open)
- Date: 2026-09-19
- Supersedes: none

## Context

The server-side BFF session adapter needs a shared store before multiple web
instances can safely serve the same browser session. The Compact profile must
remain dependency-free, while managed deployments need bounded failure behavior
and must not expose the Core session cookie to the browser or datastore.

## Decision

Provide `internal/security/bffsession.RedisStore` as an optional adapter over an
injected `redis.UniversalClient`, suitable for Redis or Valkey. It implements
the common `bffsession.SessionStore` boundary consumed by the BFF auth exchange,
so deployments can select it without changing browser-facing handlers. The browser keeps
only an opaque session cookie. The Core cookie and CSRF hash are encrypted with
AES-GCM before being written, bound to the opaque session ID, and expired with a
configured TTL. Reads, writes, and revocation use bounded operation contexts and
fail closed on timeout, missing data, invalid configuration, or decryption
failure.

The encryption key is supplied by deployment secret management, never derived
from a Redis URL. Redis is not constructed by Compact; deployments explicitly
select this adapter and configure the client, TLS, ACLs, persistence,
replication, eviction policy, and capacity limits. Provider key rotation,
failover, and multi-instance browser E2E remain release gates outside this ADR.

## Consequences

- Multiple BFF instances can share sessions without sending the Core cookie to
  the browser or storing it in plaintext in Redis.
- A Redis/Valkey outage rejects session operations rather than authenticating
  with stale or incomplete state.
- The bounded in-memory adapter remains appropriate for Compact and tests;
  production deployments must document Redis capacity and eviction behavior.
- Operational certification is still required for TLS/ACL enforcement, key
  ceremony and rotation, persistence/restore, failover, and noisy-neighbor
  protection.
