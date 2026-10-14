# ADR-0078: Publish a versioned frontend route and selector contract

## Status

Accepted

## Decision

Represent frontend routes and stable DOM selectors in a versioned manifest with
deterministic validation. The manifest is descriptive and reviewable; the
server router remains the source of truth for authorization and CSRF. Separate
frontend deployments must negotiate a compatible manifest version.

## Consequences

UI behavior can be developed against named selectors instead of brittle markup
guessing, while route security remains centralized. Adding a breaking selector
or route change requires a manifest version decision and migration notes.
