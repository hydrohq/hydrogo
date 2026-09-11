# R3 cache generation and bounded miss validation

**Date:** 2026-09-08  
**Roadmap:** R3 / S05-C  
**Status:** Local cache gate, in-flight generation bypass/revalidation, and stale refill rejection succeeded; actual blue/green traffic and measured DB capacity drill remain open.

`internal/cache.Cache.GetOrLoad` coalesces concurrent misses for the same key within the same generation into a single loader call. During `Clear`, the generation is incremented; a new request does not join an old in-flight loader, and even if the old result can return to its caller, it cannot refill the cache. The Pages public route lookup uses this contract.

The Pages service contract test runs 64 concurrent same-route requests and verifies that the repository/DB lookup count is 1. Thus, the bounded miss load is proven not only at the cache primitive level but also at the actual public lookup service boundary.

Executed checks:

```text
go test ./internal/cache ./tests/contract
go test ./...
go vet ./...
```

The remaining R3 work involves actual blue/green deploy traffic and measuring DB load with a realistic dataset on the supported Compact profile. L2/Distributed behavior is not part of this gate.
