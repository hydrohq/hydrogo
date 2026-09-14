# ADR-0059: Evidence-based dependency triage cadence

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0028](0028-core-isolation-and-licensing.md), [ADR-0057](0057-release-checklist-gates.md)

Dependency and vulnerability review occurs on every change, weekly, and before
release. Findings require reachability/license evidence and an owner; silent
suppression is not permitted. Policy severity and legal decisions remain
outside the codebase until explicitly approved.
