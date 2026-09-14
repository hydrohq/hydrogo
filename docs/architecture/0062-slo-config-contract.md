# ADR-0062: Keep SLO configuration target-neutral

## Status

Accepted

## Decision

HydroGo provides a small validated SLO configuration contract containing a
stable name, indicator, window, and optional target. It does not prescribe
target percentages, latency budgets, alert thresholds, or customer-specific
SLIs.

## Consequences

Invalid names, duplicate objectives, unsafe indicator values, and malformed
windows fail before dashboards consume the configuration. Operators retain
ownership of service objectives and can add them without changing Core.
