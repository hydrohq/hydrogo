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
still needs gateway/queue quotas backed by a shared coordination service.
