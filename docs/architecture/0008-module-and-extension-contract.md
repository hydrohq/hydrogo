# ADR-0008: Core, module and customer extension boundary

- **Status:** Superseded
- **Date:** 2026-09-05
- **Superseded by:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md)

> Historical decision: The wide interface below is not a stable SDK. Current packaging, v0/v1 maturity, module lifecycle, and presentation definitions are in ADR-0010. This text is preserved for the decision history.

## Context

To increase HydroGo's development speed and allow customer projects to receive Core updates, it must be definitive which code is a stable contract and which code is an internal implementation detail.

## Decision

The code is divided into three visibility levels:

```text
internal/     HydroGo internal implementation; no compatibility guarantee for the customer
extension/    Versioned customer and module SDK
client/       Customer-specific edition code
```

The Client Edition can only use `extension` contracts and public module APIs. CI rejects forbidden internal imports.

## Module contract

A module declares at least the following:

```go
type Module interface {
    ID() string
    Version() Version
    Requires() Requirements
    Capabilities() []Capability
    Routes() []Route
    Permissions() []Permission
    Migrations(database string) []Migration
    Admin() AdminContribution
    Jobs() []JobDefinition
    CachePolicies() []CachePolicy
}
```

- The module dependency graph is acyclic.
- An inactive module does not load any routes, migrations, assets, or workers.
- Disabling a module does not automatically delete data.
- Uninstalling a module requires an explicit data retention/deletion plan.
- The required DB/cache/storage capabilities are verified at startup.

## Extension formats

Order of preference:

1. Configuration and theme tokens
2. Versioned hook/event contracts
3. Component/view slot composition
4. Customer-specific application service or route
5. Full view override

HydroCore fork is not a supported extension format.

## Code generation

- `generated/` is completely reproducible and is not edited manually.
- `custom/` is human code and is not written by the generator.
- The generator creates standard CRUD, migration skeletons, admin forms, API, and test scaffolding.
- Business rules, authorization decisions, and destructive migrations are not automatically guessed.

## Presentation modes

The Edition explicitly chooses one of the following operating modes:

- `integrated`: Admin + Templ public site
- `headless`: Admin + API
- `hybrid`: Templ and API together

For the first commercial Site Edition, `integrated` is the default.

## Compatibility

- Published extension contracts in the same major version are preserved.
- Experimental APIs carry a separate namespace and an explicit label.
- A breaking change requires a new major version, a deprecation period, and a migration guide.
- A Core release is not published without passing the supported Client Edition contract tests.
