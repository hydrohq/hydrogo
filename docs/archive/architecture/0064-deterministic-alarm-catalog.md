# ADR-0064: Keep the operational alarm catalog deterministic

## Status

Accepted

## Decision

Operational alarms are emitted from a bounded snapshot with fixed codes and
severity. Thresholds remain supplied by configuration; the platform does not
hard-code customer SLOs. Dashboard integrations consume the catalog and must
not attach unbounded IDs or PII as labels.

## Consequences

Alarm routing is predictable across editions and profiles. Provider-specific
escalation and notification policy remain deployment-owned concerns.
