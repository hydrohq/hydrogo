# Dependency, license, and vulnerability triage cadence

## Owners and evidence

| Activity | Owner role | Cadence | Evidence | Escalation |
| --- | --- | --- | --- | --- |
| Reachable vulnerability scan | Security reviewer | Every change and weekly | `govulncheck` output in CI artifact | Block release for a reachable high/critical finding |
| SPDX SBOM and module graph | Release engineer | Every release and weekly | `hydro-sbom.spdx.json`, `go-modules.txt` artifact | Block release if generation or diff review fails |
| Dependency update review | Core maintainer | Dependabot weekly | Dependabot PR plus test/compatibility result | Defer only with dated owner and rationale |
| License/edition compatibility | Legal/compliance owner | Every new direct dependency and release | License review entry and release checklist | Hold distribution until cleared |

The scheduled workflow is `.github/workflows/dependency-review.yml` (Monday
03:23 UTC) and can be run manually for an immediate review. Artifacts are
retained for 90 days; the release record links the artifact or records why the
run was not available. No private keys, credentials, or production data belong
in this evidence.

## Every change

- Run `govulncheck ./...` and the repository verification script.
- Review new direct and transitive dependencies, their license, and whether
  they are reachable from production builds.
- Keep generated SPDX SBOM output deterministic and attached to the release.

## Weekly

- Review newly disclosed vulnerabilities and upstream advisories.
- Classify each finding as reachable, unreachable, test-only, or false positive;
  record the evidence and owner.
- Prefer a patched version or a bounded mitigation. Do not suppress a finding
  without an expiry date and reviewer.

## Release

- Re-run the scan from a clean checkout and compare the SBOM diff.
- Confirm license compatibility with the edition/license boundary.
- Record unresolved findings in the release checklist and open-gates document.

Production severity, remediation deadlines, and legal interpretation are policy
decisions; this cadence records evidence but does not invent those values.
