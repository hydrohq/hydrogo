# ADR-0098: Provider-neutral form contract

**Status:** Accepted  
**Date:** 2026-09-16

## Context

Payload-style rapid prototyping needs reusable form definitions, but storage,
email delivery, consent, and inbox retention are product and module concerns.

## Decision

Add a bounded `internal/forms` contract for form metadata, field types,
select options, and submission normalization. Unknown fields, oversized values,
duplicate IDs/options, unsupported types, and missing required values fail
closed. Normalization never logs or persists user data.

## Consequences

Modules can render and process forms consistently without a Core-owned inbox or
provider dependency. Consent, retention, delivery, spam controls, and database
migrations remain explicit module/product decisions.

## Verification

`go test ./internal/forms -count=1` covers valid submission, required/select
validation, duplicate metadata, and deterministic field ordering.
