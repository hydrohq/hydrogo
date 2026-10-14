# ADR-0312: Global email branding and legal footer

## Status

Accepted

## Decision

An administrator can configure a global company name, physical address, privacy
policy link, custom legal notice, footer text, logo asset, and accent color from
the Mailer settings page. Settings are stored as one singleton row shared by
all tenants and instances. A sender decorator reads that row at delivery time
and applies the current values to both text and HTML email bodies, including
messages already queued for delivery. Existing spool records need no rewrite.

The decorator is enabled only when Auth is active, and the page is exposed only
when the existing Mailer transport and Auth are active. It adds no worker,
service, cache, or dependency. Logo references use ready media asset IDs and
the configured public origin. URL, asset ID, color, and field lengths are
validated before storing; all inserted HTML text is escaped.

SQLite, PostgreSQL, and MySQL use the same singleton/upsert contract. The
additive migration is selected with Auth. Rollback drops the settings row and
returns to the built-in email presentation; already delivered email is
unaffected.

## Validation

Store tests cover persistence and invalid settings. Sender tests cover text and
HTML decoration, escaping, and settings read failures. The Admin Studio form is
typed Templ with CSRF and role checks; migration guard checks dialect
checksums.
