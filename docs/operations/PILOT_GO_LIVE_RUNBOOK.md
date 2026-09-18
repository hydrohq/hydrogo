# Pilot Go-Live & Incident Response Runbook

> **Operational Target:** Single-Customer Production Launch (Phase 5)  
> **Safety Discipline:** Fail-closed defaults, zero unmeasured claims, auditable break-glass.

---

## 1. Pre-Flight Launch Checklist (T-24h)

1. **Module Gating Verification:**
   - Verify `HYDRO_MODULES_ENABLED` contains only requested modules (e.g. `pages,site,media,forms`).
   - Confirm inactive modules have registered 0 routes and 0 workers (`hydro doctor --verbose`).
2. **Database & Storage Integrity:**
   - Execute dry-run migration and verify schema version matches release tag.
   - Run `hydro backup create` and test restoration onto staging target with media checksum check.
3. **MFA & Access Auditing:**
   - Ensure all administrator accounts have MFA (TOTP) enrolled.
   - Test password reset and single-use emergency recovery token flow.
4. **Monitoring & Alerting Verification:**
   - Trigger test alarm (P1 alert) to verify PagerDuty / OpsGenie / Slack notification delivery.
   - Confirm Prometheus metrics endpoint (`/metrics`) is responding to authorized scrapers.

---

## 2. Launch Sequence (T-0)

```text
[Step 1]  Set maintenance banner on public edge proxy if migrating data.
[Step 2]  Execute latest database migrations: `hydro migrate up`.
[Step 3]  Deploy HydroGo binary in Slot Green.
[Step 4]  Execute pre-traffic warmup and probe checks (`/readyz`).
[Step 5]  Switch reverse proxy upstream traffic from Blue to Green.
[Step 6]  Monitor p95 latency and error rate for 15 minutes.
[Step 7]  Retain Blue binary and old static assets for 48-hour rollback window.
```

---

## 3. Incident Severity & Escalation Matrix

### P1 - Critical Outage / Data Loss Risk
- **Action:** Immediately initiate bridge call with DevOps Lead and Product Architect.
- **Rollback Decision Rule:** If error rate > 1% or p99 latency > 1000ms for more than 5 minutes post-deploy, trigger **immediate one-command slot rollback**:
  ```bash
  hydro slot switch --target blue
  ```
- **Customer Communication:** Post status update to status page within 15 minutes.

### Break-Glass Emergency Access
- In the event of administrative lockout, execute CLI break-glass password reset from the secure server console:
  ```bash
  hydro admin reset-password --email admin@example.com --force-mfa-reenroll
  ```
- An immutable audit entry is written directly to the database and external audit log.
