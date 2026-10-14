# ADR-0023: Metadata-driven Admin and Frontend E2E

**Status:** Proposed

## Context

HydroGo's Admin Studio surface is defined by collection, field, action, and widget metadata. The headless frontend operates via a versioned API client. Independent, hand-written E2E tests for these surfaces can detach from panel and frontend changes over time. Raw HTML crawling produces brittle selectors, authorization bypasses, and false positives.

## Proposed decision

E2E test scenarios will be generated from two explicit metadata sources:

1. Admin Studio `Registry`: collection, field, widget, readonly/required information, action, and route ownership.
2. Frontend E2E manifest: route, visible page purpose, access level, stable `data-testid` values, and expected API projection.

`hydro e2e generate` or an equivalent build tool inspects this metadata and generates deterministic Playwright tests. The generated output is kept in a separate `generated/` directory; customer/custom flows are located under `custom/` and preserved during regeneration.

The generator only creates safe and defined scenarios:

- public page load and published content visibility;
- admin login/session and authorized page load;
- required/readonly/widget validations from metadata;
- form validation and optimistic-lock conflict flow;
- explicitly declared actions like Pages/News publish, schedule, and preview.

Write scenarios only run in an isolated seed database, with idempotent fixtures. Real production, real payment, real email, or external webhook calls are not included in the E2E scope. An unknown route, selector, action, or permission stops generation in a fail-closed manner.

## Boundaries

- HTML/CSS classes or text content are not used as automatic selectors; the application provides a stable `data-testid` or accessible role/label contract.
- The Admin registry does not carry direct DB access or authorization decisions.
- The Frontend manifest does not expand the API contract; it only describes the test intent and the surface to be observed.
- The browser runner and browser binary are not added to the Compact production binary; they remain only as a development/CI test tool.
- Generated tests do not replace a security proof; HTTP/service contract tests are preserved.

## Acceptance criteria

- Generating the same registry and manifest twice produces zero diff.
- Generation of unknown or unauthorized metadata is rejected fail-closed.
- The generated happy path and validation/conflict path work for Pages and News.
- The headless starter can run the generated public read test without importing Core internals.
- Actual browser execution is reported separately; if there is no browser setup, the test is not reported as "skipped", but as an open environment gate.

## Conclusion

This approach generates tests from the actual contracts of the Admin Studio and headless frontend, but does not introduce brittleness to the system with random HTML discovery or unbounded automation. Pre-implementation selector, manifest format, runner version, and fixture isolation must be additionally validated.
