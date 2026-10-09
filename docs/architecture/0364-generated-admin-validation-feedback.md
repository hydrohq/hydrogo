# ADR-0364: Generated Admin validation feedback

- **Status:** Accepted
- **Date:** 2026-10-09
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0346](0346-blueprint-admin-studio-workflows.md), [ADR-0361](0361-generated-admin-resource-capabilities.md), [ADR-0362](0362-generated-admin-read-only-record-view.md)

## Context

Generated CRUD forms replaced structured API validation failures with a generic
error response. Editors lost submitted values and had no direct path to fields
that needed correction. API error text is not a safe customer-facing contract.

## Decision

- A generated Admin API `422` response with code `validation_failed` is
  rendered back into the generated form with an accessible validation summary
  and links to registered editable fields.
- Only known field IDs are linked. The `required` code gets specific guidance;
  other codes receive generic field guidance. Unknown fields and API-provided
  messages are ignored.
- Submitted values are parsed through the existing field contract and overlaid
  on the current record. Update failures reload the current record so untouched
  fields remain populated. Submitted version tokens are retained.
- Summary copy uses embedded Turkish and English catalogs. The typed
  `ValidationSummary` component owns escaping and alert semantics; links target
  the generated control, rich-text editor, block-editor mount, media field, or
  array editor.
- Other API failures keep their existing safe behavior. No route, authorization
  policy, API contract, database schema, migration, cache, worker, or service is
  added.

## Alternatives

- Display the API message directly: rejected because it can contain internal
  details and is not a stable UI contract.
- Keep the generic error page: rejected because it discards editor input and
  leaves fields unclear.
- Add client-side validation state: rejected because server validation remains
  authoritative and generated forms require no browser runtime.

## Consequences

### Benefits

- Editors retain submitted values and can navigate from an accessible summary
  to the fields that need attention.
- Unknown API details are kept out of customer-facing HTML.

### Costs and risks

- Field-level feedback depends on the Edition API's structured validation
  shape. Unknown response formats intentionally receive generic guidance.
- HTTP tests do not replace real browser visual and assistive-technology
  verification; that remains an open Admin Studio audit gate.

## Scale impact

- **Compact:** rendering is bounded by the collection's registered field count;
  no service, worker, cache, or browser runtime is added.
- **Standard:** an update validation response may trigger one existing API GET
  to restore untouched values.
- **Distributed:** the GET uses the request-scoped API and authorization
  context already used by generated record editing.

## Database impact

- **SQLite:** no query, schema, or migration change.
- **PostgreSQL:** no query, schema, or migration change.
- **Future adapters:** no adapter-specific behavior is introduced.

No migration metadata is required because this decision changes neither stored
data nor migration behavior. No cache key, TTL, invalidation, or fallback is
added.

## Deployment and compatibility

This is additive rendering behavior for an existing API validation response.
Old and new binaries share the same routes and API shape. Rollback restores the
previous generic 400 response; no persistent state needs recovery.

## Validation

- Typed component tests check escaping, alert semantics, and field links.
- Generated Admin HTTP tests check Turkish and English responses, HTTP 422,
  field targets, input preservation, ignored unknown fields, update reload, and
  absence of API message leakage.
- Full Go tests, `go vet`, required repository guards, browser accessibility,
  and the external SQLite/PostgreSQL and Chromium gates are recorded separately.

## References

- [Admin Studio product audit and plan](../planning/ADMIN_STUDIO_PRODUCT_AUDIT_AND_PLAN.md)
- [Generated Admin validation feedback](../validation/generated-admin-validation-feedback-2026-10-09.md)
