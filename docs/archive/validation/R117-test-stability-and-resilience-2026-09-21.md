# R117 — Test stability and resilience release evidence

Date: 2026-09-21

P3 adds a three-run shuffled stability gate that writes a CI artifact without
retrying failed runs. Its runnable policy and local entry points are in
[`../TEST_STABILITY.md`](../TEST_STABILITY.md).

P4 adds the manually dispatched `Resilience Drills` workflow. It repeats
queue, worker, scheduled-news and blue/green contracts, runs Linux race checks,
and publishes a local SQLite capacity-profile artifact. Its scope and explicit
limitations are in [`../RESILIENCE_DRILLS.md`](../RESILIENCE_DRILLS.md).

These are verified repository controls, not proof of an external PostgreSQL or
Redis outage, reverse-proxy rollback, clean-target restore, or customer-load
exercise. Those gates remain pending in `release-gate-manifest.json`.
