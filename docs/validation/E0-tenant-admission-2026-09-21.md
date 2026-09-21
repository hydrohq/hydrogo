# E0 Tenant Admission Validation — 2026-09-21

`internal/security.TenantAdmission` adds a bounded process-local concurrency
quota per tenant, rejects missing tenant scope, and caps the number of active
tenant buckets. The contract test proves tenant A cannot consume tenant B's
slot and that new buckets are denied after the configured bound.
The recovery contract also proves a canceled waiter does not leak capacity:
after release, the tenant can acquire a fresh slot again.
The HTTP middleware contract additionally proves a saturated noisy tenant gets
`429` while a separate healthy tenant still receives `204`.

```text
go test ./internal/security -count=1
```

This is a single-process noisy-neighbor boundary. A distributed deployment
can now opt into `security.RedisTenantAdmission`, a Redis/Valkey sorted-set
lease with expiry cleanup and idempotent release, wired through
`server.Config.DistributedTenantAdmission`. Compact remains process-local by
default; the live Redis/Valkey multi-instance and partition drill is still an
external certification gate.

When `HYDRO_TEST_REDIS_URL` is provided, the integration gate creates two
independent Redis clients and verifies shared tenant-A saturation, tenant-B
isolation, and release recovery against the same sorted-set lease namespace.
