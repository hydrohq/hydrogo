# Production Disaster Recovery & Operational Drill Template

> **Status: not production evidence.** Replace each placeholder with an actual
> environment, immutable commit SHA, command output, artifact location, and
> operator sign-off only after the drill has been run.

---

## 1. Drill 01: Blue/Green Deployment & Rollback Drill

- **Date / Commit:** `PENDING` · exact immutable commit SHA required
- **Objective:** Verify zero-downtime binary switch, backward-compatible migration co-existence, and instant reverse-proxy rollback without data corruption.
- **Topology:** Caddy Reverse Proxy upstreaming to Slot A (`hydro-blue:8080`) and Slot B (`hydro-green:8081`).
- **Executed Steps:**
  1. Slot A active serving v1 traffic with 50 concurrent synthetic HTTP read/write clients.
  2. Applied non-destructive expand migration (`CREATE INDEX CONCURRENTLY` / nullable column additions).
  3. Deployed and warmed up Slot B (v2) with `/readyz` probe verification.
  4. Switched reverse proxy upstream slot from A to B atomically.
  5. Injected simulated failure on Slot B and triggered immediate rollback to Slot A.
- **Observed Metrics:** record request count, 502/503/504 count, write-fence
  result, rollback duration, reverse-proxy config diff, and retained-asset check.
- **Pass Status:** **PENDING EXECUTION**

---

## 2. Drill 02: Clean Target Database + Media Restore & Checksum Validation

- **Date / Commit:** `PENDING` · exact immutable commit SHA required
- **Objective:** Validate PITR restore onto an empty PostgreSQL target, verify SHA-256 media checksums, revoke all legacy sessions, and quarantine stale background jobs.
- **Evidence Command:**
  ```powershell
  hydro backup postgres --destination <new-snapshot-directory>
  hydro restore verify --snapshot <new-snapshot-directory>
  hydro restore postgres --snapshot <new-snapshot-directory> --destination <new-staging-directory> --sessions-revoked --egress-quarantined --old-writes-blocked
  hydro restore activate-media --staging <new-staging-directory> --active <active-media-directory> --rollback <rollback-media-directory>
  hydro doctor --json
  ```
- **Verification Results:** attach manifest checksum output, restored-session
  revocation evidence, quarantine marker, destination DB identity, and media
  activation result.
- **Pass Status:** **PENDING EXECUTION**

---

## 3. Drill 03: Redis Outage & Cache Invalidation Resilience

- **Date / Commit:** `PENDING` · exact immutable commit SHA required
- **Objective:** Verify bounded fallback to PostgreSQL when Valkey/Redis instance experiences complete network partition or restart.
- **Observations:** attach partition/restart timestamps, cache and rate-limit
  behavior, alert delivery evidence, reconnect log, and stale-read checks.
- **Pass Status:** **PENDING EXECUTION**

---

## 4. Drill 04: Capacity Baseline & Latency Budget (Standard Profile)

| Scenario | Concurrency | p50 Latency | p95 Latency | p99 Latency | Error Rate |
| --- | --- | --- | --- | --- | --- |
| Public cached read | `PENDING` | `PENDING` | `PENDING` | `PENDING` | `PENDING` |
| Public dynamic miss | `PENDING` | `PENDING` | `PENDING` | `PENDING` | `PENDING` |
| Form submissions | `PENDING` | `PENDING` | `PENDING` | `PENDING` | `PENDING` |
| Admin mutations | `PENDING` | `PENDING` | `PENDING` | `PENDING` | `PENDING` |

- **RPO measured:** `PENDING`
- **RTO measured:** `PENDING`
- **Pass Status:** **PENDING EXECUTION**
