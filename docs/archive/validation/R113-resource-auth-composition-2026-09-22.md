# R113 — Resource auth composition flag

Date: 2026-09-22

## Scope

Blueprints can declare `auth: true`. The declaration is metadata only: it
requires the existing HydroGo Identity/Session/RBAC composition at the Edition
boundary and never generates a parallel credential or authorization system.

## Evidence

- `go test ./internal/generator -run 'TestLoadBlueprintMapsAuthCompositionFlag|TestCoreCollectionFixturesGenerateDeterministically' -count=1` — PASS.
- Generated `Spec` and `module.yaml` carry the auth declaration.
- Generated runtime wiring contains an authenticated/anonymous policy review
  item only when the flag is enabled.
- JSON Schema documents the boolean with a false default.
- ADR index drift check covers the new decision record.

## Boundary result

The resource contract now expresses Payload-like auth intent while preserving
HydroGo's centralized, deny-by-default identity and policy boundary.
Production identity provider, MFA, tenant policy, and browser acceptance remain
Edition/deployment gates.
