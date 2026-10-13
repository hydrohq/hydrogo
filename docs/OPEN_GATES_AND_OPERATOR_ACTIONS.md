# Open Gates and Operator Actions

**Status:** Safe local verifications within the repository are automated; this document catalogues requirements that necessitate real production environments, external infrastructure, operator access, or client setups.  
**Last Updated:** 2026-09-21

## Scope & External Evidence Note

The following items cannot be fully verified within isolated, container-less dev checkouts:

- External signed extension manifests and staging HTTPS endpoints.
- Real S3-compatible provider certification: real endpoints, IAM credentials, bucket lifecycle, and KMS/retention policies (local mocks do not replace cloud vendor certification).
- Live deployment rehearsals: two-process blue/green reload and rollback topologies behind reverse proxies.
- External backups and key escrow: restore rehearsals to clean target hosts, key recovery, and access-loss drills.
- Multi-node Redis/Valkey failover, partition, and clock-skew drills.
- Isolated customer pilot environments with written SLAs and measurable RPO/RTO data.

---

## 1. Operator & Production Requirements

### P0 — Distribution & Packaging
- [x] Released under the open-source MIT License.
- [ ] Build official release binaries with embedded vendor public keys via `-ldflags` when optional cryptographic licensing is used.
- [ ] Keep private signing keys strictly within secured vendor infrastructure; never embed them into client distributions or git repositories.

### P0 — Production Security & Network Hardening
- [ ] Terminate HTTPS, enforce `HYDRO_SECURE_COOKIES=true`, configure `public_origin`, and set HSTS policies on production reverse proxies.
- [ ] Create the primary administrator account using a strong password and enforce multi-factor authentication (MFA).
- [ ] Configure production SMTP relays (e.g. Resend, Postmark, SES) and verify end-to-end delivery of password reset, invitation, and MFA recovery notifications.
- [ ] Store production secrets, session keys, and database passwords in secure environment variables or secret managers.

### P0 — Database, Backups, and Disaster Recovery
- [x] Run PostgreSQL contract test suites on PostgreSQL 18 with automated verification (`HYDRO_TEST_POSTGRES_URL`).
- [ ] Rehearse backup and restore procedures using customer-scale datasets, measuring RPO, RTO, and storage utilization.
- [ ] Verify blue/green zero-downtime restarts and schema migration rollbacks behind a production reverse proxy.
- [ ] Formally establish data retention, cryptographic erasure, and backup retention policies.

### P0 — Media Processing & Integrations
- [ ] Ensure `ffmpeg` is installed in production and configured via `HYDRO_MEDIA_FFMPEG` when rich video transcoding is required.
- [ ] Confirm graceful degradation (fail-closed, queued, or error feedback) when media transcoders are unavailable, verifying bounded memory limits.
- [ ] Verify payment provider webhooks (e.g., Stripe sandbox/production), signature validation, refund workflows, and timeout handling.

---

## 2. Test Scenarios Requiring Real Environments

- [x] Chromium Playwright E2E: isolated SQLite server, migrations, seeds, admin authentication state.
- [ ] Cross-browser responsive E2E testing across full desktop, tablet, and mobile device matrices.
- [ ] Load and endurance testing: verify connection pools, queue worker leases, and rate-limiting behaviors under multi-hour sustained load.
- [ ] Performance profiling across capacity profiles (Compact, Standard, Distributed) under real network latencies.

---

## 3. Architecture & Governance Boundaries

- [x] Minimal preset configurations: `compact-site`, `editorial`, `commerce`, and `headless` configurations that isolate unused module overhead.
- [x] Inactive modules do not execute migrations, register HTTP routes, or instantiate background workers.
- [x] Member credentials and session tokens are never persisted to insecure browser local storage.
- [x] Generic error messages are presented to clients in production to prevent leaking internal database or infrastructure details.
- [ ] Long-term historical telemetry: decide data retention policies, aggregation cadences, and GDPR consent requirements.
- [ ] Headless browser sessions: enforce same-site BFF proxy boundaries rather than permissive cross-origin credentials.

---

## 4. Locally Verified Capabilities

- [x] Full unit and integration test suite passing on Go.
- [x] Concurrency race detector (`go test -race ./...`) clean across all internal packages.
- [x] Static analysis (`go vet ./...`) and lint rules passing.
- [x] Module boundary isolation verified via `tools/moduleguard`.
- [x] Migration metadata and dialect parity verified via `tools/migrationguard`.
- [x] Safe project scaffolding (`hydro new`) without leaking core internal packages.

---

## 5. Related References

- [Production Gates](validation/PRODUCTION-GATES.md)
- [Operations Runbook](OPERATIONS_RUNBOOK.md)
- [Personal Data Policy](PERSONAL_DATA_POLICY.md)
- [Information Hierarchy](INFORMATION_HIERARCHY.md)
- [Roadmap](ROADMAP.md)
