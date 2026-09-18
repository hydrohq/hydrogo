# Service Level Objectives (SLO), RPO & RTO Specification

> **Target Release:** HydroGo Single-Customer Production Pilot (`standard` Profile)  
> **Authoritative Standard:** Site Reliability & Operational Safety

---

## 1. Objectives Overview

| Metric | Target Objective | Measurement Window | Alerting Threshold |
| :--- | :--- | :--- | :--- |
| **Availability (SLO)** | **99.9%** | 30-day rolling window | Downtime > 5 minutes triggers P1 alert |
| **RPO (Recovery Point Objective)** | **<= 5 minutes** | Any catastrophic database loss | Continuous WAL archiving + Hourly snapshots |
| **RTO (Recovery Time Objective)** | **<= 15 minutes** | Total disaster recovery | Automated restore verification in clean target |
| **p95 Public API Latency** | **< 60 ms** | 5-minute rolling window | p95 > 150 ms over 3 consecutive windows |
| **p99 Public API Latency** | **< 200 ms** | 5-minute rolling window | p99 > 500 ms over 3 consecutive windows |
| **p95 Admin API / Form Latency** | **< 150 ms** | 5-minute rolling window | p95 > 350 ms over 3 consecutive windows |
| **Public Cache Hit Ratio** | **> 92%** | Standard read workload | Cache hit ratio < 80% under steady traffic |

---

## 2. Recovery Point Objective (RPO) Strategy (<= 5 Minutes)

1. **Transactional Durability:**
   - PostgreSQL Write-Ahead Logging (`wal_level = replica`, `archive_mode = on`).
   - WAL segments continuously shipped to encrypted object storage bucket with a maximum archival timeout of 5 minutes (`archive_timeout = 300`).
2. **Snapshot Cadence:**
   - Full database dump + media manifest executed every 6 hours.
   - Point-In-Time-Recovery (PITR) supported via base backup + WAL replay to any arbitrary second within the last 30 days.
3. **Media Storage:**
   - S3 object versioning enabled on the media bucket.
   - Deleted/updated items retain previous versions in cold storage for 30 days to prevent accidental or malicious destruction.

---

## 3. Recovery Time Objective (RTO) Strategy (<= 15 Minutes)

1. **Automated Restore Preflight:**
   - CLI command `hydro restore` verifies manifest integrity, checksums, and schema safety in a temporary staging target before activating media or opening database traffic.
2. **Standardized Recovery Procedure:**
   - Step 1 (0–3 min): Provision clean target PostgreSQL database and fetch latest base snapshot.
   - Step 2 (3–8 min): Replay WAL archive up to target timestamp (PITR).
   - Step 3 (8–11 min): Execute `hydro restore verify` to validate data markers, write-fencing, and media checksums.
   - Step 4 (11–13 min): Run `hydro doctor` to verify system health, revoking old sessions and quarantining stale background jobs.
   - Step 5 (13–15 min): Switch reverse proxy upstream to new instance.
3. **Drill Frequency:**
   - Mandatory disaster recovery drill executed quarterly on staging/isolated target.

---

## 4. Latency & Resource Budgets

### Latency Profiles (Standard Profile)

```text
Request Path                          p50 (Median)    p95 Budget     p99 Budget
─────────────────────────────────────────────────────────────────────────────
Public Read (L1/L2 Cache Hit)         2 - 8 ms        15 ms          35 ms
Public Read (DB Cache Miss)          15 - 25 ms       60 ms         120 ms
Admin Mutation / Form Submit         20 - 45 ms      120 ms         250 ms
Media Upload (Presigned S3/Direct)   50 - 150 ms     300 ms         800 ms
Authentication & Password Verify     40 - 70 ms      110 ms         180 ms
```

### Resource Thresholds & Alarms

- **CPU Utilization:** Alert when sustained > 75% for 10 minutes.
- **Memory Utilization:** Alert when sustained > 80% (ensures memory headroom for buffer pools and garbage collection).
- **Database Connection Pool:** Alert when active connections > 80% of max pool size (`HYDRO_DB_MAX_OPEN_CONNS`).
- **Disk Storage (Data / Logs):** Warning at 75% capacity, Critical alert at 85% capacity.
- **Dead-Letter Jobs Queue:** Alert when unacknowledged dead-letter jobs > 0.
