# ADR-0242: Resource auth is an explicit composition capability

- **Status:** Accepted
- **Date:** 2026-09-22
- **Scope:** Blueprint `auth` metadata and generated resource manifests
- **Related:** ADR-0013, ADR-0210, ADR-0240

## Decision

A Blueprint may set `auth: true` to declare that the generated resource is
intended for an authenticated Identity/Session/RBAC surface. The flag is
copied into generated `Spec` and `module.yaml` metadata and adds a review item
to `RUNTIME_WIRING.md`.

The flag never creates a second authentication system, chooses credentials,
installs middleware, or grants permissions. The Edition composition root must
attach HydroGo's existing auth/session/policy stack and prove anonymous and
authenticated behavior before enabling the route. Generated policy remains
deny-by-default.

## Consequences

- Blueprint intent is machine-readable and reviewable without weakening the
  Core/Edition boundary.
- Payload-like “auth enabled by declaration” ergonomics are available while
  the security-sensitive identity implementation stays centralized.
- Existing Blueprints default to `auth: false` and remain source-compatible.

## Verification

Loader tests cover the mapping, generated manifests carry the flag, and the
runtime wiring checklist explicitly requires authenticated policy tests.
