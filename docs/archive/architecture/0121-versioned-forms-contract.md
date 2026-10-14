# ADR-0121: Versioned forms contract

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0020](0020-headless-first-admin-studio-and-frontend-templates.md)

## Decision

Form definitions and bounded submission normalization are exposed through
`extension/v1`. The contract supports common field types, required values and
select options without prescribing persistence, inbox, delivery, consent or
retention behavior. Those concerns remain module/Client Edition policy and do
not become hidden Core dependencies.

## Evidence and consequences

`extension/v1/forms_test.go` covers normalization, required fields, option
validation and deterministic field ordering. A durable inbox and admin form
builder still require a module-owned schema, migration and privacy decision.
