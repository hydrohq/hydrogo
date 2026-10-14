# Security Policy

## Supported versions

HydroGo is under active development. Security fixes are applied to the latest
`main` branch and to the latest published release once releases are declared
supported. Older commits and unreleased development snapshots are not a
security-support promise.

The support status of profiles and database providers is documented in
`docs/archive/SUPPORT_MATRIX.md` (historical).

## Reporting a vulnerability

Please do not open a public issue for an undisclosed vulnerability. Report it
through a private GitHub Security Advisory for this repository. If private
reporting is unavailable, contact the repository maintainers through a private
channel and include “HydroGo security report” in the subject.

Please include:

- the affected commit, release, or component;
- a clear description of the impact and attack prerequisites;
- reproduction steps or a minimal proof of concept;
- any suggested mitigation; and
- whether the issue is already known to be exploited.

Do not include real customer data, production credentials, signing keys,
session tokens, or other secrets in a report or proof of concept.

We will acknowledge a report when it is received, investigate it, and
coordinate disclosure after a fix or mitigation is available. Timelines may
depend on the severity, exploitability, affected deployment profile, and
whether an external provider must publish a coordinated fix.

## Scope and assumptions

The threat model and deployment assumptions are maintained in
`docs/archive/security/THREAT_MODEL.md`. A contract test, local benchmark, or
development-only configuration is not by itself a production security
certification. Please identify the profile, database, storage provider, and
deployment topology when reporting an operational security issue.
