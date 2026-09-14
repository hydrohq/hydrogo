# Portable content format

Portable export/import is an allowlisted content operation. It includes pages,
page revisions, media metadata, site settings, navigation, redirects, form
inbox records, and news items. Locale, slug, revision, and relation fields are
preserved; authentication users, password hashes, sessions, audit events,
outbox events, and job leases are excluded.

The format is versioned (`format_version: 1`), rejects unknown fields and
trailing JSON, applies a byte limit, and inserts parent pages before revisions.
SQLite round trips are covered locally. PostgreSQL round trips require the
real service gate before being called certified.
