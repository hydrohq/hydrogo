# Dependency, license, and vulnerability triage cadence

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
