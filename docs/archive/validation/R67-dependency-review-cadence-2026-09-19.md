# R67 dependency review cadence

**Date:** 2026-09-19  
**Status:** Workflow and ownership controls are present; the first scheduled
external run remains an operational follow-up.

HydroGo now has a weekly, manually triggerable dependency review workflow. It
verifies the module graph, runs the reachable-code vulnerability scan, creates
a deterministic SPDX SBOM, records the complete Go module graph, and retains
the evidence artifact for 90 days. Dependabot remains the update intake path;
the owner/escalation matrix is in [`docs/DEPENDENCY_TRIAGE.md`](../DEPENDENCY_TRIAGE.md).

Local verification for the workflow's repository-owned checks:

```text
go run ./tools/actionguard
go test ./tools/sbom ./tools/actionguard
go mod verify
```

This report does not claim that a scheduled GitHub run has completed; the first
workflow execution and any findings must be recorded by the release/security
owner from the actual CI artifact.
