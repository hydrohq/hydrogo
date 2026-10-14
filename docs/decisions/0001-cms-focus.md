# 0001: Refocus HydroGo as a CMS

- Status: Accepted
- Date: 2026-10-14

## Context

In its first six weeks HydroGo grew into a "Go workspace, code generator and
product engine": Client Edition scaffolding, a vendor license system, signed
extension packages, private runtime distribution, GraphQL, commerce, twelve
sectoral content modules and about 390 ADRs plus 480 validation documents.
Most of it was written by coding agents running in long unattended loops.

The result compiled and had good test coverage, but nobody could hold it in
their head, the admin UI quality stayed poor, and agents kept choosing between
four overlapping ways to define a content type (hand-written modules, blueprint
codegen, `internal/collections`, `internal/adminstudio`).

## Decision

HydroGo is a **single-binary CMS**: pages, news, media, members, themes, an
admin panel and a REST API. Anything that does not serve that sentence is
removed or frozen.

Removed in this refocus (still available in git history):

- Client Edition scaffolding (`internal/projectgen`, `hydro new/project/check/frontend`,
  the edition-aware `hydro dev`, `examples/*-edition`)
- Runtime licensing (`internal/licensing`, `cmd/hydro-issuer`, the serve gate)
- Signed extension packages (`internal/extensionpkg`, `hydro extension`)
- Private runtime distribution parts of `internal/sdkdist` (`hydro sdk generate` stays)
- GraphQL (`modules/api/graphql`); REST + OpenAPI is the headless surface
- Unwired modules: commerce, social_sync, documents and the twelve sectoral
  modules (`internal/sectoral`); their migrations stay so existing databases
  still start
- The release-gate evidence manifest and its tool

Kept on purpose although they looked like candidates: `internal/tenant`
(scope plumbing runs through every store), `internal/analytics`,
`internal/realtime` (editor presence), `internal/outbox` (pages requires it),
`internal/ai`, `internal/push`, `internal/geolocation` (small, optional,
off by default).

Historical plans, ADRs and validation reports moved to `docs/archive/`. They
describe how the code got here; they are not instructions.

## Next steps

1. One content-type schema and one generic admin renderer: `adminstudio.Collection`
   as the schema, `GeneratedAdminHandler` as the renderer, `internal/collections`
   tables as storage. Small content types (FAQ, team, …) come back as schemas.
2. One admin design system, with screenshots in the agent feedback loop.
3. Split `cmd/hydro/main.go`.

## Consequences

- New decision records go in `docs/decisions/` as `NNNN-short-title.md`, and
  only for changes to a module boundary, a public contract or a decision
  recorded here. Bug fixes and refactors inside a module need no record.
