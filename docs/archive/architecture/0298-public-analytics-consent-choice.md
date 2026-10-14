# ADR-0298: Public analytics consent choice

- Status: Accepted
- Date: 2026-09-29
- Related: [ADR-0125](0125-versioned-analytics-consent-contract.md), [ADR-0131](0131-durable-analytics-consent-store.md), [ADR-0217](0217-consent-gated-content-view-analytics.md), [ADR-0269](0269-optional-no-cookie-visitor-analytics.md)

## Context

Consent-backed content view collection requires a clear public choice. The
existing consent endpoint persists an explicit grant, rejection, or revocation,
but public page and news HTML did not offer that choice.

## Decision

When the Analytics module is active and no consent decision cookie exists,
public page/FAQ and news article HTML includes the typed Design System consent
notice. Accept and decline submit the existing `granted` and `rejected` values
with the request CSRF token. The response returns to the originating local path
after persisting the decision. Error responses keep a generic user-facing
message. Consent-dependent HTML sends `Vary: Cookie`; this prevents a static
HTML cache from reusing the notice or its absence across consent states.

The notice does not introduce script tracking, an additional identifier, a
schema, worker, or migration. Content-view events continue to require a current
granted purpose in the analytics store. Optional no-cookie aggregates remain
governed by ADR-0269 and their explicit installation configuration.

Admin reports support a custom inclusive date range in addition to the
existing seven- and thirty-day presets. The range is parsed as ISO dates,
rejects future or reversed dates, and is limited to one year before it reaches
the query layer. Custom filters use the existing indexed date projection.

## Operations and verification

The feature is inert when Analytics is inactive. It adds no memory-resident
state and relies on the existing consent store and request CSRF middleware.
SQLite analytics tests verify that recording requires current consent; page and
news tests verify the notice is shown before a decision, hidden after a
decision, and responses vary on cookies. Return destinations reject external
origins and protocol-relative paths. Date-range tests cover valid, reversed,
malformed, future, and over-one-year inputs. No migration or rollback data step
is required.
