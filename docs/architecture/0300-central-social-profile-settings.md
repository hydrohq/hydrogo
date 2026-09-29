# ADR-0300: Central Social Profile Settings

- Status: Accepted
- Date: 2026-09-29

## Context

Editions need a small set of public brand profile links without adding a social
network integration, worker, or customer-specific route. Public URLs must be
validated before they reach rendered pages.

## Decision

Store the seven supported social profile URLs in the existing tenant-scoped
`site_settings` record as one public JSON setting. The site module owns
validation and the Design System owns the typed Admin Studio form and public
links component. Each platform accepts HTTPS URLs from its own approved host
list. Empty entries are omitted. Public links use a generic accessible SVG
link mark and `noopener noreferrer`.

No new table, migration, worker, or dependency is introduced. Inactive site
module composition does not register the editor route or load settings.

## Compatibility and operations

The setting is additive and ignored by older versions. SQLite, PostgreSQL, and
MySQL use the existing `site_settings` contract unchanged. Rollback consists of
removing the setting through Admin Studio or deleting its row through the
existing site settings lifecycle. The setting is tenant-scoped and audited.

## Consequences

The public renderer reads at most one small setting and fails closed on invalid
stored JSON or URLs. Social links are rendered on page and news views when the
site module is active. This configuration does not fetch remote profile data.
