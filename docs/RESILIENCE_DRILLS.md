# Resilience drills

`Resilience Drills` is a manual GitHub Actions workflow. It repeats the
in-process queue, worker polling, scheduled-news and blue/green traffic
contracts, then publishes a separate local SQLite capacity-profile report.
The report is comparative evidence, not an SLA or a production-capacity claim.

Use a repetition count between 1 and 50. A failed iteration fails the workflow;
there is no automatic retry. Pull-request CI retains its fast deterministic
gates, while this workflow is used before a release candidate or after a
concurrency-sensitive change.

The drill does not replace an external PostgreSQL/Redis outage, multi-process
proxy drain, object-storage failure or representative customer-load exercise.
Those remain operator-run production gates.
