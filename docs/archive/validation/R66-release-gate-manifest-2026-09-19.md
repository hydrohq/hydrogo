# R66 release gate manifest

**Date:** 2026-09-19  
**Status:** Local manifest validation passed; the managed Standard profile remains a candidate.

The machine-readable [`release-gate-manifest.json`](release-gate-manifest.json)
maps each claimed profile to dated evidence. `releasegates` verifies that every
verified gate has an existing repository evidence file and rejects a `ready`
profile when any required gate is pending. It does not convert a prose plan or
local synthetic benchmark into production evidence.

P3 shuffled stability evidence and P4 repeatable resilience drills are also
represented in the manifest. CI validates the local-development readiness claim
on every run; the managed Standard profile remains fail-closed as a candidate
until its external gates are evidenced.

```text
go test ./tools/releasegates
go run ./tools/releasegates -manifest docs/validation/release-gate-manifest.json
go run ./tools/releasegates -manifest docs/validation/release-gate-manifest.json -profile local-development -require-ready
```

The Standard profile intentionally has `candidate` status because clean-target
restore/RPO-RTO, reverse-proxy blue/green and independent security testing still
require a real deployment. A readiness check for that profile must fail until
those reports are added:

```text
go run ./tools/releasegates -manifest docs/validation/release-gate-manifest.json -profile managed-single-customer-standard -require-ready
```
