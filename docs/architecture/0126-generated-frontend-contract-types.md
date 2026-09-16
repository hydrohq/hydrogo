# ADR-0126: Generated frontend contract types

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0020](0020-headless-first-admin-studio-and-frontend-templates.md), [ADR-0028](0028-generated-frontend-adapters.md)

## Decision

Every generated Client Edition includes `frontend/contracts.ts`, a dependency-
free TypeScript contract for forms, editorial workflows, search, capability
kits and analytics consent. It is metadata/type scaffolding only: authentication,
authorization, persistence, provider adapters and design-system implementation
remain in the Client Edition. The generator must not add a runtime dependency to
Core or assume a particular frontend framework.

## Evidence and consequences

`internal/projectgen/projectgen_test.go` verifies the file is emitted for every
project. Consumers can import the types from React, HTMX, vanilla or another
frontend stack without installing Node at runtime.
