# ADR-0305: Staff Invitation Share Message

- Status: Accepted
- Date: 2026-09-29

## Context

Administrators need a quick way to share onboarding details when inviting
staff. Reusable or temporary passwords must not be exposed because invitations
already use one-time, expiring credentials.

## Decision

After the existing invitation email is sent, the protected invitation endpoint
returns the recipient address and a one-time invitation URL in a `no-store`
JSON response. Admin Studio presents an accessible dialog containing a
preformatted onboarding message and a copy action. The message is held in the
dialog only and is cleared whenever the dialog closes.

## Compatibility and operations

No schema, migration, or worker is added. The public origin must be configured
for this action. Existing invitation email and acceptance flows remain intact;
the invitation retains its current one-day expiry and single-use behavior.
