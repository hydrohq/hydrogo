# ADR-0044: Sanitized content contract for example themes

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0012](0012-content-media-and-portability.md), [ADR-0018](0018-filesystem-theme-bundles.md)

## Context

`extension.PageViewData.HTMLContent` is derived by the Core rich-text
allowlist sanitizer before it reaches a theme. Example themes were escaping
that field as ordinary text, which both broke formatting and obscured the
typed safe-content boundary. At the same time, theme chrome fields such as
titles and site names remain untrusted strings.

## Decision

Example themes render `HTMLContent` as trusted HTML only at the content slot,
following the extension contract that Core supplies already-sanitized output.
Locale, title, site name, copyright, and nonce values continue to use context-
appropriate HTML escaping. Themes must not accept arbitrary unsanitized input
as `HTMLContent`; custom editions remain responsible for honoring this
contract.

## Consequences

- Rich text and structured formatting render correctly in generated editions.
- The XSS boundary remains in the Core sanitizer rather than being duplicated
  inconsistently by every theme.
- Tests verify escaped chrome and preserved sanitized markup in both examples.
