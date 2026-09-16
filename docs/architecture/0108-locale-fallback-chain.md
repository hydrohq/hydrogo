# ADR-0108: Deterministic bounded locale fallback chain

**Status:** Accepted  
**Date:** 2026-09-16

## Context

Public and admin surfaces normalize locales independently, and localized
content needs a predictable fallback without exposing unavailable translations
or probing unbounded locale lists.

## Decision

`internal/i18n.LocaleFallbackChain` accepts a requested locale, an allowlisted
supported set, and a default locale. It returns exact locale, primary-language
fallback, then default only when supported, with a 128-locale bound, duplicate
rejection, normalization, and deterministic ordering.

## Consequences

Modules can use one provider-neutral fallback policy for public rendering,
preview, and admin selectors. Missing translations remain explicit; the Core
does not invent content or select a storage provider.

## Verification

`go test ./internal/i18n -count=1` covers priority, unsupported locales,
duplicates, and unsafe input.
