# R13 WSL Redis outage and recovery evidence (2026-09-17)

## Result

On the Ubuntu WSL2 Redis/Valkey-compatible service, a controlled single-node
outage and recovery drill was completed at commit `63e4be6`.

1. Redis was stopped through the local service manager.
2. `redis-cli ping` failed as expected.
3. Limiter startup/EVAL-unavailable regression tests passed and failed closed.
4. Redis was started again; `redis-cli ping` returned `PONG`.
5. The atomic `INCR` + TTL integration test passed against the recovered service.

No application data or credentials were used. The service was restored before
the drill ended.

## Remaining scope

This proves local outage handling and recovery for one service instance. It
does not certify multi-instance failover, network partition behavior,
clock-skew handling, persistence/replay, or customer capacity; those remain
staging/production gates.
