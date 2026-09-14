# Deterministic alarm rules

HydroGo evaluates alarms from one bounded runtime snapshot. The evaluator is
deterministic: output order is stable and no alarm contains a request path,
tenant, user, error string, or provider secret.

| Code | Severity | Condition | Operator action |
|---|---|---|---|
| `backup_stale` | critical | Last successful backup is absent or older than the configured maximum age | Stop risky changes, inspect backup job and storage, then run a restore verification |
| `disk_low` | critical | Free bytes are below the configured minimum | Quarantine uploads, free/expand storage, verify media integrity |
| `migration_failed` | critical | Latest migration did not complete | Keep traffic gated, inspect migration evidence, use forward repair/restore |
| `readiness_failed` | critical | Readiness checks fail | Remove instance from traffic and inspect dependency health |
| `smtp_delivery` | warning | One or more notification deliveries failed | Review provider status and retry policy |
| `jobs_dead_lettered` | warning | Dead-letter count is non-zero | Review payload class and replay only after authorization |

## Dashboard examples

Render the alarm list from `ops.Evaluate(snapshot)` and group by severity. A
minimal dashboard should show the code, first-seen time, current state, and a
link to the relevant runbook. Do not add raw job IDs, emails, URLs, SQL, or
provider response bodies as labels.

The evaluator does not invent thresholds: backup age, disk minimum, and other
budgets come from the selected deployment profile or operator configuration.
