# R6 Redis Rate-Limit Integration Evidence (2026-09-16)

## Scope

The `internal/security` Redis limiter contract was verified by connecting to a
local Redis/Valkey-compatible server running on WSL Ubuntu. The test runs only when
`HYDRO_TEST_REDIS_URL` is explicitly provided; if the service is absent, the default test
flow does not start external dependencies and skips the test.

## Command and Results

```bash
redis-cli ping
# PONG
cd /path/to/project
HYDRO_TEST_REDIS_URL=redis://127.0.0.1:6379/15 \
  go test ./internal/security -run TestRedisRateLimiterAtomicLimitAndTTL -count=1
# ok .../internal/security
```

The test verifies that the initial request is accepted, a second request with the same key is
rejected due to the rate limit, and the Redis key has a positive TTL not exceeding 30 seconds.
The key is deleted upon test completion; no real user data is used.

This evidence covers atomic `INCR` + TTL behavior. Multi-instance failover,
partitioning, clock-skew, persistence, and capacity/soak tests remain as live deployment
gates in the [Open Gates and Operator Actions](../OPEN_GATES_AND_OPERATOR_ACTIONS.md) document.

The limiter also tests cancellation/deadline propagation fail-closed during
request-scoped `AllowContext` calls; a canceled request is prevented from holding
the Redis operation up to the safety timeout.

Without requiring stopping a local service, initial connection and `EVAL` unreachability
over a reserved loopback port are also verified via fail-closed regression tests.
Authentic failover, partition, and clock skew drills still require a multi-instance staging environment.
