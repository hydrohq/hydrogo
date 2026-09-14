# ADR-0054: Bounded observability dimensions

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0013](0013-observability-and-privacy.md)

HydroGo metrics use fixed, low-cardinality dimensions. Request paths, user IDs,
tenant IDs, job IDs, and arbitrary error strings are not metric labels. Logs
remain structured and bounded with request correlation and redaction.
