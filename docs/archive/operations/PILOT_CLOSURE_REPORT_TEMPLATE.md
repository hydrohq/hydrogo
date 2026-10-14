# Pilot Program Closure & Capacity Report Template

> **Evaluation Period:** 4–8 Weeks Post Go-Live  
> **Purpose:** Validate production stability, support burden, SLA compliance, and capacity boundaries before multi-client expansion.

---

## 1. Executive Summary & SLA Delivery

- **Customer / Brand Name:** [Customer Name]
- **Deployment Profile:** `standard` (PostgreSQL 18 + Valkey + S3)
- **Pilot Duration:** [Start Date] to [End Date] (e.g., 60 days)
- **Delivered Availability:** [e.g., 99.98%] (Target: 99.90%)
- **Total Unplanned Downtime:** [e.g., 8 minutes]

---

## 2. Capacity & Resource Consumption Metrics

| Metric | Measured Median (p50) | Measured Peak (p99) | Budget Limit | Status |
| :--- | :--- | :--- | :--- | :--- |
| **Daily Requests** | [e.g., 450,000] | [e.g., 1,200,000] | 5,000,000 | PASS |
| **API Latency (Public Read)** | [e.g., 6.2 ms] | [e.g., 34.0 ms] | 200 ms | PASS |
| **Database Pool Utilization** | [e.g., 12%] | [e.g., 44%] | 80% | PASS |
| **Redis Memory Footprint** | [e.g., 64 MB] | [e.g., 140 MB] | 256 MB | PASS |
| **Media Storage Volume** | [e.g., 18.5 GB] | [e.g., 22.0 GB] | 100 GB | PASS |

---

## 3. Incident & Support Log Summary

- **P1 Incidents:** [Count, e.g. 0]
- **P2 Incidents:** [Count, e.g. 1 - Resolved in 45 min]
- **P3/P4 Bug Reports:** [Count, e.g. 4]
- **Support Workload:** [e.g., Average 1.2 tickets/week]

---

## 4. Final Commercial & Architectural Recommendation

- [ ] **Recommendation:** Transition from Pilot to Full Commercial SLA Support.
- [ ] **Capacity Expansion Required:** Yes / No (Details: ____________).
- [ ] **Client Edition Extensions Required:** (Details: ____________).
