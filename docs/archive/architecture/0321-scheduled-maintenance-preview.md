# ADR-0321: Scheduled Maintenance and Secret Preview

- **Status:** Accepted
- **Related decisions:** ADR-0318 (partially superseded), ADR-0013

## Decision

The existing private tenant-scoped maintenance setting may include an optional
UTC start/end window, visitor headline/message, and a preview-key SHA-256
digest. Maintenance is active only when enabled and current UTC time is inside
the configured half-open interval `[start, end)`. Empty bounds preserve the
existing immediate/manual behavior. The middleware evaluates the setting per
request, so no scheduler worker or migration is needed.

Preview keys are 16–128 characters. The raw value is accepted once through the
admin form, never persisted, and compared to the digest in constant time. A
valid `?preview_key=` request receives an HttpOnly, SameSite=Strict cookie with
a four-hour expiry and a redirect that removes the key from the URL. Secure is
set according to the server cookie policy. Updating/removing the key invalidates
old preview cookies. Active admins/editors and the existing IP allowlist retain
their prior bypass behavior.

## Compatibility and recovery

This adds optional fields to the existing private JSON setting and is backward
compatible with old settings. SQLite, PostgreSQL, and MySQL physical schemas
are unchanged. No new background worker or mandatory resource is introduced.
The site Admin Studio owns editing; the scheduled state naturally expires even
if the process is restarted. An operator can disable maintenance immediately
through the same editor.

## Verification

Tests cover validation, schedule boundaries, preview-key hashing, query removal,
cookie attributes, and normal middleware behavior. The generated Templ UI and
full migration compatibility checks remain part of release verification.
